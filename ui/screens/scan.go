//go:build !libretro

package screens

import (
	"fmt"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

// ScanProgress represents progress updates from the scanner
// This mirrors the ui.ScanProgress type
type ScanProgress struct {
	Phase           int
	Progress        float64
	GamesFound      int
	ArtworkTotal    int
	ArtworkComplete int
	StatusText      string
}

// Scanner interface for decoupling
type Scanner interface {
	Cancel()
}

// ScanProgressScreen displays ROM scanning progress
type ScanProgressScreen struct {
	callback        ScreenCallback
	scanner         Scanner
	phase           int
	progress        float64
	gamesFound      int
	artworkTotal    int
	artworkComplete int
	statusText      string
	cancelPending   bool
	cancelled       bool
}

// NewScanProgressScreen creates a new scan progress screen
func NewScanProgressScreen(callback ScreenCallback) *ScanProgressScreen {
	return &ScanProgressScreen{
		callback:   callback,
		statusText: "Initializing...",
	}
}

// SetScanner sets the active scanner
func (s *ScanProgressScreen) SetScanner(scanner Scanner) {
	s.scanner = scanner
}

// UpdateProgress updates the screen with new progress information
func (s *ScanProgressScreen) UpdateProgress(p ScanProgress) {
	s.phase = p.Phase
	s.progress = p.Progress
	s.gamesFound = p.GamesFound
	s.artworkTotal = p.ArtworkTotal
	s.artworkComplete = p.ArtworkComplete
	s.statusText = p.StatusText
}

// IsCancelled returns true if cancel was requested
func (s *ScanProgressScreen) IsCancelled() bool {
	return s.cancelled
}

// Build creates the scan progress screen UI
func (s *ScanProgressScreen) Build() *widget.Container {
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

	// Status text
	statusText := s.statusText
	if statusText == "" {
		statusText = "Scanning..."
	}
	statusLabel := widget.NewText(
		widget.TextOpts.Text(statusText, getFontFace(), themeText),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(statusLabel)

	// Progress bar background
	progressBg := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeBorder)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(300, 20),
		),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
		)),
	)

	// Progress bar fill (width based on progress)
	fillWidth := int(300 * s.progress)
	if fillWidth < 1 {
		fillWidth = 1
	}
	progressFill := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themePrimary)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(fillWidth, 20),
		),
	)
	progressBg.AddChild(progressFill)

	centerContent.AddChild(progressBg)

	// Percentage text
	percentText := fmt.Sprintf("%.0f%%", s.progress*100)
	percentLabel := widget.NewText(
		widget.TextOpts.Text(percentText, getFontFace(), themeTextSecondary),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(percentLabel)

	// Found count
	foundText := fmt.Sprintf("Found: %d new games", s.gamesFound)
	foundLabel := widget.NewText(
		widget.TextOpts.Text(foundText, getFontFace(), themeText),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(foundLabel)

	// Artwork status (only during artwork phase - phase 2)
	if s.phase == 2 && s.artworkTotal > 0 {
		artworkText := fmt.Sprintf("Downloading artwork: %d/%d", s.artworkComplete, s.artworkTotal)
		artworkLabel := widget.NewText(
			widget.TextOpts.Text(artworkText, getFontFace(), themeText),
			widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		)
		centerContent.AddChild(artworkLabel)
	}

	// Cancel button
	cancelBtnImage := newButtonImage()
	if s.cancelPending {
		cancelBtnImage = newDisabledButtonImage()
	}

	cancelButton := widget.NewButton(
		widget.ButtonOpts.Image(cancelBtnImage),
		widget.ButtonOpts.Text("Cancel", getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if !s.cancelPending {
				s.cancelPending = true
				s.cancelled = true
				if s.scanner != nil {
					s.scanner.Cancel()
				}
			}
		}),
	)
	centerContent.AddChild(cancelButton)

	rootContainer.AddChild(centerContent)

	return rootContainer
}

// OnEnter is called when entering the scan progress screen
func (s *ScanProgressScreen) OnEnter() {
	s.cancelPending = false
	s.cancelled = false
	s.progress = 0
	s.gamesFound = 0
	s.artworkTotal = 0
	s.artworkComplete = 0
	s.statusText = "Initializing..."
}

// OnExit is called when leaving the scan progress screen
func (s *ScanProgressScreen) OnExit() {
	s.scanner = nil
}
