//go:build !libretro

package screens

import (
	"fmt"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

// ErrorScreen displays startup errors for corrupted config/library files
type ErrorScreen struct {
	callback ScreenCallback
	filename string // "config.json" or "library.json"
	filepath string // Full path to the corrupted file
	onDelete func() // Callback for delete and continue
}

// NewErrorScreen creates a new error screen
func NewErrorScreen(callback ScreenCallback, filename, filepath string, onDelete func()) *ErrorScreen {
	return &ErrorScreen{
		callback: callback,
		filename: filename,
		filepath: filepath,
		onDelete: onDelete,
	}
}

// Build creates the error screen UI
func (s *ErrorScreen) Build() *widget.Container {
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeBackground)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	centerContent := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(16),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	)

	// Title
	titleLabel := widget.NewText(
		widget.TextOpts.Text("Configuration Error", getFontFace(), themeText),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(titleLabel)

	// Message
	msgText := fmt.Sprintf("The file \"%s\" is invalid or corrupted.", s.filename)
	msgLabel := widget.NewText(
		widget.TextOpts.Text(msgText, getFontFace(), themeText),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(msgLabel)

	// Help text
	helpLabel := widget.NewText(
		widget.TextOpts.Text("You can delete the file and start fresh, or exit to manually fix the file.", getFontFace(), themeTextSecondary),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(helpLabel)

	// Buttons container
	buttonsContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(16),
		)),
	)

	// Delete and Continue button
	deleteButton := widget.NewButton(
		widget.ButtonOpts.Image(newButtonImage()),
		widget.ButtonOpts.Text("Delete and Continue", getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if s.onDelete != nil {
				s.onDelete()
			}
		}),
	)
	buttonsContainer.AddChild(deleteButton)

	// Exit button
	exitButton := widget.NewButton(
		widget.ButtonOpts.Image(newButtonImage()),
		widget.ButtonOpts.Text("Exit", getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.callback.Exit()
		}),
	)
	buttonsContainer.AddChild(exitButton)

	centerContent.AddChild(buttonsContainer)
	rootContainer.AddChild(centerContent)

	return rootContainer
}

// OnEnter is called when entering the error screen
func (s *ErrorScreen) OnEnter() {
	// Nothing to do
}

// OnExit is called when leaving the error screen
func (s *ErrorScreen) OnExit() {
	// Nothing to clean up
}
