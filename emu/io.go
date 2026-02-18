package emu

import "github.com/user-none/go-chip-sn76489"

// Input holds controller state (directly usable as port values)
type Input struct {
	Port1 uint8 // Port $DC - Controller 1 + partial Controller 2
	Port2 uint8 // Port $DD - Controller 2 + misc
}

type SMSIO struct {
	vdp         *VDP
	psg         *sn76489.SN76489
	Input       *Input
	nationality Nationality
	ioControl   uint8 // Port $3F: I/O port control register
}

func NewSMSIO(vdp *VDP, psg *sn76489.SN76489, nationality Nationality) *SMSIO {
	return &SMSIO{
		vdp: vdp,
		psg: psg,
		Input: &Input{
			Port1: 0xFF, // All buttons released (active low)
			Port2: 0xFF,
		},
		nationality: nationality,
		ioControl:   0xFF, // All pins high at power-on
	}
}

func (e *SMSIO) In(addr uint8) uint8 {
	// SMS uses partial address decoding
	// Bits 7 and 6 determine the port group, bit 0 determines even/odd
	switch addr & 0xC1 {
	case 0x40: // $40-$7F even: V counter
		return e.vdp.ReadVCounter()
	case 0x41: // $40-$7F odd: H counter
		return e.vdp.ReadHCounter()
	case 0x80: // $80-$BF even: VDP data
		return e.vdp.ReadData()
	case 0x81: // $80-$BF odd: VDP control (status)
		return e.vdp.ReadControl()
	case 0xC0: // $C0-$FF even: I/O port A (controller 1)
		return e.Input.Port1
	case 0xC1: // $C0-$FF odd: I/O port B (controller 2 + misc)
		return e.readPortDD()
	}
	return 0xFF
}

func (e *SMSIO) Out(addr uint8, value uint8) {
	// SMS uses partial address decoding
	switch addr & 0xC1 {
	case 0x01: // $00-$3F odd: I/O port control register
		e.ioControl = value
	case 0x40, 0x41: // $40-$7F: PSG
		if e.psg != nil {
			e.psg.Write(value)
		}
	case 0x80: // $80-$BF even: VDP data
		e.vdp.WriteData(value)
	case 0x81: // $80-$BF odd: VDP control
		e.vdp.WriteControl(value)
	}
}

// UpdateInput updates controller state from button flags
// Port $DC bits (active low - 0 = pressed):
//
//	Bit 0: P1 Up
//	Bit 1: P1 Down
//	Bit 2: P1 Left
//	Bit 3: P1 Right
//	Bit 4: P1 Button 1
//	Bit 5: P1 Button 2
//	Bit 6: P2 Up
//	Bit 7: P2 Down
func (i *Input) SetP1(up, down, left, right, btn1, btn2 bool) {
	// Update only P1 bits (0-5), preserve P2 bits (6-7)
	i.Port1 |= 0x3F
	if up {
		i.Port1 &^= 0x01
	}
	if down {
		i.Port1 &^= 0x02
	}
	if left {
		i.Port1 &^= 0x04
	}
	if right {
		i.Port1 &^= 0x08
	}
	if btn1 {
		i.Port1 &^= 0x10
	}
	if btn2 {
		i.Port1 &^= 0x20
	}
}

// SetP2 updates Player 2 controller state
// Port $DC bits 6-7: P2 Up, Down
// Port $DD bits 0-3: P2 Left, Right, Btn1, Btn2
func (i *Input) SetP2(up, down, left, right, btn1, btn2 bool) {
	// Update Port1 bits 6-7 (P2 Up/Down), preserve P1 bits
	i.Port1 |= 0xC0
	if up {
		i.Port1 &^= 0x40
	}
	if down {
		i.Port1 &^= 0x80
	}

	// Update Port2 bits 0-3 (P2 Left/Right/Btn1/Btn2)
	i.Port2 |= 0x0F
	if left {
		i.Port2 &^= 0x01
	}
	if right {
		i.Port2 &^= 0x02
	}
	if btn1 {
		i.Port2 &^= 0x04
	}
	if btn2 {
		i.Port2 &^= 0x08
	}
}

// readPortDD synthesizes the port $DD read value.
// Bits 0-5 come from controller data (Input.Port2).
// Bits 6-7 come from the I/O control register ($3F) TH output levels.
// On Japanese consoles, bits 6-7 are inverted.
func (e *SMSIO) readPortDD() uint8 {
	// Start with controller bits 0-5
	result := e.Input.Port2 & 0x3F

	// Bit 6 = Port A TH (from ioControl bit 5)
	// Bit 7 = Port B TH (from ioControl bit 7)
	if e.ioControl&0x20 != 0 {
		result |= 0x40
	}
	if e.ioControl&0x80 != 0 {
		result |= 0x80
	}

	// Japanese consoles invert TH bits
	if e.nationality == NationalityJapanese {
		result ^= 0xC0
	}

	return result
}
