//go:build !libretro

package screens

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/user-none/emkiii/ui/storage"
)

// DetailScreen displays game information and launch options
type DetailScreen struct {
	callback ScreenCallback
	library  *storage.Library
	config   *storage.Config
	game     *storage.GameEntry
}

// NewDetailScreen creates a new detail screen
func NewDetailScreen(callback ScreenCallback, library *storage.Library, config *storage.Config) *DetailScreen {
	return &DetailScreen{
		callback: callback,
		library:  library,
		config:   config,
	}
}

// SetGame sets the game to display
func (s *DetailScreen) SetGame(crc32 string) {
	s.game = s.library.GetGame(crc32)
}

// Build creates the detail screen UI
func (s *DetailScreen) Build() *widget.Container {
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeBackground)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(16)),
			widget.RowLayoutOpts.Spacing(16),
		)),
	)

	// Back button
	backButton := widget.NewButton(
		widget.ButtonOpts.Image(newButtonImage()),
		widget.ButtonOpts.Text("Back", getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(8)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.callback.SwitchToLibrary()
		}),
	)
	rootContainer.AddChild(backButton)

	if s.game == nil {
		errorLabel := widget.NewText(
			widget.TextOpts.Text("Game not found", getFontFace(), themeText),
		)
		rootContainer.AddChild(errorLabel)
		return rootContainer
	}

	// Main content container (horizontal: box art | metadata)
	contentContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(24),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
	)

	// Calculate art size based on window - use ~40% of width, max 400px
	windowWidth := s.callback.GetWindowWidth()
	artWidth := windowWidth * 40 / 100
	if artWidth < 150 {
		artWidth = 150
	}
	if artWidth > 400 {
		artWidth = 400
	}
	artHeight := artWidth * 4 / 3 // 3:4 aspect ratio for box art

	// Box art container with black background (per design spec)
	artContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(colorBlack)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(artWidth, artHeight),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// Try to load box art image (scaled to fit)
	artImage := s.loadBoxArtScaled(artWidth, artHeight)
	if artImage != nil {
		// Display the actual artwork
		artGraphic := widget.NewGraphic(
			widget.GraphicOpts.Image(artImage),
			widget.GraphicOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
					HorizontalPosition: widget.AnchorLayoutPositionCenter,
					VerticalPosition:   widget.AnchorLayoutPositionCenter,
				}),
			),
		)
		artContainer.AddChild(artGraphic)
	} else {
		// Show placeholder text if no artwork
		artPlaceholder := widget.NewText(
			widget.TextOpts.Text(s.game.DisplayName, getFontFace(), themeTextSecondary),
			widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			})),
		)
		artContainer.AddChild(artPlaceholder)
	}
	contentContainer.AddChild(artContainer)

	// Metadata container
	metadataContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
	)

	// Calculate max characters for metadata text based on available width
	// Available width = window width - art width - padding/spacing
	metadataWidth := windowWidth - artWidth - 80 // 80 for padding and spacing
	maxChars := metadataWidth / 7                // ~7 pixels per character with basic font
	if maxChars < 20 {
		maxChars = 20
	}
	if maxChars > 80 {
		maxChars = 80
	}

	// Title (with warning icon if missing)
	titleText := s.game.DisplayName
	if s.game.Missing {
		titleText = "[!] " + titleText
	}
	metadataContainer.AddChild(s.createMetadataLabel(titleText, maxChars, themeText))

	// Full name (No-Intro format)
	if s.game.Name != "" && s.game.Name != s.game.DisplayName {
		metadataContainer.AddChild(s.createMetadataLabel("Name: "+s.game.Name, maxChars, themeTextSecondary))
	}

	// Region
	region := strings.ToUpper(s.game.Region)
	if region == "" {
		region = "Unknown"
	}
	metadataContainer.AddChild(s.createMetadataLabel("Region: "+region, maxChars, themeTextSecondary))

	// Developer
	if s.game.Developer != "" {
		metadataContainer.AddChild(s.createMetadataLabel("Developer: "+s.game.Developer, maxChars, themeTextSecondary))
	}

	// Publisher
	if s.game.Publisher != "" {
		metadataContainer.AddChild(s.createMetadataLabel("Publisher: "+s.game.Publisher, maxChars, themeTextSecondary))
	}

	// Genre
	if s.game.Genre != "" {
		metadataContainer.AddChild(s.createMetadataLabel("Genre: "+s.game.Genre, maxChars, themeTextSecondary))
	}

	// Franchise
	if s.game.Franchise != "" {
		metadataContainer.AddChild(s.createMetadataLabel("Franchise: "+s.game.Franchise, maxChars, themeTextSecondary))
	}

	// Release Date
	if s.game.ReleaseDate != "" {
		metadataContainer.AddChild(s.createMetadataLabel("Released: "+s.game.ReleaseDate, maxChars, themeTextSecondary))
	}

	// ESRB Rating
	if s.game.ESRBRating != "" {
		metadataContainer.AddChild(s.createMetadataLabel("ESRB: "+s.game.ESRBRating, maxChars, themeTextSecondary))
	}

	// Play time
	metadataContainer.AddChild(s.createMetadataLabel("Play Time: "+s.formatPlayTime(s.game.PlayTimeSeconds), maxChars, themeTextSecondary))

	// Last played
	metadataContainer.AddChild(s.createMetadataLabel("Last Played: "+s.formatLastPlayed(s.game.LastPlayed), maxChars, themeTextSecondary))

	// Added date
	metadataContainer.AddChild(s.createMetadataLabel("Added: "+s.formatDate(s.game.Added), maxChars, themeTextSecondary))

	// Missing ROM warning
	if s.game.Missing {
		warningLabel := widget.NewText(
			widget.TextOpts.Text("ROM file not found", getFontFace(), themeTextSecondary),
		)
		metadataContainer.AddChild(warningLabel)
	}

	contentContainer.AddChild(metadataContainer)
	rootContainer.AddChild(contentContainer)

	// Button container
	buttonContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(16),
		)),
	)

	// Check if resume state exists
	hasResume := s.hasResumeState()

	if !s.game.Missing {
		// Play button
		playButton := widget.NewButton(
			widget.ButtonOpts.Image(newPrimaryButtonImage()),
			widget.ButtonOpts.Text("Play", getFontFace(), &widget.ButtonTextColor{
				Idle:     themeText,
				Disabled: themeTextSecondary,
			}),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				s.callback.LaunchGame(s.game.CRC32, false)
			}),
		)
		buttonContainer.AddChild(playButton)

		// Resume button (enabled only if resume state exists)
		resumeImage := newButtonImage()
		if !hasResume {
			resumeImage = newDisabledButtonImage()
		}

		resumeButton := widget.NewButton(
			widget.ButtonOpts.Image(resumeImage),
			widget.ButtonOpts.Text("Resume", getFontFace(), &widget.ButtonTextColor{
				Idle:     themeText,
				Disabled: themeTextSecondary,
			}),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				if hasResume {
					s.callback.LaunchGame(s.game.CRC32, true)
				}
			}),
		)
		buttonContainer.AddChild(resumeButton)
	} else {
		// Remove from Library button for missing games
		removeButton := widget.NewButton(
			widget.ButtonOpts.Image(newButtonImage()),
			widget.ButtonOpts.Text("Remove from Library", getFontFace(), &widget.ButtonTextColor{
				Idle:     themeText,
				Disabled: themeTextSecondary,
			}),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				s.library.RemoveGame(s.game.CRC32)
				storage.SaveLibrary(s.library)
				s.callback.SwitchToLibrary()
			}),
		)
		buttonContainer.AddChild(removeButton)
	}

	// Favorite toggle
	favText := "Add to Favorites"
	if s.game.Favorite {
		favText = "Remove from Favorites"
	}
	favButton := widget.NewButton(
		widget.ButtonOpts.Image(newButtonImage()),
		widget.ButtonOpts.Text(favText, getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.game.Favorite = !s.game.Favorite
			storage.SaveLibrary(s.library)
		}),
	)
	buttonContainer.AddChild(favButton)

	rootContainer.AddChild(buttonContainer)

	return rootContainer
}

// loadBoxArtScaled loads and scales the box art image for the current game
func (s *DetailScreen) loadBoxArtScaled(maxWidth, maxHeight int) *ebiten.Image {
	if s.game == nil {
		return nil
	}

	artworkPath, err := storage.GetGameArtworkPath(s.game.CRC32)
	if err != nil {
		return nil
	}

	f, err := os.Open(artworkPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil
	}

	// Scale image to fit within maxWidth x maxHeight
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	scaleX := float64(maxWidth) / float64(srcWidth)
	scaleY := float64(maxHeight) / float64(srcHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newWidth := int(float64(srcWidth) * scale)
	newHeight := int(float64(srcHeight) * scale)
	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	srcEbiten := ebiten.NewImageFromImage(img)
	dst := ebiten.NewImage(newWidth, newHeight)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(srcEbiten, op)

	return dst
}

// hasResumeState checks if a resume state exists for the current game
func (s *DetailScreen) hasResumeState() bool {
	saveDir, err := storage.GetGameSaveDir(s.game.CRC32)
	if err != nil {
		return false
	}
	resumePath := filepath.Join(saveDir, "resume.state")
	_, err = os.Stat(resumePath)
	return err == nil
}

// formatPlayTime formats seconds into human-readable format
func (s *DetailScreen) formatPlayTime(seconds int64) string {
	if seconds == 0 {
		return "Never played"
	}
	if seconds < 60 {
		return "< 1m"
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// formatLastPlayed formats a Unix timestamp for display
func (s *DetailScreen) formatLastPlayed(timestamp int64) string {
	if timestamp == 0 {
		return "Never"
	}

	t := time.Unix(timestamp, 0)
	now := time.Now()

	// Check if same day
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return "Today"
	}

	// Check if yesterday
	yesterday := now.AddDate(0, 0, -1)
	if t.Year() == yesterday.Year() && t.YearDay() == yesterday.YearDay() {
		return "Yesterday"
	}

	// This year - show month and day
	if t.Year() == now.Year() {
		return t.Format("Jan 2")
	}

	// Previous years - show full date
	return t.Format("Jan 2, 2006")
}

// formatDate formats a Unix timestamp as a date
func (s *DetailScreen) formatDate(timestamp int64) string {
	if timestamp == 0 {
		return "Unknown"
	}
	return time.Unix(timestamp, 0).Format("Jan 2, 2006")
}

// OnEnter is called when entering the detail screen
func (s *DetailScreen) OnEnter() {
	// Nothing specific to do
}

// OnExit is called when leaving the detail screen
func (s *DetailScreen) OnExit() {
	// Nothing to clean up
}

// Helper functions for button images
func newPrimaryButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:     image.NewNineSliceColor(themePrimary),
		Hover:    image.NewNineSliceColor(themePrimaryHover),
		Pressed:  image.NewNineSliceColor(themeSurface),
		Disabled: image.NewNineSliceColor(themeBorder),
	}
}

func newDisabledButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:     image.NewNineSliceColor(themeBorder),
		Hover:    image.NewNineSliceColor(themeBorder),
		Pressed:  image.NewNineSliceColor(themeBorder),
		Disabled: image.NewNineSliceColor(themeBorder),
	}
}

// colorBlack for box art background
var colorBlack = color.NRGBA{0x00, 0x00, 0x00, 0xff}

// createMetadataLabel creates a text label with optional truncation and tooltip
func (s *DetailScreen) createMetadataLabel(text string, maxChars int, textColor color.Color) *widget.Text {
	displayText := text
	needsTooltip := false

	if len(text) > maxChars {
		displayText = text[:maxChars-3] + "..."
		needsTooltip = true
	}

	opts := []widget.TextOpt{
		widget.TextOpts.Text(displayText, getFontFace(), textColor),
	}

	if needsTooltip {
		tooltipContainer := widget.NewContainer(
			widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeBorder)),
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(8)),
			)),
		)
		tooltipLabel := widget.NewText(
			widget.TextOpts.Text(text, getFontFace(), themeText),
		)
		tooltipContainer.AddChild(tooltipLabel)

		opts = append(opts, widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.ToolTip(
				widget.NewToolTip(
					widget.ToolTipOpts.Content(tooltipContainer),
				),
			),
		))
	}

	return widget.NewText(opts...)
}
