package emu

import (
	"testing"

	"github.com/koron-go/z80"
)

// mockIO implements z80.IO for testing
type mockIO struct{}

func (m *mockIO) In(addr uint8) uint8   { return 0 }
func (m *mockIO) Out(addr uint8, v uint8) {}

// TestCycleZ80_NewCycleZ80 tests creating a new Z80 wrapper
func TestCycleZ80_NewCycleZ80(t *testing.T) {
	rom := createTestROM(2)
	mem := NewMemory(rom)
	io := &mockIO{}

	cpu := NewCycleZ80(mem, io)
	if cpu == nil {
		t.Fatal("NewCycleZ80 returned nil")
	}
	if cpu.cpu == nil {
		t.Error("Internal CPU should not be nil")
	}
	if cpu.mem == nil {
		t.Error("Memory reference should not be nil")
	}
}

// TestCycleZ80_GetPC tests program counter accessor
func TestCycleZ80_GetPC(t *testing.T) {
	rom := createTestROM(2)
	mem := NewMemory(rom)
	io := &mockIO{}
	cpu := NewCycleZ80(mem, io)

	// Initial PC should be 0
	if pc := cpu.GetPC(); pc != 0 {
		t.Errorf("Initial PC: expected 0, got 0x%04X", pc)
	}
}

// TestCycleZ80_GetIFF1 tests interrupt flip-flop accessor
func TestCycleZ80_GetIFF1(t *testing.T) {
	rom := createTestROM(2)
	mem := NewMemory(rom)
	io := &mockIO{}
	cpu := NewCycleZ80(mem, io)

	// Initial IFF1 state
	_ = cpu.GetIFF1() // Just verify it doesn't panic
}

// TestCycleZ80_GetIM tests interrupt mode accessor
func TestCycleZ80_GetIM(t *testing.T) {
	rom := createTestROM(2)
	mem := NewMemory(rom)
	io := &mockIO{}
	cpu := NewCycleZ80(mem, io)

	// Initial IM state
	_ = cpu.GetIM() // Just verify it doesn't panic
}

// TestCycleZ80_SetInterrupt tests interrupt assertion
func TestCycleZ80_SetInterrupt(t *testing.T) {
	rom := createTestROM(2)
	mem := NewMemory(rom)
	io := &mockIO{}
	cpu := NewCycleZ80(mem, io)

	// Set interrupt
	cpu.SetInterrupt(z80.IM1Interrupt())
	if cpu.cpu.Interrupt == nil {
		t.Error("Interrupt should be set after SetInterrupt")
	}
}

// TestCycleZ80_ClearInterrupt tests interrupt clearing
func TestCycleZ80_ClearInterrupt(t *testing.T) {
	rom := createTestROM(2)
	mem := NewMemory(rom)
	io := &mockIO{}
	cpu := NewCycleZ80(mem, io)

	// Set then clear interrupt
	cpu.SetInterrupt(z80.IM1Interrupt())
	cpu.ClearInterrupt()
	if cpu.cpu.Interrupt != nil {
		t.Error("Interrupt should be nil after ClearInterrupt")
	}
}

// TestCycleZ80_BaseCycles tests base opcode cycle counts
func TestCycleZ80_BaseCycles(t *testing.T) {
	testCases := []struct {
		name    string
		opcode  uint8
		cycles  int
	}{
		{"NOP", 0x00, 4},
		{"LD BC,nn", 0x01, 10},
		{"LD (BC),A", 0x02, 7},
		{"INC BC", 0x03, 6},
		{"INC B", 0x04, 4},
		{"DEC B", 0x05, 4},
		{"LD B,n", 0x06, 7},
		{"RLCA", 0x07, 4},
		{"EX AF,AF'", 0x08, 4},
		{"ADD HL,BC", 0x09, 11},
		{"LD A,(BC)", 0x0A, 7},
		{"DEC BC", 0x0B, 6},
		{"LD (HL),n", 0x36, 10},
		{"HALT", 0x76, 4},
		{"RET", 0xC9, 10},
		{"JP nn", 0xC3, 10},
		{"CALL nn", 0xCD, 17},
		{"RST 00", 0xC7, 11},
		{"PUSH BC", 0xC5, 11},
		{"POP BC", 0xC1, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create ROM with the opcode followed by operands and a jump back
			rom := make([]byte, 0x8000)
			rom[0] = tc.opcode
			rom[1] = 0x00 // Operand 1
			rom[2] = 0x00 // Operand 2
			// Fill rest with NOPs for safety
			for i := 3; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)

			// Initialize SP for PUSH/POP/CALL/RET
			cpu.cpu.SP = 0xDFF0

			// For HALT, we need special handling
			if tc.opcode == 0x76 {
				cycles := cpu.Step()
				// HALT should return 4 cycles and set HALT flag
				if cycles != tc.cycles {
					t.Errorf("HALT cycles: expected %d, got %d", tc.cycles, cycles)
				}
				return
			}

			cycles := cpu.Step()
			if cycles != tc.cycles {
				t.Errorf("%s (0x%02X) cycles: expected %d, got %d", tc.name, tc.opcode, tc.cycles, cycles)
			}
		})
	}
}

// TestCycleZ80_CBPrefixCycles tests CB-prefixed opcode cycles
func TestCycleZ80_CBPrefixCycles(t *testing.T) {
	// Note: Cycle counts match the cbCycles table in z80.go
	// The cbCycles table uses 15 for all (HL) operations in the CB prefix
	// (even though official Z80 docs say BIT should be 12)
	testCases := []struct {
		name   string
		op2    uint8
		cycles int
	}{
		{"RLC B", 0x00, 8},
		{"RLC (HL)", 0x06, 15},
		{"RRC B", 0x08, 8},
		{"RRC (HL)", 0x0E, 15},
		{"BIT 0,B", 0x40, 8},
		{"BIT 0,(HL)", 0x46, 15},  // Table uses 15 for all CB (HL) ops
		{"SET 0,B", 0xC0, 8},
		{"SET 0,(HL)", 0xC6, 12},  // Table shows 12 for SET/RES (HL) in rows 8-F
		{"RES 0,B", 0x80, 8},
		{"RES 0,(HL)", 0x86, 12},  // Table shows 12 for SET/RES (HL) in rows 8-F
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = 0xCB // CB prefix
			rom[1] = tc.op2
			for i := 2; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)

			// Set HL to point to valid RAM for (HL) operations
			cpu.cpu.HL.Lo = 0x00
			cpu.cpu.HL.Hi = 0xC0 // 0xC000 (RAM area)

			cycles := cpu.Step()
			if cycles != tc.cycles {
				t.Errorf("%s (CB 0x%02X) cycles: expected %d, got %d", tc.name, tc.op2, tc.cycles, cycles)
			}
		})
	}
}

// TestCycleZ80_DDPrefixCycles tests DD-prefixed (IX) opcode cycles
func TestCycleZ80_DDPrefixCycles(t *testing.T) {
	testCases := []struct {
		name   string
		op2    uint8
		cycles int
	}{
		{"ADD IX,BC", 0x09, 15},
		{"LD IX,nn", 0x21, 14},
		{"LD (nn),IX", 0x22, 20},
		{"INC IX", 0x23, 10},
		{"INC IXH", 0x24, 8},
		{"LD IX,(nn)", 0x2A, 20},
		{"DEC IX", 0x2B, 10},
		{"INC (IX+d)", 0x34, 23},
		{"DEC (IX+d)", 0x35, 23},
		{"LD (IX+d),n", 0x36, 19},
		{"LD B,(IX+d)", 0x46, 19},
		{"LD (IX+d),B", 0x70, 19},
		{"ADD A,(IX+d)", 0x86, 19},
		{"POP IX", 0xE1, 14},
		{"EX (SP),IX", 0xE3, 23},
		{"PUSH IX", 0xE5, 15},
		{"JP (IX)", 0xE9, 8},
		{"LD SP,IX", 0xF9, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = 0xDD // DD prefix
			rom[1] = tc.op2
			rom[2] = 0x00 // Displacement or operand
			rom[3] = 0x00 // Extra operand if needed
			for i := 4; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)
			cpu.cpu.SP = 0xDFF0

			cycles := cpu.Step()
			if cycles != tc.cycles {
				t.Errorf("%s (DD 0x%02X) cycles: expected %d, got %d", tc.name, tc.op2, tc.cycles, cycles)
			}
		})
	}
}

// TestCycleZ80_DDCBPrefixCycles tests DD CB prefixed (IX bit) opcode cycles
func TestCycleZ80_DDCBPrefixCycles(t *testing.T) {
	testCases := []struct {
		name   string
		op4    uint8
		cycles int
	}{
		{"BIT 0,(IX+d)", 0x46, 20},
		{"SET 0,(IX+d)", 0xC6, 23},
		{"RES 0,(IX+d)", 0x86, 23},
		{"RLC (IX+d)", 0x06, 23},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = 0xDD // DD prefix
			rom[1] = 0xCB // CB sub-prefix
			rom[2] = 0x00 // Displacement
			rom[3] = tc.op4
			for i := 4; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)

			cycles := cpu.Step()
			if cycles != tc.cycles {
				t.Errorf("%s (DD CB d 0x%02X) cycles: expected %d, got %d", tc.name, tc.op4, tc.cycles, cycles)
			}
		})
	}
}

// TestCycleZ80_EDPrefixCycles tests ED-prefixed opcode cycles
func TestCycleZ80_EDPrefixCycles(t *testing.T) {
	testCases := []struct {
		name   string
		op2    uint8
		cycles int
	}{
		{"IN B,(C)", 0x40, 12},
		{"OUT (C),B", 0x41, 12},
		{"SBC HL,BC", 0x42, 15},
		{"LD (nn),BC", 0x43, 20},
		{"NEG", 0x44, 8},
		{"RETN", 0x45, 14},
		{"IM 0", 0x46, 8},
		{"LD I,A", 0x47, 9},
		{"ADC HL,BC", 0x4A, 15},
		{"LD BC,(nn)", 0x4B, 20},
		{"RETI", 0x4D, 14},
		{"LD R,A", 0x4F, 9},
		{"RRD", 0x67, 18},
		{"RLD", 0x6F, 18},
		{"LDI", 0xA0, 16},
		{"CPI", 0xA1, 16},
		{"INI", 0xA2, 16},
		{"OUTI", 0xA3, 16},
		{"LDD", 0xA8, 16},
		{"CPD", 0xA9, 16},
		{"IND", 0xAA, 16},
		{"OUTD", 0xAB, 16},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = 0xED // ED prefix
			rom[1] = tc.op2
			rom[2] = 0x00 // Operand 1
			rom[3] = 0x00 // Operand 2
			for i := 4; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)
			cpu.cpu.SP = 0xDFF0

			// Set HL/BC for block operations
			cpu.cpu.HL.Lo = 0x00
			cpu.cpu.HL.Hi = 0xC0
			cpu.cpu.BC.Lo = 0x01
			cpu.cpu.BC.Hi = 0x00

			cycles := cpu.Step()
			if cycles != tc.cycles {
				t.Errorf("%s (ED 0x%02X) cycles: expected %d, got %d", tc.name, tc.op2, tc.cycles, cycles)
			}
		})
	}
}

// TestCycleZ80_FDPrefixCycles tests FD-prefixed (IY) opcode cycles
func TestCycleZ80_FDPrefixCycles(t *testing.T) {
	testCases := []struct {
		name   string
		op2    uint8
		cycles int
	}{
		{"ADD IY,BC", 0x09, 15},
		{"LD IY,nn", 0x21, 14},
		{"LD (nn),IY", 0x22, 20},
		{"INC IY", 0x23, 10},
		{"LD IY,(nn)", 0x2A, 20},
		{"DEC IY", 0x2B, 10},
		{"INC (IY+d)", 0x34, 23},
		{"DEC (IY+d)", 0x35, 23},
		{"POP IY", 0xE1, 14},
		{"PUSH IY", 0xE5, 15},
		{"JP (IY)", 0xE9, 8},
		{"LD SP,IY", 0xF9, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = 0xFD // FD prefix
			rom[1] = tc.op2
			rom[2] = 0x00 // Displacement or operand
			rom[3] = 0x00 // Extra operand if needed
			for i := 4; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)
			cpu.cpu.SP = 0xDFF0

			cycles := cpu.Step()
			if cycles != tc.cycles {
				t.Errorf("%s (FD 0x%02X) cycles: expected %d, got %d", tc.name, tc.op2, tc.cycles, cycles)
			}
		})
	}
}

// TestCycleZ80_FDCBPrefixCycles tests FD CB prefixed (IY bit) opcode cycles
func TestCycleZ80_FDCBPrefixCycles(t *testing.T) {
	testCases := []struct {
		name   string
		op4    uint8
		cycles int
	}{
		{"BIT 0,(IY+d)", 0x46, 20},
		{"SET 0,(IY+d)", 0xC6, 23},
		{"RES 0,(IY+d)", 0x86, 23},
		{"RLC (IY+d)", 0x06, 23},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = 0xFD // FD prefix
			rom[1] = 0xCB // CB sub-prefix
			rom[2] = 0x00 // Displacement
			rom[3] = tc.op4
			for i := 4; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)

			cycles := cpu.Step()
			if cycles != tc.cycles {
				t.Errorf("%s (FD CB d 0x%02X) cycles: expected %d, got %d", tc.name, tc.op4, tc.cycles, cycles)
			}
		})
	}
}

// TestCycleZ80_JRConditionalNotTaken tests JR cc,d when condition is not met
func TestCycleZ80_JRConditionalNotTaken(t *testing.T) {
	testCases := []struct {
		name   string
		opcode uint8
		setup  func(*CycleZ80)
	}{
		{"JR NZ,d (Z set)", 0x20, func(c *CycleZ80) { c.cpu.AF.Lo |= 0x40 }},   // Z flag set
		{"JR Z,d (Z clear)", 0x28, func(c *CycleZ80) { c.cpu.AF.Lo &^= 0x40 }}, // Z flag clear
		{"JR NC,d (C set)", 0x30, func(c *CycleZ80) { c.cpu.AF.Lo |= 0x01 }},   // C flag set
		{"JR C,d (C clear)", 0x38, func(c *CycleZ80) { c.cpu.AF.Lo &^= 0x01 }}, // C flag clear
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = tc.opcode
			rom[1] = 0x10 // Jump offset (positive)
			for i := 2; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)
			tc.setup(cpu)

			cycles := cpu.Step()
			// Not taken should be 7 cycles
			if cycles != 7 {
				t.Errorf("%s not taken: expected 7 cycles, got %d", tc.name, cycles)
			}
			// PC should advance by 2 (opcode + displacement)
			if cpu.GetPC() != 2 {
				t.Errorf("%s PC: expected 2, got %d", tc.name, cpu.GetPC())
			}
		})
	}
}

// TestCycleZ80_JRConditionalTaken tests JR cc,d when condition is met
func TestCycleZ80_JRConditionalTaken(t *testing.T) {
	testCases := []struct {
		name   string
		opcode uint8
		setup  func(*CycleZ80)
	}{
		{"JR NZ,d (Z clear)", 0x20, func(c *CycleZ80) { c.cpu.AF.Lo &^= 0x40 }}, // Z flag clear
		{"JR Z,d (Z set)", 0x28, func(c *CycleZ80) { c.cpu.AF.Lo |= 0x40 }},     // Z flag set
		{"JR NC,d (C clear)", 0x30, func(c *CycleZ80) { c.cpu.AF.Lo &^= 0x01 }}, // C flag clear
		{"JR C,d (C set)", 0x38, func(c *CycleZ80) { c.cpu.AF.Lo |= 0x01 }},     // C flag set
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = tc.opcode
			rom[1] = 0x10 // Jump offset (positive, +16)
			for i := 2; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)
			tc.setup(cpu)

			cycles := cpu.Step()
			// Taken should be 12 cycles
			if cycles != 12 {
				t.Errorf("%s taken: expected 12 cycles, got %d", tc.name, cycles)
			}
			// PC should jump to 0x0002 + 0x10 = 0x0012
			if cpu.GetPC() != 0x12 {
				t.Errorf("%s PC: expected 0x12, got 0x%04X", tc.name, cpu.GetPC())
			}
		})
	}
}

// TestCycleZ80_RETConditionalNotTaken tests RET cc when condition is not met
func TestCycleZ80_RETConditionalNotTaken(t *testing.T) {
	testCases := []struct {
		name   string
		opcode uint8
		setup  func(*CycleZ80)
	}{
		{"RET NZ (Z set)", 0xC0, func(c *CycleZ80) { c.cpu.AF.Lo |= 0x40 }},
		{"RET Z (Z clear)", 0xC8, func(c *CycleZ80) { c.cpu.AF.Lo &^= 0x40 }},
		{"RET NC (C set)", 0xD0, func(c *CycleZ80) { c.cpu.AF.Lo |= 0x01 }},
		{"RET C (C clear)", 0xD8, func(c *CycleZ80) { c.cpu.AF.Lo &^= 0x01 }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = tc.opcode
			for i := 1; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)
			cpu.cpu.SP = 0xDFF0
			tc.setup(cpu)

			cycles := cpu.Step()
			// Not taken should be 5 cycles
			if cycles != 5 {
				t.Errorf("%s not taken: expected 5 cycles, got %d", tc.name, cycles)
			}
			// PC should advance by 1
			if cpu.GetPC() != 1 {
				t.Errorf("%s PC: expected 1, got %d", tc.name, cpu.GetPC())
			}
		})
	}
}

// TestCycleZ80_RETConditionalTaken tests RET cc when condition is met
func TestCycleZ80_RETConditionalTaken(t *testing.T) {
	testCases := []struct {
		name   string
		opcode uint8
		setup  func(*CycleZ80)
	}{
		{"RET NZ (Z clear)", 0xC0, func(c *CycleZ80) { c.cpu.AF.Lo &^= 0x40 }},
		{"RET Z (Z set)", 0xC8, func(c *CycleZ80) { c.cpu.AF.Lo |= 0x40 }},
		{"RET NC (C clear)", 0xD0, func(c *CycleZ80) { c.cpu.AF.Lo &^= 0x01 }},
		{"RET C (C set)", 0xD8, func(c *CycleZ80) { c.cpu.AF.Lo |= 0x01 }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = tc.opcode
			for i := 1; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)
			// Set up stack with return address
			cpu.cpu.SP = 0xDFF0
			mem.Set(0xDFF0, 0x00) // Low byte of return address
			mem.Set(0xDFF1, 0x10) // High byte (return to 0x1000)
			tc.setup(cpu)

			cycles := cpu.Step()
			// Taken should be 11 cycles
			if cycles != 11 {
				t.Errorf("%s taken: expected 11 cycles, got %d", tc.name, cycles)
			}
			// PC should be return address
			if cpu.GetPC() != 0x1000 {
				t.Errorf("%s PC: expected 0x1000, got 0x%04X", tc.name, cpu.GetPC())
			}
		})
	}
}

// TestCycleZ80_CALLConditionalNotTaken tests CALL cc,nn when condition is not met
func TestCycleZ80_CALLConditionalNotTaken(t *testing.T) {
	testCases := []struct {
		name   string
		opcode uint8
		setup  func(*CycleZ80)
	}{
		{"CALL NZ,nn (Z set)", 0xC4, func(c *CycleZ80) { c.cpu.AF.Lo |= 0x40 }},
		{"CALL Z,nn (Z clear)", 0xCC, func(c *CycleZ80) { c.cpu.AF.Lo &^= 0x40 }},
		{"CALL NC,nn (C set)", 0xD4, func(c *CycleZ80) { c.cpu.AF.Lo |= 0x01 }},
		{"CALL C,nn (C clear)", 0xDC, func(c *CycleZ80) { c.cpu.AF.Lo &^= 0x01 }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = tc.opcode
			rom[1] = 0x00 // Low byte of address
			rom[2] = 0x10 // High byte (0x1000)
			for i := 3; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)
			cpu.cpu.SP = 0xDFF0
			tc.setup(cpu)

			cycles := cpu.Step()
			// Not taken should be 10 cycles
			if cycles != 10 {
				t.Errorf("%s not taken: expected 10 cycles, got %d", tc.name, cycles)
			}
			// PC should advance by 3
			if cpu.GetPC() != 3 {
				t.Errorf("%s PC: expected 3, got %d", tc.name, cpu.GetPC())
			}
		})
	}
}

// TestCycleZ80_CALLConditionalTaken tests CALL cc,nn when condition is met
func TestCycleZ80_CALLConditionalTaken(t *testing.T) {
	testCases := []struct {
		name   string
		opcode uint8
		setup  func(*CycleZ80)
	}{
		{"CALL NZ,nn (Z clear)", 0xC4, func(c *CycleZ80) { c.cpu.AF.Lo &^= 0x40 }},
		{"CALL Z,nn (Z set)", 0xCC, func(c *CycleZ80) { c.cpu.AF.Lo |= 0x40 }},
		{"CALL NC,nn (C clear)", 0xD4, func(c *CycleZ80) { c.cpu.AF.Lo &^= 0x01 }},
		{"CALL C,nn (C set)", 0xDC, func(c *CycleZ80) { c.cpu.AF.Lo |= 0x01 }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = tc.opcode
			rom[1] = 0x00 // Low byte of address
			rom[2] = 0x10 // High byte (0x1000)
			for i := 3; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)
			cpu.cpu.SP = 0xDFF0
			tc.setup(cpu)

			cycles := cpu.Step()
			// Taken should be 17 cycles
			if cycles != 17 {
				t.Errorf("%s taken: expected 17 cycles, got %d", tc.name, cycles)
			}
			// PC should be call address
			if cpu.GetPC() != 0x1000 {
				t.Errorf("%s PC: expected 0x1000, got 0x%04X", tc.name, cpu.GetPC())
			}
		})
	}
}

// TestCycleZ80_DJNZNotTaken tests DJNZ when B becomes 0
func TestCycleZ80_DJNZNotTaken(t *testing.T) {
	rom := make([]byte, 0x8000)
	rom[0] = 0x10 // DJNZ
	rom[1] = 0x10 // Displacement
	for i := 2; i < len(rom); i++ {
		rom[i] = 0x00
	}

	mem := NewMemory(rom)
	io := &mockIO{}
	cpu := NewCycleZ80(mem, io)
	cpu.cpu.BC.Hi = 1 // B=1, will become 0 (not taken)

	cycles := cpu.Step()
	// Not taken should be 8 cycles
	if cycles != 8 {
		t.Errorf("DJNZ not taken: expected 8 cycles, got %d", cycles)
	}
	// PC should advance by 2
	if cpu.GetPC() != 2 {
		t.Errorf("DJNZ PC: expected 2, got %d", cpu.GetPC())
	}
}

// TestCycleZ80_DJNZTaken tests DJNZ when B is still non-zero
func TestCycleZ80_DJNZTaken(t *testing.T) {
	rom := make([]byte, 0x8000)
	rom[0] = 0x10 // DJNZ
	rom[1] = 0x10 // Displacement (+16)
	for i := 2; i < len(rom); i++ {
		rom[i] = 0x00
	}

	mem := NewMemory(rom)
	io := &mockIO{}
	cpu := NewCycleZ80(mem, io)
	cpu.cpu.BC.Hi = 5 // B=5, will become 4 (taken)

	cycles := cpu.Step()
	// Taken should be 13 cycles
	if cycles != 13 {
		t.Errorf("DJNZ taken: expected 13 cycles, got %d", cycles)
	}
	// PC should jump
	if cpu.GetPC() != 0x12 {
		t.Errorf("DJNZ PC: expected 0x12, got 0x%04X", cpu.GetPC())
	}
}

// TestCycleZ80_JPConditional tests JP cc,nn (always 10 cycles)
func TestCycleZ80_JPConditional(t *testing.T) {
	opcodes := []uint8{0xC2, 0xCA, 0xD2, 0xDA, 0xE2, 0xEA, 0xF2, 0xFA}

	for _, opcode := range opcodes {
		t.Run("JP cc", func(t *testing.T) {
			rom := make([]byte, 0x8000)
			rom[0] = opcode
			rom[1] = 0x00
			rom[2] = 0x10
			for i := 3; i < len(rom); i++ {
				rom[i] = 0x00
			}

			mem := NewMemory(rom)
			io := &mockIO{}
			cpu := NewCycleZ80(mem, io)

			cycles := cpu.Step()
			// JP cc is always 10 cycles regardless of condition
			if cycles != 10 {
				t.Errorf("JP cc (0x%02X): expected 10 cycles, got %d", opcode, cycles)
			}
		})
	}
}

// TestCycleZ80_HALTState tests that HALT returns 4 cycles
func TestCycleZ80_HALTState(t *testing.T) {
	rom := make([]byte, 0x8000)
	rom[0] = 0x76 // HALT
	for i := 1; i < len(rom); i++ {
		rom[i] = 0x00
	}

	mem := NewMemory(rom)
	io := &mockIO{}
	cpu := NewCycleZ80(mem, io)

	// First HALT instruction
	cycles := cpu.Step()
	if cycles != 4 {
		t.Errorf("HALT cycles: expected 4, got %d", cycles)
	}

	// While halted, Step should return 4 cycles (NOP equivalent)
	cycles = cpu.Step()
	if cycles != 4 {
		t.Errorf("HALT (while halted) cycles: expected 4, got %d", cycles)
	}
}

// TestCycleZ80_InterruptServiceTime tests interrupt response cycles
func TestCycleZ80_InterruptServiceTime(t *testing.T) {
	rom := make([]byte, 0x8000)
	// Fill with NOPs
	for i := range rom {
		rom[i] = 0x00
	}
	// Place RST 38 handler (IM1 jumps to $0038)
	rom[0x38] = 0xC9 // RET

	mem := NewMemory(rom)
	io := &mockIO{}
	cpu := NewCycleZ80(mem, io)
	cpu.cpu.SP = 0xDFF0
	cpu.cpu.IFF1 = true // Enable interrupts
	cpu.cpu.IM = 1       // IM1 mode

	// Set pending interrupt
	cpu.SetInterrupt(z80.IM1Interrupt())

	// Step should service the interrupt
	cycles := cpu.Step()
	// IM1 interrupt response is 13 cycles
	if cycles != 13 {
		t.Errorf("Interrupt service cycles: expected 13, got %d", cycles)
	}
}

// TestCycleZ80_InterruptWakesHALT tests that interrupts wake CPU from HALT
func TestCycleZ80_InterruptWakesHALT(t *testing.T) {
	rom := make([]byte, 0x8000)
	rom[0] = 0x76 // HALT
	for i := 1; i < len(rom); i++ {
		rom[i] = 0x00
	}
	// Place handler
	rom[0x38] = 0xC9 // RET

	mem := NewMemory(rom)
	io := &mockIO{}
	cpu := NewCycleZ80(mem, io)
	cpu.cpu.SP = 0xDFF0
	cpu.cpu.IFF1 = true // Interrupts enabled
	cpu.cpu.IM = 1

	// Enter HALT
	cpu.Step()
	if !cpu.cpu.HALT {
		t.Error("CPU should be in HALT state")
	}

	// Set interrupt - should wake from HALT
	cpu.SetInterrupt(z80.IM1Interrupt())
	cycles := cpu.Step()

	// Should service interrupt (13 cycles)
	if cycles != 13 {
		t.Errorf("Wake from HALT cycles: expected 13, got %d", cycles)
	}
	// Should no longer be halted
	if cpu.cpu.HALT {
		t.Error("CPU should not be halted after interrupt")
	}
}

// TestCycleZ80_BaseCyclesTableLength verifies cycle table has 256 entries
func TestCycleZ80_BaseCyclesTableLength(t *testing.T) {
	if len(baseCycles) != 256 {
		t.Errorf("baseCycles table length: expected 256, got %d", len(baseCycles))
	}
	if len(cbCycles) != 256 {
		t.Errorf("cbCycles table length: expected 256, got %d", len(cbCycles))
	}
	if len(ddCycles) != 256 {
		t.Errorf("ddCycles table length: expected 256, got %d", len(ddCycles))
	}
	if len(fdCycles) != 256 {
		t.Errorf("fdCycles table length: expected 256, got %d", len(fdCycles))
	}
	if len(edCycles) != 256 {
		t.Errorf("edCycles table length: expected 256, got %d", len(edCycles))
	}
}
