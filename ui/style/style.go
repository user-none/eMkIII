//go:build !libretro

package style

import (
	goimage "image"
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
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

// ScrollSlider creates a vertical scroll slider bound to a scroll container.
// The needsScroll function should return true when content exceeds view height.
// Returns the slider widget.
func ScrollSlider(scrollContainer *widget.ScrollContainer, needsScroll func() bool) *widget.Slider {
	return widget.NewSlider(
		widget.SliderOpts.TabOrder(-1), // Non-focusable for gamepad navigation
		widget.SliderOpts.Direction(widget.DirectionVertical),
		widget.SliderOpts.MinMax(0, 1000),
		widget.SliderOpts.Images(
			&widget.SliderTrackImage{
				Idle:  image.NewNineSliceColor(Border),
				Hover: image.NewNineSliceColor(Border),
			},
			SliderButtonImage(),
		),
		widget.SliderOpts.FixedHandleSize(40),
		widget.SliderOpts.PageSizeFunc(func() int {
			if !needsScroll() {
				return 1000 // Handle fills track - no scrolling needed
			}
			viewHeight := scrollContainer.ViewRect().Dy()
			contentHeight := scrollContainer.ContentRect().Dy()
			return int(float64(viewHeight) / float64(contentHeight) * 1000)
		}),
		widget.SliderOpts.ChangedHandler(func(args *widget.SliderChangedEventArgs) {
			if !needsScroll() {
				scrollContainer.ScrollTop = 0
				return
			}
			scrollContainer.ScrollTop = float64(args.Current) / 1000
		}),
	)
}

// SetupScrollHandler adds mouse wheel scroll support to a scroll container.
// The slider's Current value is kept in sync with scroll position.
func SetupScrollHandler(scrollContainer *widget.ScrollContainer, vSlider *widget.Slider, needsScroll func() bool) {
	scrollContainer.GetWidget().ScrolledEvent.AddHandler(func(args interface{}) {
		if !needsScroll() {
			scrollContainer.ScrollTop = 0
			return
		}
		a := args.(*widget.WidgetScrolledEventArgs)
		p := scrollContainer.ScrollTop + (a.Y * 0.05)
		if p < 0 {
			p = 0
		}
		if p > 1 {
			p = 1
		}
		scrollContainer.ScrollTop = p
		vSlider.Current = int(p * 1000)
	})
}

// DisabledSidebarItem creates a non-focusable sidebar item with the given label.
// Used for future/coming-soon menu items.
func DisabledSidebarItem(label string) *widget.Container {
	item := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(Border)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(8)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)
	item.AddChild(widget.NewText(
		widget.TextOpts.Text(label, FontFace(), TextSecondary),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	))
	return item
}

// ScaleImage scales an image to fit within maxWidth x maxHeight while preserving aspect ratio.
// Returns an ebiten.Image suitable for display.
func ScaleImage(src goimage.Image, maxWidth, maxHeight int) *ebiten.Image {
	bounds := src.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Calculate scale to fit within max dimensions
	scaleX := float64(maxWidth) / float64(srcWidth)
	scaleY := float64(maxHeight) / float64(srcHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Calculate new dimensions
	newWidth := int(float64(srcWidth) * scale)
	newHeight := int(float64(srcHeight) * scale)

	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	// Create source ebiten image
	srcEbiten := ebiten.NewImageFromImage(src)

	// Create destination image and draw scaled
	dst := ebiten.NewImage(newWidth, newHeight)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(srcEbiten, op)

	return dst
}

// TruncateStart truncates a string from the start, keeping the end portion.
// Returns the truncated string and whether truncation occurred.
// Useful for file paths where the end (filename) is most relevant.
func TruncateStart(s string, maxLen int) (string, bool) {
	if len(s) <= maxLen {
		return s, false
	}
	if maxLen <= 3 {
		return s[len(s)-maxLen:], true
	}
	return "..." + s[len(s)-maxLen+3:], true
}

// TruncateEnd truncates a string from the end, keeping the start portion.
// Returns the truncated string and whether truncation occurred.
// Useful for titles where the beginning is most relevant.
func TruncateEnd(s string, maxLen int) (string, bool) {
	if len(s) <= maxLen {
		return s, false
	}
	if maxLen <= 3 {
		return s[:maxLen], true
	}
	return s[:maxLen-3] + "...", true
}
