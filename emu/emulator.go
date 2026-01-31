package emu

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"math"

	"github.com/koron-go/z80"
)

const (
	ScreenWidth     = 256
	MaxScreenHeight = 224
	sampleRate      = 48000
)

// Save state format constants
const (
	stateVersion    = 1
	stateMagic      = "eMkIIISState"
	stateHeaderSize = 22 // magic(12) + version(2) + romCRC(4) + dataCRC(4)
)

// EmulatorBase contains fields shared by all platform implementations
type EmulatorBase struct {
	cpu                 *CycleZ80
	mem                 *Memory
	vdp                 *VDP
	psg                 *PSG
	io                  *SMSIO
	cyclesPerFrame      int
	cyclesPerScanline   int
	cyclesPerScanlineFP int // Fixed-point (16 fractional bits) for accurate timing

	// Region timing
	region    Region
	timing    RegionTiming
	scanlines int

	// Audio buffer for accumulating samples (shared between builds)
	audioBuffer []int16
}

// initEmulatorBase creates and initializes the shared emulator components
func initEmulatorBase(rom []byte, region Region) EmulatorBase {
	mem := NewMemory(rom)
	vdp := NewVDP()

	timing := GetTimingForRegion(region)
	vdp.SetTotalScanlines(timing.Scanlines)

	samplesPerFrame := sampleRate / timing.FPS
	psg := NewPSG(timing.CPUClockHz, sampleRate, samplesPerFrame*2)

	io := NewSMSIO(vdp, psg)
	cpu := NewCycleZ80(mem, io)

	cyclesPerFrame := timing.CPUClockHz / timing.FPS
	cyclesPerScanline := cyclesPerFrame / timing.Scanlines
	cyclesPerScanlineFP := (timing.CPUClockHz * 65536) / timing.FPS / timing.Scanlines

	return EmulatorBase{
		cpu:                 cpu,
		mem:                 mem,
		vdp:                 vdp,
		psg:                 psg,
		io:                  io,
		cyclesPerFrame:      cyclesPerFrame,
		cyclesPerScanline:   cyclesPerScanline,
		cyclesPerScanlineFP: cyclesPerScanlineFP,
		region:              region,
		timing:              timing,
		scanlines:           timing.Scanlines,
	}
}

// runScanlines executes one frame of CPU/VDP/PSG emulation and returns audio samples
func (e *EmulatorBase) runScanlines() []float32 {
	activeHeight := e.vdp.ActiveHeight()

	var targetCyclesFP int = 0
	var executedCycles int = 0
	var prevTargetCycles int = 0

	// Collect all audio samples for the frame
	frameSamples := make([]float32, 0, 900) // ~800 samples per frame at 48kHz/60fps

	for i := 0; i < e.scanlines; i++ {
		targetCyclesFP += e.cyclesPerScanlineFP
		targetCycles := targetCyclesFP >> 16

		e.vdp.SetVCounter(uint16(i))

		if i == 0 {
			e.vdp.LatchVScrollForFrame()
		}

		// Flags to track per-scanline interrupt triggers
		lineIntChecked := false
		vblankChecked := false
		isVBlankLine := (i == activeHeight)

		scanlineCycles := 0
		for executedCycles < targetCycles {
			scanlineProgress := executedCycles - prevTargetCycles

			// Check VBlank at cycle 0 (only on vblank line)
			if !vblankChecked && isVBlankLine && scanlineProgress >= VBlankInterruptCycle {
				e.vdp.SetVBlank()
				vblankChecked = true
				// Check interrupt state after VBlank trigger
				if e.vdp.InterruptPending() {
					e.cpu.SetInterrupt(z80.IM1Interrupt())
				} else {
					e.cpu.ClearInterrupt()
				}
			}

			// Check line interrupt at cycle 8
			if !lineIntChecked && scanlineProgress >= LineInterruptCycle {
				e.vdp.UpdateLineCounter()
				lineIntChecked = true
				// Check interrupt state after line counter update
				if e.vdp.InterruptPending() {
					e.cpu.SetInterrupt(z80.IM1Interrupt())
				} else {
					e.cpu.ClearInterrupt()
				}
			}

			e.vdp.SetHCounter(GetHCounterForCycle(scanlineProgress))
			cycles := e.cpu.Step()
			executedCycles += cycles
			scanlineCycles += cycles
		}

		// Handle any interrupt checks that didn't trigger during short scanlines
		if !lineIntChecked {
			e.vdp.UpdateLineCounter()
		}
		if !vblankChecked && isVBlankLine {
			e.vdp.SetVBlank()
		}

		if i < activeHeight {
			e.vdp.RenderScanline()
		}

		prevTargetCycles = targetCycles

		e.psg.GenerateSamples(scanlineCycles)
		buffer, count := e.psg.GetBuffer()
		if count > 0 {
			frameSamples = append(frameSamples, buffer[:count]...)
		}
	}

	return frameSamples
}

// SetInput sets Player 1 controller state from external source
func (e *EmulatorBase) SetInput(up, down, left, right, btn1, btn2 bool) {
	e.io.Input.SetP1(up, down, left, right, btn1, btn2)
}

// SetInputP2 sets Player 2 controller state from external source
func (e *EmulatorBase) SetInputP2(up, down, left, right, btn1, btn2 bool) {
	e.io.Input.SetP2(up, down, left, right, btn1, btn2)
}

// GetFramebuffer returns raw RGBA pixel data for current frame
func (e *EmulatorBase) GetFramebuffer() []byte {
	return e.vdp.framebuffer.Pix
}

// GetFramebufferStride returns the stride (bytes per row) of the framebuffer
func (e *EmulatorBase) GetFramebufferStride() int {
	return e.vdp.framebuffer.Stride
}

// GetActiveHeight returns the current active display height (192 or 224)
func (e *EmulatorBase) GetActiveHeight() int {
	return e.vdp.ActiveHeight()
}

// GetRegion returns the emulator's region setting
func (e *EmulatorBase) GetRegion() Region {
	return e.region
}

// GetTiming returns the region timing configuration
func (e *EmulatorBase) GetTiming() RegionTiming {
	return e.timing
}

// SetRegion updates the emulator's region configuration
func (e *EmulatorBase) SetRegion(region Region) {
	e.region = region
	e.timing = GetTimingForRegion(region)
	e.scanlines = e.timing.Scanlines
	e.vdp.SetTotalScanlines(e.timing.Scanlines)
	e.cyclesPerFrame = e.timing.CPUClockHz / e.timing.FPS
	e.cyclesPerScanline = e.cyclesPerFrame / e.timing.Scanlines
	e.cyclesPerScanlineFP = (e.timing.CPUClockHz * 65536) / e.timing.FPS / e.timing.Scanlines
}

// =============================================================================
// Shared Emulation Methods
// =============================================================================

// ConvertAudioSamples converts float32 mono samples to int16 stereo.
func ConvertAudioSamples(samples []float32) []int16 {
	result := make([]int16, len(samples)*2)
	for i, sample := range samples {
		intSample := int16(sample * 32767)
		result[i*2] = intSample   // Left
		result[i*2+1] = intSample // Right (duplicate for stereo)
	}
	return result
}

// RunFrame executes one frame of emulation without Ebiten or SDL.
// Audio samples are accumulated in the internal buffer.
func (e *EmulatorBase) RunFrame() {
	// Reset audio buffer for this frame
	e.audioBuffer = e.audioBuffer[:0]

	// Run the core emulation loop
	frameSamples := e.runScanlines()

	// Convert float32 samples to 16-bit stereo
	e.audioBuffer = append(e.audioBuffer, ConvertAudioSamples(frameSamples)...)
}

// GetAudioSamples returns accumulated audio samples as 16-bit stereo PCM.
func (e *EmulatorBase) GetAudioSamples() []int16 {
	return e.audioBuffer
}

// LeftColumnBlankEnabled returns whether VDP has left column blank enabled.
func (e *EmulatorBase) LeftColumnBlankEnabled() bool {
	return e.vdp.LeftColumnBlankEnabled()
}

// GetSystemRAM returns a pointer to the 8KB system RAM.
// Used by libretro for RetroAchievements memory exposure.
func (e *EmulatorBase) GetSystemRAM() *[0x2000]uint8 {
	return e.mem.GetSystemRAM()
}

// GetCartRAM returns a pointer to the 32KB cartridge RAM.
// Used by libretro for battery-backed save RAM persistence.
func (e *EmulatorBase) GetCartRAM() *[0x8000]uint8 {
	return e.mem.GetCartRAM()
}

// SetPause triggers the SMS pause button (NMI) for one frame.
// The NMI is triggered on the next frame start.
func (e *EmulatorBase) SetPause() {
	e.cpu.TriggerNMI()
}

// =============================================================================
// Save State Serialization
// =============================================================================

// SerializeSize returns the total size in bytes needed for a save state.
func (e *EmulatorBase) SerializeSize() int {
	// Header: 22 bytes
	// CPU: ~32 bytes
	// Memory: 8KB RAM + 32KB cartRAM + 3 bankSlot + 1 ramControl = 40964 bytes
	// VDP: 16KB VRAM + 32 CRAM + 16 regs + misc = ~16571 bytes
	// PSG: ~45 bytes
	// Input: 2 bytes

	return stateHeaderSize + // 22
		32 + // CPU state
		0x2000 + // RAM (8KB)
		0x8000 + // Cart RAM (32KB)
		3 + // bankSlot
		1 + // ramControl
		0x4000 + // VRAM (16KB)
		0x20 + // CRAM (32 bytes)
		16 + // VDP registers
		2 + // addr
		4 + // addrLatch, writeLatch, codeReg, readBuffer
		1 + // status
		2 + // vCounter
		1 + // hCounter
		2 + // lineCounter
		1 + // lineIntPending
		3 + // hScrollLatch, reg2Latch, vScrollLatch
		45 + // PSG state
		2 // Input ports
}

// Serialize creates a save state and returns it as a byte slice.
func (e *EmulatorBase) Serialize() ([]byte, error) {
	size := e.SerializeSize()
	data := make([]byte, size)

	// Write header
	copy(data[0:12], stateMagic)
	binary.LittleEndian.PutUint16(data[12:14], stateVersion)
	binary.LittleEndian.PutUint32(data[14:18], e.mem.GetROMCRC32())
	// Data CRC will be written at the end

	offset := stateHeaderSize

	// Serialize CPU state
	offset = e.serializeCPU(data, offset)

	// Serialize Memory state
	offset = e.serializeMemory(data, offset)

	// Serialize VDP state
	offset = e.serializeVDP(data, offset)

	// Serialize PSG state
	offset = e.serializePSG(data, offset)

	// Serialize Input state
	offset = e.serializeInput(data, offset)

	// Calculate and write data CRC32 (over everything after header)
	dataCRC := crc32.ChecksumIEEE(data[stateHeaderSize:])
	binary.LittleEndian.PutUint32(data[18:22], dataCRC)

	return data, nil
}

// Deserialize restores emulator state from a save state byte slice.
// Note: Region is NOT restored - the current region setting is preserved.
func (e *EmulatorBase) Deserialize(data []byte) error {
	if err := e.VerifyState(data); err != nil {
		return err
	}

	offset := stateHeaderSize

	// Deserialize CPU state
	offset = e.deserializeCPU(data, offset)

	// Deserialize Memory state
	offset = e.deserializeMemory(data, offset)

	// Deserialize VDP state
	offset = e.deserializeVDP(data, offset)

	// Deserialize PSG state
	offset = e.deserializePSG(data, offset)

	// Deserialize Input state
	e.deserializeInput(data, offset)

	return nil
}

// VerifyState checks if a save state is valid without loading it.
func (e *EmulatorBase) VerifyState(data []byte) error {
	// Check minimum length (must be at least header + expected state data)
	expectedSize := e.SerializeSize()
	if len(data) < expectedSize {
		return errors.New("save state too short")
	}

	// Check magic bytes
	if string(data[0:12]) != stateMagic {
		return errors.New("invalid save state magic")
	}

	// Check version
	version := binary.LittleEndian.Uint16(data[12:14])
	if version > stateVersion {
		return errors.New("unsupported save state version")
	}

	// Check ROM CRC32
	romCRC := binary.LittleEndian.Uint32(data[14:18])
	if romCRC != e.mem.GetROMCRC32() {
		return errors.New("save state is for a different ROM")
	}

	// Check data CRC32
	expectedCRC := binary.LittleEndian.Uint32(data[18:22])
	actualCRC := crc32.ChecksumIEEE(data[stateHeaderSize:])
	if expectedCRC != actualCRC {
		return errors.New("save state data is corrupted")
	}

	return nil
}

// serializeCPU writes CPU state to the data buffer
func (e *EmulatorBase) serializeCPU(data []byte, offset int) int {
	cpu := e.cpu.cpu

	// PC, SP (4 bytes)
	binary.LittleEndian.PutUint16(data[offset:], cpu.PC)
	offset += 2
	binary.LittleEndian.PutUint16(data[offset:], cpu.SP)
	offset += 2

	// Main registers AF, BC, DE, HL (8 bytes)
	binary.LittleEndian.PutUint16(data[offset:], cpu.AF.U16())
	offset += 2
	binary.LittleEndian.PutUint16(data[offset:], cpu.BC.U16())
	offset += 2
	binary.LittleEndian.PutUint16(data[offset:], cpu.DE.U16())
	offset += 2
	binary.LittleEndian.PutUint16(data[offset:], cpu.HL.U16())
	offset += 2

	// IX, IY (4 bytes) - these are uint16 directly
	binary.LittleEndian.PutUint16(data[offset:], cpu.IX)
	offset += 2
	binary.LittleEndian.PutUint16(data[offset:], cpu.IY)
	offset += 2

	// Alternate registers AF', BC', DE', HL' (8 bytes)
	binary.LittleEndian.PutUint16(data[offset:], cpu.Alternate.AF.U16())
	offset += 2
	binary.LittleEndian.PutUint16(data[offset:], cpu.Alternate.BC.U16())
	offset += 2
	binary.LittleEndian.PutUint16(data[offset:], cpu.Alternate.DE.U16())
	offset += 2
	binary.LittleEndian.PutUint16(data[offset:], cpu.Alternate.HL.U16())
	offset += 2

	// I, R (2 bytes)
	data[offset] = cpu.IR.Hi
	offset++
	data[offset] = cpu.IR.Lo
	offset++

	// IFF1, IFF2 (2 bytes)
	if cpu.IFF1 {
		data[offset] = 1
	} else {
		data[offset] = 0
	}
	offset++
	if cpu.IFF2 {
		data[offset] = 1
	} else {
		data[offset] = 0
	}
	offset++

	// IM (1 byte)
	data[offset] = byte(cpu.IM)
	offset++

	// HALT (1 byte)
	if cpu.HALT {
		data[offset] = 1
	} else {
		data[offset] = 0
	}
	offset++

	// Interrupt pending (1 byte)
	if cpu.Interrupt != nil {
		data[offset] = 1
	} else {
		data[offset] = 0
	}
	offset++

	return offset
}

// deserializeCPU reads CPU state from the data buffer
func (e *EmulatorBase) deserializeCPU(data []byte, offset int) int {
	cpu := e.cpu.cpu

	// PC, SP
	cpu.PC = binary.LittleEndian.Uint16(data[offset:])
	offset += 2
	cpu.SP = binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	// Main registers AF, BC, DE, HL
	cpu.AF.SetU16(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2
	cpu.BC.SetU16(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2
	cpu.DE.SetU16(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2
	cpu.HL.SetU16(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2

	// IX, IY - these are uint16 directly
	cpu.IX = binary.LittleEndian.Uint16(data[offset:])
	offset += 2
	cpu.IY = binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	// Alternate registers
	cpu.Alternate.AF.SetU16(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2
	cpu.Alternate.BC.SetU16(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2
	cpu.Alternate.DE.SetU16(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2
	cpu.Alternate.HL.SetU16(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2

	// I, R
	cpu.IR.Hi = data[offset]
	offset++
	cpu.IR.Lo = data[offset]
	offset++

	// IFF1, IFF2
	cpu.IFF1 = data[offset] != 0
	offset++
	cpu.IFF2 = data[offset] != 0
	offset++

	// IM
	cpu.IM = int(data[offset])
	offset++

	// HALT
	cpu.HALT = data[offset] != 0
	offset++

	// Interrupt pending
	if data[offset] != 0 {
		cpu.Interrupt = z80.IM1Interrupt()
	} else {
		cpu.Interrupt = nil
	}
	offset++

	return offset
}

// serializeMemory writes Memory state to the data buffer
func (e *EmulatorBase) serializeMemory(data []byte, offset int) int {
	// System RAM (8KB)
	copy(data[offset:], e.mem.ram[:])
	offset += len(e.mem.ram)

	// Cart RAM (32KB)
	copy(data[offset:], e.mem.cartRAM[:])
	offset += len(e.mem.cartRAM)

	// Bank slots (3 bytes)
	copy(data[offset:], e.mem.bankSlot[:])
	offset += len(e.mem.bankSlot)

	// RAM control (1 byte)
	data[offset] = e.mem.ramControl
	offset++

	return offset
}

// deserializeMemory reads Memory state from the data buffer
func (e *EmulatorBase) deserializeMemory(data []byte, offset int) int {
	// System RAM (8KB)
	copy(e.mem.ram[:], data[offset:offset+len(e.mem.ram)])
	offset += len(e.mem.ram)

	// Cart RAM (32KB)
	copy(e.mem.cartRAM[:], data[offset:offset+len(e.mem.cartRAM)])
	offset += len(e.mem.cartRAM)

	// Bank slots (3 bytes)
	copy(e.mem.bankSlot[:], data[offset:offset+len(e.mem.bankSlot)])
	offset += len(e.mem.bankSlot)

	// RAM control (1 byte)
	e.mem.ramControl = data[offset]
	offset++

	return offset
}

// serializeVDP writes VDP state to the data buffer
func (e *EmulatorBase) serializeVDP(data []byte, offset int) int {
	// VRAM (16KB)
	copy(data[offset:], e.vdp.vram[:])
	offset += len(e.vdp.vram)

	// CRAM (32 bytes)
	copy(data[offset:], e.vdp.cram[:])
	offset += len(e.vdp.cram)

	// Registers (16 bytes)
	copy(data[offset:], e.vdp.register[:])
	offset += len(e.vdp.register)

	// Address (2 bytes)
	binary.LittleEndian.PutUint16(data[offset:], e.vdp.addr)
	offset += 2

	// addrLatch, writeLatch, codeReg, readBuffer (4 bytes)
	data[offset] = e.vdp.addrLatch
	offset++
	if e.vdp.writeLatch {
		data[offset] = 1
	} else {
		data[offset] = 0
	}
	offset++
	data[offset] = e.vdp.codeReg
	offset++
	data[offset] = e.vdp.readBuffer
	offset++

	// Status (1 byte)
	data[offset] = e.vdp.status
	offset++

	// vCounter (2 bytes)
	binary.LittleEndian.PutUint16(data[offset:], e.vdp.vCounter)
	offset += 2

	// hCounter (1 byte)
	data[offset] = e.vdp.hCounter
	offset++

	// lineCounter (2 bytes, signed)
	binary.LittleEndian.PutUint16(data[offset:], uint16(e.vdp.lineCounter))
	offset += 2

	// lineIntPending (1 byte)
	if e.vdp.lineIntPending {
		data[offset] = 1
	} else {
		data[offset] = 0
	}
	offset++

	// Latched values (3 bytes)
	data[offset] = e.vdp.hScrollLatch
	offset++
	data[offset] = e.vdp.reg2Latch
	offset++
	data[offset] = e.vdp.vScrollLatch
	offset++

	return offset
}

// deserializeVDP reads VDP state from the data buffer
func (e *EmulatorBase) deserializeVDP(data []byte, offset int) int {
	// VRAM (16KB)
	copy(e.vdp.vram[:], data[offset:offset+len(e.vdp.vram)])
	offset += len(e.vdp.vram)

	// CRAM (32 bytes)
	copy(e.vdp.cram[:], data[offset:offset+len(e.vdp.cram)])
	offset += len(e.vdp.cram)

	// Registers (16 bytes)
	copy(e.vdp.register[:], data[offset:offset+len(e.vdp.register)])
	offset += len(e.vdp.register)

	// Address (2 bytes)
	e.vdp.addr = binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	// addrLatch, writeLatch, codeReg, readBuffer (4 bytes)
	e.vdp.addrLatch = data[offset]
	offset++
	e.vdp.writeLatch = data[offset] != 0
	offset++
	e.vdp.codeReg = data[offset]
	offset++
	e.vdp.readBuffer = data[offset]
	offset++

	// Status (1 byte)
	e.vdp.status = data[offset]
	offset++

	// vCounter (2 bytes)
	e.vdp.vCounter = binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	// hCounter (1 byte)
	e.vdp.hCounter = data[offset]
	offset++

	// lineCounter (2 bytes, signed)
	e.vdp.lineCounter = int16(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2

	// lineIntPending (1 byte)
	e.vdp.lineIntPending = data[offset] != 0
	offset++

	// Latched values (3 bytes)
	e.vdp.hScrollLatch = data[offset]
	offset++
	e.vdp.reg2Latch = data[offset]
	offset++
	e.vdp.vScrollLatch = data[offset]
	offset++

	return offset
}

// serializePSG writes PSG state to the data buffer
func (e *EmulatorBase) serializePSG(data []byte, offset int) int {
	// Tone registers (3 x 2 bytes = 6 bytes)
	for i := 0; i < 3; i++ {
		binary.LittleEndian.PutUint16(data[offset:], e.psg.toneReg[i])
		offset += 2
	}

	// Tone counters (3 x 2 bytes = 6 bytes)
	for i := 0; i < 3; i++ {
		binary.LittleEndian.PutUint16(data[offset:], e.psg.toneCounter[i])
		offset += 2
	}

	// Tone outputs (3 bytes)
	for i := 0; i < 3; i++ {
		if e.psg.toneOutput[i] {
			data[offset] = 1
		} else {
			data[offset] = 0
		}
		offset++
	}

	// Noise register (1 byte)
	data[offset] = e.psg.noiseReg
	offset++

	// Noise counter (2 bytes)
	binary.LittleEndian.PutUint16(data[offset:], e.psg.noiseCounter)
	offset += 2

	// Noise shift register (2 bytes)
	binary.LittleEndian.PutUint16(data[offset:], e.psg.noiseShift)
	offset += 2

	// Noise output (1 byte)
	if e.psg.noiseOutput {
		data[offset] = 1
	} else {
		data[offset] = 0
	}
	offset++

	// Volume (4 bytes)
	copy(data[offset:], e.psg.volume[:])
	offset += len(e.psg.volume)

	// Latch state (2 bytes)
	data[offset] = e.psg.latchedChannel
	offset++
	data[offset] = e.psg.latchedType
	offset++

	// Clock counter (8 bytes, float64)
	binary.LittleEndian.PutUint64(data[offset:], math.Float64bits(e.psg.clockCounter))
	offset += 8

	// Clock divider (4 bytes, int)
	binary.LittleEndian.PutUint32(data[offset:], uint32(e.psg.clockDivider))
	offset += 4

	return offset
}

// deserializePSG reads PSG state from the data buffer
func (e *EmulatorBase) deserializePSG(data []byte, offset int) int {
	// Tone registers (3 x 2 bytes = 6 bytes)
	for i := 0; i < 3; i++ {
		e.psg.toneReg[i] = binary.LittleEndian.Uint16(data[offset:])
		offset += 2
	}

	// Tone counters (3 x 2 bytes = 6 bytes)
	for i := 0; i < 3; i++ {
		e.psg.toneCounter[i] = binary.LittleEndian.Uint16(data[offset:])
		offset += 2
	}

	// Tone outputs (3 bytes)
	for i := 0; i < 3; i++ {
		e.psg.toneOutput[i] = data[offset] != 0
		offset++
	}

	// Noise register (1 byte)
	e.psg.noiseReg = data[offset]
	offset++

	// Noise counter (2 bytes)
	e.psg.noiseCounter = binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	// Noise shift register (2 bytes)
	e.psg.noiseShift = binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	// Noise output (1 byte)
	e.psg.noiseOutput = data[offset] != 0
	offset++

	// Volume (4 bytes)
	copy(e.psg.volume[:], data[offset:offset+len(e.psg.volume)])
	offset += len(e.psg.volume)

	// Latch state (2 bytes)
	e.psg.latchedChannel = data[offset]
	offset++
	e.psg.latchedType = data[offset]
	offset++

	// Clock counter (8 bytes, float64)
	e.psg.clockCounter = math.Float64frombits(binary.LittleEndian.Uint64(data[offset:]))
	offset += 8

	// Clock divider (4 bytes, int)
	e.psg.clockDivider = int(binary.LittleEndian.Uint32(data[offset:]))
	offset += 4

	return offset
}

// serializeInput writes Input state to the data buffer
func (e *EmulatorBase) serializeInput(data []byte, offset int) int {
	data[offset] = e.io.Input.Port1
	offset++
	data[offset] = e.io.Input.Port2
	offset++
	return offset
}

// deserializeInput reads Input state from the data buffer
func (e *EmulatorBase) deserializeInput(data []byte, offset int) int {
	e.io.Input.Port1 = data[offset]
	offset++
	e.io.Input.Port2 = data[offset]
	offset++
	return offset
}
