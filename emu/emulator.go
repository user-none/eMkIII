package emu

import (
	"encoding/binary"
	"errors"
	"hash/crc32"

	emucore "github.com/user-none/eblitui/api"
	"github.com/user-none/go-chip-sn76489"
	"github.com/user-none/go-chip-z80"
)

// Compile-time interface checks.
var _ emucore.Emulator = (*Emulator)(nil)
var _ emucore.SaveStater = (*Emulator)(nil)
var _ emucore.BatterySaver = (*Emulator)(nil)
var _ emucore.MemoryInspector = (*Emulator)(nil)
var _ emucore.MemoryMapper = (*Emulator)(nil)

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

// Emulator contains the emulator core components.
type Emulator struct {
	cpu                 *z80.CPU
	mem                 *Memory
	vdp                 *VDP
	psg                 *sn76489.SN76489
	io                  *SMSIO
	cyclesPerScanlineFP int // Fixed-point (16 fractional bits) for accurate timing

	// Region timing
	region    Region
	timing    RegionTiming
	scanlines int

	// Input edge detection for pause button
	prevButtons [2]uint32

	// Crop border support
	cropBorder bool
	cropBuffer []byte

	// Pre-allocated audio buffers to avoid per-frame allocations
	frameSamples []float32 // Collects float32 samples during scanline emulation
	audioBuffer  []int16   // Final int16 stereo output for external consumption
}

// NewEmulator creates and initializes the emulator components.
func NewEmulator(rom []byte, region Region) (Emulator, error) {
	mem := NewMemory(rom)
	vdp := NewVDP()

	timing := GetTimingForRegion(region)
	vdp.SetTotalScanlines(timing.Scanlines)

	samplesPerFrame := sampleRate / timing.FPS
	psg := sn76489.New(timing.CPUClockHz, sampleRate, samplesPerFrame*2, sn76489.Sega)

	nationality := DetectNationalityFromROM(rom)
	io := NewSMSIO(vdp, psg, nationality)
	bus := NewSMSBus(mem, io)
	cpu := z80.New(bus)

	cyclesPerScanlineFP := (timing.CPUClockHz * 65536) / timing.FPS / timing.Scanlines

	return Emulator{
		cpu:                 cpu,
		mem:                 mem,
		vdp:                 vdp,
		psg:                 psg,
		io:                  io,
		cyclesPerScanlineFP: cyclesPerScanlineFP,
		region:              region,
		timing:              timing,
		scanlines:           timing.Scanlines,
		cropBuffer:          make([]byte, (ScreenWidth-8)*MaxScreenHeight*4),
		// Pre-allocate audio buffers: ~800 samples/frame at 48kHz/60fps
		frameSamples: make([]float32, 0, 1024),
		audioBuffer:  make([]int16, 0, 2048),
	}, nil
}

// checkAndSetInterrupt updates CPU interrupt state based on VDP pending interrupts
func (e *Emulator) checkAndSetInterrupt() {
	e.cpu.INT(e.vdp.InterruptPending(), 0xFF)
}

// runScanlines executes one frame of CPU/VDP/PSG emulation.
// Audio samples are accumulated in e.frameSamples.
func (e *Emulator) runScanlines() {
	activeHeight := e.vdp.ActiveHeight()

	var targetCyclesFP int = 0
	var prevTarget int = 0

	// Reset pre-allocated buffer for this frame
	e.frameSamples = e.frameSamples[:0]

	for i := 0; i < e.scanlines; i++ {
		targetCyclesFP += e.cyclesPerScanlineFP
		target := targetCyclesFP >> 16
		scanlineBudget := target - prevTarget
		prevTarget = target

		e.vdp.SetVCounter(uint16(i))

		if i == 0 {
			e.vdp.LatchVScrollForFrame()
		}

		// Flags to track per-scanline interrupt triggers
		vblankChecked := false
		lineInterruptChecked := false
		cramLatched := false
		// frame interrupt fires at V-counter $C1 (line 193) for
		// 192-line mode and $E1 (line 225) for 224-line mode, one line after
		// the last active display line.
		isVBlankLine := (i == activeHeight+1)

		consumed := 0
		for consumed < scanlineBudget {
			// Check VBlank at cycle VBlankInterruptCycle (only on vblank line)
			if !vblankChecked && isVBlankLine && consumed >= VBlankInterruptCycle {
				e.vdp.SetVBlank()
				vblankChecked = true
				// Check interrupt state after VBlank trigger
				e.checkAndSetInterrupt()
			}

			// Line counter decrements at LineInterruptCycle (~cycle 8)
			// This is when line interrupts fire on real hardware
			if !lineInterruptChecked && consumed >= LineInterruptCycle {
				e.vdp.UpdateLineCounter()
				lineInterruptChecked = true
				e.checkAndSetInterrupt()
			}

			// Latch CRAM and per-line registers at cycle 14 (after line interrupt handler can modify them)
			if !cramLatched && consumed >= CRAMLatchCycle {
				e.vdp.LatchCRAM()
				e.vdp.LatchPerLineRegisters()
				cramLatched = true
			}

			e.vdp.SetHCounter(GetHCounterForCycle(consumed))
			consumed += e.cpu.StepCycles(scanlineBudget - consumed)

			// Check if VDP register write requires interrupt state update.
			// SMS interrupt line is level-triggered, so enabling interrupts via
			// register write should immediately assert pending interrupts.
			if e.vdp.InterruptCheckRequired() {
				e.checkAndSetInterrupt()
			}

			// Update interrupt state only when VDP status was read (level-triggered).
			// Reading status clears flags, which should de-assert the interrupt line.
			if e.vdp.StatusWasRead() {
				e.checkAndSetInterrupt()
			}
		}

		if !vblankChecked && isVBlankLine {
			e.vdp.SetVBlank()
			e.checkAndSetInterrupt()
		}

		// Ensure line counter is updated even for short scanlines
		if !lineInterruptChecked {
			e.vdp.UpdateLineCounter()
			e.checkAndSetInterrupt()
		}

		if i < activeHeight {
			e.vdp.RenderScanline()
		}

		e.psg.GenerateSamples(scanlineBudget)
		buffer, count := e.psg.GetBuffer()
		if count > 0 {
			e.frameSamples = append(e.frameSamples, buffer[:count]...)
		}
	}
}

// SetInput unpacks a button bitmask and sets controller state for the given player.
func (e *Emulator) SetInput(player int, buttons uint32) {
	up := buttons&(1<<emucore.ButtonUp) != 0
	down := buttons&(1<<emucore.ButtonDown) != 0
	left := buttons&(1<<emucore.ButtonLeft) != 0
	right := buttons&(1<<emucore.ButtonRight) != 0
	btn1 := buttons&(1<<4) != 0
	btn2 := buttons&(1<<5) != 0

	switch player {
	case 0:
		e.io.Input.SetP1(up, down, left, right, btn1, btn2)
		// Edge detect pause (bit 7): trigger NMI on press (0->1)
		pauseNow := buttons&(1<<7) != 0
		pausePrev := e.prevButtons[0]&(1<<7) != 0
		if pauseNow && !pausePrev {
			e.cpu.NMI()
		}
	case 1:
		e.io.Input.SetP2(up, down, left, right, btn1, btn2)
	}

	if player < 2 {
		e.prevButtons[player] = buttons
	}
}

// GetFramebuffer returns raw RGBA pixel data for current frame.
// When crop border is enabled and the VDP has left column blank active,
// the left 8 pixels are stripped from each row.
func (e *Emulator) GetFramebuffer() []byte {
	if e.cropBorder && e.vdp.LeftColumnBlankEnabled() {
		srcStride := e.vdp.framebuffer.Stride
		dstStride := (ScreenWidth - 8) * 4
		activeHeight := e.vdp.ActiveHeight()
		for y := 0; y < activeHeight; y++ {
			srcOff := y*srcStride + 8*4 // skip 8 pixels
			dstOff := y * dstStride
			copy(e.cropBuffer[dstOff:dstOff+dstStride], e.vdp.framebuffer.Pix[srcOff:srcOff+dstStride])
		}
		return e.cropBuffer[:dstStride*activeHeight]
	}
	return e.vdp.framebuffer.Pix
}

// GetFramebufferStride returns the stride (bytes per row) of the framebuffer.
func (e *Emulator) GetFramebufferStride() int {
	if e.cropBorder && e.vdp.LeftColumnBlankEnabled() {
		return (ScreenWidth - 8) * 4
	}
	return e.vdp.framebuffer.Stride
}

// GetActiveHeight returns the current active display height (192 or 224)
func (e *Emulator) GetActiveHeight() int {
	return e.vdp.ActiveHeight()
}

// GetRegion returns the emulator's region setting
func (e *Emulator) GetRegion() Region {
	return e.region
}

// GetTiming returns FPS and scanline count for the current region.
func (e *Emulator) GetTiming() emucore.Timing {
	return emucore.Timing{
		FPS:       e.timing.FPS,
		Scanlines: e.timing.Scanlines,
	}
}

// SetRegion updates the emulator's region configuration
func (e *Emulator) SetRegion(region Region) {
	e.region = region
	e.timing = GetTimingForRegion(region)
	e.scanlines = e.timing.Scanlines
	e.vdp.SetTotalScanlines(e.timing.Scanlines)
	e.cyclesPerScanlineFP = (e.timing.CPUClockHz * 65536) / e.timing.FPS / e.timing.Scanlines
}

// SetOption applies a core option change identified by key.
func (e *Emulator) SetOption(key string, value string) {
	switch key {
	case "crop_border":
		e.cropBorder = value == "true"
	}
}

// Close releases any resources held by the emulator.
func (e *Emulator) Close() {}

// =============================================================================
// Shared Emulation Methods
// =============================================================================

// RunFrame executes one frame of emulation.
// Audio samples are accumulated in the internal buffer.
func (e *Emulator) RunFrame() {
	// Reset audio buffer for this frame
	e.audioBuffer = e.audioBuffer[:0]

	// Run the core emulation loop (populates e.frameSamples)
	e.runScanlines()

	// Convert float32 mono samples to int16 stereo in-place
	// Attenuate by 0.5 to compensate for acoustic summing when both speakers
	// play the same signal (mono duplicated to L+R doubles perceived loudness)
	for _, sample := range e.frameSamples {
		intSample := int16(sample * 32767 * 0.5)
		e.audioBuffer = append(e.audioBuffer, intSample, intSample)
	}
}

// GetAudioSamples returns accumulated audio samples as 16-bit stereo PCM.
func (e *Emulator) GetAudioSamples() []int16 {
	return e.audioBuffer
}

// HasSRAM reports whether the loaded ROM uses battery-backed save.
// SMS cartridges always have 32KB cart RAM available.
func (e *Emulator) HasSRAM() bool {
	return true
}

// GetSRAM returns a copy of the current SRAM contents.
func (e *Emulator) GetSRAM() []byte {
	sram := make([]byte, len(e.mem.cartRAM))
	copy(sram, e.mem.cartRAM[:])
	return sram
}

// SetSRAM loads SRAM contents into the emulator.
func (e *Emulator) SetSRAM(data []byte) {
	copy(e.mem.cartRAM[:], data)
}

// =============================================================================
// Save State Serialization
// =============================================================================

// SerializeSize returns the total size in bytes needed for a save state.
func SerializeSize() int {
	// Header: 22 bytes
	// CPU: 47 bytes (library SerializeSize)
	// Memory: 8KB RAM + 32KB cartRAM + 3 bankSlot + 1 ramControl = 40964 bytes
	// VDP: 16KB VRAM + 32 CRAM + 16 regs + misc = ~16571 bytes
	// PSG: 40 bytes (library SerializeSize)
	// Input: 3 bytes (Port1, Port2, ioControl)

	return stateHeaderSize + // 22
		z80.SerializeSize + // CPU state
		0x2000 + // RAM (8KB)
		0x8000 + // Cart RAM (32KB)
		3 + // bankSlot
		1 + // ramControl
		0x4000 + // VRAM (16KB)
		0x20 + // CRAM (32 bytes)
		0x20 + // CRAM latch (32 bytes)
		16 + // VDP registers
		2 + // addr
		4 + // addrLatch, writeLatch, codeReg, readBuffer
		1 + // status
		2 + // vCounter
		1 + // hCounter
		2 + // lineCounter
		1 + // lineIntPending
		4 + // hScrollLatch, reg2Latch, reg7Latch, vScrollLatch
		1 + // interruptCheckRequired
		sn76489.SerializeSize + // PSG state
		3 // Input ports (2) + ioControl (1)
}

// Serialize creates a save state and returns it as a byte slice.
func (e *Emulator) Serialize() ([]byte, error) {
	size := SerializeSize()
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
func (e *Emulator) Deserialize(data []byte) error {
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
func (e *Emulator) VerifyState(data []byte) error {
	// Check minimum length (must be at least header + expected state data)
	expectedSize := SerializeSize()
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
func (e *Emulator) serializeCPU(data []byte, offset int) int {
	e.cpu.Serialize(data[offset:])
	return offset + z80.SerializeSize
}

// deserializeCPU reads CPU state from the data buffer
func (e *Emulator) deserializeCPU(data []byte, offset int) int {
	e.cpu.Deserialize(data[offset:])
	return offset + z80.SerializeSize
}

// serializeMemory writes Memory state to the data buffer
func (e *Emulator) serializeMemory(data []byte, offset int) int {
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
func (e *Emulator) deserializeMemory(data []byte, offset int) int {
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
func (e *Emulator) serializeVDP(data []byte, offset int) int {
	// VRAM (16KB)
	copy(data[offset:], e.vdp.vram[:])
	offset += len(e.vdp.vram)

	// CRAM (32 bytes)
	copy(data[offset:], e.vdp.cram[:])
	offset += len(e.vdp.cram)

	// CRAM latch (32 bytes)
	copy(data[offset:], e.vdp.cramLatch[:])
	offset += len(e.vdp.cramLatch)

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

	// Latched values (4 bytes)
	data[offset] = e.vdp.hScrollLatch
	offset++
	data[offset] = e.vdp.reg2Latch
	offset++
	data[offset] = e.vdp.reg7Latch
	offset++
	data[offset] = e.vdp.vScrollLatch
	offset++

	// interruptCheckRequired (1 byte)
	if e.vdp.interruptCheckRequired {
		data[offset] = 1
	} else {
		data[offset] = 0
	}
	offset++

	return offset
}

// deserializeVDP reads VDP state from the data buffer
func (e *Emulator) deserializeVDP(data []byte, offset int) int {
	// VRAM (16KB)
	copy(e.vdp.vram[:], data[offset:offset+len(e.vdp.vram)])
	offset += len(e.vdp.vram)

	// CRAM (32 bytes)
	copy(e.vdp.cram[:], data[offset:offset+len(e.vdp.cram)])
	offset += len(e.vdp.cram)

	// CRAM latch (32 bytes)
	copy(e.vdp.cramLatch[:], data[offset:offset+len(e.vdp.cramLatch)])
	offset += len(e.vdp.cramLatch)

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

	// Latched values (4 bytes)
	e.vdp.hScrollLatch = data[offset]
	offset++
	e.vdp.reg2Latch = data[offset]
	offset++
	e.vdp.reg7Latch = data[offset]
	offset++
	e.vdp.vScrollLatch = data[offset]
	offset++

	// interruptCheckRequired (1 byte)
	e.vdp.interruptCheckRequired = data[offset] != 0
	offset++

	return offset
}

// serializePSG writes PSG state to the data buffer
func (e *Emulator) serializePSG(data []byte, offset int) int {
	e.psg.Serialize(data[offset:])
	return offset + sn76489.SerializeSize
}

// deserializePSG reads PSG state from the data buffer
func (e *Emulator) deserializePSG(data []byte, offset int) int {
	e.psg.Deserialize(data[offset:])
	return offset + sn76489.SerializeSize
}

// serializeInput writes Input state to the data buffer
func (e *Emulator) serializeInput(data []byte, offset int) int {
	data[offset] = e.io.Input.Port1
	offset++
	data[offset] = e.io.Input.Port2
	offset++
	data[offset] = e.io.ioControl
	offset++
	return offset
}

// deserializeInput reads Input state from the data buffer
func (e *Emulator) deserializeInput(data []byte, offset int) int {
	e.io.Input.Port1 = data[offset]
	offset++
	e.io.Input.Port2 = data[offset]
	offset++
	e.io.ioControl = data[offset]
	offset++
	return offset
}

// =============================================================================
// MemoryInspector interface
// =============================================================================

// Flat address boundaries for ReadMemory.
const (
	systemRAMStart = 0x0000
	systemRAMEnd   = 0x1FFF
)

// ReadMemory reads from a flat address into buf and returns the number
// of bytes read. SMS flat address mapping for RetroAchievements:
// 0x0000-0x1FFF -> System RAM (8KB)
func (e *Emulator) ReadMemory(addr uint32, buf []byte) uint32 {
	var count uint32
	for i := range buf {
		cur := addr + uint32(i)
		if cur >= systemRAMStart && cur <= systemRAMEnd {
			buf[i] = e.mem.ram[cur]
			count++
		} else {
			return count
		}
	}
	return count
}

// =============================================================================
// MemoryMapper interface
// =============================================================================

// MemoryMap returns a list of available memory regions with sizes.
func (e *Emulator) MemoryMap() []emucore.MemoryRegion {
	return []emucore.MemoryRegion{
		{Type: emucore.MemorySystemRAM, Size: 0x2000},
		{Type: emucore.MemorySaveRAM, Size: 0x8000},
	}
}

// ReadRegion returns a copy of the specified memory region.
func (e *Emulator) ReadRegion(regionType int) []byte {
	switch regionType {
	case emucore.MemorySystemRAM:
		out := make([]byte, len(e.mem.ram))
		copy(out, e.mem.ram[:])
		return out
	case emucore.MemorySaveRAM:
		return e.GetSRAM()
	default:
		return nil
	}
}

// WriteRegion writes data to the specified memory region.
func (e *Emulator) WriteRegion(regionType int, data []byte) {
	switch regionType {
	case emucore.MemorySystemRAM:
		copy(e.mem.ram[:], data)
	case emucore.MemorySaveRAM:
		e.SetSRAM(data)
	}
}
