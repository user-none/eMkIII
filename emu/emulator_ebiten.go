//go:build !libretro

package emu

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/veandco/go-sdl2/sdl"
)

// Emulator wraps EmulatorBase with Ebiten/SDL specific functionality
type Emulator struct {
	EmulatorBase

	audioDeviceID sdl.AudioDeviceID
	offscreen     *ebiten.Image // Offscreen buffer for native resolution rendering
}

// NewEmulator creates a new emulator instance with Ebiten/SDL audio
func NewEmulator(rom []byte, region Region) *Emulator {
	base := initEmulatorBase(rom, region)

	// Initialize SDL audio subsystem
	if err := sdl.Init(sdl.INIT_AUDIO); err != nil {
		panic(fmt.Sprintf("failed to init SDL audio: %v", err))
	}

	// Configure audio device for queue-based (push) audio
	spec := sdl.AudioSpec{
		Freq:     sampleRate,
		Format:   sdl.AUDIO_S16LSB, // 16-bit signed little-endian
		Channels: 2,                // Stereo
		Samples:  1024,             // Buffer size hint
	}

	var obtained sdl.AudioSpec
	deviceID, err := sdl.OpenAudioDevice("", false, &spec, &obtained, 0)
	if err != nil {
		sdl.Quit()
		panic(fmt.Sprintf("failed to open audio device: %v", err))
	}

	// Start playback immediately - we'll queue samples each frame
	sdl.PauseAudioDevice(deviceID, false)

	return &Emulator{
		EmulatorBase:  base,
		audioDeviceID: deviceID,
	}
}

// Close cleans up the emulator resources
func (e *Emulator) Close() {
	sdl.CloseAudioDevice(e.audioDeviceID)
	sdl.Quit()
}

// QueueAudio sends the accumulated audio samples from RunFrame() to SDL.
// This should be called after RunFrame() when using the UI mode.
func (e *Emulator) QueueAudio() {
	samples := e.GetAudioSamples()
	if len(samples) == 0 {
		return
	}

	// Convert int16 stereo samples to bytes for SDL
	audioBytes := make([]byte, len(samples)*2) // 2 bytes per int16 sample
	for i, sample := range samples {
		audioBytes[i*2] = byte(sample)
		audioBytes[i*2+1] = byte(sample >> 8)
	}
	sdl.QueueAudio(e.audioDeviceID, audioBytes)
}

// DrawToScreen renders the emulator framebuffer to the given screen.
// Handles scaling, centering, and optional border cropping.
// This method encapsulates rendering logic for use by both the runner and UI.
func (e *Emulator) DrawToScreen(screen *ebiten.Image, cropBorder bool) {
	activeHeight := e.vdp.ActiveHeight()

	// Create or resize offscreen buffer if needed
	if e.offscreen == nil || e.offscreen.Bounds().Dy() != activeHeight {
		e.offscreen = ebiten.NewImage(ScreenWidth, activeHeight)
	}

	// Copy VDP framebuffer to offscreen buffer
	stride := e.vdp.framebuffer.Stride
	e.offscreen.WritePixels(e.vdp.framebuffer.Pix[:stride*activeHeight])

	// Determine source image and native width
	var srcImage *ebiten.Image
	nativeW := float64(ScreenWidth)

	// Crop left border if enabled and VDP has left column blank active
	if cropBorder && e.vdp.LeftColumnBlankEnabled() {
		srcImage = e.offscreen.SubImage(image.Rect(8, 0, ScreenWidth, activeHeight)).(*ebiten.Image)
		nativeW = float64(ScreenWidth - 8) // 248 pixels
	} else {
		srcImage = e.offscreen
	}

	// Calculate scaling to fit window while preserving aspect ratio
	screenW, screenH := screen.Bounds().Dx(), screen.Bounds().Dy()
	nativeH := float64(activeHeight)

	scaleX := float64(screenW) / nativeW
	scaleY := float64(screenH) / nativeH
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Calculate offset to center the image
	scaledW := nativeW * scale
	scaledH := nativeH * scale
	offsetX := (float64(screenW) - scaledW) / 2
	offsetY := (float64(screenH) - scaledH) / 2

	// Draw scaled image centered in window
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(offsetX, offsetY)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(srcImage, op)
}

func (e *Emulator) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Return window size so we control scaling in Draw()
	return outsideWidth, outsideHeight
}

// GetFramebuffer returns the VDP framebuffer as an ebiten.Image at native resolution.
// If cropBorder is true and the VDP has left column blank enabled, the 8-pixel
// left border is cropped from the returned image.
func (e *Emulator) GetFramebuffer(cropBorder bool) *ebiten.Image {
	activeHeight := e.vdp.ActiveHeight()

	// Create or resize offscreen buffer if needed
	if e.offscreen == nil || e.offscreen.Bounds().Dy() != activeHeight {
		e.offscreen = ebiten.NewImage(ScreenWidth, activeHeight)
	}

	// Copy VDP framebuffer to offscreen buffer
	stride := e.vdp.framebuffer.Stride
	e.offscreen.WritePixels(e.vdp.framebuffer.Pix[:stride*activeHeight])

	// Crop left border if enabled and VDP has left column blank active
	if cropBorder && e.vdp.LeftColumnBlankEnabled() {
		return e.offscreen.SubImage(image.Rect(8, 0, ScreenWidth, activeHeight)).(*ebiten.Image)
	}
	return e.offscreen
}
