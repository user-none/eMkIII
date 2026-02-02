//go:build !libretro

// Package types provides shared interfaces used across UI packages.
// This package exists to avoid import cycles between screens and sub-packages.
package types

import (
	"github.com/ebitenui/ebitenui/widget"
)

// ScreenCallback provides callbacks for screen navigation
type ScreenCallback interface {
	SwitchToLibrary()
	SwitchToDetail(gameCRC string)
	SwitchToSettings()
	SwitchToScanProgress(rescanAll bool)
	LaunchGame(gameCRC string, resume bool)
	Exit()
	GetWindowWidth() int             // For responsive layout calculations
	RequestRebuild()                 // Request UI rebuild after state changes
	GetPlaceholderImageData() []byte // Get raw placeholder image data for missing artwork
}

// FocusRestorer is implemented by screens that support focus restoration after rebuilds
type FocusRestorer interface {
	// GetPendingFocusButton returns the button that should receive focus after rebuild
	GetPendingFocusButton() *widget.Button
	// ClearPendingFocus clears the pending focus state
	ClearPendingFocus()
}

// FocusManager interface for focus restoration and scroll management.
// Implemented by BaseScreen, used by sub-sections that need to register
// focusable buttons and manage scroll position.
type FocusManager interface {
	RegisterFocusButton(key string, btn *widget.Button)
	SetPendingFocus(key string)
	SetScrollWidgets(sc *widget.ScrollContainer, slider *widget.Slider)
	SaveScrollPosition()
	RestoreScrollPosition()
}
