//go:build !libretro

package screens

import (
	"github.com/ebitenui/ebitenui/widget"
)

// Screen is the interface that all screens implement
type Screen interface {
	// Build creates and returns the root container for this screen
	Build() *widget.Container
	// OnEnter is called when the screen becomes active
	OnEnter()
	// OnExit is called when the screen is being left
	OnExit()
}

// ScreenCallback provides callbacks for screen navigation
type ScreenCallback interface {
	SwitchToLibrary()
	SwitchToDetail(gameCRC string)
	SwitchToSettings()
	SwitchToScanProgress(rescanAll bool)
	LaunchGame(gameCRC string, resume bool)
	Exit()
	GetWindowWidth() int // For responsive layout calculations
	RequestRebuild()     // Request UI rebuild after state changes
}

// FocusRestorer is implemented by screens that support focus restoration after rebuilds
type FocusRestorer interface {
	// GetPendingFocusButton returns the button that should receive focus after rebuild
	GetPendingFocusButton() *widget.Button
	// ClearPendingFocus clears the pending focus state
	ClearPendingFocus()
}
