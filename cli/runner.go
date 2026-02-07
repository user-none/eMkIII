//go:build !libretro

// Package cli provides a command-line runner for the emulator.
// It handles input polling and runs the emulator in a window without the full UI.
package cli

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/user-none/emkiii/emu"
	"github.com/user-none/emkiii/ui"
)

// Runner wraps an emulator for command-line mode.
// It handles input polling (emulator doesn't poll input itself).
// This follows the libretro pattern where the frontend is responsible
// for polling input and passing it to the emulator via SetInput().
type Runner struct {
	emulator    *emu.Emulator
	audioPlayer *ui.AudioPlayer
	cropBorder  bool
}

// NewRunner creates a new Runner wrapping the given emulator.
func NewRunner(e *emu.Emulator, cropBorder bool) *Runner {
	player, err := ui.NewAudioPlayer(false)
	if err != nil {
		panic(err)
	}
	return &Runner{
		emulator:    e,
		audioPlayer: player,
		cropBorder:  cropBorder,
	}
}

// Close cleans up the runner's resources.
func (r *Runner) Close() {
	if r.audioPlayer != nil {
		r.audioPlayer.Close()
		r.audioPlayer = nil
	}
}

// Update implements ebiten.Game.
func (r *Runner) Update() error {
	if !ebiten.IsFocused() {
		return nil
	}

	// Poll input (runner responsibility, not emulator)
	r.pollInput()

	// Run one frame of emulation
	r.emulator.RunFrame()

	// Queue audio samples to SDL
	r.audioPlayer.QueueSamples(r.emulator.GetAudioSamples())

	return nil
}

// Draw implements ebiten.Game.
func (r *Runner) Draw(screen *ebiten.Image) {
	r.emulator.DrawToScreen(screen, r.cropBorder)
}

// Layout implements ebiten.Game.
func (r *Runner) Layout(outsideWidth, outsideHeight int) (int, int) {
	return r.emulator.Layout(outsideWidth, outsideHeight)
}

// pollInput reads keyboard and gamepad input and passes it to the emulator.
func (r *Runner) pollInput() {
	// Keyboard (WASD + arrows for movement, J/Z and K/X for buttons)
	up := ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp)
	down := ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown)
	left := ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
	right := ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight)
	btn1 := ebiten.IsKeyPressed(ebiten.KeyJ) || ebiten.IsKeyPressed(ebiten.KeyZ)
	btn2 := ebiten.IsKeyPressed(ebiten.KeyK) || ebiten.IsKeyPressed(ebiten.KeyX)

	// Gamepad support (all connected gamepads)
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}

		// D-pad
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			up = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			down = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
			left = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight) {
			right = true
		}

		// Face buttons: A/Cross = Button 1, B/Circle = Button 2
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightBottom) {
			btn1 = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightRight) {
			btn2 = true
		}

		// Left analog stick (with deadzone)
		const deadzone = 0.5
		axisX := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		axisY := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
		if axisX < -deadzone {
			left = true
		}
		if axisX > deadzone {
			right = true
		}
		if axisY < -deadzone {
			up = true
		}
		if axisY > deadzone {
			down = true
		}
	}

	r.emulator.SetInput(up, down, left, right, btn1, btn2)

	// SMS Pause button (Enter key triggers NMI)
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		r.emulator.SetPause()
	}
}
