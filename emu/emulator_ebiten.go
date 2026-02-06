//go:build !libretro

package emu

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/hajimehoshi/ebiten/v2"
)

var sdlOnce sync.Once

// loadSDLLibrary attempts to load the SDL3 library from multiple locations.
// It tries each path in priority order until one succeeds.
func loadSDLLibrary() error {
	paths := sdlLibrarySearchPaths()
	var lastErr error
	for _, path := range paths {
		if err := sdl.LoadLibrary(path); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	// Final fallback: try default name (let dlopen search system paths)
	if err := sdl.LoadLibrary(sdl.Path()); err == nil {
		return nil
	} else if lastErr == nil {
		lastErr = err
	}
	return fmt.Errorf("failed to load SDL3 library from any location: %w", lastErr)
}

// sdlLibrarySearchPaths returns a list of paths to search for the SDL3 library.
func sdlLibrarySearchPaths() []string {
	switch runtime.GOOS {
	case "darwin":
		paths := []string{}
		// 1. App bundle Frameworks directory (for .app distribution)
		if exe, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exe)
			// Inside .app bundle: Contents/MacOS/emkiii -> Contents/Frameworks/
			paths = append(paths, filepath.Join(exeDir, "..", "Frameworks", "libSDL3.dylib"))
			// Same directory as executable
			paths = append(paths, filepath.Join(exeDir, "libSDL3.dylib"))
		}
		// 2. Homebrew locations (for development)
		if runtime.GOARCH == "arm64" {
			paths = append(paths, "/opt/homebrew/lib/libSDL3.dylib")
		} else {
			paths = append(paths, "/usr/local/lib/libSDL3.dylib")
		}
		// 3. System locations
		paths = append(paths, "/usr/lib/libSDL3.dylib")
		return paths
	case "linux", "freebsd":
		paths := []string{}
		if exe, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exe)
			paths = append(paths, filepath.Join(exeDir, "libSDL3.so.0"))
		}
		// Standard library paths (dlopen will search these anyway)
		paths = append(paths, "/usr/local/lib/libSDL3.so.0")
		paths = append(paths, "/usr/lib/libSDL3.so.0")
		return paths
	case "windows":
		paths := []string{}
		if exe, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exe)
			paths = append(paths, filepath.Join(exeDir, "SDL3.dll"))
		}
		return paths
	default:
		return []string{}
	}
}

// Emulator wraps EmulatorBase with Ebiten/SDL specific functionality
type Emulator struct {
	EmulatorBase

	audioStream *sdl.AudioStream
	offscreen   *ebiten.Image // Offscreen buffer for native resolution rendering
}

// NewEmulator creates a new emulator instance with Ebiten/SDL3 audio
func NewEmulator(rom []byte, region Region) *Emulator {
	base := initEmulatorBase(rom, region)

	// Load SDL3 library once (required before any SDL calls)
	sdlOnce.Do(func() {
		if err := loadSDLLibrary(); err != nil {
			panic(err.Error())
		}
	})

	// Initialize SDL audio subsystem
	if err := sdl.Init(sdl.INIT_AUDIO); err != nil {
		panic(fmt.Sprintf("failed to init SDL audio: %v", err))
	}

	// Configure audio spec for push-based audio
	spec := sdl.AudioSpec{
		Freq:     sampleRate,
		Format:   sdl.AUDIO_S16LE, // 16-bit signed little-endian
		Channels: 2,               // Stereo
	}

	// Open audio stream on default playback device
	// 0 callback means push-based audio via PutData()
	audioStream := sdl.AUDIO_DEVICE_DEFAULT_PLAYBACK.OpenAudioDeviceStream(&spec, 0)
	if audioStream == nil {
		sdl.Quit()
		panic("failed to open audio stream")
	}

	// Start playback immediately - we'll queue samples each frame
	if err := audioStream.ResumeDevice(); err != nil {
		audioStream.Destroy()
		sdl.Quit()
		panic(fmt.Sprintf("failed to resume audio device: %v", err))
	}

	return &Emulator{
		EmulatorBase: base,
		audioStream:  audioStream,
	}
}

// Close cleans up the emulator resources
func (e *Emulator) Close() {
	if e.audioStream != nil {
		e.audioStream.Destroy()
	}
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

	// Queue audio data to the stream
	if err := e.audioStream.PutData(audioBytes); err != nil {
		log.Printf("warning: failed to queue audio: %v", err)
	}
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
