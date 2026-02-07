//go:build !libretro

package emu

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Emulator wraps EmulatorBase with Ebiten-specific functionality
type Emulator struct {
	EmulatorBase

	offscreen *ebiten.Image           // Offscreen buffer for native resolution rendering
	drawOpts  ebiten.DrawImageOptions // Pre-allocated draw options to avoid per-frame allocation
}

// NewEmulator creates a new emulator instance with Ebiten rendering.
// Audio is managed separately via AudioPlayer.
func NewEmulator(rom []byte, region Region) *Emulator {
	base := initEmulatorBase(rom, region)

	return &Emulator{
		EmulatorBase: base,
	}
}

// Close cleans up the emulator resources
func (e *Emulator) Close() {
	// Emulator no longer manages audio - AudioPlayer handles it
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

	// Draw scaled image centered in window using pre-allocated options
	e.drawOpts = ebiten.DrawImageOptions{}
	e.drawOpts.GeoM.Scale(scale, scale)
	e.drawOpts.GeoM.Translate(offsetX, offsetY)
	e.drawOpts.Filter = ebiten.FilterNearest
	screen.DrawImage(srcImage, &e.drawOpts)
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
