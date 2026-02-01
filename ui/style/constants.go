//go:build !libretro

package style

import "time"

// Layout constants used across screens
const (
	// Standard spacing and padding values
	DefaultPadding = 16
	DefaultSpacing = 16
	SmallSpacing   = 8
	TinySpacing    = 4
	LargeSpacing   = 24

	// Scrollbar dimensions
	ScrollbarWidth = 20

	// Button padding
	ButtonPaddingSmall  = 8
	ButtonPaddingMedium = 12
)

// List view constants for table-style layouts
const (
	ListRowHeight    = 30
	ListHeaderHeight = 28

	// Column widths for library list view
	ListColFavorite   = 24
	ListColGenre      = 100
	ListColRegion     = 50
	ListColPlayTime   = 80
	ListColLastPlayed = 100
)

// Icon view constants for grid layouts
const (
	IconMinCardWidth       = 200
	IconCardTextHeight     = 24
	IconDefaultWindowWidth = 800 // Fallback when window width unavailable
)

// Detail screen constants
const (
	DetailArtWidthSmall = 150
	DetailArtWidthLarge = 400
)

// Settings screen constants
const (
	SettingsRowHeight           = 28
	SettingsSidebarMinWidth     = 160
	SettingsFolderListMinHeight = 100
)

// Progress bar constants
const (
	ProgressBarWidth  = 300
	ProgressBarHeight = 20
)

// Scroll estimation constants
const (
	// Used when estimating scroll position before layout is complete
	EstimatedViewportHeight = 400
)

// Gamepad navigation timing constants
const (
	NavInitialDelay  = 400 * time.Millisecond // Delay before repeat starts
	NavStartInterval = 200 * time.Millisecond // Initial repeat interval
	NavMinInterval   = 25 * time.Millisecond  // Fastest repeat (cap)
	NavAcceleration  = 20 * time.Millisecond  // Speed increase per repeat
)

// Auto-save and timing constants
const (
	AutoSaveInterval = 5 * time.Second
	HTTPTimeout      = 10 * time.Second
)

// Mouse wheel scroll sensitivity
const (
	ScrollWheelSensitivity = 0.05
)
