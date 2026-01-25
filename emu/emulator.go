package emu

import (
	"github.com/koron-go/z80"
)

const (
	ScreenWidth     = 256
	MaxScreenHeight = 224
	sampleRate      = 48000
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

// SetInput sets controller state from external source
func (e *EmulatorBase) SetInput(up, down, left, right, btn1, btn2 bool) {
	e.io.Input.SetP1(up, down, left, right, btn1, btn2)
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
