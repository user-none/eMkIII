//go:build !libretro

package ui

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/user-none/emkiii/ui/style"
)

// UINavigation represents the result of UI input polling
type UINavigation struct {
	Direction    int  // 0=none, 1=prev, 2=next
	Activate     bool // A/Cross button just pressed
	Back         bool // B/Circle button just pressed
	OpenSettings bool // Start button just pressed
	FocusChanged bool // True if navigation caused focus change this frame
}

// InputManager handles all input for UI navigation.
// It tracks gamepad state, handles repeat navigation, and provides
// a clean interface for UI code to query input state.
type InputManager struct {
	// Navigation state for repeat handling
	direction    int           // 0=none, 1=prev, 2=next
	startTime    time.Time     // When direction was first pressed
	lastMove     time.Time     // When last move occurred
	repeatDelay  time.Duration // Current repeat interval
	focusChanged bool          // Track if focus changed this frame
}

// NewInputManager creates a new input manager
func NewInputManager() *InputManager {
	return &InputManager{
		repeatDelay: style.NavStartInterval,
	}
}

// Update polls input state. Should be called once per frame.
// Returns true if F12 (screenshot) was just pressed.
func (im *InputManager) Update() (screenshotRequested bool) {
	// Check for F12 screenshot (global, works everywhere)
	screenshotRequested = inpututil.IsKeyJustPressed(ebiten.KeyF12)
	return screenshotRequested
}

// GetUINavigation returns the current UI navigation state.
// This handles gamepad D-pad/analog stick with repeat navigation,
// and A/B/Start button presses.
func (im *InputManager) GetUINavigation() UINavigation {
	result := UINavigation{}

	gamepadIDs := ebiten.AppendGamepadIDs(nil)
	if len(gamepadIDs) == 0 {
		// No gamepad - reset navigation state
		im.direction = 0
		im.repeatDelay = style.NavStartInterval
		return result
	}

	// Use first connected gamepad
	id := gamepadIDs[0]

	// Determine current navigation direction from D-pad and analog stick
	navPrev := false
	navNext := false

	// D-pad
	if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop) ||
		ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
		navPrev = true
	}
	if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom) ||
		ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight) {
		navNext = true
	}

	// Analog stick (UI uses 0.5 threshold)
	axisY := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
	axisX := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
	if axisY < -0.5 || axisX < -0.5 {
		navPrev = true
	}
	if axisY > 0.5 || axisX > 0.5 {
		navNext = true
	}

	// Determine desired direction (prev takes priority if both pressed)
	desiredDir := 0
	if navPrev {
		desiredDir = 1
	} else if navNext {
		desiredDir = 2
	}

	now := time.Now()
	im.focusChanged = false

	if desiredDir == 0 {
		// No direction pressed - reset state
		im.direction = 0
		im.repeatDelay = style.NavStartInterval
	} else if desiredDir != im.direction {
		// Direction changed - move immediately and start tracking
		im.direction = desiredDir
		im.startTime = now
		im.lastMove = now
		im.repeatDelay = style.NavStartInterval
		im.focusChanged = true
		result.Direction = desiredDir
	} else {
		// Same direction held - check for repeat
		holdDuration := now.Sub(im.startTime)
		timeSinceLastMove := now.Sub(im.lastMove)

		if holdDuration >= style.NavInitialDelay && timeSinceLastMove >= im.repeatDelay {
			// Time to repeat
			im.focusChanged = true
			im.lastMove = now
			result.Direction = desiredDir

			// Accelerate (decrease interval)
			im.repeatDelay -= style.NavAcceleration
			if im.repeatDelay < style.NavMinInterval {
				im.repeatDelay = style.NavMinInterval
			}
		}
	}

	result.FocusChanged = im.focusChanged

	// A/Cross button activates focused widget
	result.Activate = inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom)

	// B/Circle button for back navigation
	result.Back = inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightRight)

	// Start button for settings
	result.OpenSettings = inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight)

	return result
}
