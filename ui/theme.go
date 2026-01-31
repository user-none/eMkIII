//go:build !libretro

package ui

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/user-none/emkiii/ui/style"
)

// ThemeColors contains all theme color definitions
// Re-exported from style package for backwards compatibility
type ThemeColors struct {
	Background    color.Color
	Surface       color.Color
	Primary       color.Color
	PrimaryHover  color.Color
	Text          color.Color
	TextSecondary color.Color
	Accent        color.Color
	Border        color.Color
}

// Theme is the global theme configuration
// Re-exported from style package for backwards compatibility
var Theme = ThemeColors{
	Background:    style.Background,
	Surface:       style.Surface,
	Primary:       style.Primary,
	PrimaryHover:  style.PrimaryHover,
	Text:          style.Text,
	TextSecondary: style.TextSecondary,
	Accent:        style.Accent,
	Border:        style.Border,
}

// GetFontFace returns the font face to use for UI text
func GetFontFace() text.Face {
	return style.FontFace()
}

// NewButtonImage creates a standard button image set
func NewButtonImage() *widget.ButtonImage {
	return style.ButtonImage()
}

// NewPrimaryButtonImage creates a prominent button image set
func NewPrimaryButtonImage() *widget.ButtonImage {
	return style.PrimaryButtonImage()
}

// NewSliderTrackImage creates a slider track image
func NewSliderTrackImage() *widget.SliderTrackImage {
	return style.SliderTrackImage()
}

// NewSliderButtonImage creates a slider button image
func NewSliderButtonImage() *widget.ButtonImage {
	return style.SliderButtonImage()
}
