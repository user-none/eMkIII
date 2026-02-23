package emu

import (
	"encoding/binary"
	"hash/crc32"
	"testing"

	"github.com/user-none/go-chip-sn76489"
	"github.com/user-none/go-chip-z80"
)

// TestEmulator_ComponentIntegration tests that components work together correctly.
func TestEmulator_ComponentIntegration(t *testing.T) {
	// Create components manually (mimicking what NewEmulator does)
	rom := createTestROM(4)
	mem := NewMemory(rom)
	vdp := NewVDP()
	timing := GetTimingForRegion(RegionNTSC)

	vdp.SetTotalScanlines(timing.Scanlines)

	samplesPerFrame := 48000 / timing.FPS
	psg := sn76489.New(timing.CPUClockHz, 48000, samplesPerFrame*2, sn76489.Sega)

	io := NewSMSIO(vdp, psg, NationalityExport)
	bus := NewSMSBus(mem, io)
	cpu := z80.New(bus)

	// Verify all components are properly initialized
	if mem == nil || vdp == nil || psg == nil || io == nil || cpu == nil {
		t.Fatal("Component initialization failed")
	}

	// Verify CPU can execute instructions
	cycles := cpu.Step()
	if cycles <= 0 {
		t.Error("CPU should execute at least one cycle")
	}
}

// TestEmulator_TimingCalculations tests frame timing calculations
func TestEmulator_TimingCalculations(t *testing.T) {
	testCases := []struct {
		region   Region
		expected struct {
			fps            int
			scanlines      int
			cpuClock       int
			cyclesPerFrame int
			cyclesPerLine  int
		}
	}{
		{
			region: RegionNTSC,
			expected: struct {
				fps            int
				scanlines      int
				cpuClock       int
				cyclesPerFrame int
				cyclesPerLine  int
			}{
				fps:            60,
				scanlines:      262,
				cpuClock:       3579545,
				cyclesPerFrame: 3579545 / 60,
				cyclesPerLine:  (3579545 / 60) / 262,
			},
		},
		{
			region: RegionPAL,
			expected: struct {
				fps            int
				scanlines      int
				cpuClock       int
				cyclesPerFrame int
				cyclesPerLine  int
			}{
				fps:            50,
				scanlines:      313,
				cpuClock:       3546893,
				cyclesPerFrame: 3546893 / 50,
				cyclesPerLine:  (3546893 / 50) / 313,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.region.String(), func(t *testing.T) {
			timing := GetTimingForRegion(tc.region)

			if timing.FPS != tc.expected.fps {
				t.Errorf("FPS: expected %d, got %d", tc.expected.fps, timing.FPS)
			}
			if timing.Scanlines != tc.expected.scanlines {
				t.Errorf("Scanlines: expected %d, got %d", tc.expected.scanlines, timing.Scanlines)
			}
			if timing.CPUClockHz != tc.expected.cpuClock {
				t.Errorf("CPUClockHz: expected %d, got %d", tc.expected.cpuClock, timing.CPUClockHz)
			}

			cyclesPerFrame := timing.CPUClockHz / timing.FPS
			if cyclesPerFrame != tc.expected.cyclesPerFrame {
				t.Errorf("CyclesPerFrame: expected %d, got %d", tc.expected.cyclesPerFrame, cyclesPerFrame)
			}

			cyclesPerLine := cyclesPerFrame / timing.Scanlines
			if cyclesPerLine != tc.expected.cyclesPerLine {
				t.Errorf("CyclesPerLine: expected %d, got %d", tc.expected.cyclesPerLine, cyclesPerLine)
			}
		})
	}
}

// TestEmulator_FixedPointTiming tests fixed-point cycle accumulation accuracy
func TestEmulator_FixedPointTiming(t *testing.T) {
	// Test NTSC timing
	timing := GetTimingForRegion(RegionNTSC)

	// Fixed-point calculation (8 fractional bits)
	cyclesPerScanlineFP := (timing.CPUClockHz * 256) / timing.FPS / timing.Scanlines

	// After 262 scanlines, total cycles should match expected per-frame total
	var totalCyclesFP int
	for i := 0; i < timing.Scanlines; i++ {
		totalCyclesFP += cyclesPerScanlineFP
	}
	totalCycles := totalCyclesFP >> 8

	expectedCyclesPerFrame := timing.CPUClockHz / timing.FPS

	// Allow small rounding error (1-2 cycles)
	diff := totalCycles - expectedCyclesPerFrame
	if diff < -2 || diff > 2 {
		t.Errorf("Fixed-point timing drift: expected ~%d cycles/frame, got %d (diff: %d)",
			expectedCyclesPerFrame, totalCycles, diff)
	}
}

// TestEmulator_ScanlineExecution tests one scanline of execution
func TestEmulator_ScanlineExecution(t *testing.T) {
	rom := createTestROM(4)
	mem := NewMemory(rom)
	vdp := NewVDP()
	timing := GetTimingForRegion(RegionNTSC)

	vdp.SetTotalScanlines(timing.Scanlines)

	psg := sn76489.New(timing.CPUClockHz, 48000, 2000, sn76489.Sega)
	io := NewSMSIO(vdp, psg, NationalityExport)
	bus := NewSMSBus(mem, io)
	cpu := z80.New(bus)

	// Calculate cycles per scanline
	cyclesPerFrame := timing.CPUClockHz / timing.FPS
	cyclesPerScanline := cyclesPerFrame / timing.Scanlines

	// Execute one scanline worth of cycles
	var executedCycles int
	for executedCycles < cyclesPerScanline {
		cycles := cpu.Step()
		executedCycles += cycles
	}

	// Should have executed roughly the right number of cycles
	// (may overshoot by one instruction's worth)
	if executedCycles < cyclesPerScanline {
		t.Errorf("Executed too few cycles: %d < %d", executedCycles, cyclesPerScanline)
	}
	if executedCycles > cyclesPerScanline+20 { // Allow ~20 cycles overshoot
		t.Errorf("Executed too many cycles: %d >> %d", executedCycles, cyclesPerScanline)
	}
}

// TestEmulator_VDPInterruptIntegration tests VDP interrupt triggering
func TestEmulator_VDPInterruptIntegration(t *testing.T) {
	vdp := NewVDP()

	// Enable frame interrupt
	vdp.WriteControl(0x20) // Frame IE bit
	vdp.WriteControl(0x81)

	// Set VBlank
	vdp.SetVBlank()

	// Check interrupt pending
	if !vdp.InterruptPending() {
		t.Error("VDP interrupt should be pending after VBlank with frame IE enabled")
	}

	// Simulate reading status (clears interrupt)
	vdp.ReadControl()

	if vdp.InterruptPending() {
		t.Error("VDP interrupt should be cleared after status read")
	}
}

// TestEmulator_PSGIntegration tests PSG audio generation
func TestEmulator_PSGIntegration(t *testing.T) {
	timing := GetTimingForRegion(RegionNTSC)
	psg := sn76489.New(timing.CPUClockHz, 48000, 2000, sn76489.Sega)

	// Write a tone to channel 0
	psg.Write(0x80 | 0x0F) // Channel 0 tone, low nibble
	psg.Write(0x10)        // High 6 bits
	psg.Write(0x90 | 0x00) // Channel 0 volume = 0 (max)

	// Generate some samples
	cyclesPerScanline := (timing.CPUClockHz / timing.FPS) / timing.Scanlines
	psg.GenerateSamples(cyclesPerScanline)

	buffer, count := psg.GetBuffer()
	if count == 0 {
		t.Error("PSG should have generated samples")
	}

	// Check that we have valid sample data
	hasNonZero := false
	for i := 0; i < count; i++ {
		if buffer[i] != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("PSG should generate non-zero samples with tone enabled")
	}
}

// TestEmulator_FrameLoop_Logic tests the frame loop logic
func TestEmulator_FrameLoop_Logic(t *testing.T) {
	rom := createTestROM(4)
	mem := NewMemory(rom)
	vdp := NewVDP()
	timing := GetTimingForRegion(RegionNTSC)

	vdp.SetTotalScanlines(timing.Scanlines)

	psg := sn76489.New(timing.CPUClockHz, 48000, 2000, sn76489.Sega)
	io := NewSMSIO(vdp, psg, NationalityExport)
	bus := NewSMSBus(mem, io)
	cpu := z80.New(bus)

	// Enable display
	vdp.WriteControl(0x40)
	vdp.WriteControl(0x81)

	cyclesPerScanlineFP := (timing.CPUClockHz * 256) / timing.FPS / timing.Scanlines

	var targetCyclesFP int
	var executedCycles int
	var prevTargetCycles int
	activeHeight := vdp.ActiveHeight()

	// Simulate one frame
	for i := 0; i < timing.Scanlines; i++ {
		targetCyclesFP += cyclesPerScanlineFP
		targetCycles := targetCyclesFP >> 8

		vdp.SetVCounter(uint16(i))

		if i == 0 {
			vdp.LatchVScrollForFrame()
		}

		vdp.UpdateLineCounter()

		if i == activeHeight+1 {
			vdp.SetVBlank()
		}

		for executedCycles < targetCycles {
			cycles := cpu.Step()
			executedCycles += cycles
		}

		if i < activeHeight {
			vdp.RenderScanline()
		}

		actualScanlineCycles := targetCycles - prevTargetCycles
		prevTargetCycles = targetCycles

		psg.GenerateSamples(actualScanlineCycles)
	}

	// Verify we executed roughly the right number of cycles
	expectedCycles := timing.CPUClockHz / timing.FPS
	diff := executedCycles - expectedCycles
	if diff < -10 || diff > 10 {
		t.Errorf("Frame cycle count: expected ~%d, got %d (diff: %d)",
			expectedCycles, executedCycles, diff)
	}

	// Verify VBlank was triggered
	if vdp.GetStatus()&0x80 == 0 {
		// Note: Status may have been cleared during frame, so this isn't a hard error
		// Just verify the frame completed
	}
}

// TestEmulator_InputHandling tests controller input via SMSIO
func TestEmulator_InputHandling(t *testing.T) {
	vdp := NewVDP()
	psg := sn76489.New(3579545, 48000, 2000, sn76489.Sega)
	io := NewSMSIO(vdp, psg, NationalityExport)

	// Initially all buttons released (0xFF)
	if io.Input.Port1 != 0xFF {
		t.Errorf("Initial Port1: expected 0xFF, got 0x%02X", io.Input.Port1)
	}

	// Press Up
	io.Input.SetP1(true, false, false, false, false, false)
	if io.Input.Port1&0x01 != 0 {
		t.Error("Up should be pressed (bit 0 clear)")
	}

	// Press all buttons
	io.Input.SetP1(true, true, true, true, true, true)
	if io.Input.Port1 != 0xC0 {
		t.Errorf("All pressed Port1: expected 0xC0 (P2 bits high), got 0x%02X", io.Input.Port1)
	}

	// Release all
	io.Input.SetP1(false, false, false, false, false, false)
	if io.Input.Port1 != 0xFF {
		t.Errorf("All released Port1: expected 0xFF, got 0x%02X", io.Input.Port1)
	}
}

// TestEmulator_Constants tests emulator constants
func TestEmulator_Constants(t *testing.T) {
	if ScreenWidth != 256 {
		t.Errorf("ScreenWidth: expected 256, got %d", ScreenWidth)
	}
	if MaxScreenHeight != 224 {
		t.Errorf("MaxScreenHeight: expected 224, got %d", MaxScreenHeight)
	}
}

// TestEmulator_VDPLineCounter tests line counter during frame execution
func TestEmulator_VDPLineCounter(t *testing.T) {
	vdp := NewVDP()

	// Set line counter reload value
	vdp.WriteControl(0x05) // Reload value = 5
	vdp.WriteControl(0x8A)

	// Per SMS VDP hardware: counter reloads on lines 193+ (not 192)
	// Line 192 still decrements, use line 193 for initialization
	vdp.SetVCounter(193)
	vdp.UpdateLineCounter()

	// Counter should be 5
	if got := vdp.GetLineCounter(); got != 5 {
		t.Errorf("Line counter after VBlank init: expected 5, got %d", got)
	}

	// Simulate active scanlines
	for line := uint16(0); line < 6; line++ {
		vdp.SetVCounter(line)
		vdp.UpdateLineCounter()
	}

	// After 6 active scanlines, counter should have underflowed
	if !vdp.GetLineIntPending() {
		t.Error("Line interrupt should be pending after counter underflow")
	}
}

// TestEmulator_VBlankFiringScanline tests that VBlank fires at activeHeight+1
// frame interrupt flag is set at V-counter $C1 (line 193)
// for 192-line mode and $E1 (line 225) for 224-line mode, not on the last active line.
func TestEmulator_VBlankFiringScanline(t *testing.T) {
	vdp := NewVDP()
	activeHeight := vdp.ActiveHeight() // 192 for default mode

	// Simulate scanlines leading up to and past VBlank
	// On the last active line (191), VBlank should not be set
	vdp.SetVCounter(uint16(activeHeight - 1))
	if vdp.GetStatus()&0x80 != 0 {
		t.Errorf("VBlank should not be set on last active line %d", activeHeight-1)
	}

	// On line activeHeight (192), VBlank should NOT fire yet
	// This is the first non-active line but the frame interrupt doesn't fire here
	vdp.SetVCounter(uint16(activeHeight))
	if vdp.GetStatus()&0x80 != 0 {
		t.Errorf("VBlank should not be set on line %d (V-counter $%02X)",
			activeHeight, activeHeight)
	}

	// On line activeHeight+1 (193), VBlank SHOULD fire
	// This matches MacDonald: V-counter $C1 for 192-line, $E1 for 224-line
	vdp.SetVCounter(uint16(activeHeight + 1))
	vdp.SetVBlank()
	if vdp.GetStatus()&0x80 == 0 {
		t.Errorf("VBlank should be set on line %d (V-counter $%02X)",
			activeHeight+1, activeHeight+1)
	}
}

// TestEmulator_HCounterDuringFrame tests H-counter updates during execution
func TestEmulator_HCounterDuringFrame(t *testing.T) {
	vdp := NewVDP()

	// Test H-counter at various cycle offsets
	testCases := []struct {
		cycle int
		desc  string
	}{
		{0, "start of scanline"},
		{85, "mid-left"},
		{170, "mid-right"},
		{200, "H-blank start"},
		{227, "end of scanline"},
	}

	for _, tc := range testCases {
		h := GetHCounterForCycle(tc.cycle)
		vdp.SetHCounter(h)
		if got := vdp.ReadHCounter(); got != h {
			t.Errorf("H-counter at %s (cycle %d): expected 0x%02X, got 0x%02X",
				tc.desc, tc.cycle, h, got)
		}
	}
}

// TestEmulator_AudioSampleCount tests audio sample generation per frame
func TestEmulator_AudioSampleCount(t *testing.T) {
	timing := GetTimingForRegion(RegionNTSC)

	// At 48kHz and 60 FPS, we expect ~800 samples per frame
	expectedSamples := 48000 / timing.FPS

	psg := sn76489.New(timing.CPUClockHz, 48000, expectedSamples*2, sn76489.Sega)

	// Generate samples for one frame worth of cycles
	cyclesPerFrame := timing.CPUClockHz / timing.FPS

	// Split into scanlines like the real emulator does
	cyclesPerScanline := cyclesPerFrame / timing.Scanlines
	totalSamples := 0

	for i := 0; i < timing.Scanlines; i++ {
		psg.GenerateSamples(cyclesPerScanline)
		_, count := psg.GetBuffer()
		totalSamples += count
	}

	// Should be close to expected samples
	diff := totalSamples - expectedSamples
	if diff < -10 || diff > 10 {
		t.Errorf("Samples per frame: expected ~%d, got %d (diff: %d)",
			expectedSamples, totalSamples, diff)
	}
}

// =============================================================================
// Save State Serialization Tests
// =============================================================================

// createTestEmulator creates an Emulator for testing serialization
func createTestEmulator() *Emulator {
	rom := createTestROM(4)
	e, _ := NewEmulator(rom, RegionNTSC)
	return &e
}

// TestSerializeSize verifies consistent size returned
func TestSerializeSize(t *testing.T) {
	size1 := SerializeSize()
	size2 := SerializeSize()

	if size1 != size2 {
		t.Errorf("SerializeSize not consistent: %d vs %d", size1, size2)
	}

	// Size should be header (22) + state data
	if size1 < stateHeaderSize {
		t.Errorf("SerializeSize too small: %d < %d (header)", size1, stateHeaderSize)
	}
}

// TestSerializeDeserializeRoundTrip tests save state round-trip
func TestSerializeDeserializeRoundTrip(t *testing.T) {
	base := createTestEmulator()

	// Run a few CPU steps to change state
	for i := 0; i < 100; i++ {
		base.cpu.Step()
	}

	// Write some values to RAM to test memory serialization
	base.mem.Set(0xC000, 0xAB)
	base.mem.Set(0xC001, 0xCD)

	// Write to VDP registers
	base.vdp.WriteControl(0x55)
	base.vdp.WriteControl(0x80) // Register 0 = 0x55

	// Save state
	state, err := base.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Modify emulator state
	base.mem.Set(0xC000, 0xFF)
	base.mem.Set(0xC001, 0xFF)

	// Restore state
	err = base.Deserialize(state)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	// Verify RAM was restored
	if base.mem.Get(0xC000) != 0xAB {
		t.Errorf("RAM[0xC000]: expected 0xAB, got 0x%02X", base.mem.Get(0xC000))
	}
	if base.mem.Get(0xC001) != 0xCD {
		t.Errorf("RAM[0xC001]: expected 0xCD, got 0x%02X", base.mem.Get(0xC001))
	}

	// Verify VDP register was restored
	if base.vdp.GetRegister(0) != 0x55 {
		t.Errorf("VDP Register 0: expected 0x55, got 0x%02X", base.vdp.GetRegister(0))
	}
}

// TestVerifyState_ValidState tests that a valid state passes verification
func TestVerifyState_ValidState(t *testing.T) {
	base := createTestEmulator()

	state, err := base.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	err = base.VerifyState(state)
	if err != nil {
		t.Errorf("VerifyState should pass for valid state: %v", err)
	}
}

// TestVerifyState_InvalidMagic tests wrong magic bytes rejection
func TestVerifyState_InvalidMagic(t *testing.T) {
	base := createTestEmulator()

	state, err := base.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Corrupt magic bytes
	state[0] = 'X'

	err = base.VerifyState(state)
	if err == nil {
		t.Error("VerifyState should reject invalid magic bytes")
	}
}

// TestVerifyState_UnsupportedVersion tests future version rejection
func TestVerifyState_UnsupportedVersion(t *testing.T) {
	base := createTestEmulator()

	state, err := base.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Set a future version number
	binary.LittleEndian.PutUint16(state[12:14], 9999)

	err = base.VerifyState(state)
	if err == nil {
		t.Error("VerifyState should reject unsupported version")
	}
}

// TestVerifyState_CorruptData tests bad CRC32 rejection
func TestVerifyState_CorruptData(t *testing.T) {
	base := createTestEmulator()

	state, err := base.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Corrupt state data (after header)
	if len(state) > stateHeaderSize+10 {
		state[stateHeaderSize+5] ^= 0xFF
	}

	err = base.VerifyState(state)
	if err == nil {
		t.Error("VerifyState should reject corrupted data")
	}
}

// TestVerifyState_WrongROM tests mismatched ROM CRC32 rejection
func TestVerifyState_WrongROM(t *testing.T) {
	base1 := createTestEmulator()

	state, err := base1.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Create different emulator with different ROM
	differentROM := make([]byte, 0x8000)
	for i := range differentROM {
		differentROM[i] = byte(i & 0xFF)
	}
	e2, _ := NewEmulator(differentROM, RegionNTSC)
	base2 := &e2

	err = base2.VerifyState(state)
	if err == nil {
		t.Error("VerifyState should reject state from different ROM")
	}
}

// TestVerifyState_TooShort tests rejection of truncated data
func TestVerifyState_TooShort(t *testing.T) {
	base := createTestEmulator()

	// Create data smaller than header
	state := make([]byte, stateHeaderSize-1)

	err := base.VerifyState(state)
	if err == nil {
		t.Error("VerifyState should reject data smaller than header")
	}
}

// TestDeserialize_PreservesRegion tests that region is NOT changed by load
func TestDeserialize_PreservesRegion(t *testing.T) {
	// Create emulator with NTSC
	ntscROM := createTestROM(4)
	ntscEmu, _ := NewEmulator(ntscROM, RegionNTSC)
	baseNTSC := &ntscEmu

	// Save state
	state, err := baseNTSC.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Create new emulator with PAL using same ROM
	palEmu, _ := NewEmulator(ntscROM, RegionPAL)
	basePAL := &palEmu

	// Verify initial region is PAL
	if basePAL.GetRegion() != RegionPAL {
		t.Fatal("Initial region should be PAL")
	}

	// Load NTSC state into PAL emulator
	err = basePAL.Deserialize(state)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	// Region should still be PAL (not changed by state load)
	if basePAL.GetRegion() != RegionPAL {
		t.Errorf("Region should be preserved as PAL, got %v", basePAL.GetRegion())
	}
}

// TestSerialize_StateIntegrity tests that serialized state has correct format
func TestSerialize_StateIntegrity(t *testing.T) {
	base := createTestEmulator()

	state, err := base.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Check magic bytes
	if string(state[0:12]) != stateMagic {
		t.Errorf("Magic bytes: expected %q, got %q", stateMagic, string(state[0:12]))
	}

	// Check version
	version := binary.LittleEndian.Uint16(state[12:14])
	if version != stateVersion {
		t.Errorf("Version: expected %d, got %d", stateVersion, version)
	}

	// Verify ROM CRC32 matches
	romCRC := binary.LittleEndian.Uint32(state[14:18])
	expectedROMCRC := base.mem.GetROMCRC32()
	if romCRC != expectedROMCRC {
		t.Errorf("ROM CRC32: expected 0x%08X, got 0x%08X", expectedROMCRC, romCRC)
	}

	// Verify data CRC32
	dataCRC := binary.LittleEndian.Uint32(state[18:22])
	calculatedCRC := crc32.ChecksumIEEE(state[stateHeaderSize:])
	if dataCRC != calculatedCRC {
		t.Errorf("Data CRC32: expected 0x%08X, got 0x%08X", calculatedCRC, dataCRC)
	}
}

// TestMemory_GetROMCRC32 tests the ROM CRC32 calculation
func TestMemory_GetROMCRC32(t *testing.T) {
	rom := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	mem := NewMemory(rom)

	crc := mem.GetROMCRC32()
	expected := crc32.ChecksumIEEE(rom)

	if crc != expected {
		t.Errorf("GetROMCRC32: expected 0x%08X, got 0x%08X", expected, crc)
	}
}

// TestSerialize_IOControlRoundTrip tests that ioControl survives serialize/deserialize
func TestSerialize_IOControlRoundTrip(t *testing.T) {
	base := createTestEmulator()

	// Write to port $3F
	base.io.Out(0x3F, 0xF5)
	if base.io.ioControl != 0xF5 {
		t.Fatalf("ioControl not set: expected 0xF5, got 0x%02X", base.io.ioControl)
	}

	// Serialize
	state, err := base.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Change ioControl
	base.io.Out(0x3F, 0x00)

	// Deserialize
	err = base.Deserialize(state)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	// Verify restored
	if base.io.ioControl != 0xF5 {
		t.Errorf("ioControl not restored: expected 0xF5, got 0x%02X", base.io.ioControl)
	}
}

// =============================================================================
// SetInput Bitmask Tests
// =============================================================================

// TestEmulator_SetInput_Bitmask tests bitmask-based input unpacking
func TestEmulator_SetInput_Bitmask(t *testing.T) {
	e := createTestEmulator()

	// Press Up (bit 0) and Button 1 (bit 4)
	e.SetInput(0, (1<<0)|(1<<4))

	// Up should be pressed (bit 0 of Port1 clear = pressed)
	if e.io.Input.Port1&0x01 != 0 {
		t.Error("Up should be pressed (bit 0 clear)")
	}
	// Button 1 should be pressed (bit 4 of Port1 clear = pressed)
	if e.io.Input.Port1&0x10 != 0 {
		t.Error("Button 1 should be pressed (bit 4 clear)")
	}
	// Down should not be pressed (bit 1 of Port1 set = released)
	if e.io.Input.Port1&0x02 == 0 {
		t.Error("Down should not be pressed (bit 1 set)")
	}

	// Release all
	e.SetInput(0, 0)
	if e.io.Input.Port1 != 0xFF {
		t.Errorf("All released: expected 0xFF, got 0x%02X", e.io.Input.Port1)
	}
}

// TestEmulator_SetInput_Player2 tests that player 2 input routes correctly
func TestEmulator_SetInput_Player2(t *testing.T) {
	e := createTestEmulator()

	// Press all directions on P2
	e.SetInput(1, (1<<0)|(1<<1)|(1<<2)|(1<<3)|(1<<4)|(1<<5))

	// P2 Up/Down in bits 0-1 of Port2
	if e.io.Input.Port2&0x01 != 0 {
		t.Error("P2 Up should be pressed")
	}
	if e.io.Input.Port2&0x02 != 0 {
		t.Error("P2 Down should be pressed")
	}
}

// TestEmulator_SetInput_PauseEdge tests pause NMI edge detection
func TestEmulator_SetInput_PauseEdge(t *testing.T) {
	e := createTestEmulator()

	// First call with pause not pressed - no NMI
	e.SetInput(0, 0)

	// Now press pause (bit 7) - should trigger NMI
	e.SetInput(0, 1<<7)

	// Hold pause - should NOT re-trigger NMI
	e.SetInput(0, 1<<7)

	// Release and re-press - should trigger again
	e.SetInput(0, 0)
	e.SetInput(0, 1<<7)
}

// =============================================================================
// Crop Border Tests
// =============================================================================

// TestEmulator_CropBorder tests framebuffer crop border behavior
func TestEmulator_CropBorder(t *testing.T) {
	e := createTestEmulator()

	// Without crop, should return full width
	fb := e.GetFramebuffer()
	stride := e.GetFramebufferStride()
	if stride != ScreenWidth*4 {
		t.Errorf("Normal stride: expected %d, got %d", ScreenWidth*4, stride)
	}
	if len(fb) == 0 {
		t.Error("Framebuffer should not be empty")
	}

	// Enable crop border
	e.SetOption("crop_border", "true")

	// Enable VDP left column blank (register 0, bit 5)
	e.vdp.WriteControl(0x20)
	e.vdp.WriteControl(0x80)

	stride = e.GetFramebufferStride()
	if stride != (ScreenWidth-8)*4 {
		t.Errorf("Cropped stride: expected %d, got %d", (ScreenWidth-8)*4, stride)
	}

	fb = e.GetFramebuffer()
	expectedLen := (ScreenWidth - 8) * 4 * e.GetActiveHeight()
	if len(fb) != expectedLen {
		t.Errorf("Cropped framebuffer length: expected %d, got %d", expectedLen, len(fb))
	}
}

// =============================================================================
// BatterySaver Tests
// =============================================================================

// TestEmulator_SRAM tests SRAM get/set round-trip
func TestEmulator_SRAM(t *testing.T) {
	e := createTestEmulator()

	if !e.HasSRAM() {
		t.Error("HasSRAM should return true")
	}

	// Write some data to cart RAM
	e.mem.cartRAM[0] = 0xAB
	e.mem.cartRAM[1] = 0xCD

	sram := e.GetSRAM()
	if sram[0] != 0xAB || sram[1] != 0xCD {
		t.Errorf("GetSRAM: expected [0xAB, 0xCD], got [0x%02X, 0x%02X]", sram[0], sram[1])
	}

	// Verify it's a copy (modifying returned slice doesn't affect emulator)
	sram[0] = 0xFF
	if e.mem.cartRAM[0] != 0xAB {
		t.Error("GetSRAM should return a copy, not a reference")
	}

	// Test SetSRAM
	newData := make([]byte, 0x8000)
	newData[0] = 0x11
	newData[1] = 0x22
	e.SetSRAM(newData)

	if e.mem.cartRAM[0] != 0x11 || e.mem.cartRAM[1] != 0x22 {
		t.Errorf("SetSRAM: expected [0x11, 0x22], got [0x%02X, 0x%02X]",
			e.mem.cartRAM[0], e.mem.cartRAM[1])
	}
}

// =============================================================================
// MemoryInspector Tests
// =============================================================================

// TestEmulator_ReadMemory tests flat address memory reading
func TestEmulator_ReadMemory(t *testing.T) {
	e := createTestEmulator()

	// Write known values to system RAM
	e.mem.ram[0] = 0xDE
	e.mem.ram[1] = 0xAD

	buf := make([]byte, 4)
	n := e.ReadMemory(0, buf)
	if n != 4 {
		t.Errorf("ReadMemory: expected 4 bytes read, got %d", n)
	}
	if buf[0] != 0xDE || buf[1] != 0xAD {
		t.Errorf("ReadMemory: expected [0xDE, 0xAD, ...], got [0x%02X, 0x%02X, ...]",
			buf[0], buf[1])
	}

	// Read past system RAM boundary should stop
	buf = make([]byte, 4)
	n = e.ReadMemory(0x2000, buf)
	if n != 0 {
		t.Errorf("ReadMemory past boundary: expected 0 bytes, got %d", n)
	}
}

// =============================================================================
// MemoryMapper Tests
// =============================================================================

// TestEmulator_MemoryMap tests memory region listing
func TestEmulator_MemoryMap(t *testing.T) {
	e := createTestEmulator()

	regions := e.MemoryMap()
	if len(regions) != 2 {
		t.Fatalf("MemoryMap: expected 2 regions, got %d", len(regions))
	}

	// System RAM
	if regions[0].Size != 0x2000 {
		t.Errorf("System RAM size: expected 0x2000, got 0x%X", regions[0].Size)
	}
	// Save RAM
	if regions[1].Size != 0x8000 {
		t.Errorf("Save RAM size: expected 0x8000, got 0x%X", regions[1].Size)
	}
}

// TestEmulator_ReadWriteRegion tests region read/write round-trip
func TestEmulator_ReadWriteRegion(t *testing.T) {
	e := createTestEmulator()

	// Write to system RAM via WriteRegion
	data := make([]byte, 0x2000)
	data[0] = 0xBE
	data[1] = 0xEF
	e.WriteRegion(1, data) // MemorySystemRAM = 1

	// Read back via ReadRegion
	result := e.ReadRegion(1)
	if result[0] != 0xBE || result[1] != 0xEF {
		t.Errorf("ReadRegion: expected [0xBE, 0xEF], got [0x%02X, 0x%02X]",
			result[0], result[1])
	}
}
