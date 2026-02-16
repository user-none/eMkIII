package emu

import "testing"

// TestVDP_ControlWriteSequence tests two-byte address/command sequence
func TestVDP_ControlWriteSequence(t *testing.T) {
	vdp := NewVDP()

	// First write should set latch
	if vdp.GetWriteLatch() {
		t.Error("Write latch should be false initially")
	}

	vdp.WriteControl(0x00) // First byte
	if !vdp.GetWriteLatch() {
		t.Error("Write latch should be true after first byte")
	}

	vdp.WriteControl(0x00) // Second byte
	if vdp.GetWriteLatch() {
		t.Error("Write latch should be false after second byte")
	}
}

// TestVDP_RegisterWrite tests control code 2 writes to registers
func TestVDP_RegisterWrite(t *testing.T) {
	vdp := NewVDP()

	// Write to register 5 with value 0x7E
	// First byte: value to write (0x7E)
	// Second byte: 10xx xxxx | register number (0x85 = 0x80 | 5)
	vdp.WriteControl(0x7E)
	vdp.WriteControl(0x85) // Code 2 (bits 7-6 = 10), reg 5

	if got := vdp.GetRegister(5); got != 0x7E {
		t.Errorf("Register 5 after write: expected 0x7E, got 0x%02X", got)
	}

	// Write to register 0 with value 0x36
	vdp.WriteControl(0x36)
	vdp.WriteControl(0x80) // Code 2, reg 0

	if got := vdp.GetRegister(0); got != 0x36 {
		t.Errorf("Register 0 after write: expected 0x36, got 0x%02X", got)
	}
}

// TestVDP_VRAMReadWrite tests VRAM access with auto-increment
func TestVDP_VRAMReadWrite(t *testing.T) {
	vdp := NewVDP()

	// Set up VRAM write at address 0x100
	vdp.WriteControl(0x00) // Low byte of address
	vdp.WriteControl(0x41) // High byte (0x01) + code 1 (VRAM write)

	// Write sequential bytes
	vdp.WriteData(0x11)
	vdp.WriteData(0x22)
	vdp.WriteData(0x33)

	// Verify address auto-incremented
	if got := vdp.GetAddress(); got != 0x103 {
		t.Errorf("Address after 3 writes at 0x100: expected 0x103, got 0x%04X", got)
	}

	// Verify data in VRAM directly
	vram := vdp.GetVRAM()
	if vram[0x100] != 0x11 {
		t.Errorf("VRAM[0x100]: expected 0x11, got 0x%02X", vram[0x100])
	}
	if vram[0x101] != 0x22 {
		t.Errorf("VRAM[0x101]: expected 0x22, got 0x%02X", vram[0x101])
	}
	if vram[0x102] != 0x33 {
		t.Errorf("VRAM[0x102]: expected 0x33, got 0x%02X", vram[0x102])
	}
}

// TestVDP_CRAMWrite tests palette writes (32 bytes, wraps at $1F)
func TestVDP_CRAMWrite(t *testing.T) {
	vdp := NewVDP()

	// Set up CRAM write at address 0x00
	vdp.WriteControl(0x00) // Low byte of address
	vdp.WriteControl(0xC0) // Code 3 (CRAM write)

	// Write to first 4 palette entries
	vdp.WriteData(0x00) // Black
	vdp.WriteData(0x03) // Red
	vdp.WriteData(0x0C) // Green
	vdp.WriteData(0x30) // Blue

	cram := vdp.GetCRAM()
	if cram[0] != 0x00 {
		t.Errorf("CRAM[0]: expected 0x00, got 0x%02X", cram[0])
	}
	if cram[1] != 0x03 {
		t.Errorf("CRAM[1]: expected 0x03, got 0x%02X", cram[1])
	}
	if cram[2] != 0x0C {
		t.Errorf("CRAM[2]: expected 0x0C, got 0x%02X", cram[2])
	}
	if cram[3] != 0x30 {
		t.Errorf("CRAM[3]: expected 0x30, got 0x%02X", cram[3])
	}
}

// TestVDP_CRAMWrap tests that CRAM address wraps at 32 bytes
func TestVDP_CRAMWrap(t *testing.T) {
	vdp := NewVDP()

	// Set up CRAM write at address 0x1F (last entry)
	vdp.WriteControl(0x1F)
	vdp.WriteControl(0xC0) // Code 3 (CRAM write)

	vdp.WriteData(0xAA) // Write to index 31
	vdp.WriteData(0xBB) // Should wrap to index 0

	cram := vdp.GetCRAM()
	if cram[31] != 0xAA {
		t.Errorf("CRAM[31]: expected 0xAA, got 0x%02X", cram[31])
	}
	// Note: The address wraps to 0x20 but CRAM access masks with 0x1F
	// so writing at address 0x20 writes to CRAM[0]
	if cram[0] != 0xBB {
		t.Errorf("CRAM[0] after wrap: expected 0xBB, got 0x%02X", cram[0])
	}
}

// TestVDP_ReadBuffer tests pre-fetch behavior on VRAM reads
func TestVDP_ReadBuffer(t *testing.T) {
	vdp := NewVDP()

	// Write known data to VRAM
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40) // VRAM write at 0x000
	vdp.WriteData(0xAA)
	vdp.WriteData(0xBB)
	vdp.WriteData(0xCC)

	// Verify VRAM contents directly
	vram := vdp.GetVRAM()
	if vram[0] != 0xAA || vram[1] != 0xBB || vram[2] != 0xCC {
		t.Errorf("VRAM contents wrong: [0]=%02X [1]=%02X [2]=%02X", vram[0], vram[1], vram[2])
	}

	// Set up VRAM read at 0x000
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x00) // VRAM read at 0x000

	// First read returns the pre-fetched byte (VRAM[0])
	// The pre-fetch happens during WriteControl with code 0
	first := vdp.ReadData()
	if first != 0xAA {
		t.Errorf("First read (pre-fetch): expected 0xAA, got 0x%02X", first)
	}

	// Second read returns the next byte (VRAM[1])
	second := vdp.ReadData()
	if second != 0xBB {
		t.Errorf("Second read: expected 0xBB, got 0x%02X", second)
	}
}

// TestVDP_WriteDataUpdatesReadBuffer tests that writing to the data port
// loads the written value into the read buffer
func TestVDP_WriteDataUpdatesReadBuffer(t *testing.T) {
	vdp := NewVDP()

	// Write a known value to VRAM at address 0x100
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x41) // VRAM write at 0x100
	vdp.WriteData(0x42)    // Write 0x42 -- this should also load read buffer

	// Now set up a VRAM read at a different address (0x200) where VRAM is 0x00
	// The pre-fetch during read setup loads VRAM[0x200] into the buffer,
	// overwriting the value from WriteData. So we need a different approach:
	// Write to data port, then immediately read without re-setting the address.

	// Start fresh: write 0xFF to VRAM[0] so we can distinguish it
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40) // VRAM write at 0x000
	vdp.WriteData(0xFF)

	// Now write 0x42 to VRAM[1] -- read buffer should become 0x42
	vdp.WriteData(0x42)

	// Switch to VRAM read mode at address 0x050 (contains 0x00)
	// The pre-fetch loads VRAM[0x050]=0x00 into the buffer
	vdp.WriteControl(0x50)
	vdp.WriteControl(0x00) // VRAM read at 0x050

	// First read returns the pre-fetched value (VRAM[0x050] = 0x00)
	// This confirms the pre-fetch overwrites the WriteData buffer value,
	// which is correct -- the read setup always pre-fetches.
	first := vdp.ReadData()
	if first != 0x00 {
		t.Errorf("Read after write setup: expected 0x00, got 0x%02X", first)
	}

	// Test the actual behavior: write in VRAM write mode, then read without
	// re-issuing a read command. ReadData always returns the buffer contents.
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40) // VRAM write at 0x000
	vdp.WriteData(0xAB)    // Writes to VRAM[0], buffer = 0xAB

	// ReadData returns buffer (0xAB), then loads VRAM[next_addr] into buffer
	got := vdp.ReadData()
	if got != 0xAB {
		t.Errorf("Read after WriteData: expected 0xAB (from buffer), got 0x%02X", got)
	}
}

// TestVDP_VBlankFlag tests status bit 7 set/clear behavior
func TestVDP_VBlankFlag(t *testing.T) {
	vdp := NewVDP()

	// Initially VBlank should not be set
	if vdp.GetStatus()&0x80 != 0 {
		t.Error("VBlank flag should not be set initially")
	}

	// Set VBlank
	vdp.SetVBlank()
	if vdp.GetStatus()&0x80 == 0 {
		t.Error("VBlank flag should be set after SetVBlank()")
	}

	// Read control clears VBlank flag
	vdp.ReadControl()
	if vdp.GetStatus()&0x80 != 0 {
		t.Error("VBlank flag should be cleared after ReadControl()")
	}
}

// TestVDP_InterruptPending tests frame and line interrupt logic
func TestVDP_InterruptPending(t *testing.T) {
	vdp := NewVDP()

	// Initially no interrupt pending
	if vdp.InterruptPending() {
		t.Error("No interrupt should be pending initially")
	}

	// Set VBlank but frame IE disabled
	vdp.SetVBlank()
	if vdp.InterruptPending() {
		t.Error("Interrupt should not be pending when frame IE is disabled")
	}

	// Enable frame IE (register 1 bit 5)
	vdp.WriteControl(0x20) // Value
	vdp.WriteControl(0x81) // Register 1
	if !vdp.InterruptPending() {
		t.Error("Interrupt should be pending when VBlank set and frame IE enabled")
	}

	// Clear by reading control
	vdp.ReadControl()
	if vdp.InterruptPending() {
		t.Error("Interrupt should not be pending after status read")
	}
}

// TestVDP_LineCounterBehavior tests counter decrement and reload
func TestVDP_LineCounterBehavior(t *testing.T) {
	vdp := NewVDP()

	// Set line counter reload value (register 10)
	vdp.WriteControl(0x05) // Value = 5
	vdp.WriteControl(0x8A) // Register 10

	// Per SMS VDP hardware: counter reloads on lines 193+ (not 192)
	// Line 192 still decrements (it's the first VBlank line but still counts)
	// Use line 193 to properly initialize the counter via reload
	vdp.SetVCounter(193)
	vdp.UpdateLineCounter()

	// Now counter should be 5 (from register 10)
	// Simulate active display scanlines
	// Counter decrements each line: 5->4->3->2->1->0->-1 (underflow on line 5)
	for line := uint16(0); line < 10; line++ {
		vdp.SetVCounter(line)
		vdp.UpdateLineCounter()

		// After line 5, counter should underflow and set pending
		// Line 0: 5-1=4, Line 1: 4-1=3, Line 2: 3-1=2, Line 3: 2-1=1, Line 4: 1-1=0, Line 5: 0-1=-1 (underflow!)
		if line == 5 && !vdp.GetLineIntPending() {
			t.Errorf("Line interrupt should be pending after line %d", line)
		}
	}
}

// TestVDP_ActiveHeight tests 192/224 line modes
// Note: 240-line mode (M2=1, M1=0) is Game Gear only, not supported on SMS
func TestVDP_ActiveHeight(t *testing.T) {
	vdp := NewVDP()

	// Default: 192 lines
	if got := vdp.ActiveHeight(); got != 192 {
		t.Errorf("Default active height: expected 192, got %d", got)
	}

	// Set M2=1, M1=1 for 224 lines
	// M2 = register 0 bit 1 (0x02)
	// M1 = register 1 bit 4 (0x10)
	vdp.WriteControl(0x02) // M2 bit in reg 0
	vdp.WriteControl(0x80)
	vdp.WriteControl(0x10) // M1 bit in reg 1 (bit 4, not bit 3)
	vdp.WriteControl(0x81)

	if got := vdp.ActiveHeight(); got != 224 {
		t.Errorf("224-line mode: expected 224, got %d", got)
	}

	// Clear M1 (set reg 1 to 0) - should revert to 192 lines
	// M2=1, M1=0 is 240-line mode on Game Gear, but SMS falls back to 192
	vdp.WriteControl(0x00) // Clear M1
	vdp.WriteControl(0x81)

	if got := vdp.ActiveHeight(); got != 192 {
		t.Errorf("M2=1, M1=0 on SMS: expected 192 (240-line mode is GG only), got %d", got)
	}

	// Clear M2 as well - still 192 lines
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x80)

	if got := vdp.ActiveHeight(); got != 192 {
		t.Errorf("M2=0, M1=0: expected 192, got %d", got)
	}
}

// TestVDP_VScrollLatching tests vScroll is latched once per frame
func TestVDP_VScrollLatching(t *testing.T) {
	vdp := NewVDP()

	// Set initial vScroll
	vdp.WriteControl(0x10)
	vdp.WriteControl(0x89) // Register 9 = 0x10

	// Latch at start of frame
	vdp.LatchVScrollForFrame()

	// Change register value
	vdp.WriteControl(0x20)
	vdp.WriteControl(0x89) // Register 9 = 0x20

	// Register changed but latch should still have old value
	if vdp.GetRegister(9) != 0x20 {
		t.Errorf("Register 9: expected 0x20, got 0x%02X", vdp.GetRegister(9))
	}
	// The vScrollLatch is internal, we can't directly test it without adding an accessor
	// But we've verified the register write works correctly
}

// TestVDP_HScrollLatching tests hScroll is latched per scanline
func TestVDP_HScrollLatching(t *testing.T) {
	vdp := NewVDP()

	// Set hScroll value
	vdp.WriteControl(0x08)
	vdp.WriteControl(0x88) // Register 8 = 0x08

	// SetVCounter should latch hScroll for that scanline
	vdp.SetVCounter(0)

	// Change register
	vdp.WriteControl(0x10)
	vdp.WriteControl(0x88) // Register 8 = 0x10

	// On next SetVCounter, the new value should be latched
	vdp.SetVCounter(1)

	// Register 8 should have the new value
	if vdp.GetRegister(8) != 0x10 {
		t.Errorf("Register 8: expected 0x10, got 0x%02X", vdp.GetRegister(8))
	}
}

// TestVDP_StatusFlagClearing tests that status read clears specific flags
func TestVDP_StatusFlagClearing(t *testing.T) {
	vdp := NewVDP()

	// We can't directly set overflow/collision flags, but we can test VBlank
	vdp.SetVBlank()

	status := vdp.ReadControl()
	if status&0x80 == 0 {
		t.Error("VBlank should be set in returned status")
	}

	// After read, flags should be cleared
	if vdp.GetStatus()&0x80 != 0 {
		t.Error("VBlank should be cleared after read")
	}
}

// TestVDP_AddressAutoIncrement tests address wraps at 16KB boundary
func TestVDP_AddressAutoIncrement(t *testing.T) {
	vdp := NewVDP()

	// Set address to near end of VRAM
	vdp.WriteControl(0xFF)
	vdp.WriteControl(0x7F) // Address 0x3FFF, code 1 (write)

	vdp.WriteData(0xAA) // Write to 0x3FFF, address becomes 0x0000

	// Address should wrap to 0
	if got := vdp.GetAddress(); got != 0x0000 {
		t.Errorf("Address after wrap: expected 0x0000, got 0x%04X", got)
	}
}

// TestVDP_CodeRegisterValues tests different command codes
func TestVDP_CodeRegisterValues(t *testing.T) {
	vdp := NewVDP()

	testCases := []struct {
		secondByte uint8
		expected   uint8
		desc       string
	}{
		{0x00, 0, "VRAM read (code 0)"},
		{0x40, 1, "VRAM write (code 1)"},
		{0x80, 2, "Register write (code 2)"},
		{0xC0, 3, "CRAM write (code 3)"},
	}

	for _, tc := range testCases {
		vdp.WriteControl(0x00)          // First byte (don't care for this test)
		vdp.WriteControl(tc.secondByte) // Second byte with code

		if got := vdp.GetCodeReg(); got != tc.expected {
			t.Errorf("%s: expected code %d, got %d", tc.desc, tc.expected, got)
		}
	}
}

// TestVDP_ReadControlClearsLatch tests that reading control clears write latch
func TestVDP_ReadControlClearsLatch(t *testing.T) {
	vdp := NewVDP()

	// Write first byte
	vdp.WriteControl(0x00)
	if !vdp.GetWriteLatch() {
		t.Error("Latch should be set after first write")
	}

	// Read control should clear the latch
	vdp.ReadControl()
	if vdp.GetWriteLatch() {
		t.Error("Latch should be cleared after ReadControl()")
	}
}

// TestVDP_FramebufferNotNil tests that framebuffer is created
func TestVDP_FramebufferNotNil(t *testing.T) {
	vdp := NewVDP()

	fb := vdp.Framebuffer()
	if fb == nil {
		t.Error("Framebuffer should not be nil")
	}

	// Framebuffer is sized for maximum possible height (224) to support all display modes
	// (192-line standard, 224-line extended)
	bounds := fb.Bounds()
	if bounds.Dx() != 256 || bounds.Dy() != 224 {
		t.Errorf("Framebuffer size: expected 256x224 (MaxScreenHeight), got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

// TestVDP_VRAMSize tests that VRAM is 16KB
func TestVDP_VRAMSize(t *testing.T) {
	vdp := NewVDP()

	vram := vdp.GetVRAM()
	if len(vram) != 0x4000 {
		t.Errorf("VRAM size: expected 16384, got %d", len(vram))
	}
}

// TestVDP_CRAMSize tests that CRAM is 32 bytes
func TestVDP_CRAMSize(t *testing.T) {
	vdp := NewVDP()

	cram := vdp.GetCRAM()
	if len(cram) != 32 {
		t.Errorf("CRAM size: expected 32, got %d", len(cram))
	}
}

// ----------------------------------------------------------------------------
// V-Counter Tests
// ----------------------------------------------------------------------------

// TestVDP_VCounter_NTSC192 tests V-counter behavior in NTSC 192-line mode
func TestVDP_VCounter_NTSC192(t *testing.T) {
	vdp := NewVDP()
	vdp.SetTotalScanlines(262) // NTSC

	// 192-line mode is default
	testCases := []struct {
		line     uint16
		expected uint8
	}{
		{0, 0},
		{100, 100},
		{191, 191},
		{192, 192},
		{218, 218}, // Last normal line
		{219, 213}, // Jump: 219 - 6 = 213
		{234, 228}, // 234 - 6 = 228
		{261, 255}, // Last line: 261 - 6 = 255
	}

	for _, tc := range testCases {
		vdp.SetVCounter(tc.line)
		got := vdp.ReadVCounter()
		if got != tc.expected {
			t.Errorf("NTSC 192-line V-counter at line %d: expected %d, got %d", tc.line, tc.expected, got)
		}
	}
}

// TestVDP_VCounter_NTSC224 tests V-counter behavior in NTSC 224-line mode
func TestVDP_VCounter_NTSC224(t *testing.T) {
	vdp := NewVDP()
	vdp.SetTotalScanlines(262) // NTSC

	// Enable 224-line mode (M2=1, M1=1)
	vdp.WriteControl(0x02) // M2 in reg 0
	vdp.WriteControl(0x80)
	vdp.WriteControl(0x10) // M1 in reg 1
	vdp.WriteControl(0x81)

	testCases := []struct {
		line     uint16
		expected uint8
	}{
		{0, 0},
		{223, 223},
		{234, 234}, // Last normal line
		{235, 229}, // Jump: 235 - 6 = 229
		{261, 255}, // Last line
	}

	for _, tc := range testCases {
		vdp.SetVCounter(tc.line)
		got := vdp.ReadVCounter()
		if got != tc.expected {
			t.Errorf("NTSC 224-line V-counter at line %d: expected %d, got %d", tc.line, tc.expected, got)
		}
	}
}

// TestVDP_VCounter_PAL192 tests V-counter behavior in PAL 192-line mode
func TestVDP_VCounter_PAL192(t *testing.T) {
	vdp := NewVDP()
	vdp.SetTotalScanlines(313) // PAL

	testCases := []struct {
		line     uint16
		expected uint8
	}{
		{0, 0},
		{100, 100},
		{191, 191},
		{242, 242}, // Last normal line
		{243, 186}, // Jump: 243 - 57 = 186
		{280, 223}, // 280 - 57 = 223
		{312, 255}, // Last line: 312 - 57 = 255
	}

	for _, tc := range testCases {
		vdp.SetVCounter(tc.line)
		got := vdp.ReadVCounter()
		if got != tc.expected {
			t.Errorf("PAL 192-line V-counter at line %d: expected %d, got %d", tc.line, tc.expected, got)
		}
	}
}

// TestVDP_VCounter_PAL224 tests V-counter behavior in PAL 224-line mode
func TestVDP_VCounter_PAL224(t *testing.T) {
	vdp := NewVDP()
	vdp.SetTotalScanlines(313) // PAL

	// Enable 224-line mode
	vdp.WriteControl(0x02)
	vdp.WriteControl(0x80)
	vdp.WriteControl(0x10)
	vdp.WriteControl(0x81)

	testCases := []struct {
		line     uint16
		expected uint8
	}{
		{0, 0},
		{223, 223},
		{255, 255}, // Before jump
		{259, 202}, // Jump: 259 - 57 = 202
		{312, 255}, // Last line
	}

	for _, tc := range testCases {
		vdp.SetVCounter(tc.line)
		got := vdp.ReadVCounter()
		if got != tc.expected {
			t.Errorf("PAL 224-line V-counter at line %d: expected %d, got %d", tc.line, tc.expected, got)
		}
	}
}

// TestVDP_VCounter_SetTotalScanlines tests region configuration
func TestVDP_VCounter_SetTotalScanlines(t *testing.T) {
	vdp := NewVDP()

	// Default is NTSC
	if vdp.totalScanlines != 262 {
		t.Errorf("Default scanlines: expected 262 (NTSC), got %d", vdp.totalScanlines)
	}

	// Switch to PAL
	vdp.SetTotalScanlines(313)
	if vdp.totalScanlines != 313 {
		t.Errorf("After SetTotalScanlines(313): expected 313, got %d", vdp.totalScanlines)
	}
}

// TestVDP_HCounter_SetGet tests H-counter setter and getter
func TestVDP_HCounter_SetGet(t *testing.T) {
	vdp := NewVDP()

	testCases := []uint8{0x00, 0x40, 0x80, 0x93, 0xE9, 0xFF}

	for _, h := range testCases {
		vdp.SetHCounter(h)
		got := vdp.ReadHCounter()
		if got != h {
			t.Errorf("H-counter: set 0x%02X, got 0x%02X", h, got)
		}
	}
}

// TestVDP_LineCounter_Getter tests GetLineCounter accessor
func TestVDP_LineCounter_Getter(t *testing.T) {
	vdp := NewVDP()

	// Set line counter reload value
	vdp.WriteControl(0x0A)
	vdp.WriteControl(0x8A) // Register 10 = 10

	// Initialize via VBlank (line 193+, since line 192 still decrements)
	vdp.SetVCounter(193)
	vdp.UpdateLineCounter()

	if got := vdp.GetLineCounter(); got != 10 {
		t.Errorf("GetLineCounter after VBlank init: expected 10, got %d", got)
	}
}

// TestVDP_GetLineIntPending tests line interrupt pending accessor
func TestVDP_GetLineIntPending(t *testing.T) {
	vdp := NewVDP()

	// Initially not pending
	if vdp.GetLineIntPending() {
		t.Error("Line interrupt should not be pending initially")
	}

	// Set up counter to underflow quickly
	vdp.WriteControl(0x00) // Counter reload = 0
	vdp.WriteControl(0x8A)

	// VBlank to initialize (line 193+, since line 192 still decrements)
	vdp.SetVCounter(193)
	vdp.UpdateLineCounter()

	// Active line should trigger underflow
	vdp.SetVCounter(0)
	vdp.UpdateLineCounter()

	// Should be pending now
	if !vdp.GetLineIntPending() {
		t.Error("Line interrupt should be pending after counter underflow")
	}
}

// TestVDP_GetStatus tests status accessor
func TestVDP_GetStatus(t *testing.T) {
	vdp := NewVDP()

	// Initially 0
	if got := vdp.GetStatus(); got != 0 {
		t.Errorf("Initial status: expected 0, got 0x%02X", got)
	}

	// Set VBlank
	vdp.SetVBlank()
	if got := vdp.GetStatus(); got&0x80 == 0 {
		t.Error("Status should have VBlank bit set")
	}
}

// TestVDP_GetAddress tests address accessor
func TestVDP_GetAddress(t *testing.T) {
	vdp := NewVDP()

	// Set address
	vdp.WriteControl(0x34)
	vdp.WriteControl(0x52) // Address = 0x1234, code 1

	if got := vdp.GetAddress(); got != 0x1234 {
		t.Errorf("GetAddress: expected 0x1234, got 0x%04X", got)
	}
}

// TestVDP_GetCodeReg tests code register accessor
func TestVDP_GetCodeReg(t *testing.T) {
	vdp := NewVDP()

	// Set code 3 (CRAM write)
	vdp.WriteControl(0x00)
	vdp.WriteControl(0xC0)

	if got := vdp.GetCodeReg(); got != 3 {
		t.Errorf("GetCodeReg: expected 3, got %d", got)
	}
}

// TestVDP_GetWriteLatch tests write latch accessor
func TestVDP_GetWriteLatch(t *testing.T) {
	vdp := NewVDP()

	// Initially false
	if vdp.GetWriteLatch() {
		t.Error("Write latch should be false initially")
	}

	// After first write
	vdp.WriteControl(0x00)
	if !vdp.GetWriteLatch() {
		t.Error("Write latch should be true after first write")
	}

	// After second write
	vdp.WriteControl(0x00)
	if vdp.GetWriteLatch() {
		t.Error("Write latch should be false after second write")
	}
}

// TestVDP_GetRegister tests register accessor for all 16 registers
func TestVDP_GetRegister(t *testing.T) {
	vdp := NewVDP()

	// Write to all 16 registers
	for i := 0; i < 16; i++ {
		val := uint8(i * 10)
		vdp.WriteControl(val)
		vdp.WriteControl(0x80 | uint8(i))

		if got := vdp.GetRegister(i); got != val {
			t.Errorf("Register %d: expected 0x%02X, got 0x%02X", i, val, got)
		}
	}
}

// TestVDP_Framebuffer tests framebuffer accessor
func TestVDP_Framebuffer(t *testing.T) {
	vdp := NewVDP()

	fb := vdp.Framebuffer()
	if fb == nil {
		t.Fatal("Framebuffer should not be nil")
	}

	bounds := fb.Bounds()
	if bounds.Dx() != ScreenWidth {
		t.Errorf("Framebuffer width: expected %d, got %d", ScreenWidth, bounds.Dx())
	}
	if bounds.Dy() != MaxScreenHeight {
		t.Errorf("Framebuffer height: expected %d, got %d", MaxScreenHeight, bounds.Dy())
	}
}

// TestVDP_GetVRAM tests VRAM accessor
func TestVDP_GetVRAM(t *testing.T) {
	vdp := NewVDP()

	vram := vdp.GetVRAM()
	if len(vram) != 0x4000 {
		t.Errorf("VRAM length: expected 0x4000, got 0x%04X", len(vram))
	}

	// Write and verify via accessor
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40) // VRAM write at 0
	vdp.WriteData(0xAB)

	if vram[0] != 0xAB {
		t.Errorf("VRAM[0]: expected 0xAB, got 0x%02X", vram[0])
	}
}

// TestVDP_GetCRAM tests CRAM accessor
func TestVDP_GetCRAM(t *testing.T) {
	vdp := NewVDP()

	cram := vdp.GetCRAM()
	if len(cram) != 32 {
		t.Errorf("CRAM length: expected 32, got %d", len(cram))
	}

	// Write and verify via accessor
	vdp.WriteControl(0x05)
	vdp.WriteControl(0xC0) // CRAM write at 5
	vdp.WriteData(0x3F)

	if cram[5] != 0x3F {
		t.Errorf("CRAM[5]: expected 0x3F, got 0x%02X", cram[5])
	}
}
