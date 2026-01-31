//go:build !libretro

package ui

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"
)

// ThemeColors contains all theme color definitions
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
var Theme = ThemeColors{
	Background:    color.NRGBA{0x1a, 0x1a, 0x2e, 0xff}, // Dark blue-gray
	Surface:       color.NRGBA{0x25, 0x25, 0x3a, 0xff}, // Slightly lighter
	Primary:       color.NRGBA{0x4a, 0x4a, 0x8a, 0xff}, // Muted purple
	PrimaryHover:  color.NRGBA{0x5a, 0x5a, 0x9a, 0xff},
	Text:          color.NRGBA{0xff, 0xff, 0xff, 0xff},
	TextSecondary: color.NRGBA{0xaa, 0xaa, 0xaa, 0xff},
	Accent:        color.NRGBA{0xff, 0xd7, 0x00, 0xff}, // Gold for favorites
	Border:        color.NRGBA{0x3a, 0x3a, 0x5a, 0xff},
}

// fontFace is the cached font face
var fontFace text.Face

// GetFontFace returns the font face to use for UI text
func GetFontFace() text.Face {
	if fontFace == nil {
		fontFace = text.NewGoXFace(basicfont.Face7x13)
	}
	return fontFace
}

// NewButtonImage creates a standard button image set
func NewButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:     image.NewNineSliceColor(Theme.Surface),
		Hover:    image.NewNineSliceColor(Theme.PrimaryHover),
		Pressed:  image.NewNineSliceColor(Theme.Primary),
		Disabled: image.NewNineSliceColor(Theme.Border),
	}
}

// NewListButtonImage creates a button image set for list items
func NewListButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:     image.NewNineSliceColor(Theme.Surface),
		Hover:    image.NewNineSliceColor(Theme.PrimaryHover),
		Pressed:  image.NewNineSliceColor(Theme.Primary),
		Disabled: image.NewNineSliceColor(Theme.Border),
	}
}

// NewPrimaryButtonImage creates a prominent button image set
func NewPrimaryButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:     image.NewNineSliceColor(Theme.Primary),
		Hover:    image.NewNineSliceColor(Theme.PrimaryHover),
		Pressed:  image.NewNineSliceColor(Theme.Surface),
		Disabled: image.NewNineSliceColor(Theme.Border),
	}
}

// NewListEntryColor creates a color set for list entries
func NewListEntryColor() *widget.ListEntryColor {
	return &widget.ListEntryColor{
		Unselected:                 Theme.Text,
		Selected:                   Theme.Text,
		DisabledUnselected:         Theme.TextSecondary,
		DisabledSelected:           Theme.TextSecondary,
		SelectingBackground:        Theme.PrimaryHover,
		SelectedBackground:         Theme.Primary,
		FocusedBackground:          Theme.PrimaryHover,
		SelectingFocusedBackground: Theme.PrimaryHover,
		SelectedFocusedBackground:  Theme.Primary,
		DisabledSelectedBackground: Theme.Border,
	}
}

// NewScrollContainerImage creates a scroll container image
func NewScrollContainerImage() *widget.ScrollContainerImage {
	return &widget.ScrollContainerImage{
		Idle: image.NewNineSliceColor(Theme.Surface),
		Mask: image.NewNineSliceColor(Theme.Surface),
	}
}

// NewSliderTrackImage creates a slider track image
func NewSliderTrackImage() *widget.SliderTrackImage {
	return &widget.SliderTrackImage{
		Idle:     image.NewNineSliceColor(Theme.Border),
		Hover:    image.NewNineSliceColor(Theme.Border),
		Disabled: image.NewNineSliceColor(Theme.Border),
	}
}

// NewSliderButtonImage creates a slider button image
func NewSliderButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:     image.NewNineSliceColor(Theme.Primary),
		Hover:    image.NewNineSliceColor(Theme.PrimaryHover),
		Pressed:  image.NewNineSliceColor(Theme.Surface),
		Disabled: image.NewNineSliceColor(Theme.Border),
	}
}
