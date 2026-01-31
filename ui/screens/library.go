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
	"github.com/user-none/emkiii/ui/storage"
	"github.com/user-none/emkiii/ui/style"
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
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Background)),
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
		widget.ButtonOpts.Image(style.ActiveButtonImage(s.config.Library.ViewMode == "icon")),
		widget.ButtonOpts.Text("Icon", style.FontFace(), &widget.ButtonTextColor{
			Idle:     style.Text,
			Disabled: style.TextSecondary,
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
		widget.ButtonOpts.Image(style.ActiveButtonImage(s.config.Library.ViewMode == "list")),
		widget.ButtonOpts.Text("List", style.FontFace(), &widget.ButtonTextColor{
			Idle:     style.Text,
			Disabled: style.TextSecondary,
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
		widget.ButtonOpts.Text(sortOptions[currentSortIdx], style.FontFace(), &widget.ButtonTextColor{
			Idle:     style.Text,
			Disabled: style.TextSecondary,
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
		widget.ButtonOpts.Image(style.ActiveButtonImage(s.config.Library.FavoritesFilter)),
		widget.ButtonOpts.Text(favText, style.FontFace(), &widget.ButtonTextColor{
			Idle:     style.Text,
			Disabled: style.TextSecondary,
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
		widget.ButtonOpts.Image(style.ButtonImage()),
		widget.ButtonOpts.Text("Settings", style.FontFace(), &widget.ButtonTextColor{
			Idle:     style.Text,
			Disabled: style.TextSecondary,
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
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Surface)),
	)
	header.AddChild(style.TableHeaderCell("", colFav, headerHeight)) // Favorite column (no header text)
	header.AddChild(style.TableHeaderCell("Title", 0, headerHeight)) // Title stretches
	header.AddChild(style.TableHeaderCell("Genre", colGenre, headerHeight))
	header.AddChild(style.TableHeaderCell("Region", colRegion, headerHeight))
	header.AddChild(style.TableHeaderCell("Play Time", colPlayTime, headerHeight))
	header.AddChild(style.TableHeaderCell("Last Played", colLastPlayed, headerHeight))

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
			rowIdleBg = style.Background
		} else {
			rowIdleBg = style.Surface
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
		row.AddChild(style.TableCell(fav, colFav, rowHeight, style.Accent))
		row.AddChild(style.TableCell(g.DisplayName, 0, rowHeight, style.Text))
		row.AddChild(style.TableCell(genre, colGenre, rowHeight, style.TextSecondary))
		row.AddChild(style.TableCell(region, colRegion, rowHeight, style.TextSecondary))
		row.AddChild(style.TableCell(playTime, colPlayTime, rowHeight, style.TextSecondary))
		row.AddChild(style.TableCell(lastPlayed, colLastPlayed, rowHeight, style.TextSecondary))

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

	// Create scrollable container (we use custom layout for header alignment, so ignore wrapper)
	scrollContainer, vSlider, scrollRow := style.ScrollableContainer(style.ScrollableOpts{
		Content: listContent,
		BgColor: style.Background,
		Spacing: 4,
	})

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

	// Create scrollable container
	scrollContainer, vSlider, wrapper := style.ScrollableContainer(style.ScrollableOpts{
		Content: gridContainer,
		BgColor: style.Background,
		Spacing: 4,
	})

	// Store references for scroll preservation
	s.scrollContainer = scrollContainer
	s.vSlider = vSlider

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

	return style.ScaleImage(img, maxWidth, maxHeight)
}

// getPlaceholderImageSized returns a placeholder image of specific size
func getPlaceholderImageSized(width, height int) *ebiten.Image {
	img := ebiten.NewImage(width, height)
	img.Fill(style.Surface)
	return img
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

