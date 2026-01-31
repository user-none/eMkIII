//go:build !libretro

package screens

import (
	"bytes"
	"fmt"
	goimage "image"
	"image/color"
	"os"
	"strings"
	"time"

	_ "image/png"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/user-none/emkiii/ui/storage"
	"golang.org/x/image/font/basicfont"
)

// LibraryScreen displays the game library
type LibraryScreen struct {
	callback ScreenCallback
	library  *storage.Library
	config   *storage.Config

	// UI state
	selectedIndex int
	games         []*storage.GameEntry

	// Selection and scroll preservation (independent for each view)
	iconSelectedCRC string  // CRC of selected game in icon view
	listSelectedCRC string  // CRC of selected game in list view
	iconScrollTop   float64 // Scroll position for icon view
	listScrollTop   float64 // Scroll position for list view

	// Widget references for scroll preservation
	scrollContainer     *widget.ScrollContainer // icon view
	vSlider             *widget.Slider          // icon view
	listScrollContainer *widget.ScrollContainer // list view
	listVSlider         *widget.Slider          // list view

	// Button references for focus restoration (maps CRC to button)
	gameButtons map[string]*widget.Button

	// CRC of game to restore focus to after rebuild
	pendingFocusCRC string

	// Toolbar button references for focus restoration
	toolbarButtons map[string]*widget.Button

	// Key of toolbar button to restore focus to after rebuild
	pendingToolbarFocus string
}

// NewLibraryScreen creates a new library screen
func NewLibraryScreen(callback ScreenCallback, library *storage.Library, config *storage.Config) *LibraryScreen {
	return &LibraryScreen{
		callback:      callback,
		library:       library,
		config:        config,
		selectedIndex: 0,
	}
}

// SetLibrary updates the library reference
func (s *LibraryScreen) SetLibrary(library *storage.Library) {
	s.library = library
}

// SetConfig updates the config reference
func (s *LibraryScreen) SetConfig(config *storage.Config) {
	s.config = config
}

// Build creates the library screen UI
func (s *LibraryScreen) Build() *widget.Container {
	// Initialize button maps for focus restoration
	s.gameButtons = make(map[string]*widget.Button)
	s.toolbarButtons = make(map[string]*widget.Button)

	// Get sorted games
	s.games = s.library.GetGamesSorted(s.config.Library.SortBy, s.config.Library.FavoritesFilter)

	// Check if library is truly empty vs filtered empty
	totalGames := s.library.GameCount()

	// Use anchor layout for root to fill entire window
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeBackground)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// Inner container with vertical layout for toolbar + content
	innerContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(16)),
			widget.GridLayoutOpts.Spacing(16, 16),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				StretchHorizontal: true,
				StretchVertical:   true,
			}),
		),
	)

	if totalGames == 0 {
		// Library is truly empty - no games at all
		innerContainer.AddChild(s.buildEmptyState())
	} else if len(s.games) == 0 {
		// Library has games but filter shows none (e.g., favorites filter with no favorites)
		innerContainer.AddChild(s.buildToolbar())
		innerContainer.AddChild(s.buildFilteredEmptyState())
	} else {
		// Toolbar (row 0 - doesn't stretch)
		innerContainer.AddChild(s.buildToolbar())

		// Game list or grid (row 1 - stretches to fill)
		if s.config.Library.ViewMode == "list" {
			innerContainer.AddChild(s.buildListView())
		} else {
			innerContainer.AddChild(s.buildIconView())
		}
	}

	rootContainer.AddChild(innerContainer)
	return rootContainer
}

// buildEmptyState creates the empty library display
func (s *LibraryScreen) buildEmptyState() *widget.Container {
	emptyContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
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

	titleLabel := widget.NewText(
		widget.TextOpts.Text("No games in library", getFontFace(), themeText),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(titleLabel)

	subtitleLabel := widget.NewText(
		widget.TextOpts.Text("Add a ROM folder in Settings", getFontFace(), themeTextSecondary),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(subtitleLabel)

	settingsButton := widget.NewButton(
		widget.ButtonOpts.Image(newButtonImage()),
		widget.ButtonOpts.Text("Open Settings", getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.callback.SwitchToSettings()
		}),
	)
	centerContent.AddChild(settingsButton)

	emptyContainer.AddChild(centerContent)
	return emptyContainer
}

// buildFilteredEmptyState creates the display when filters hide all games
func (s *LibraryScreen) buildFilteredEmptyState() *widget.Container {
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
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

	titleLabel := widget.NewText(
		widget.TextOpts.Text("No favorites yet", getFontFace(), themeText),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(titleLabel)

	subtitleLabel := widget.NewText(
		widget.TextOpts.Text("Turn off the favorites filter to see all games", getFontFace(), themeTextSecondary),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(subtitleLabel)

	container.AddChild(centerContent)
	return container
}

// buildToolbar creates the library toolbar
func (s *LibraryScreen) buildToolbar() *widget.Container {
	// Use GridLayout with 3 columns: left (view toggles), center (sort/favorites), right (settings)
	toolbar := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{false, true, false}, nil),
			widget.GridLayoutOpts.Spacing(8, 0),
		)),
	)

	// LEFT SECTION: View mode toggles
	leftSection := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)

	iconViewBtn := widget.NewButton(
		widget.ButtonOpts.Image(s.getViewButtonImage(s.config.Library.ViewMode == "icon")),
		widget.ButtonOpts.Text("Icon", getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(8)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.config.Library.ViewMode = "icon"
			storage.SaveConfig(s.config)
			s.pendingToolbarFocus = "icon"
			s.callback.RequestRebuild()
		}),
	)
	s.toolbarButtons["icon"] = iconViewBtn
	leftSection.AddChild(iconViewBtn)

	listViewBtn := widget.NewButton(
		widget.ButtonOpts.Image(s.getViewButtonImage(s.config.Library.ViewMode == "list")),
		widget.ButtonOpts.Text("List", getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(8)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.config.Library.ViewMode = "list"
			storage.SaveConfig(s.config)
			s.pendingToolbarFocus = "list"
			s.callback.RequestRebuild()
		}),
	)
	s.toolbarButtons["list"] = listViewBtn
	leftSection.AddChild(listViewBtn)

	toolbar.AddChild(leftSection)

	// CENTER SECTION: Sort and Favorites
	centerSection := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	centerContent := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	)

	// Sort label with vertical centering
	sortLabelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(0, 32),
		),
	)
	sortLabel := widget.NewText(
		widget.TextOpts.Text("Sort:", getFontFace(), themeText),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				VerticalPosition: widget.AnchorLayoutPositionCenter,
			}),
		),
	)
	sortLabelContainer.AddChild(sortLabel)
	centerContent.AddChild(sortLabelContainer)

	// Sort button
	sortOptions := []string{"Title", "Last Played", "Play Time"}
	sortValues := []string{"title", "lastPlayed", "playTime"}

	currentSortIdx := 0
	for i, v := range sortValues {
		if v == s.config.Library.SortBy {
			currentSortIdx = i
			break
		}
	}

	sortButton := widget.NewButton(
		widget.ButtonOpts.Image(newButtonImage()),
		widget.ButtonOpts.Text(sortOptions[currentSortIdx], getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(8)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			currentSortIdx = (currentSortIdx + 1) % len(sortOptions)
			s.config.Library.SortBy = sortValues[currentSortIdx]
			storage.SaveConfig(s.config)
			s.pendingToolbarFocus = "sort"
			s.callback.RequestRebuild()
		}),
	)
	s.toolbarButtons["sort"] = sortButton
	centerContent.AddChild(sortButton)

	// Favorites button
	favText := "Favorites"
	if s.config.Library.FavoritesFilter {
		favText = "[*] Favorites"
	}
	favButton := widget.NewButton(
		widget.ButtonOpts.Image(s.getViewButtonImage(s.config.Library.FavoritesFilter)),
		widget.ButtonOpts.Text(favText, getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(8)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.config.Library.FavoritesFilter = !s.config.Library.FavoritesFilter
			storage.SaveConfig(s.config)
			s.pendingToolbarFocus = "favorites"
			s.callback.RequestRebuild()
		}),
	)
	s.toolbarButtons["favorites"] = favButton
	centerContent.AddChild(favButton)

	centerSection.AddChild(centerContent)
	toolbar.AddChild(centerSection)

	// RIGHT SECTION: Settings button
	rightSection := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	settingsButton := widget.NewButton(
		widget.ButtonOpts.Image(newButtonImage()),
		widget.ButtonOpts.Text("Settings", getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(8)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.callback.SwitchToSettings()
		}),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionEnd,
			}),
		),
	)
	rightSection.AddChild(settingsButton)

	toolbar.AddChild(rightSection)

	return toolbar
}

// buildListView creates the list view of games using custom ScrollContainer for scroll control
func (s *LibraryScreen) buildListView() widget.PreferredSizeLocateableWidget {
	rowHeight := 30 // Height of each list row
	headerHeight := 28
	selectedIndex := -1

	// Column widths
	colFav := 24
	colGenre := 100
	colRegion := 50
	colPlayTime := 80
	colLastPlayed := 100

	// Helper to create a text cell with proper styling
	createCell := func(text string, width int, stretch bool, textColor color.Color) *widget.Container {
		cell := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(width, rowHeight),
			),
		)
		label := widget.NewText(
			widget.TextOpts.Text(text, getFontFace(), textColor),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
					VerticalPosition: widget.AnchorLayoutPositionCenter,
				}),
			),
		)
		cell.AddChild(label)
		return cell
	}

	// Helper to create header cell
	createHeaderCell := func(text string, width int) *widget.Container {
		cell := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(width, headerHeight),
			),
		)
		label := widget.NewText(
			widget.TextOpts.Text(text, getFontFace(), themeTextSecondary),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
					VerticalPosition: widget.AnchorLayoutPositionCenter,
				}),
			),
		)
		cell.AddChild(label)
		return cell
	}

	// Build header row
	header := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(6),
			widget.GridLayoutOpts.Stretch([]bool{false, true, false, false, false, false}, nil),
			widget.GridLayoutOpts.Spacing(8, 0),
			widget.GridLayoutOpts.Padding(widget.Insets{Left: 8, Right: 8}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(0, headerHeight),
		),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeSurface)),
	)
	header.AddChild(createHeaderCell("", colFav)) // Favorite column (no header text)
	header.AddChild(createHeaderCell("Title", 0)) // Title stretches
	header.AddChild(createHeaderCell("Genre", colGenre))
	header.AddChild(createHeaderCell("Region", colRegion))
	header.AddChild(createHeaderCell("Play Time", colPlayTime))
	header.AddChild(createHeaderCell("Last Played", colLastPlayed))

	// Create vertical container for all game rows
	listContent := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(0),
		)),
	)

	// Add a row for each game
	for i, game := range s.games {
		idx := i
		g := game

		// Track selected index for scroll centering
		if g.CRC32 == s.listSelectedCRC {
			selectedIndex = idx
		}

		// Format cell values
		fav := ""
		if g.Favorite {
			fav = "*"
		}
		region := strings.ToUpper(g.Region)
		if region == "" {
			region = "-"
		}
		genre := g.Genre
		if genre == "" {
			genre = "-"
		}
		playTime := formatPlayTime(g.PlayTimeSeconds)
		lastPlayed := formatLastPlayed(g.LastPlayed)

		// Determine row background color for alternating rows
		var rowIdleBg color.Color
		if idx%2 == 0 {
			rowIdleBg = themeBackground
		} else {
			rowIdleBg = themeSurface
		}

		// Create row container with grid layout (transparent background - button handles colors)
		row := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(6),
				widget.GridLayoutOpts.Stretch([]bool{false, true, false, false, false, false}, nil),
				widget.GridLayoutOpts.Spacing(8, 0),
				widget.GridLayoutOpts.Padding(widget.Insets{Left: 8, Right: 8}),
			)),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(0, rowHeight),
			),
		)

		// Add cells
		row.AddChild(createCell(fav, colFav, false, themeAccent))
		row.AddChild(createCell(g.DisplayName, 0, true, themeText))
		row.AddChild(createCell(genre, colGenre, false, themeTextSecondary))
		row.AddChild(createCell(region, colRegion, false, themeTextSecondary))
		row.AddChild(createCell(playTime, colPlayTime, false, themeTextSecondary))
		row.AddChild(createCell(lastPlayed, colLastPlayed, false, themeTextSecondary))

		// Create button with alternating row color as idle, focus/hover colors for interaction
		gameCRC := g.CRC32 // Capture for closure
		rowButton := widget.NewButton(
			widget.ButtonOpts.Image(&widget.ButtonImage{
				Idle:    image.NewNineSliceColor(rowIdleBg),
				Hover:   image.NewNineSliceColor(themePrimaryHover),
				Pressed: image.NewNineSliceColor(themePrimary),
			}),
			widget.ButtonOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Stretch: true,
				}),
				widget.WidgetOpts.MinSize(0, rowHeight),
			),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				if s.listScrollContainer != nil {
					s.listScrollTop = s.listScrollContainer.ScrollTop
				}
				s.listSelectedCRC = gameCRC
				s.pendingFocusCRC = gameCRC // Remember for focus restoration
				s.callback.SwitchToDetail(gameCRC)
			}),
		)

		// Store button reference for focus restoration
		s.gameButtons[gameCRC] = rowButton

		// Stack: button at bottom (shows background), row content on top (transparent)
		rowWrapper := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewStackedLayout()),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Stretch: true,
				}),
				widget.WidgetOpts.MinSize(0, rowHeight),
			),
		)
		rowWrapper.AddChild(rowButton)
		rowWrapper.AddChild(row)

		listContent.AddChild(rowWrapper)
	}

	// Create scroll container
	scrollContainer := widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(listContent),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
			Idle: image.NewNineSliceColor(themeBackground),
			Mask: image.NewNineSliceColor(themeBackground),
		}),
	)

	// Helper to check if scrolling is needed
	needsScroll := func() bool {
		contentHeight := scrollContainer.ContentRect().Dy()
		viewHeight := scrollContainer.ViewRect().Dy()
		return contentHeight > 0 && viewHeight > 0 && contentHeight > viewHeight
	}

	// Create vertical slider for scrolling (TabOrder -1 makes it non-focusable for gamepad)
	vSlider := widget.NewSlider(
		widget.SliderOpts.TabOrder(-1),
		widget.SliderOpts.Direction(widget.DirectionVertical),
		widget.SliderOpts.MinMax(0, 1000),
		widget.SliderOpts.Images(
			&widget.SliderTrackImage{
				Idle:  image.NewNineSliceColor(themeBorder),
				Hover: image.NewNineSliceColor(themeBorder),
			},
			newSliderButtonImage(),
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

	// Store references for scroll preservation
	s.listScrollContainer = scrollContainer
	s.listVSlider = vSlider

	// Restore or calculate scroll position
	if s.listScrollTop > 0 {
		scrollContainer.ScrollTop = s.listScrollTop
		vSlider.Current = int(s.listScrollTop * 1000)
	} else if selectedIndex >= 0 && len(s.games) > 0 {
		totalHeight := len(s.games) * rowHeight
		selectedY := selectedIndex * rowHeight
		viewportHeight := 400
		targetScrollY := selectedY - (viewportHeight / 2) + (rowHeight / 2)
		if targetScrollY < 0 {
			targetScrollY = 0
		}
		if totalHeight > viewportHeight && targetScrollY > totalHeight-viewportHeight {
			targetScrollY = totalHeight - viewportHeight
		}
		if totalHeight > 0 {
			scrollTop := float64(targetScrollY) / float64(totalHeight)
			if scrollTop > 1 {
				scrollTop = 1
			}
			if scrollTop < 0 {
				scrollTop = 0
			}
			scrollContainer.ScrollTop = scrollTop
			vSlider.Current = int(scrollTop * 1000)
		}
	}

	// Sync scroll container to slider on mouse wheel - only scroll if content exceeds view
	scrollContainer.GetWidget().ScrolledEvent.AddHandler(func(args any) {
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

	// Slider width constant
	sliderWidth := 20

	// Header row with spacer for slider alignment
	headerRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Spacing(4, 0),
			widget.GridLayoutOpts.Stretch([]bool{true, false}, nil),
		)),
	)
	headerRow.AddChild(header)
	// Empty spacer matching slider width
	headerSpacer := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(sliderWidth, 0),
		),
	)
	headerRow.AddChild(headerSpacer)

	// Scroll area with slider
	scrollRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Spacing(4, 0),
			widget.GridLayoutOpts.Stretch([]bool{true, false}, []bool{true}),
		)),
	)
	scrollRow.AddChild(scrollContainer)
	scrollRow.AddChild(vSlider)

	// Main container: header row + scroll area
	mainContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true}),
			widget.GridLayoutOpts.Spacing(0, 4),
		)),
	)
	mainContainer.AddChild(headerRow)
	mainContainer.AddChild(scrollRow)

	return mainContainer
}

// formatLastPlayed formats a Unix timestamp for display
func formatLastPlayed(timestamp int64) string {
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

	// Previous years - show year
	return t.Format("2006")
}

// buildIconView creates the icon/grid view of games with artwork
func (s *LibraryScreen) buildIconView() widget.PreferredSizeLocateableWidget {
	// Calculate responsive grid dimensions
	windowWidth := s.callback.GetWindowWidth()
	if windowWidth < 400 {
		windowWidth = 800 // Default if not yet available
	}

	// Layout constants
	padding := 16        // Padding on sides
	scrollbarWidth := 20 // Width for scrollbar
	minCardWidth := 200  // Minimum card width (matches image 9 look)
	spacing := 8         // Fixed spacing between cards
	textHeight := 24     // Height for title text

	// Available width for cards (subtract padding and scrollbar)
	availableWidth := windowWidth - (padding * 2) - scrollbarWidth

	// Calculate number of columns that fit with minimum card width
	// Formula: columns = floor((availableWidth + spacing) / (minCardWidth + spacing))
	columns := (availableWidth + spacing) / (minCardWidth + spacing)
	if columns < 2 {
		columns = 2
	}

	// Calculate exact card width to fill the available space
	// Formula: cardWidth = (availableWidth - (columns - 1) * spacing) / columns
	cardWidth := (availableWidth - (columns-1)*spacing) / columns

	// Card height maintains ~4:3 aspect ratio for artwork + text
	artHeight := cardWidth * 4 / 3
	cardHeight := artHeight + textHeight

	// Create stretch array - all columns stretch equally to fill width
	columnStretches := make([]bool, columns)
	for i := range columnStretches {
		columnStretches[i] = true
	}

	// Grid container for the cards - columns stretch to fill available width
	gridContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(columns),
			widget.GridLayoutOpts.Spacing(spacing, spacing),
			widget.GridLayoutOpts.Stretch(columnStretches, nil),
		)),
	)

	// Add game cards with calculated dimensions
	for _, game := range s.games {
		card := s.buildGameCardSized(game, cardWidth, cardHeight, artHeight)
		gridContainer.AddChild(card)
	}

	// Create scroll container
	scrollContainer := widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(gridContainer),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
			Idle: image.NewNineSliceColor(themeBackground),
			Mask: image.NewNineSliceColor(themeBackground),
		}),
	)

	// Helper to check if scrolling is needed
	needsScroll := func() bool {
		contentHeight := scrollContainer.ContentRect().Dy()
		viewHeight := scrollContainer.ViewRect().Dy()
		return contentHeight > 0 && viewHeight > 0 && contentHeight > viewHeight
	}

	// Create vertical slider for scrolling (TabOrder -1 makes it non-focusable for gamepad)
	vSlider := widget.NewSlider(
		widget.SliderOpts.TabOrder(-1),
		widget.SliderOpts.Direction(widget.DirectionVertical),
		widget.SliderOpts.MinMax(0, 1000),
		widget.SliderOpts.Images(
			&widget.SliderTrackImage{
				Idle:  image.NewNineSliceColor(themeBorder),
				Hover: image.NewNineSliceColor(themeBorder),
			},
			newSliderButtonImage(),
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

	// Store references for scroll preservation
	s.scrollContainer = scrollContainer
	s.vSlider = vSlider

	// Restore icon view scroll position if we have one
	if s.iconScrollTop > 0 {
		scrollContainer.ScrollTop = s.iconScrollTop
		vSlider.Current = int(s.iconScrollTop * 1000)
	}

	// Sync scroll container to slider on mouse wheel - only scroll if content exceeds view
	scrollContainer.GetWidget().ScrolledEvent.AddHandler(func(args any) {
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

	// Use GridLayout with 2 columns: stretching scroll area + fixed width slider
	scrollRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Spacing(4, 0),
			widget.GridLayoutOpts.Stretch([]bool{true, false}, []bool{true}),
		)),
	)

	scrollRow.AddChild(scrollContainer)
	scrollRow.AddChild(vSlider)

	return scrollRow
}

// buildGameCardSized creates a game card with specific dimensions
func (s *LibraryScreen) buildGameCardSized(game *storage.GameEntry, cardWidth, cardHeight, artHeight int) *widget.Container {
	// Load artwork scaled to fit
	artwork := s.loadGameArtworkSized(game.CRC32, cardWidth, artHeight)

	// Inner card content
	cardContent := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(2),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(cardWidth, cardHeight),
		),
	)

	// Artwork button (clickable)
	gameCRC := game.CRC32 // Capture for closure
	artButton := widget.NewButton(
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle:    image.NewNineSliceColor(themeSurface),
			Hover:   image.NewNineSliceColor(themePrimaryHover),
			Pressed: image.NewNineSliceColor(themePrimary),
		}),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(cardWidth, artHeight),
		),
		widget.ButtonOpts.Graphic(artwork),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			// Save scroll position and selected game before navigating
			s.iconSelectedCRC = gameCRC
			s.pendingFocusCRC = gameCRC // Remember for focus restoration
			if s.scrollContainer != nil {
				s.iconScrollTop = s.scrollContainer.ScrollTop
			}
			s.callback.SwitchToDetail(gameCRC)
		}),
	)

	// Store button reference for focus restoration
	s.gameButtons[gameCRC] = artButton

	cardContent.AddChild(artButton)

	// Game title (truncated based on card width)
	maxChars := cardWidth / 7 // Approximate chars that fit
	if maxChars < 10 {
		maxChars = 10
	}
	displayName := truncateString(game.DisplayName, maxChars)
	titleLabel := widget.NewText(
		widget.TextOpts.Text(displayName, getFontFace(), themeText),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionStart),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
	)
	cardContent.AddChild(titleLabel)

	// Wrapper with AnchorLayout to center the card content in the grid cell
	card := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	card.AddChild(cardContent)

	return card
}

// loadGameArtworkSized loads artwork scaled to specific dimensions
func (s *LibraryScreen) loadGameArtworkSized(crc32 string, maxWidth, maxHeight int) *ebiten.Image {
	artPath, err := storage.GetGameArtworkPath(crc32)
	if err != nil {
		return getPlaceholderImageSized(maxWidth, maxHeight)
	}

	data, err := os.ReadFile(artPath)
	if err != nil {
		return getPlaceholderImageSized(maxWidth, maxHeight)
	}

	img, _, err := goimage.Decode(bytes.NewReader(data))
	if err != nil {
		return getPlaceholderImageSized(maxWidth, maxHeight)
	}

	return scaleImage(img, maxWidth, maxHeight)
}

// getPlaceholderImageSized returns a placeholder image of specific size
func getPlaceholderImageSized(width, height int) *ebiten.Image {
	img := ebiten.NewImage(width, height)
	img.Fill(themeSurface)
	return img
}

// scaleImage scales an image to fit within maxWidth x maxHeight while preserving aspect ratio
func scaleImage(src goimage.Image, maxWidth, maxHeight int) *ebiten.Image {
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

// getViewButtonImage returns button image based on active state
func (s *LibraryScreen) getViewButtonImage(active bool) *widget.ButtonImage {
	if active {
		return &widget.ButtonImage{
			Idle:     image.NewNineSliceColor(themePrimary),
			Hover:    image.NewNineSliceColor(themePrimaryHover),
			Pressed:  image.NewNineSliceColor(themePrimary),
			Disabled: image.NewNineSliceColor(themeBorder),
		}
	}
	return newButtonImage()
}

// SaveScrollPosition saves the current scroll position before a rebuild
// This should be called before rebuildCurrentScreen
func (s *LibraryScreen) SaveScrollPosition() {
	if s.config.Library.ViewMode == "icon" {
		if s.scrollContainer != nil {
			s.iconScrollTop = s.scrollContainer.ScrollTop
		}
	} else {
		if s.listScrollContainer != nil {
			s.listScrollTop = s.listScrollContainer.ScrollTop
		}
	}
}

// OnEnter is called when entering the library screen
func (s *LibraryScreen) OnEnter() {
	// Refresh games list
	s.games = s.library.GetGamesSorted(s.config.Library.SortBy, s.config.Library.FavoritesFilter)
}

// GetPendingFocusButton returns the button that should receive focus after rebuild
// Returns nil if no pending focus or button not found
func (s *LibraryScreen) GetPendingFocusButton() *widget.Button {
	// Check toolbar focus first (higher priority for toolbar actions)
	if s.pendingToolbarFocus != "" {
		return s.toolbarButtons[s.pendingToolbarFocus]
	}
	// Then check game button focus
	if s.pendingFocusCRC == "" {
		return nil
	}
	return s.gameButtons[s.pendingFocusCRC]
}

// ClearPendingFocus clears all pending focus state
func (s *LibraryScreen) ClearPendingFocus() {
	s.pendingFocusCRC = ""
	s.pendingToolbarFocus = ""
}

// EnsureFocusedVisible scrolls the view to ensure the focused widget is visible
// This is called after gamepad navigation changes focus
func (s *LibraryScreen) EnsureFocusedVisible(focused widget.Focuser) {
	if focused == nil {
		return
	}

	// Check if this is a game button (not toolbar)
	// Only game buttons should trigger scrolling
	isGameButton := false
	if btn, ok := focused.(*widget.Button); ok {
		for _, gameBtn := range s.gameButtons {
			if gameBtn == btn {
				isGameButton = true
				break
			}
		}
	}
	if !isGameButton {
		return
	}

	// Get the appropriate scroll container based on view mode
	var scrollContainer *widget.ScrollContainer
	var vSlider *widget.Slider
	if s.config.Library.ViewMode == "icon" {
		scrollContainer = s.scrollContainer
		vSlider = s.vSlider
	} else {
		scrollContainer = s.listScrollContainer
		vSlider = s.listVSlider
	}

	if scrollContainer == nil {
		return
	}

	// Get the focused widget's rectangle
	focusWidget := focused.GetWidget()
	if focusWidget == nil {
		return
	}
	focusRect := focusWidget.Rect

	// Get the scroll container's view rect (visible area on screen)
	viewRect := scrollContainer.ViewRect()
	contentRect := scrollContainer.ContentRect()

	// If content fits in view, no scrolling needed
	if contentRect.Dy() <= viewRect.Dy() {
		return
	}

	// Current scroll offset in pixels
	maxScroll := contentRect.Dy() - viewRect.Dy()
	scrollOffset := int(scrollContainer.ScrollTop * float64(maxScroll))

	// Widget's position relative to view top
	widgetTopInView := focusRect.Min.Y - viewRect.Min.Y
	widgetBottomInView := focusRect.Max.Y - viewRect.Min.Y
	viewHeight := viewRect.Dy()

	// Check if widget top is above the visible area
	if widgetTopInView < 0 {
		// Scroll up: align widget top with view top
		newScrollOffset := scrollOffset + widgetTopInView
		if newScrollOffset < 0 {
			newScrollOffset = 0
		}
		scrollContainer.ScrollTop = float64(newScrollOffset) / float64(maxScroll)
		if vSlider != nil {
			vSlider.Current = int(scrollContainer.ScrollTop * 1000)
		}
	} else if widgetBottomInView > viewHeight {
		// Scroll down: align widget bottom with view bottom (minimal scroll)
		newScrollOffset := scrollOffset + (widgetBottomInView - viewHeight)
		if newScrollOffset > maxScroll {
			newScrollOffset = maxScroll
		}
		scrollContainer.ScrollTop = float64(newScrollOffset) / float64(maxScroll)
		if vSlider != nil {
			vSlider.Current = int(scrollContainer.ScrollTop * 1000)
		}
	}
}

// OnExit is called when leaving the library screen
func (s *LibraryScreen) OnExit() {
	// Nothing to clean up
}

// formatPlayTime formats seconds into human-readable format
func formatPlayTime(seconds int64) string {
	if seconds == 0 {
		return "Never"
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

// Theme colors (duplicated for package isolation)
var (
	themeBackground    = color.NRGBA{0x1a, 0x1a, 0x2e, 0xff}
	themeSurface       = color.NRGBA{0x25, 0x25, 0x3a, 0xff}
	themePrimary       = color.NRGBA{0x4a, 0x4a, 0x8a, 0xff}
	themePrimaryHover  = color.NRGBA{0x5a, 0x5a, 0x9a, 0xff}
	themeText          = color.NRGBA{0xff, 0xff, 0xff, 0xff}
	themeTextSecondary = color.NRGBA{0xaa, 0xaa, 0xaa, 0xff}
	themeAccent        = color.NRGBA{0xff, 0xd7, 0x00, 0xff} // Gold for favorites
	themeBorder        = color.NRGBA{0x3a, 0x3a, 0x5a, 0xff}
)

var fontFace text.Face

func getFontFace() text.Face {
	if fontFace == nil {
		fontFace = text.NewGoXFace(basicfont.Face7x13)
	}
	return fontFace
}

func newButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:     image.NewNineSliceColor(themeSurface),
		Hover:    image.NewNineSliceColor(themePrimaryHover),
		Pressed:  image.NewNineSliceColor(themePrimary),
		Disabled: image.NewNineSliceColor(themeBorder),
	}
}

func newSliderButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:     image.NewNineSliceColor(themePrimary),
		Hover:    image.NewNineSliceColor(themePrimaryHover),
		Pressed:  image.NewNineSliceColor(themePrimary),
		Disabled: image.NewNineSliceColor(themeBorder),
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
