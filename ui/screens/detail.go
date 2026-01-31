//go:build !libretro

package screens

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/user-none/emkiii/ui/storage"
	"github.com/user-none/emkiii/ui/style"
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
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Background)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(style.DefaultPadding)),
			widget.RowLayoutOpts.Spacing(style.DefaultSpacing),
		)),
	)

	// Back button
	backButton := style.TextButton("Back", style.ButtonPaddingSmall, func(args *widget.ButtonClickedEventArgs) {
		s.callback.SwitchToLibrary()
	})
	rootContainer.AddChild(backButton)

	if s.game == nil {
		errorLabel := widget.NewText(
			widget.TextOpts.Text("Game not found", style.FontFace(), style.Text),
		)
		rootContainer.AddChild(errorLabel)
		return rootContainer
	}

	// Main content container (horizontal: box art | metadata)
	contentContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(style.LargeSpacing),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
	)

	// Calculate art size based on window - use ~40% of width, with min/max bounds
	windowWidth := s.callback.GetWindowWidth()
	artWidth := windowWidth * 40 / 100
	if artWidth < style.DetailArtWidthSmall {
		artWidth = style.DetailArtWidthSmall
	}
	if artWidth > style.DetailArtWidthLarge {
		artWidth = style.DetailArtWidthLarge
	}
	artHeight := artWidth * 4 / 3 // 3:4 aspect ratio for box art

	// Box art container with black background (per design spec)
	artContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Black)),
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
			widget.TextOpts.Text(s.game.DisplayName, style.FontFace(), style.TextSecondary),
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
			widget.RowLayoutOpts.Spacing(style.SmallSpacing),
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
	metadataContainer.AddChild(s.createMetadataLabel(titleText, maxChars, style.Text))

	// Full name (No-Intro format)
	if s.game.Name != "" && s.game.Name != s.game.DisplayName {
		metadataContainer.AddChild(s.createMetadataLabel("Name: "+s.game.Name, maxChars, style.TextSecondary))
	}

	// Region
	region := strings.ToUpper(s.game.Region)
	if region == "" {
		region = "Unknown"
	}
	metadataContainer.AddChild(s.createMetadataLabel("Region: "+region, maxChars, style.TextSecondary))

	// Developer
	if s.game.Developer != "" {
		metadataContainer.AddChild(s.createMetadataLabel("Developer: "+s.game.Developer, maxChars, style.TextSecondary))
	}

	// Publisher
	if s.game.Publisher != "" {
		metadataContainer.AddChild(s.createMetadataLabel("Publisher: "+s.game.Publisher, maxChars, style.TextSecondary))
	}

	// Genre
	if s.game.Genre != "" {
		metadataContainer.AddChild(s.createMetadataLabel("Genre: "+s.game.Genre, maxChars, style.TextSecondary))
	}

	// Franchise
	if s.game.Franchise != "" {
		metadataContainer.AddChild(s.createMetadataLabel("Franchise: "+s.game.Franchise, maxChars, style.TextSecondary))
	}

	// Release Date
	if s.game.ReleaseDate != "" {
		metadataContainer.AddChild(s.createMetadataLabel("Released: "+s.game.ReleaseDate, maxChars, style.TextSecondary))
	}

	// ESRB Rating
	if s.game.ESRBRating != "" {
		metadataContainer.AddChild(s.createMetadataLabel("ESRB: "+s.game.ESRBRating, maxChars, style.TextSecondary))
	}

	// Play time
	metadataContainer.AddChild(s.createMetadataLabel("Play Time: "+style.FormatPlayTime(s.game.PlayTimeSeconds), maxChars, style.TextSecondary))

	// Last played
	metadataContainer.AddChild(s.createMetadataLabel("Last Played: "+style.FormatLastPlayed(s.game.LastPlayed), maxChars, style.TextSecondary))

	// Added date
	metadataContainer.AddChild(s.createMetadataLabel("Added: "+style.FormatDate(s.game.Added), maxChars, style.TextSecondary))

	// Missing ROM warning
	if s.game.Missing {
		warningLabel := widget.NewText(
			widget.TextOpts.Text("ROM file not found", style.FontFace(), style.TextSecondary),
		)
		metadataContainer.AddChild(warningLabel)
	}

	contentContainer.AddChild(metadataContainer)
	rootContainer.AddChild(contentContainer)

	// Button container
	buttonContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(style.DefaultSpacing),
		)),
	)

	// Check if resume state exists
	hasResume := s.hasResumeState()

	if !s.game.Missing {
		// Play button
		playButton := style.PrimaryTextButton("Play", style.ButtonPaddingMedium, func(args *widget.ButtonClickedEventArgs) {
			s.callback.LaunchGame(s.game.CRC32, false)
		})
		buttonContainer.AddChild(playButton)

		// Resume button (enabled only if resume state exists)
		resumeImage := style.ButtonImage()
		if !hasResume {
			resumeImage = style.DisabledButtonImage()
		}

		resumeButton := widget.NewButton(
			widget.ButtonOpts.Image(resumeImage),
			widget.ButtonOpts.Text("Resume", style.FontFace(), style.ButtonTextColor()),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingMedium)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				if hasResume {
					s.callback.LaunchGame(s.game.CRC32, true)
				}
			}),
		)
		buttonContainer.AddChild(resumeButton)
	} else {
		// Remove from Library button for missing games
		removeButton := style.TextButton("Remove from Library", style.ButtonPaddingMedium, func(args *widget.ButtonClickedEventArgs) {
			s.library.RemoveGame(s.game.CRC32)
			storage.SaveLibrary(s.library)
			s.callback.SwitchToLibrary()
		})
		buttonContainer.AddChild(removeButton)
	}

	// Favorite toggle
	favText := "Add to Favorites"
	if s.game.Favorite {
		favText = "Remove from Favorites"
	}
	favButton := style.TextButton(favText, 12, func(args *widget.ButtonClickedEventArgs) {
		s.game.Favorite = !s.game.Favorite
		storage.SaveLibrary(s.library)
	})
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

	return style.ScaleImage(img, maxWidth, maxHeight)
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

// OnEnter is called when entering the detail screen
func (s *DetailScreen) OnEnter() {
	// Nothing specific to do
}

// OnExit is called when leaving the detail screen
func (s *DetailScreen) OnExit() {
	// Nothing to clean up
}

// createMetadataLabel creates a text label with optional truncation and tooltip
func (s *DetailScreen) createMetadataLabel(text string, maxChars int, textColor color.Color) *widget.Text {
	displayText := text
	needsTooltip := false

	if len(text) > maxChars {
		displayText = text[:maxChars-3] + "..."
		needsTooltip = true
	}

	opts := []widget.TextOpt{
		widget.TextOpts.Text(displayText, style.FontFace(), textColor),
	}

	if needsTooltip {
		opts = append(opts, widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.ToolTip(
				widget.NewToolTip(
					widget.ToolTipOpts.Content(style.TooltipContent(text)),
				),
			),
		))
	}

	return widget.NewText(opts...)
}
