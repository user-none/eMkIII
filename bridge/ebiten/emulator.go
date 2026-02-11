//go:build !libretro && !ios

// Package ebiten provides an Ebiten-specific wrapper for the emulator.
package ebiten

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/user-none/emkiii/emu"
)

// Emulator wraps emu.EmulatorBase with Ebiten-specific functionality
type Emulator struct {
	emu.EmulatorBase

	offscreen *ebiten.Image           // Offscreen buffer for native resolution rendering
	drawOpts  ebiten.DrawImageOptions // Pre-allocated draw options to avoid per-frame allocation
}

// NewEmulator creates a new emulator instance with Ebiten rendering.
// Audio is managed separately via AudioPlayer.
func NewEmulator(rom []byte, region emu.Region) *Emulator {
	base := emu.InitEmulatorBase(rom, region)

	return &Emulator{
		EmulatorBase: base,
	}
}

// Close cleans up the emulator resources
func (e *Emulator) Close() {
	// Emulator no longer manages audio - AudioPlayer handles it
}

func (e *Emulator) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Return window size so we control scaling in Draw()
	return outsideWidth, outsideHeight
}

// DrawCachedFramebuffer renders pre-cached pixel data to the screen.
// Used by the ADT architecture where the emulation goroutine writes pixels
// to a shared framebuffer, and the Ebiten Draw() thread renders them.
func (e *Emulator) DrawCachedFramebuffer(screen *ebiten.Image, pixels []byte, stride, activeHeight int, leftColumnBlank, cropBorder bool) {
	if activeHeight == 0 || stride == 0 {
		return
	}

	requiredLen := stride * activeHeight
	if len(pixels) < requiredLen {
		return
	}

	// Create or resize offscreen buffer if needed
	if e.offscreen == nil || e.offscreen.Bounds().Dy() != activeHeight {
		e.offscreen = ebiten.NewImage(emu.ScreenWidth, activeHeight)
	}

	e.offscreen.WritePixels(pixels[:requiredLen])

	// Determine source image and native width
	var srcImage *ebiten.Image
	nativeW := float64(emu.ScreenWidth)

	if cropBorder && leftColumnBlank {
		srcImage = e.offscreen.SubImage(image.Rect(8, 0, emu.ScreenWidth, activeHeight)).(*ebiten.Image)
		nativeW = float64(emu.ScreenWidth - 8)
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

	scaledW := nativeW * scale
	scaledH := nativeH * scale
	offsetX := (float64(screenW) - scaledW) / 2
	offsetY := (float64(screenH) - scaledH) / 2

	e.drawOpts = ebiten.DrawImageOptions{}
	e.drawOpts.GeoM.Scale(scale, scale)
	e.drawOpts.GeoM.Translate(offsetX, offsetY)
	e.drawOpts.Filter = ebiten.FilterNearest
	screen.DrawImage(srcImage, &e.drawOpts)
}

// GetCachedFramebufferImage returns pre-cached pixel data as an ebiten.Image
// at native resolution. Used for xBR shader processing with ADT.
func (e *Emulator) GetCachedFramebufferImage(pixels []byte, stride, activeHeight int, leftColumnBlank, cropBorder bool) *ebiten.Image {
	if activeHeight == 0 || stride == 0 {
		return nil
	}

	requiredLen := stride * activeHeight
	if len(pixels) < requiredLen {
		return nil
	}

	// Create or resize offscreen buffer if needed
	if e.offscreen == nil || e.offscreen.Bounds().Dy() != activeHeight {
		e.offscreen = ebiten.NewImage(emu.ScreenWidth, activeHeight)
	}

	e.offscreen.WritePixels(pixels[:requiredLen])

	if cropBorder && leftColumnBlank {
		return e.offscreen.SubImage(image.Rect(8, 0, emu.ScreenWidth, activeHeight)).(*ebiten.Image)
	}
	return e.offscreen
}
