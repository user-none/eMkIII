package emu

import "testing"

// TestIO_ControllerDefaultState tests that all buttons released = $FF
func TestIO_ControllerDefaultState(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	if io.Input.Port1 != 0xFF {
		t.Errorf("Default Port1: expected 0xFF, got 0x%02X", io.Input.Port1)
	}
	if io.Input.Port2 != 0xFF {
		t.Errorf("Default Port2: expected 0xFF, got 0x%02X", io.Input.Port2)
	}
}

// TestIO_ControllerInput tests active-low button encoding
func TestIO_ControllerInput(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	// Test individual buttons (active low - 0 = pressed)
	testCases := []struct {
		up, down, left, right, btn1, btn2 bool
		expectedPort1                     uint8
	}{
		{true, false, false, false, false, false, 0xFE},  // Up: bit 0 clear
		{false, true, false, false, false, false, 0xFD},  // Down: bit 1 clear
		{false, false, true, false, false, false, 0xFB},  // Left: bit 2 clear
		{false, false, false, true, false, false, 0xF7},  // Right: bit 3 clear
		{false, false, false, false, true, false, 0xEF},  // Button 1: bit 4 clear
		{false, false, false, false, false, true, 0xDF},  // Button 2: bit 5 clear
		{true, false, true, false, true, false, 0xEA},    // Up + Left + Btn1
		{false, false, false, false, false, false, 0xFF}, // All released
	}

	for i, tc := range testCases {
		io.Input.SetP1(tc.up, tc.down, tc.left, tc.right, tc.btn1, tc.btn2)
		if io.Input.Port1 != tc.expectedPort1 {
			t.Errorf("Test %d: expected Port1=0x%02X, got 0x%02X", i, tc.expectedPort1, io.Input.Port1)
		}
	}
}

// TestIO_PortDecoding tests correct routing for port ranges
func TestIO_PortDecoding(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	// Set up known state
	io.Input.Port1 = 0xAA
	io.Input.Port2 = 0x55

	// Test controller port reads ($C0-$FF range)
	// Even ports ($C0, $C2, etc.) return Port1
	if got := io.In(0xC0); got != 0xAA {
		t.Errorf("In($C0): expected 0xAA (Port1), got 0x%02X", got)
	}
	if got := io.In(0xDC); got != 0xAA {
		t.Errorf("In($DC): expected 0xAA (Port1), got 0x%02X", got)
	}

	// Odd ports ($C1, $C3, etc.) return Port2
	if got := io.In(0xC1); got != 0x55 {
		t.Errorf("In($C1): expected 0x55 (Port2), got 0x%02X", got)
	}
	if got := io.In(0xDD); got != 0x55 {
		t.Errorf("In($DD): expected 0x55 (Port2), got 0x%02X", got)
	}
}

// TestIO_VCounterRead tests that port $7E returns V counter
func TestIO_VCounterRead(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	// Set VDP V counter via SetVCounter
	vdp.SetVCounter(100)

	// Read V counter via I/O port $40 (or $7E - both in same range)
	if got := io.In(0x40); got != 100 {
		t.Errorf("V counter read via $40: expected 100, got %d", got)
	}

	vdp.SetVCounter(200)
	if got := io.In(0x7E); got != 200 {
		t.Errorf("V counter read via $7E: expected 200, got %d", got)
	}
}

// TestIO_HCounterRead tests that port $7F returns H counter
func TestIO_HCounterRead(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	// Set VDP H counter
	vdp.SetHCounter(50)

	// Read H counter via I/O port $41 or $7F (odd ports in $40-$7F range)
	if got := io.In(0x41); got != 50 {
		t.Errorf("H counter read via $41: expected 50, got %d", got)
	}

	vdp.SetHCounter(128)
	if got := io.In(0x7F); got != 128 {
		t.Errorf("H counter read via $7F: expected 128, got %d", got)
	}
}

// TestIO_VDPDataRouting tests that port $BE routes to VDP data
func TestIO_VDPDataRouting(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	// Set up VDP for VRAM write (code = 1)
	vdp.WriteControl(0x00) // Low address byte
	vdp.WriteControl(0x40) // High address byte + code 1 (VRAM write)

	// Write data via I/O port
	io.Out(0xBE, 0x42)

	// Verify data was written to VRAM
	vram := vdp.GetVRAM()
	if vram[0] != 0x42 {
		t.Errorf("VRAM write via $BE: expected 0x42, got 0x%02X", vram[0])
	}
}

// TestIO_VDPControlRouting tests that port $BF routes to VDP control
func TestIO_VDPControlRouting(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	// Write to control port via I/O
	io.Out(0xBF, 0x00) // First byte of two-byte sequence
	if !vdp.GetWriteLatch() {
		t.Error("VDP write latch should be set after first control write")
	}

	io.Out(0xBF, 0x82) // Second byte: code 2 (register write), reg 0, value 0x00
	if vdp.GetWriteLatch() {
		t.Error("VDP write latch should be clear after second control write")
	}
}

// TestIO_PSGWrite tests that port $7F writes route to PSG
func TestIO_PSGWrite(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	// Write volume to channel 0 via PSG port
	io.Out(0x7F, 0x9F) // Channel 0 volume = 0x0F (silent)
	if got := psg.GetVolume(0); got != 0x0F {
		t.Errorf("PSG write via $7F: expected volume 0x0F, got 0x%02X", got)
	}

	// Also test $40 (even ports in $40-$7F range go to PSG)
	io.Out(0x40, 0x95) // Channel 0 volume = 0x05
	if got := psg.GetVolume(0); got != 0x05 {
		t.Errorf("PSG write via $40: expected volume 0x05, got 0x%02X", got)
	}
}

// TestIO_VDPDataRead tests VDP data read routing
func TestIO_VDPDataRead(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	// Write some data to VRAM first
	vdp.WriteControl(0x00) // Low address = 0
	vdp.WriteControl(0x40) // High address = 0, code = 1 (VRAM write)
	vdp.WriteData(0xAB)
	vdp.WriteData(0xCD) // Write a second byte

	// Set up for VRAM read
	vdp.WriteControl(0x00) // Low address = 0
	vdp.WriteControl(0x00) // High address = 0, code = 0 (VRAM read)

	// Read via I/O port - first read returns pre-fetched byte at address 0
	if got := io.In(0xBE); got != 0xAB {
		t.Errorf("VRAM read via $BE: expected 0xAB, got 0x%02X", got)
	}
	// Second read returns byte at address 1
	if got := io.In(0x80); got != 0xCD {
		t.Errorf("VRAM read via $80: expected 0xCD, got 0x%02X", got)
	}
}

// TestIO_VDPStatusRead tests VDP status read routing
func TestIO_VDPStatusRead(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	// Set VBlank flag
	vdp.SetVBlank()

	// Read status via I/O port $BF (odd ports in $80-$BF range)
	status := io.In(0xBF)
	if status&0x80 == 0 {
		t.Error("VBlank flag should be set in status")
	}

	// Status read should clear the VBlank flag
	status2 := io.In(0x81)
	if status2&0x80 != 0 {
		t.Error("VBlank flag should be cleared after read")
	}
}

// TestIO_ControllerP2Input tests Player 2 active-low button encoding
func TestIO_ControllerP2Input(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	// Test individual P2 buttons (active low - 0 = pressed)
	// P2 Up: Port1 bit 6 clear (0xBF)
	// P2 Down: Port1 bit 7 clear (0x7F)
	// P2 Left: Port2 bit 0 clear (0xFE)
	// P2 Right: Port2 bit 1 clear (0xFD)
	// P2 Button 1: Port2 bit 2 clear (0xFB)
	// P2 Button 2: Port2 bit 3 clear (0xF7)
	testCases := []struct {
		up, down, left, right, btn1, btn2 bool
		expectedPort1                     uint8
		expectedPort2                     uint8
	}{
		{true, false, false, false, false, false, 0xBF, 0xFF},  // Up: Port1 bit 6 clear
		{false, true, false, false, false, false, 0x7F, 0xFF},  // Down: Port1 bit 7 clear
		{false, false, true, false, false, false, 0xFF, 0xFE},  // Left: Port2 bit 0 clear
		{false, false, false, true, false, false, 0xFF, 0xFD},  // Right: Port2 bit 1 clear
		{false, false, false, false, true, false, 0xFF, 0xFB},  // Button 1: Port2 bit 2 clear
		{false, false, false, false, false, true, 0xFF, 0xF7},  // Button 2: Port2 bit 3 clear
		{true, true, true, true, true, true, 0x3F, 0xF0},       // All P2 pressed
		{false, false, false, false, false, false, 0xFF, 0xFF}, // All released
	}

	for i, tc := range testCases {
		// Reset ports
		io.Input.Port1 = 0xFF
		io.Input.Port2 = 0xFF
		io.Input.SetP2(tc.up, tc.down, tc.left, tc.right, tc.btn1, tc.btn2)
		if io.Input.Port1 != tc.expectedPort1 {
			t.Errorf("Test %d: expected Port1=0x%02X, got 0x%02X", i, tc.expectedPort1, io.Input.Port1)
		}
		if io.Input.Port2 != tc.expectedPort2 {
			t.Errorf("Test %d: expected Port2=0x%02X, got 0x%02X", i, tc.expectedPort2, io.Input.Port2)
		}
	}
}

// TestIO_ControllerP1P2Combined tests P1 and P2 inputs don't interfere
func TestIO_ControllerP1P2Combined(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	// Set P1: Up + Button 1 (Port1 bits 0 and 4 clear = 0xEE)
	// Set P2: Down + Button 2 (Port1 bit 7 clear, Port2 bit 3 clear)
	// Expected Port1: P1 bits (0xEE) + P2 bits (0x7F) = 0x6E (both masks applied)
	// Expected Port2: P2 Button 2 (0xF7)

	io.Input.SetP1(true, false, false, false, true, false)
	io.Input.SetP2(false, true, false, false, false, true)

	// Port1 should have P1 Up (bit 0 clear) + P1 Btn1 (bit 4 clear) + P2 Down (bit 7 clear)
	// = 0xFF & ~0x01 & ~0x10 & ~0x80 = 0x6E
	expectedPort1 := uint8(0x6E)
	if io.Input.Port1 != expectedPort1 {
		t.Errorf("Combined input: expected Port1=0x%02X, got 0x%02X", expectedPort1, io.Input.Port1)
	}

	// Port2 should have P2 Button 2 (bit 3 clear) = 0xF7
	expectedPort2 := uint8(0xF7)
	if io.Input.Port2 != expectedPort2 {
		t.Errorf("Combined input: expected Port2=0x%02X, got 0x%02X", expectedPort2, io.Input.Port2)
	}

	// Test reverse order: SetP2 then SetP1 should still work correctly
	io.Input.Port1 = 0xFF
	io.Input.Port2 = 0xFF
	io.Input.SetP2(true, false, false, false, false, false) // P2 Up
	io.Input.SetP1(false, true, false, false, false, false) // P1 Down

	// Port1: P1 Down (bit 1 clear) + P2 Up (bit 6 clear) = 0xFF & ~0x02 & ~0x40 = 0xBD
	expectedPort1 = 0xBD
	if io.Input.Port1 != expectedPort1 {
		t.Errorf("Reverse order: expected Port1=0x%02X, got 0x%02X", expectedPort1, io.Input.Port1)
	}
}

// TestIO_PartialAddressDecoding tests that port decoding uses partial address
func TestIO_PartialAddressDecoding(t *testing.T) {
	vdp := NewVDP()
	psg := NewPSG(3579545, 48000, 800)
	io := NewSMSIO(vdp, psg)

	io.Input.Port1 = 0x12
	io.Input.Port2 = 0x34

	// All these should return the same as $DC/$DD due to partial decoding
	// The decoding only looks at bits 7, 6, and 0
	ports := []struct {
		port   uint8
		expect uint8
		desc   string
	}{
		{0xC0, 0x12, "Port1"},
		{0xC2, 0x12, "Port1"},
		{0xDC, 0x12, "Port1"},
		{0xFE, 0x12, "Port1"},
		{0xC1, 0x34, "Port2"},
		{0xC3, 0x34, "Port2"},
		{0xDD, 0x34, "Port2"},
		{0xFF, 0x34, "Port2"},
	}

	for _, p := range ports {
		if got := io.In(p.port); got != p.expect {
			t.Errorf("In(0x%02X): expected 0x%02X (%s), got 0x%02X", p.port, p.expect, p.desc, got)
		}
	}
}
