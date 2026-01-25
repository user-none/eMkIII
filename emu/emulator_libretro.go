//go:build libretro

package emu

// Emulator wraps EmulatorBase with libretro-specific functionality
type Emulator struct {
	EmulatorBase

	// Audio buffer for libretro (accumulated per frame)
	audioBuffer []int16
}

// NewEmulatorForLibretro creates an emulator instance for libretro use (no SDL/Ebiten)
func NewEmulatorForLibretro(rom []byte, region Region) *Emulator {
	base := initEmulatorBase(rom, region)

	return &Emulator{
		EmulatorBase: base,
	}
}

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

// RunFrame executes one frame of emulation without Ebiten or SDL
func (e *Emulator) RunFrame() {
	// Reset audio buffer for this frame
	e.audioBuffer = e.audioBuffer[:0]

	// Run the core emulation loop
	frameSamples := e.runScanlines()

	// Convert float32 samples to 16-bit stereo
	e.audioBuffer = append(e.audioBuffer, ConvertAudioSamples(frameSamples)...)
}

// GetAudioSamples returns accumulated audio samples as 16-bit stereo PCM
func (e *Emulator) GetAudioSamples() []int16 {
	return e.audioBuffer
}

// LeftColumnBlankEnabled returns whether VDP has left column blank enabled
func (e *Emulator) LeftColumnBlankEnabled() bool {
	return e.vdp.LeftColumnBlankEnabled()
}
