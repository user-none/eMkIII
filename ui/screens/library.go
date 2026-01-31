//go:build !libretro

package screens

import (
	"bytes"
	goimage "image"
	"os"
	"strings"

	_ "image/png"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/user-none/emkiii/ui/storage"
	"github.com/user-none/emkiii/ui/style"
)

// LibraryScreen displays the game library
type LibraryScreen struct {
	BaseScreen // Embedded for focus restoration

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

	// Widget references for scroll preservation (dual view mode)
	iconScrollContainer *widget.ScrollContainer
	iconVSlider         *widget.Slider
	listScrollContainer *widget.ScrollContainer
	listVSlider         *widget.Slider
}

// NewLibraryScreen creates a new library screen
func NewLibraryScreen(callback ScreenCallback, library *storage.Library, config *storage.Config) *LibraryScreen {
	s := &LibraryScreen{
		callback:      callback,
		library:       library,
		config:        config,
		selectedIndex: 0,
	}
	s.InitBase()
	return s
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
	// Clear button references for fresh build
	s.ClearFocusButtons()

	// Get sorted games
	s.games = s.library.GetGamesSorted(s.config.Library.SortBy, s.config.Library.FavoritesFilter)

	// Check if library is truly empty vs filtered empty
	totalGames := s.library.GameCount()

	// Use standard screen container pattern
	rootContainer := style.ScreenContainer()
	innerContainer := style.ScreenContentContainer([]bool{false, true}) // toolbar=fixed, content=stretch

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
	button := style.TextButton("Open Settings", 12, func(args *widget.ButtonClickedEventArgs) {
		s.callback.SwitchToSettings()
	})
	return style.EmptyState("No games in library", "Add a ROM folder in Settings", button)
}

// buildFilteredEmptyState creates the display when filters hide all games
func (s *LibraryScreen) buildFilteredEmptyState() *widget.Container {
	return style.EmptyState("No favorites yet", "Turn off the favorites filter to see all games", nil)
}

// buildToolbar creates the library toolbar
func (s *LibraryScreen) buildToolbar() *widget.Container {
	// Use GridLayout with 3 columns: left (view toggles), center (sort/favorites), right (settings)
	toolbar := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{false, true, false}, nil),
			widget.GridLayoutOpts.Spacing(style.SmallSpacing, 0),
		)),
	)

	// LEFT SECTION: View mode toggles
	leftSection := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(style.SmallSpacing),
		)),
	)

	iconViewBtn := style.ToggleButton("Icon", s.config.Library.ViewMode == "icon", func(args *widget.ButtonClickedEventArgs) {
		s.config.Library.ViewMode = "icon"
		storage.SaveConfig(s.config)
		s.SetPendingFocus("toolbar-icon")
		s.callback.RequestRebuild()
	})
	s.RegisterFocusButton("toolbar-icon", iconViewBtn)
	leftSection.AddChild(iconViewBtn)

	listViewBtn := style.ToggleButton("List", s.config.Library.ViewMode == "list", func(args *widget.ButtonClickedEventArgs) {
		s.config.Library.ViewMode = "list"
		storage.SaveConfig(s.config)
		s.SetPendingFocus("toolbar-list")
		s.callback.RequestRebuild()
	})
	s.RegisterFocusButton("toolbar-list", listViewBtn)
	leftSection.AddChild(listViewBtn)

	toolbar.AddChild(leftSection)

	// CENTER SECTION: Sort and Favorites
	centerSection := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	centerContent := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(style.SmallSpacing),
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
		widget.TextOpts.Text("Sort:", style.FontFace(), style.Text),
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
		widget.ButtonOpts.Image(style.ButtonImage()),
		widget.ButtonOpts.Text(sortOptions[currentSortIdx], style.FontFace(), style.ButtonTextColor()),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingSmall)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			currentSortIdx = (currentSortIdx + 1) % len(sortOptions)
			s.config.Library.SortBy = sortValues[currentSortIdx]
			storage.SaveConfig(s.config)
			s.SetPendingFocus("toolbar-sort")
			s.callback.RequestRebuild()
		}),
	)
	s.RegisterFocusButton("toolbar-sort", sortButton)
	centerContent.AddChild(sortButton)

	// Favorites button
	favText := "Favorites"
	if s.config.Library.FavoritesFilter {
		favText = "[*] Favorites"
	}
	favButton := style.ToggleButton(favText, s.config.Library.FavoritesFilter, func(args *widget.ButtonClickedEventArgs) {
		s.config.Library.FavoritesFilter = !s.config.Library.FavoritesFilter
		storage.SaveConfig(s.config)
		s.SetPendingFocus("toolbar-favorites")
		s.callback.RequestRebuild()
	})
	s.RegisterFocusButton("toolbar-favorites", favButton)
	centerContent.AddChild(favButton)

	centerSection.AddChild(centerContent)
	toolbar.AddChild(centerSection)

	// RIGHT SECTION: Settings button
	rightSection := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	settingsButton := widget.NewButton(
		widget.ButtonOpts.Image(style.ButtonImage()),
		widget.ButtonOpts.Text("Settings", style.FontFace(), style.ButtonTextColor()),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingSmall)),
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
	selectedIndex := -1

	// Build header row
	header := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(6),
			widget.GridLayoutOpts.Stretch([]bool{false, true, false, false, false, false}, nil),
			widget.GridLayoutOpts.Spacing(style.SmallSpacing, 0),
			widget.GridLayoutOpts.Padding(&widget.Insets{Left: style.SmallSpacing, Right: style.SmallSpacing}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(0, style.ListHeaderHeight),
		),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Surface)),
	)
	header.AddChild(style.TableHeaderCell("", style.ListColFavorite, style.ListHeaderHeight)) // Favorite column (no header text)
	header.AddChild(style.TableHeaderCell("Title", 0, style.ListHeaderHeight))                // Title stretches
	header.AddChild(style.TableHeaderCell("Genre", style.ListColGenre, style.ListHeaderHeight))
	header.AddChild(style.TableHeaderCell("Region", style.ListColRegion, style.ListHeaderHeight))
	header.AddChild(style.TableHeaderCell("Play Time", style.ListColPlayTime, style.ListHeaderHeight))
	header.AddChild(style.TableHeaderCell("Last Played", style.ListColLastPlayed, style.ListHeaderHeight))

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
		playTime := style.FormatPlayTime(g.PlayTimeSeconds)
		lastPlayed := style.FormatLastPlayed(g.LastPlayed)

		// Determine row background color for alternating rows
		rowIdleBg := style.AlternatingRowColor(idx)

		// Create row container with grid layout (transparent background - button handles colors)
		row := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(6),
				widget.GridLayoutOpts.Stretch([]bool{false, true, false, false, false, false}, nil),
				widget.GridLayoutOpts.Spacing(style.SmallSpacing, 0),
				widget.GridLayoutOpts.Padding(&widget.Insets{Left: style.SmallSpacing, Right: style.SmallSpacing}),
			)),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(0, style.ListRowHeight),
			),
		)

		// Add cells
		row.AddChild(style.TableCell(fav, style.ListColFavorite, style.ListRowHeight, style.Accent))
		row.AddChild(style.TableCell(g.DisplayName, 0, style.ListRowHeight, style.Text))
		row.AddChild(style.TableCell(genre, style.ListColGenre, style.ListRowHeight, style.TextSecondary))
		row.AddChild(style.TableCell(region, style.ListColRegion, style.ListRowHeight, style.TextSecondary))
		row.AddChild(style.TableCell(playTime, style.ListColPlayTime, style.ListRowHeight, style.TextSecondary))
		row.AddChild(style.TableCell(lastPlayed, style.ListColLastPlayed, style.ListRowHeight, style.TextSecondary))

		// Create button with alternating row color as idle, focus/hover colors for interaction
		gameCRC := g.CRC32 // Capture for closure
		rowButton := widget.NewButton(
			widget.ButtonOpts.Image(&widget.ButtonImage{
				Idle:    image.NewNineSliceColor(rowIdleBg),
				Hover:   image.NewNineSliceColor(style.PrimaryHover),
				Pressed: image.NewNineSliceColor(style.Primary),
			}),
			widget.ButtonOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Stretch: true,
				}),
				widget.WidgetOpts.MinSize(0, style.ListRowHeight),
			),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				if s.listScrollContainer != nil {
					s.listScrollTop = s.listScrollContainer.ScrollTop
				}
				s.listSelectedCRC = gameCRC
				s.SetPendingFocus("game-" + gameCRC)
				s.callback.SwitchToDetail(gameCRC)
			}),
		)

		// Store button reference for focus restoration
		s.RegisterFocusButton("game-"+gameCRC, rowButton)

		// Stack: button at bottom (shows background), row content on top (transparent)
		rowWrapper := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewStackedLayout()),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Stretch: true,
				}),
				widget.WidgetOpts.MinSize(0, style.ListRowHeight),
			),
		)
		rowWrapper.AddChild(rowButton)
		rowWrapper.AddChild(row)

		listContent.AddChild(rowWrapper)
	}

	// Create scrollable container (we use custom layout for header alignment, so ignore wrapper)
	scrollContainer, vSlider, scrollRow := style.ScrollableContainer(style.ScrollableOpts{
		Content: listContent,
		BgColor: style.Background,
		Spacing: style.TinySpacing,
	})

	// Store references for scroll preservation
	s.listScrollContainer = scrollContainer
	s.listVSlider = vSlider

	// Restore or calculate scroll position
	if s.listScrollTop > 0 {
		scrollContainer.ScrollTop = s.listScrollTop
		vSlider.Current = int(s.listScrollTop * 1000)
	} else if selectedIndex >= 0 && len(s.games) > 0 {
		totalHeight := len(s.games) * style.ListRowHeight
		selectedY := selectedIndex * style.ListRowHeight
		viewportHeight := style.EstimatedViewportHeight
		targetScrollY := selectedY - (viewportHeight / 2) + (style.ListRowHeight / 2)
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

	// Header row with spacer for slider alignment
	headerRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Spacing(style.TinySpacing, 0),
			widget.GridLayoutOpts.Stretch([]bool{true, false}, nil),
		)),
	)
	headerRow.AddChild(header)
	// Empty spacer matching slider width
	headerSpacer := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(style.ScrollbarWidth, 0),
		),
	)
	headerRow.AddChild(headerSpacer)

	// Main container: header row + scroll area
	mainContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true}),
			widget.GridLayoutOpts.Spacing(0, style.TinySpacing),
		)),
	)
	mainContainer.AddChild(headerRow)
	mainContainer.AddChild(scrollRow)

	return mainContainer
}

// buildIconView creates the icon/grid view of games with artwork
func (s *LibraryScreen) buildIconView() widget.PreferredSizeLocateableWidget {
	// Calculate responsive grid dimensions
	windowWidth := s.callback.GetWindowWidth()
	if windowWidth < 400 {
		windowWidth = style.IconDefaultWindowWidth
	}

	// Available width for cards (subtract padding and scrollbar)
	availableWidth := windowWidth - (style.DefaultPadding * 2) - style.ScrollbarWidth

	// Calculate number of columns that fit with minimum card width
	// Formula: columns = floor((availableWidth + spacing) / (minCardWidth + spacing))
	columns := (availableWidth + style.SmallSpacing) / (style.IconMinCardWidth + style.SmallSpacing)
	if columns < 2 {
		columns = 2
	}

	// Calculate exact card width to fill the available space
	// Formula: cardWidth = (availableWidth - (columns - 1) * spacing) / columns
	cardWidth := (availableWidth - (columns-1)*style.SmallSpacing) / columns

	// Card height maintains ~4:3 aspect ratio for artwork + text
	artHeight := cardWidth * 4 / 3
	cardHeight := artHeight + style.IconCardTextHeight

	// Create stretch array - all columns stretch equally to fill width
	columnStretches := make([]bool, columns)
	for i := range columnStretches {
		columnStretches[i] = true
	}

	// Grid container for the cards - columns stretch to fill available width
	gridContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(columns),
			widget.GridLayoutOpts.Spacing(style.SmallSpacing, style.SmallSpacing),
			widget.GridLayoutOpts.Stretch(columnStretches, nil),
		)),
	)

	// Add game cards with calculated dimensions
	for _, game := range s.games {
		card := s.buildGameCardSized(game, cardWidth, cardHeight, artHeight)
		gridContainer.AddChild(card)
	}

	// Create scrollable container
	scrollContainer, vSlider, wrapper := style.ScrollableContainer(style.ScrollableOpts{
		Content: gridContainer,
		BgColor: style.Background,
		Spacing: 4,
	})

	// Store references for scroll preservation
	s.iconScrollContainer = scrollContainer
	s.iconVSlider = vSlider

	// Restore icon view scroll position if we have one
	if s.iconScrollTop > 0 {
		scrollContainer.ScrollTop = s.iconScrollTop
		vSlider.Current = int(s.iconScrollTop * 1000)
	}

	return wrapper
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
			Idle:    image.NewNineSliceColor(style.Surface),
			Hover:   image.NewNineSliceColor(style.PrimaryHover),
			Pressed: image.NewNineSliceColor(style.Primary),
		}),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(cardWidth, artHeight),
		),
		widget.ButtonOpts.Graphic(&widget.GraphicImage{Idle: artwork}),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			// Save scroll position and selected game before navigating
			s.iconSelectedCRC = gameCRC
			s.SetPendingFocus("game-" + gameCRC)
			if s.iconScrollContainer != nil {
				s.iconScrollTop = s.iconScrollContainer.ScrollTop
			}
			s.callback.SwitchToDetail(gameCRC)
		}),
	)

	// Store button reference for focus restoration
	s.RegisterFocusButton("game-"+gameCRC, artButton)

	cardContent.AddChild(artButton)

	// Game title (truncated based on card width)
	maxChars := cardWidth / 7 // Approximate chars that fit
	if maxChars < 10 {
		maxChars = 10
	}
	displayName, _ := style.TruncateEnd(game.DisplayName, maxChars)
	titleLabel := widget.NewText(
		widget.TextOpts.Text(displayName, style.FontFace(), style.Text),
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
func (s *LibraryScreen) loadGameArtworkSized(gameCRC string, maxWidth, maxHeight int) *ebiten.Image {
	artPath, err := storage.GetGameArtworkPath(gameCRC)
	if err != nil {
		return s.getPlaceholderImageSized(maxWidth, maxHeight)
	}

	data, err := os.ReadFile(artPath)
	if err != nil {
		return s.getPlaceholderImageSized(maxWidth, maxHeight)
	}

	img, _, err := goimage.Decode(bytes.NewReader(data))
	if err != nil {
		return s.getPlaceholderImageSized(maxWidth, maxHeight)
	}

	return style.ScaleImage(img, maxWidth, maxHeight)
}

// getPlaceholderImageSized returns the placeholder image scaled to the specified size
func (s *LibraryScreen) getPlaceholderImageSized(width, height int) *ebiten.Image {
	data := s.callback.GetPlaceholderImageData()
	if data == nil {
		// Fallback to solid color if no placeholder data
		img := ebiten.NewImage(width, height)
		img.Fill(style.Surface)
		return img
	}

	img, _, err := goimage.Decode(bytes.NewReader(data))
	if err != nil {
		// Fallback to solid color on decode error
		fallback := ebiten.NewImage(width, height)
		fallback.Fill(style.Surface)
		return fallback
	}

	return style.ScaleImage(img, width, height)
}

// SaveScrollPosition saves the current scroll position before a rebuild
// This should be called before rebuildCurrentScreen
func (s *LibraryScreen) SaveScrollPosition() {
	if s.config.Library.ViewMode == "icon" {
		if s.iconScrollContainer != nil {
			s.iconScrollTop = s.iconScrollContainer.ScrollTop
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

// isGameButton returns true if the button is a game button (not a toolbar button)
func (s *LibraryScreen) isGameButton(btn *widget.Button) bool {
	// Game buttons have keys starting with "game-"
	for key, b := range s.focusButtons {
		if b == btn && len(key) > 5 && key[:5] == "game-" {
			return true
		}
	}
	return false
}

// EnsureFocusedVisible scrolls the view to ensure the focused widget is visible
// This is called after gamepad navigation changes focus
func (s *LibraryScreen) EnsureFocusedVisible(focused widget.Focuser) {
	if focused == nil {
		return
	}

	// Check if this is a game button (not toolbar)
	// Only game buttons should trigger scrolling
	btn, ok := focused.(*widget.Button)
	if !ok || !s.isGameButton(btn) {
		return
	}

	// Get the appropriate scroll container based on view mode
	var scrollContainer *widget.ScrollContainer
	var vSlider *widget.Slider
	if s.config.Library.ViewMode == "icon" {
		scrollContainer = s.iconScrollContainer
		vSlider = s.iconVSlider
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
