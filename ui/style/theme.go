//go:build !libretro

package style

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"
)

// Theme colors
var (
	Background    = color.NRGBA{0x1a, 0x1a, 0x2e, 0xff} // Dark blue-gray
	Surface       = color.NRGBA{0x25, 0x25, 0x3a, 0xff} // Slightly lighter
	Primary       = color.NRGBA{0x4a, 0x4a, 0x8a, 0xff} // Muted purple
	PrimaryHover  = color.NRGBA{0x5a, 0x5a, 0x9a, 0xff}
	Text          = color.NRGBA{0xff, 0xff, 0xff, 0xff}
	TextSecondary = color.NRGBA{0xaa, 0xaa, 0xaa, 0xff}
	Accent        = color.NRGBA{0xff, 0xd7, 0x00, 0xff} // Gold for favorites
	Border        = color.NRGBA{0x3a, 0x3a, 0x5a, 0xff}
	Black         = color.NRGBA{0x00, 0x00, 0x00, 0xff}
)

// fontFace is the cached font face
var fontFace text.Face

// FontFace returns the font face to use for UI text
func FontFace() text.Face {
	if fontFace == nil {
		fontFace = text.NewGoXFace(basicfont.Face7x13)
	}
	return fontFace
}

// ButtonImage creates a standard button image set
func ButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:     image.NewNineSliceColor(Surface),
		Hover:    image.NewNineSliceColor(PrimaryHover),
		Pressed:  image.NewNineSliceColor(Primary),
		Disabled: image.NewNineSliceColor(Border),
	}
}

// PrimaryButtonImage creates a prominent button image set
func PrimaryButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:     image.NewNineSliceColor(Primary),
		Hover:    image.NewNineSliceColor(PrimaryHover),
		Pressed:  image.NewNineSliceColor(Surface),
		Disabled: image.NewNineSliceColor(Border),
	}
}

// DisabledButtonImage creates a disabled-looking button image set
func DisabledButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:     image.NewNineSliceColor(Border),
		Hover:    image.NewNineSliceColor(Border),
		Pressed:  image.NewNineSliceColor(Border),
		Disabled: image.NewNineSliceColor(Border),
	}
}

// ActiveButtonImage returns a button image based on active state.
// Used for toggle buttons like view mode selectors and sidebar items.
func ActiveButtonImage(active bool) *widget.ButtonImage {
	if active {
		return PrimaryButtonImage()
	}
	return ButtonImage()
}

// SliderButtonImage creates a slider handle button image
func SliderButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:     image.NewNineSliceColor(Primary),
		Hover:    image.NewNineSliceColor(PrimaryHover),
		Pressed:  image.NewNineSliceColor(Primary),
		Disabled: image.NewNineSliceColor(Border),
	}
}

// SliderTrackImage creates a slider track image
func SliderTrackImage() *widget.SliderTrackImage {
	return &widget.SliderTrackImage{
		Idle:  image.NewNineSliceColor(Border),
		Hover: image.NewNineSliceColor(Border),
	}
}

// ScrollContainerImage creates a scroll container image
func ScrollContainerImage() *widget.ScrollContainerImage {
	return &widget.ScrollContainerImage{
		Idle: image.NewNineSliceColor(Background),
		Mask: image.NewNineSliceColor(Background),
	}
}

// ButtonTextColor returns the standard button text colors
func ButtonTextColor() *widget.ButtonTextColor {
	return &widget.ButtonTextColor{
		Idle:     Text,
		Disabled: TextSecondary,
	}
}
