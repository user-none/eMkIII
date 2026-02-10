//go:build !libretro

package screens

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/user-none/emkiii/romloader"
	"github.com/user-none/emkiii/ui/achievements"
	"github.com/user-none/emkiii/ui/storage"
	"github.com/user-none/emkiii/ui/style"
	"github.com/user-none/go-rcheevos"
)

// DetailScreen displays game information and launch options
type DetailScreen struct {
	BaseScreen // Embedded for focus restoration

	callback           ScreenCallback
	library            *storage.Library
	config             *storage.Config
	game               *storage.GameEntry
	achievementManager *achievements.Manager

	// Achievement loading state
	achMu       sync.Mutex
	achLoading  bool
	achLoadErr  error
	achFound    bool
	achProgress *rcheevos.UserProgressEntry
}

// NewDetailScreen creates a new detail screen
func NewDetailScreen(callback ScreenCallback, library *storage.Library, config *storage.Config, achievementManager *achievements.Manager) *DetailScreen {
	s := &DetailScreen{
		callback:           callback,
		library:            library,
		config:             config,
		achievementManager: achievementManager,
	}
	s.InitBase()
	return s
}

// SetGame sets the game to display
func (s *DetailScreen) SetGame(gameCRC string) {
	s.game = s.library.GetGame(gameCRC)

	// Reset achievement state
	s.achMu.Lock()
	s.achLoading = false
	s.achLoadErr = nil
	s.achFound = false
	s.achProgress = nil
	s.achMu.Unlock()

	// Start async achievement lookup if logged in
	if s.achievementManager != nil && s.achievementManager.IsLoggedIn() {
		s.achMu.Lock()
		s.achLoading = true
		s.achMu.Unlock()
		go s.loadAchievementProgress()
	}
}

// loadAchievementProgress loads achievement progress for the current game
func (s *DetailScreen) loadAchievementProgress() {
	if s.game == nil || s.achievementManager == nil {
		s.achMu.Lock()
		s.achLoading = false
		s.achMu.Unlock()
		return
	}

	// Ensure libraries are loaded first
	done := make(chan bool, 1)
	s.achievementManager.EnsureLibrariesLoaded(func(success bool) {
		done <- success
	})
	if !<-done {
		s.achMu.Lock()
		s.achLoading = false
		s.achLoadErr = fmt.Errorf("failed to load achievement data")
		s.achMu.Unlock()
		s.callback.RequestRebuild()
		return
	}

	// If achievements were unlocked during gameplay, refresh the cached progress
	if s.achievementManager.IsProgressDirty() {
		refreshDone := make(chan bool, 1)
		s.achievementManager.RefreshUserProgress(func(success bool) {
			refreshDone <- success
		})
		<-refreshDone
		s.achievementManager.ClearProgressDirty()
	}

	// Get MD5 from RDB (fast path - no ROM loading needed)
	var md5Hash string
	rdb := s.callback.GetRDB()
	if rdb != nil {
		crc32, _ := strconv.ParseUint(s.game.CRC32, 16, 32)
		md5Hash = rdb.GetMD5ByCRC32(uint32(crc32))
	}

	// Fallback: compute hash from ROM if not in RDB
	if md5Hash == "" {
		romData, _, err := romloader.LoadROM(s.game.File)
		if err != nil {
			s.achMu.Lock()
			s.achLoading = false
			s.achLoadErr = err
			s.achMu.Unlock()
			s.callback.RequestRebuild()
			return
		}
		md5Hash = s.achievementManager.ComputeGameHash(romData)
	}

	// Look up progress using MD5
	found, progress := s.achievementManager.LookupGameProgress(md5Hash)
	s.achMu.Lock()
	s.achFound = found
	s.achProgress = progress
	s.achLoading = false
	s.achMu.Unlock()
	s.callback.RequestRebuild()
}

// Build creates the detail screen UI
func (s *DetailScreen) Build() *widget.Container {
	s.ClearFocusButtons()

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

	// Outer metadata container that anchors content to top-left
	metadataOuter := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
	)

	// Calculate available width for metadata based on window size
	// Use all available space: window - art - padding (left + right) - spacing between art and metadata
	metadataWidth := windowWidth - artWidth - style.DefaultPadding*2 - style.LargeSpacing
	if metadataWidth < 200 {
		metadataWidth = 200
	}

	// Calculate max characters for value text
	// Value column = metadataWidth - label width (80) - grid spacing (16) - grid padding (16)
	valueWidth := metadataWidth - 80 - style.DefaultSpacing - style.SmallSpacing*2
	maxValueChars := valueWidth / 7 // ~7 pixels per character
	if maxValueChars < 15 {
		maxValueChars = 15
	}

	// Inner metadata container with fixed width
	metadataContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(style.SmallSpacing),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
			}),
			widget.WidgetOpts.MinSize(metadataWidth, 0),
		),
	)

	// Game Info section
	metadataContainer.AddChild(s.buildSectionHeader("Game Info"))

	// Title (with warning icon if missing)
	titleText := s.game.DisplayName
	if s.game.Missing {
		titleText = "[!] " + titleText
	}
	metadataContainer.AddChild(s.buildMetadataRow("Title", titleText, maxValueChars))

	if s.game.Name != "" && s.game.Name != s.game.DisplayName {
		metadataContainer.AddChild(s.buildMetadataRow("Name", s.game.Name, maxValueChars))
	}
	region := strings.ToUpper(s.game.Region)
	if region == "" {
		region = "Unknown"
	}
	metadataContainer.AddChild(s.buildMetadataRow("Region", region, maxValueChars))

	// Production section
	hasProduction := s.game.Developer != "" || s.game.Publisher != "" ||
		s.game.Genre != "" || s.game.Franchise != "" ||
		s.game.ReleaseDate != "" || s.game.ESRBRating != ""
	if hasProduction {
		metadataContainer.AddChild(s.buildSectionHeader("Production"))
		if s.game.Developer != "" {
			metadataContainer.AddChild(s.buildMetadataRow("Developer", s.game.Developer, maxValueChars))
		}
		if s.game.Publisher != "" {
			metadataContainer.AddChild(s.buildMetadataRow("Publisher", s.game.Publisher, maxValueChars))
		}
		if s.game.Genre != "" {
			metadataContainer.AddChild(s.buildMetadataRow("Genre", s.game.Genre, maxValueChars))
		}
		if s.game.Franchise != "" {
			metadataContainer.AddChild(s.buildMetadataRow("Franchise", s.game.Franchise, maxValueChars))
		}
		if s.game.ReleaseDate != "" {
			metadataContainer.AddChild(s.buildMetadataRow("Released", s.game.ReleaseDate, maxValueChars))
		}
		if s.game.ESRBRating != "" {
			metadataContainer.AddChild(s.buildMetadataRow("ESRB", s.game.ESRBRating, maxValueChars))
		}
	}

	// Activity section
	metadataContainer.AddChild(s.buildSectionHeader("Activity"))
	metadataContainer.AddChild(s.buildMetadataRow("Play Time", style.FormatPlayTime(s.game.PlayTimeSeconds), maxValueChars))
	metadataContainer.AddChild(s.buildMetadataRow("Last Played", style.FormatLastPlayed(s.game.LastPlayed), maxValueChars))
	metadataContainer.AddChild(s.buildMetadataRow("Added", style.FormatDate(s.game.Added), maxValueChars))

	// Achievements section (only if logged in)
	if s.achievementManager != nil && s.achievementManager.IsLoggedIn() {
		metadataContainer.AddChild(s.buildSectionHeader("Achievements"))
		metadataContainer.AddChild(s.buildAchievementSection(maxValueChars))
	}

	// Missing ROM warning
	if s.game.Missing {
		metadataContainer.AddChild(s.buildSectionHeader("Warning"))
		warningRow := widget.NewContainer(
			widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Surface)),
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(style.SmallSpacing)),
			)),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
			),
		)
		warningLabel := widget.NewText(
			widget.TextOpts.Text("ROM file not found", style.FontFace(), style.Accent),
		)
		warningRow.AddChild(warningLabel)
		metadataContainer.AddChild(warningRow)
	}

	metadataOuter.AddChild(metadataContainer)
	contentContainer.AddChild(metadataOuter)
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
		s.RegisterFocusButton("play", playButton)
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
		s.RegisterFocusButton("remove", removeButton)
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
		s.callback.RequestRebuild()
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
	if s.game != nil && !s.game.Missing {
		s.SetPendingFocus("play")
	} else {
		s.SetPendingFocus("remove")
	}
}

// buildSectionHeader creates a section header label with accent color
func (s *DetailScreen) buildSectionHeader(title string) *widget.Container {
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(style.TinySpacing),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)

	label := widget.NewText(
		widget.TextOpts.Text(title, style.FontFace(), style.Accent),
	)
	container.AddChild(label)

	return container
}

// buildMetadataRow creates a metadata row with background, label on left, value on right
// maxValueChars specifies the maximum characters for the value before truncation
func (s *DetailScreen) buildMetadataRow(label, value string, maxValueChars int) *widget.Container {
	row := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Surface)),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{false, true}, []bool{true}),
			widget.GridLayoutOpts.Spacing(style.DefaultSpacing, 0),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(style.SmallSpacing)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)

	// Label (fixed width for alignment)
	labelText := widget.NewText(
		widget.TextOpts.Text(label, style.FontFace(), style.TextSecondary),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				VerticalPosition: widget.GridLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(80, 0),
		),
	)
	row.AddChild(labelText)

	// Value (truncated if necessary)
	displayValue, wasTruncated := style.TruncateEnd(value, maxValueChars)

	valueOpts := []widget.TextOpt{
		widget.TextOpts.Text(displayValue, style.FontFace(), style.Text),
	}

	// Add tooltip with full value if truncated
	if wasTruncated {
		valueOpts = append(valueOpts, widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				VerticalPosition: widget.GridLayoutPositionCenter,
			}),
			widget.WidgetOpts.ToolTip(
				widget.NewToolTip(
					widget.ToolTipOpts.Content(style.TooltipContent(value)),
				),
			),
		))
	} else {
		valueOpts = append(valueOpts, widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				VerticalPosition: widget.GridLayoutPositionCenter,
			}),
		))
	}

	valueText := widget.NewText(valueOpts...)
	row.AddChild(valueText)

	return row
}

// buildAchievementSection creates the achievements section content
func (s *DetailScreen) buildAchievementSection(maxValueChars int) *widget.Container {
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(style.SmallSpacing),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)

	s.achMu.Lock()
	loading := s.achLoading
	loadErr := s.achLoadErr
	found := s.achFound
	progress := s.achProgress
	s.achMu.Unlock()

	if loading {
		container.AddChild(s.buildMetadataRow("Status", "Loading...", maxValueChars))
		return container
	}

	if loadErr != nil {
		container.AddChild(s.buildMetadataRow("Status", "Unable to load", maxValueChars))
		return container
	}

	if !found {
		container.AddChild(s.buildMetadataRow("Status", "Not found", maxValueChars))
		return container
	}

	if progress == nil || progress.NumAchievements == 0 {
		container.AddChild(s.buildMetadataRow("Status", "No achievements", maxValueChars))
		return container
	}

	// Has achievements - show progress
	pct := 0
	if progress.NumAchievements > 0 {
		pct = int(progress.NumUnlockedAchievements * 100 / progress.NumAchievements)
	}
	progressText := fmt.Sprintf("%d / %d (%d%%)",
		progress.NumUnlockedAchievements, progress.NumAchievements, pct)
	container.AddChild(s.buildMetadataRow("Progress", progressText, maxValueChars))

	return container
}
