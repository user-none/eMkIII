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

// Font-dependent layout values (updated by ApplyFontSize)
var (
	ListRowHeight    = 40
	ListHeaderHeight = 38

	// Column widths for library list view
	ListColFavorite   = 24
	ListColGenre      = 100
	ListColRegion     = 50
	ListColPlayTime   = 80
	ListColLastPlayed = 100

	// Icon view
	IconCardTextHeight = 34
)

// Icon view constants for grid layouts
const (
	IconMinCardWidth       = 200
	IconDefaultWindowWidth = 800 // Fallback when window width unavailable
)

// Detail screen constants
const (
	DetailArtWidthSmall = 150
	DetailArtWidthLarge = 400
)

// Settings screen constants
const (
	SettingsSidebarMinWidth     = 180
	SettingsFolderListMinHeight = 100
)

// Font-dependent settings layout value (updated by ApplyFontSize)
var SettingsRowHeight = 38

// Progress bar constants
const (
	ProgressBarWidth  = 300
	ProgressBarHeight = 20
)

// Font-dependent scroll estimation (updated by ApplyFontSize)
var EstimatedViewportHeight = 400

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

// Achievement UI constants
const (
	AchievementRowSpacing = 4
)

// Font-dependent achievement values (updated by ApplyFontSize)
var (
	AchievementBadgeSize      = 56
	AchievementRowHeight      = 92
	AchievementOverlayWidth   = 500
	AchievementOverlayPadding = 16
)
