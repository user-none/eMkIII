//go:build !libretro

package emu

import (
	"fmt"
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/veandco/go-sdl2/sdl"
)

// Emulator wraps EmulatorBase with Ebiten/SDL specific functionality
type Emulator struct {
	EmulatorBase

	audioDeviceID sdl.AudioDeviceID
	offscreen     *ebiten.Image // Offscreen buffer for native resolution rendering
	cropBorder    bool          // Crop left 8-pixel border when VDP has left column blank enabled

	// Ebiten-specific diagnostics
	lastFrameDuration    time.Duration
	lastAudioBufferMs    float64
	fpsUpdateTime time.Time
	fpsFrameCount int
	currentFPS    float64
}

// NewEmulator creates a new emulator instance with Ebiten/SDL audio
func NewEmulator(rom []byte, region Region, cropBorder bool) *Emulator {
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
		cropBorder:    cropBorder,
		fpsUpdateTime: time.Now(),
	}
}

func (e *Emulator) Update() error {
	if !ebiten.IsFocused() {
		return nil
	}

	// Poll keyboard for controller input
	// WASD = movement, J = button 1, K = button 2
	kUp := ebiten.IsKeyPressed(ebiten.KeyW)
	kDown := ebiten.IsKeyPressed(ebiten.KeyS)
	kLeft := ebiten.IsKeyPressed(ebiten.KeyA)
	kRight := ebiten.IsKeyPressed(ebiten.KeyD)
	kBtn1 := ebiten.IsKeyPressed(ebiten.KeyJ)
	kBtn2 := ebiten.IsKeyPressed(ebiten.KeyK)

	// Poll gamepad for controller input (first connected gamepad)
	// Supports PlayStation 3-5, Xbox 360-Series X, and other standard controllers
	var gUp, gDown, gLeft, gRight, gBtn1, gBtn2 bool
	gamepadIDs := ebiten.AppendGamepadIDs(nil)
	if len(gamepadIDs) > 0 {
		gp := gamepadIDs[0]
		if ebiten.IsStandardGamepadLayoutAvailable(gp) {
			// D-pad
			gUp = ebiten.IsStandardGamepadButtonPressed(gp, ebiten.StandardGamepadButtonLeftTop)
			gDown = ebiten.IsStandardGamepadButtonPressed(gp, ebiten.StandardGamepadButtonLeftBottom)
			gLeft = ebiten.IsStandardGamepadButtonPressed(gp, ebiten.StandardGamepadButtonLeftLeft)
			gRight = ebiten.IsStandardGamepadButtonPressed(gp, ebiten.StandardGamepadButtonLeftRight)

			// Face buttons: A/Cross = Button 1, B/Circle = Button 2
			gBtn1 = ebiten.IsStandardGamepadButtonPressed(gp, ebiten.StandardGamepadButtonRightBottom)
			gBtn2 = ebiten.IsStandardGamepadButtonPressed(gp, ebiten.StandardGamepadButtonRightRight)

			// Left analog stick (with deadzone)
			const deadzone = 0.5
			lx := ebiten.StandardGamepadAxisValue(gp, ebiten.StandardGamepadAxisLeftStickHorizontal)
			ly := ebiten.StandardGamepadAxisValue(gp, ebiten.StandardGamepadAxisLeftStickVertical)
			if lx < -deadzone {
				gLeft = true
			}
			if lx > deadzone {
				gRight = true
			}
			if ly < -deadzone {
				gUp = true
			}
			if ly > deadzone {
				gDown = true
			}
		}
	}

	// Combine keyboard and gamepad inputs
	e.io.Input.SetP1(
		kUp || gUp,
		kDown || gDown,
		kLeft || gLeft,
		kRight || gRight,
		kBtn1 || gBtn1,
		kBtn2 || gBtn2,
	)

	// Track frame timing for diagnostics
	frameStart := time.Now()
	e.fpsFrameCount++

	// Update FPS every second
	if time.Since(e.fpsUpdateTime) >= time.Second {
		e.currentFPS = float64(e.fpsFrameCount) / time.Since(e.fpsUpdateTime).Seconds()
		e.fpsFrameCount = 0
		e.fpsUpdateTime = time.Now()
	}

	// Run the core emulation loop
	frameSamples := e.runScanlines()

	// Queue audio samples to SDL
	if len(frameSamples) > 0 {
		// Convert float32 samples to 16-bit stereo PCM bytes
		audioBytes := make([]byte, len(frameSamples)*4) // 4 bytes per stereo frame
		for i, sample := range frameSamples {
			intSample := int16(sample * 32767)
			offset := i * 4
			audioBytes[offset] = byte(intSample)
			audioBytes[offset+1] = byte(intSample >> 8)
			audioBytes[offset+2] = byte(intSample) // Same for both channels (mono source)
			audioBytes[offset+3] = byte(intSample >> 8)
		}
		sdl.QueueAudio(e.audioDeviceID, audioBytes)
	}

	// Record diagnostics
	e.lastFrameDuration = time.Since(frameStart)
	e.lastAudioBufferMs = float64(sdl.GetQueuedAudioSize(e.audioDeviceID)/4) / float64(sampleRate) * 1000.0

	return nil
}

// Close cleans up the emulator resources
func (e *Emulator) Close() {
	sdl.CloseAudioDevice(e.audioDeviceID)
	sdl.Quit()
}

func (e *Emulator) Draw(screen *ebiten.Image) {
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
	if e.cropBorder && e.vdp.LeftColumnBlankEnabled() {
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
	screen.DrawImage(srcImage, op)
}

func (e *Emulator) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Return window size so we control scaling in Draw()
	return outsideWidth, outsideHeight
}
