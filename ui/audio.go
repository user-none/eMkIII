//go:build !ios && !libretro

package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/Zyko0/go-sdl3/sdl"
)

const audioSampleRate = 48000

// AudioPlayer manages SDL audio playback separately from the emulator.
// This allows callers to control when audio is initialized, supporting
// use cases like iOS mute settings that need to be checked before opening
// the audio device.
type AudioPlayer struct {
	stream     *sdl.AudioStream
	audioBytes []byte // Pre-allocated buffer for byte conversion
}

// SDL audio state
var (
	sdlInitOnce   sync.Once
	sdlAvailable  bool
	sdlInitFailed bool
)

// ensureSDLAudio initializes SDL audio on first use. Returns true if available.
func ensureSDLAudio() bool {
	sdlInitOnce.Do(func() {
		if err := loadSDLLibrary(); err != nil {
			log.Printf("Warning: Failed to load SDL library: %v", err)
			sdlInitFailed = true
			return
		}
		if err := sdl.Init(sdl.INIT_AUDIO); err != nil {
			log.Printf("Warning: Failed to init SDL audio: %v", err)
			sdlInitFailed = true
			return
		}
		sdlAvailable = true
	})
	return sdlAvailable
}

// NewAudioPlayer creates and initializes SDL audio.
func NewAudioPlayer() (*AudioPlayer, error) {
	if !ensureSDLAudio() {
		return nil, fmt.Errorf("SDL audio not available")
	}

	spec := sdl.AudioSpec{
		Freq:     audioSampleRate,
		Format:   sdl.AUDIO_S16LE,
		Channels: 2,
	}

	stream := sdl.AUDIO_DEVICE_DEFAULT_PLAYBACK.OpenAudioDeviceStream(&spec, 0)
	if stream == nil {
		return nil, fmt.Errorf("failed to open audio stream")
	}

	if err := stream.ResumeDevice(); err != nil {
		stream.Destroy()
		return nil, fmt.Errorf("failed to resume audio: %w", err)
	}

	return &AudioPlayer{
		stream:     stream,
		audioBytes: make([]byte, 0, 4096), // Pre-allocate for ~1600 stereo samples
	}, nil
}

// QueueSamples sends int16 stereo samples to SDL.
func (a *AudioPlayer) QueueSamples(samples []int16) {
	if len(samples) == 0 {
		return
	}

	// Convert int16 stereo samples to bytes for SDL using pre-allocated buffer
	a.audioBytes = a.audioBytes[:0]
	for _, sample := range samples {
		a.audioBytes = append(a.audioBytes, byte(sample), byte(sample>>8))
	}

	if err := a.stream.PutData(a.audioBytes); err != nil {
		log.Printf("warning: failed to queue audio: %v", err)
	}
}

// ClearQueue flushes any buffered audio from the SDL stream.
// Used when entering rewind mode to prevent stale audio playback.
func (a *AudioPlayer) ClearQueue() {
	if a.stream != nil {
		a.stream.Clear()
	}
}

// Close cleans up SDL audio resources.
func (a *AudioPlayer) Close() {
	if a.stream != nil {
		a.stream.Destroy()
		a.stream = nil
	}
	// Don't call sdl.Quit() - other audio streams may still be active
}

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
