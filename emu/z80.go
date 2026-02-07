package emu

import (
	"github.com/koron-go/z80"
)

// CycleZ80 wraps the koron-go/z80 CPU and provides accurate cycle counting.
type CycleZ80 struct {
	cpu          *z80.CPU
	mem          *Memory
	afterEI      bool           // True if we just executed EI (interrupt delay)
	cachedIM1Int *z80.Interrupt // Cached IM1 interrupt to avoid per-call allocation
}

// NewCycleZ80 creates a new cycle-counting Z80 wrapper.
func NewCycleZ80(mem *Memory, io z80.IO) *CycleZ80 {
	return &CycleZ80{
		cpu: &z80.CPU{
			Memory: mem,
			IO:     io,
		},
		mem:          mem,
		cachedIM1Int: z80.IM1Interrupt(), // Cache once at creation
	}
}

// SetInterrupt sets a pending interrupt on the CPU.
func (c *CycleZ80) SetInterrupt(interrupt *z80.Interrupt) {
	c.cpu.Interrupt = interrupt
}

// SetIM1Interrupt sets the cached IM1 interrupt on the CPU.
// This avoids allocating a new Interrupt object on each call.
func (c *CycleZ80) SetIM1Interrupt() {
	c.cpu.Interrupt = c.cachedIM1Int
}

// ClearInterrupt clears any pending interrupt on the CPU.
// This is used to emulate the level-triggered nature of the SMS VDP interrupt.
func (c *CycleZ80) ClearInterrupt() {
	c.cpu.Interrupt = nil
}

// GetPC returns the current program counter.
func (c *CycleZ80) GetPC() uint16 {
	return c.cpu.PC
}

// GetIFF1 returns the interrupt flip-flop 1 state.
func (c *CycleZ80) GetIFF1() bool {
	return c.cpu.IFF1
}

// GetIM returns the interrupt mode.
func (c *CycleZ80) GetIM() int {
	return c.cpu.IM
}

// TriggerNMI triggers a non-maskable interrupt.
// On the SMS, this is connected to the Pause button.
// The NMI pushes PC to stack and jumps to 0x0066.
// Returns 11 T-states consumed by the NMI response.
func (c *CycleZ80) TriggerNMI() int {
	// NMI is edge-triggered and non-maskable
	// It disables interrupts (IFF1=0) and jumps to 0x0066
	c.cpu.IFF1 = false
	// Push PC to stack
	c.cpu.SP--
	c.mem.Set(c.cpu.SP, uint8(c.cpu.PC>>8))
	c.cpu.SP--
	c.mem.Set(c.cpu.SP, uint8(c.cpu.PC&0xFF))
	// Jump to NMI vector
	c.cpu.PC = 0x0066
	// Wake from HALT if halted
	c.cpu.HALT = false
	// NMI response takes 11 T-states on Z80
	return 11
}

// Step executes one instruction and returns the number of T-states (cycles) consumed.
func (c *CycleZ80) Step() int {
	// ==========================================================================
	// WORKAROUND: koron-go/z80 library missing EI instruction delay
	// ==========================================================================
	//
	// Per the Zilog Z80 CPU User Manual (UM0080, page 175):
	//   "When an EI instruction is executed, any pending interrupt request
	//    is not accepted until after the instruction following EI is executed."
	//
	// This one-instruction delay is critical for code patterns like:
	//     EI        ; Enable interrupts
	//     HALT      ; Wait for interrupt
	//
	// Without the delay, if an interrupt is pending when EI executes, the CPU
	// would service it immediately BEFORE executing HALT. The program would then
	// need TWO interrupts to proceed past HALT (one serviced after EI, another
	// to wake from HALT). This causes games like Fantastic Dizzy to run at half
	// speed during sections that use EI;HALT for frame synchronization.
	//
	// The koron-go/z80 library sets IFF1=true immediately on EI without any
	// delay, which is incorrect per Z80 specifications. Since we cannot modify
	// the library, we work around this by temporarily hiding any pending
	// interrupt during the instruction immediately following EI.
	//
	// Note: This workaround only affects maskable interrupts (INT). NMI is
	// handled separately via TriggerNMI() and is not affected by EI delay.
	// ==========================================================================
	var savedInterrupt *z80.Interrupt
	if c.afterEI && c.cpu.Interrupt != nil {
		savedInterrupt = c.cpu.Interrupt
		c.cpu.Interrupt = nil
	}
	c.afterEI = false

	// Check for pending interrupt (after EI delay handling)
	if c.cpu.Interrupt != nil {
		// Wake from HALT if halted - HALT exits on any interrupt signal
		if c.cpu.HALT {
			c.cpu.HALT = false
			c.cpu.PC++ // Advance past HALT instruction
		}

		// Only service the interrupt if IFF1 is set
		if c.cpu.IFF1 {
			c.cpu.Step()
			return 13 // IM1 interrupt response cycles
		}
		// IFF1=0: interrupt woke HALT but isn't serviced yet
		// Fall through to execute the next instruction
	}

	// If halted and no interrupt pending, just burn cycles
	if c.cpu.HALT {
		return 4 // HALT executes NOPs internally
	}

	// Peek at opcode before stepping
	pc := c.cpu.PC
	opcode := c.mem.Get(pc)

	// Determine base cycles from opcode
	var cycles int
	switch opcode {
	case 0xCB:
		op2 := c.mem.Get(pc + 1)
		cycles = cbCycles[op2]
	case 0xDD:
		op2 := c.mem.Get(pc + 1)
		if op2 == 0xCB {
			// DD CB d op - indexed bit operations, 23 cycles (20 for BIT)
			op4 := c.mem.Get(pc + 3)
			if op4 >= 0x40 && op4 <= 0x7F {
				cycles = 20 // BIT instructions
			} else {
				cycles = 23 // SET/RES instructions
			}
		} else {
			cycles = ddCycles[op2]
		}
	case 0xED:
		op2 := c.mem.Get(pc + 1)
		cycles = edCycles[op2]
	case 0xFD:
		op2 := c.mem.Get(pc + 1)
		if op2 == 0xCB {
			// FD CB d op - indexed bit operations
			op4 := c.mem.Get(pc + 3)
			if op4 >= 0x40 && op4 <= 0x7F {
				cycles = 20 // BIT instructions
			} else {
				cycles = 23 // SET/RES instructions
			}
		} else {
			cycles = fdCycles[op2]
		}
	default:
		cycles = baseCycles[opcode]
	}

	// Execute the instruction
	c.cpu.Step()

	// Set EI delay flag when EI (0xFB) is executed
	if opcode == 0xFB {
		c.afterEI = true
	}

	// Adjust for conditional instructions
	cycles = c.adjustConditional(opcode, pc, cycles)

	// Restore the interrupt that was hidden during EI delay
	// It will be serviced on the next Step() call
	if savedInterrupt != nil {
		c.cpu.Interrupt = savedInterrupt
	}

	return cycles
}

// adjustConditional adjusts cycle count for conditional instructions
// based on whether the condition was taken.
func (c *CycleZ80) adjustConditional(opcode uint8, pcBefore uint16, cycles int) int {
	pcAfter := c.cpu.PC

	switch opcode {
	// JR cc,d - 12 cycles if taken, 7 if not
	case 0x20, 0x28, 0x30, 0x38:
		if pcAfter == pcBefore+2 {
			return 7 // Not taken
		}
		return 12 // Taken

	// RET cc - 11 cycles if taken, 5 if not
	case 0xC0, 0xC8, 0xD0, 0xD8, 0xE0, 0xE8, 0xF0, 0xF8:
		if pcAfter == pcBefore+1 {
			return 5 // Not taken
		}
		return 11 // Taken

	// JP cc,nn - always 10 cycles
	case 0xC2, 0xCA, 0xD2, 0xDA, 0xE2, 0xEA, 0xF2, 0xFA:
		return 10

	// CALL cc,nn - 17 cycles if taken, 10 if not
	case 0xC4, 0xCC, 0xD4, 0xDC, 0xE4, 0xEC, 0xF4, 0xFC:
		if pcAfter == pcBefore+3 {
			return 10 // Not taken
		}
		return 17 // Taken

	// DJNZ - 13 cycles if taken, 8 if not
	case 0x10:
		if pcAfter == pcBefore+2 {
			return 8 // Not taken (B became 0)
		}
		return 13 // Taken

	// ED-prefixed block instructions
	case 0xED:
		op2 := c.mem.Get(pcBefore + 1)
		// Block repeat instructions: LDIR (B0), CPIR (B1), INIR (B2), OTIR (B3),
		// LDDR (B8), CPDR (B9), INDR (BA), OTDR (BB)
		switch op2 {
		case 0xB0, 0xB1, 0xB2, 0xB3, 0xB8, 0xB9, 0xBA, 0xBB:
			if pcAfter == pcBefore {
				return 21 // Repeating (PC unchanged, will execute again)
			}
			return 16 // Completed (PC advanced past instruction)
		}
	}

	return cycles
}

// Base opcode cycles (no prefix)
var baseCycles = [256]int{
	//  0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	4, 10, 7, 6, 4, 4, 7, 4, 4, 11, 7, 6, 4, 4, 7, 4, // 0x
	8, 10, 7, 6, 4, 4, 7, 4, 12, 11, 7, 6, 4, 4, 7, 4, // 1x (DJNZ/JR use conditional)
	7, 10, 16, 6, 4, 4, 7, 4, 7, 11, 16, 6, 4, 4, 7, 4, // 2x (JR cc use conditional)
	7, 10, 13, 6, 11, 11, 10, 4, 7, 11, 13, 6, 4, 4, 7, 4, // 3x
	4, 4, 4, 4, 4, 4, 7, 4, 4, 4, 4, 4, 4, 4, 7, 4, // 4x
	4, 4, 4, 4, 4, 4, 7, 4, 4, 4, 4, 4, 4, 4, 7, 4, // 5x
	4, 4, 4, 4, 4, 4, 7, 4, 4, 4, 4, 4, 4, 4, 7, 4, // 6x
	7, 7, 7, 7, 7, 7, 4, 7, 4, 4, 4, 4, 4, 4, 7, 4, // 7x (HALT is 4)
	4, 4, 4, 4, 4, 4, 7, 4, 4, 4, 4, 4, 4, 4, 7, 4, // 8x
	4, 4, 4, 4, 4, 4, 7, 4, 4, 4, 4, 4, 4, 4, 7, 4, // 9x
	4, 4, 4, 4, 4, 4, 7, 4, 4, 4, 4, 4, 4, 4, 7, 4, // Ax
	4, 4, 4, 4, 4, 4, 7, 4, 4, 4, 4, 4, 4, 4, 7, 4, // Bx
	5, 10, 10, 10, 10, 11, 7, 11, 5, 10, 10, 0, 10, 17, 7, 11, // Cx (RET cc/CALL cc conditional)
	5, 10, 10, 11, 10, 11, 7, 11, 5, 4, 10, 11, 10, 0, 7, 11, // Dx
	5, 10, 10, 19, 10, 11, 7, 11, 5, 4, 10, 4, 10, 0, 7, 11, // Ex
	5, 10, 10, 4, 10, 11, 7, 11, 5, 6, 10, 4, 10, 0, 7, 11, // Fx
}

// CB prefix cycles (bit operations)
var cbCycles = [256]int{
	//  0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	8, 8, 8, 8, 8, 8, 15, 8, 8, 8, 8, 8, 8, 8, 15, 8, // 0x RLC
	8, 8, 8, 8, 8, 8, 15, 8, 8, 8, 8, 8, 8, 8, 15, 8, // 1x RRC
	8, 8, 8, 8, 8, 8, 15, 8, 8, 8, 8, 8, 8, 8, 15, 8, // 2x RL
	8, 8, 8, 8, 8, 8, 15, 8, 8, 8, 8, 8, 8, 8, 15, 8, // 3x RR
	8, 8, 8, 8, 8, 8, 15, 8, 8, 8, 8, 8, 8, 8, 15, 8, // 4x SLA
	8, 8, 8, 8, 8, 8, 15, 8, 8, 8, 8, 8, 8, 8, 15, 8, // 5x SRA
	8, 8, 8, 8, 8, 8, 15, 8, 8, 8, 8, 8, 8, 8, 15, 8, // 6x SLL
	8, 8, 8, 8, 8, 8, 15, 8, 8, 8, 8, 8, 8, 8, 15, 8, // 7x SRL
	8, 8, 8, 8, 8, 8, 12, 8, 8, 8, 8, 8, 8, 8, 12, 8, // 8x BIT 0 (BIT n,(HL) = 12)
	8, 8, 8, 8, 8, 8, 12, 8, 8, 8, 8, 8, 8, 8, 12, 8, // 9x BIT 1
	8, 8, 8, 8, 8, 8, 12, 8, 8, 8, 8, 8, 8, 8, 12, 8, // Ax BIT 2
	8, 8, 8, 8, 8, 8, 12, 8, 8, 8, 8, 8, 8, 8, 12, 8, // Bx BIT 3
	8, 8, 8, 8, 8, 8, 12, 8, 8, 8, 8, 8, 8, 8, 12, 8, // Cx BIT 4
	8, 8, 8, 8, 8, 8, 12, 8, 8, 8, 8, 8, 8, 8, 12, 8, // Dx BIT 5
	8, 8, 8, 8, 8, 8, 12, 8, 8, 8, 8, 8, 8, 8, 12, 8, // Ex BIT 6
	8, 8, 8, 8, 8, 8, 12, 8, 8, 8, 8, 8, 8, 8, 12, 8, // Fx BIT 7
}

// DD prefix cycles (IX register operations)
// Most are base cycles + 4, some have specific timings
var ddCycles = [256]int{
	//  0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	4, 4, 4, 4, 4, 4, 4, 4, 4, 15, 4, 4, 4, 4, 4, 4, // 0x
	4, 4, 4, 4, 4, 4, 4, 4, 4, 15, 4, 4, 4, 4, 4, 4, // 1x
	4, 14, 20, 10, 8, 8, 11, 4, 4, 15, 20, 10, 8, 8, 11, 4, // 2x
	4, 4, 4, 4, 23, 23, 19, 4, 4, 15, 4, 4, 4, 4, 4, 4, // 3x
	4, 4, 4, 4, 8, 8, 19, 4, 4, 4, 4, 4, 8, 8, 19, 4, // 4x
	4, 4, 4, 4, 8, 8, 19, 4, 4, 4, 4, 4, 8, 8, 19, 4, // 5x
	8, 8, 8, 8, 8, 8, 19, 8, 8, 8, 8, 8, 8, 8, 19, 8, // 6x
	19, 19, 19, 19, 19, 19, 4, 19, 4, 4, 4, 4, 8, 8, 19, 4, // 7x
	4, 4, 4, 4, 8, 8, 19, 4, 4, 4, 4, 4, 8, 8, 19, 4, // 8x
	4, 4, 4, 4, 8, 8, 19, 4, 4, 4, 4, 4, 8, 8, 19, 4, // 9x
	4, 4, 4, 4, 8, 8, 19, 4, 4, 4, 4, 4, 8, 8, 19, 4, // Ax
	4, 4, 4, 4, 8, 8, 19, 4, 4, 4, 4, 4, 8, 8, 19, 4, // Bx
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 0, 4, 4, 4, 4, // Cx (CB handled separately)
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // Dx
	4, 14, 4, 23, 4, 15, 4, 4, 4, 8, 4, 4, 4, 4, 4, 4, // Ex
	4, 4, 4, 4, 4, 4, 4, 4, 4, 10, 4, 4, 4, 4, 4, 4, // Fx
}

// FD prefix cycles (IY register operations) - same as DD
var fdCycles = [256]int{
	//  0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	4, 4, 4, 4, 4, 4, 4, 4, 4, 15, 4, 4, 4, 4, 4, 4, // 0x
	4, 4, 4, 4, 4, 4, 4, 4, 4, 15, 4, 4, 4, 4, 4, 4, // 1x
	4, 14, 20, 10, 8, 8, 11, 4, 4, 15, 20, 10, 8, 8, 11, 4, // 2x
	4, 4, 4, 4, 23, 23, 19, 4, 4, 15, 4, 4, 4, 4, 4, 4, // 3x
	4, 4, 4, 4, 8, 8, 19, 4, 4, 4, 4, 4, 8, 8, 19, 4, // 4x
	4, 4, 4, 4, 8, 8, 19, 4, 4, 4, 4, 4, 8, 8, 19, 4, // 5x
	8, 8, 8, 8, 8, 8, 19, 8, 8, 8, 8, 8, 8, 8, 19, 8, // 6x
	19, 19, 19, 19, 19, 19, 4, 19, 4, 4, 4, 4, 8, 8, 19, 4, // 7x
	4, 4, 4, 4, 8, 8, 19, 4, 4, 4, 4, 4, 8, 8, 19, 4, // 8x
	4, 4, 4, 4, 8, 8, 19, 4, 4, 4, 4, 4, 8, 8, 19, 4, // 9x
	4, 4, 4, 4, 8, 8, 19, 4, 4, 4, 4, 4, 8, 8, 19, 4, // Ax
	4, 4, 4, 4, 8, 8, 19, 4, 4, 4, 4, 4, 8, 8, 19, 4, // Bx
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 0, 4, 4, 4, 4, // Cx (CB handled separately)
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // Dx
	4, 14, 4, 23, 4, 15, 4, 4, 4, 8, 4, 4, 4, 4, 4, 4, // Ex
	4, 4, 4, 4, 4, 4, 4, 4, 4, 10, 4, 4, 4, 4, 4, 4, // Fx
}

// ED prefix cycles (extended operations)
var edCycles = [256]int{
	//  0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, // 0x (NOP equivalents)
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, // 1x (NOP equivalents)
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, // 2x (NOP equivalents)
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, // 3x (NOP equivalents)
	12, 12, 15, 20, 8, 14, 8, 9, 12, 12, 15, 20, 8, 14, 8, 9, // 4x
	12, 12, 15, 20, 8, 14, 8, 9, 12, 12, 15, 20, 8, 14, 8, 9, // 5x
	12, 12, 15, 20, 8, 14, 8, 18, 12, 12, 15, 20, 8, 14, 8, 18, // 6x
	12, 12, 15, 20, 8, 14, 8, 8, 12, 12, 15, 20, 8, 14, 8, 8, // 7x
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, // 8x (NOP equivalents)
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, // 9x (NOP equivalents)
	16, 16, 16, 16, 8, 8, 8, 8, 16, 16, 16, 16, 8, 8, 8, 8, // Ax (LDI, CPI, INI, OUTI, etc.)
	21, 21, 21, 21, 8, 8, 8, 8, 21, 21, 21, 21, 8, 8, 8, 8, // Bx (LDIR, CPIR, etc.) - 21 if repeat, 16 if done
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, // Cx (NOP equivalents)
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, // Dx (NOP equivalents)
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, // Ex (NOP equivalents)
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, // Fx (NOP equivalents)
}
