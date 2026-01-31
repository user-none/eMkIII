//go:build !libretro

package screens

import (
	"fmt"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/sqweek/dialog"
	"github.com/user-none/emkiii/ui/storage"
	"github.com/user-none/emkiii/ui/style"
)

// SettingsScreen displays application settings
type SettingsScreen struct {
	callback        ScreenCallback
	library         *storage.Library
	config          *storage.Config
	selectedSection int
	selectedDirs    map[int]bool // Multi-select: indices of selected directories
	pendingScan     bool         // True when a directory was added and scan should start

	// Button references for focus restoration
	folderButtons     map[int]*widget.Button
	pendingFocusIndex int // Index to restore focus to after rebuild
}

// NewSettingsScreen creates a new settings screen
func NewSettingsScreen(callback ScreenCallback, library *storage.Library, config *storage.Config) *SettingsScreen {
	return &SettingsScreen{
		callback:          callback,
		library:           library,
		config:            config,
		selectedSection:   0,
		selectedDirs:      make(map[int]bool),
		folderButtons:     make(map[int]*widget.Button),
		pendingFocusIndex: -1,
	}
}

// GetPendingFocusButton returns the button that should receive focus after rebuild
func (s *SettingsScreen) GetPendingFocusButton() *widget.Button {
	if s.pendingFocusIndex >= 0 {
		if btn, ok := s.folderButtons[s.pendingFocusIndex]; ok {
			return btn
		}
	}
	return nil
}

// ClearPendingFocus clears the pending focus index
func (s *SettingsScreen) ClearPendingFocus() {
	s.pendingFocusIndex = -1
}

// HasPendingScan returns true if a scan should be triggered
func (s *SettingsScreen) HasPendingScan() bool {
	return s.pendingScan
}

// ClearPendingScan clears the pending scan flag
func (s *SettingsScreen) ClearPendingScan() {
	s.pendingScan = false
}

// Build creates the settings screen UI
func (s *SettingsScreen) Build() *widget.Container {
	// Use GridLayout for the root to properly constrain sizes
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Background)),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			// Row 0 (header) = fixed, Row 1 (main content) = stretch
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true}),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(16)),
			widget.GridLayoutOpts.Spacing(16, 16),
		)),
	)

	// Header with back button and title
	header := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(16),
		)),
	)

	backButton := style.TextButton("Back", 8, func(args *widget.ButtonClickedEventArgs) {
		s.callback.SwitchToLibrary()
	})
	header.AddChild(backButton)

	rootContainer.AddChild(header)

	// Main content area with sidebar and content - use GridLayout for proper sizing
	mainContent := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			// Col 0 (sidebar) = fixed, Col 1 (content) = stretch
			// Row stretches vertically
			widget.GridLayoutOpts.Stretch([]bool{false, true}, []bool{true}),
			widget.GridLayoutOpts.Spacing(16, 0),
		)),
	)

	// Sidebar
	sidebar := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Surface)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(8)),
			widget.RowLayoutOpts.Spacing(4),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(160, 0),
		),
	)

	// Library section button
	libraryBtn := widget.NewButton(
		widget.ButtonOpts.Image(style.ActiveButtonImage(s.selectedSection == 0)),
		widget.ButtonOpts.Text("Library", style.FontFace(), &widget.ButtonTextColor{
			Idle:     style.Text,
			Disabled: style.TextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(8)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.selectedSection = 0
		}),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)
	sidebar.AddChild(libraryBtn)

	// Future sections (disabled) - use containers instead of buttons so they're not focusable
	sidebar.AddChild(style.DisabledSidebarItem("Video*"))
	sidebar.AddChild(style.DisabledSidebarItem("Audio*"))
	sidebar.AddChild(style.DisabledSidebarItem("Input*"))

	// Future note
	futureNote := widget.NewText(
		widget.TextOpts.Text("* Coming soon", style.FontFace(), style.TextSecondary),
	)
	sidebar.AddChild(futureNote)

	mainContent.AddChild(sidebar)

	// Content area - use GridLayout to constrain the library section
	contentArea := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{true}),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(16)),
		)),
	)

	// Library section content
	if s.selectedSection == 0 {
		contentArea.AddChild(s.buildLibrarySection())
	}

	mainContent.AddChild(contentArea)
	rootContainer.AddChild(mainContent)

	return rootContainer
}


// buildLibrarySection creates the library settings section
func (s *SettingsScreen) buildLibrarySection() *widget.Container {
	// Use GridLayout so we can make the list stretch to fill available space
	section := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			// Row stretch: label=no, list=YES, buttons=no, count=no
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true, false, false}),
			widget.GridLayoutOpts.Spacing(0, 12),
		)),
	)

	// ROM Folders label
	dirLabel := widget.NewText(
		widget.TextOpts.Text("ROM Folders", style.FontFace(), style.Text),
	)
	section.AddChild(dirLabel)

	// Create the folder list
	section.AddChild(s.buildFolderList())

	// Button row: Add Folder | Scan Library | Remove (centered)
	buttonRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(12),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionCenter,
			}),
		),
	)

	// Add Folder button
	addDirBtn := style.TextButton("Add Folder...", 12, func(args *widget.ButtonClickedEventArgs) {
		s.onAddDirectoryClick()
	})
	buttonRow.AddChild(addDirBtn)

	// Scan Library button
	scanBtn := style.PrimaryTextButton("Scan Library", 12, func(args *widget.ButtonClickedEventArgs) {
		s.callback.SwitchToScanProgress(true)
	})
	buttonRow.AddChild(scanBtn)

	// Remove button - disabled when nothing selected, removes all selected folders
	removeButtonImage := style.ButtonImage()
	if len(s.selectedDirs) == 0 {
		removeButtonImage = style.DisabledButtonImage()
	}
	removeBtn := widget.NewButton(
		widget.ButtonOpts.Image(removeButtonImage),
		widget.ButtonOpts.Text("Remove", style.FontFace(), style.ButtonTextColor()),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if len(s.selectedDirs) > 0 {
				// Collect paths to remove (iterate in reverse to avoid index shifting issues)
				for idx := len(s.library.ScanDirectories) - 1; idx >= 0; idx-- {
					if s.selectedDirs[idx] {
						path := s.library.ScanDirectories[idx].Path
						s.library.RemoveScanDirectory(path)
					}
				}
				s.selectedDirs = make(map[int]bool) // Clear selection
				storage.SaveLibrary(s.library)
				s.callback.RequestRebuild()
			}
		}),
	)
	buttonRow.AddChild(removeBtn)

	section.AddChild(buttonRow)

	// Game count
	gameCount := len(s.library.Games)
	countText := "No games in library"
	if gameCount == 1 {
		countText = "1 game in library"
	} else if gameCount > 1 {
		countText = fmt.Sprintf("%d games in library", gameCount)
	}

	countLabel := widget.NewText(
		widget.TextOpts.Text(countText, style.FontFace(), style.TextSecondary),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionCenter,
			}),
		),
	)
	section.AddChild(countLabel)

	return section
}

// buildFolderList creates a selectable folder list with scrolling
func (s *SettingsScreen) buildFolderList() widget.PreferredSizeLocateableWidget {
	rowHeight := 28
	maxPathChars := 70

	// Clear button references for fresh build
	s.folderButtons = make(map[int]*widget.Button)

	// Create list content container
	listContent := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(0),
		)),
	)

	if len(s.library.ScanDirectories) == 0 {
		// Empty state - centered text
		emptyContainer := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(0, 100),
			),
		)
		emptyLabel := widget.NewText(
			widget.TextOpts.Text("No folders added", style.FontFace(), style.TextSecondary),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
					HorizontalPosition: widget.AnchorLayoutPositionCenter,
					VerticalPosition:   widget.AnchorLayoutPositionCenter,
				}),
			),
		)
		emptyContainer.AddChild(emptyLabel)
		listContent.AddChild(emptyContainer)
	} else {
		for i, dir := range s.library.ScanDirectories {
			idx := i
			dirPath := dir.Path
			displayPath, wasTruncated := style.TruncateStart(dirPath, maxPathChars)

			// Determine row background based on selection state
			var rowBg = style.Background
			if s.selectedDirs[idx] {
				rowBg = style.Primary // Selected items show primary color
			} else if idx%2 == 1 {
				rowBg = style.Surface // Alternating colors for unselected
			}

			// Create row content with path label (no background - button handles colors for focus states)
			rowContent := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewAnchorLayout(
					widget.AnchorLayoutOpts.Padding(widget.Insets{Left: 12, Right: 12}),
				)),
				widget.ContainerOpts.WidgetOpts(
					widget.WidgetOpts.MinSize(0, rowHeight),
				),
			)

			// Build path label widget options
			pathWidgetOpts := []widget.WidgetOpt{
				widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
					HorizontalPosition: widget.AnchorLayoutPositionStart,
					VerticalPosition:   widget.AnchorLayoutPositionCenter,
				}),
			}

			// Add tooltip if path was truncated
			if wasTruncated {
				pathWidgetOpts = append(pathWidgetOpts, widget.WidgetOpts.ToolTip(
					widget.NewToolTip(
						widget.ToolTipOpts.Content(style.TooltipContent(dirPath)),
					),
				))
			}

			pathLabel := widget.NewText(
				widget.TextOpts.Text(displayPath, style.FontFace(), style.Text),
				widget.TextOpts.WidgetOpts(pathWidgetOpts...),
			)
			rowContent.AddChild(pathLabel)

			// Wrap in a button for click handling (selection)
			rowButton := widget.NewButton(
				widget.ButtonOpts.Image(&widget.ButtonImage{
					Idle:    image.NewNineSliceColor(rowBg),
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
					// Toggle selection - click to select, click again to deselect
					if s.selectedDirs[idx] {
						delete(s.selectedDirs, idx)
					} else {
						s.selectedDirs[idx] = true
					}
					s.pendingFocusIndex = idx
					s.callback.RequestRebuild()
				}),
			)

			// Store button reference for focus restoration
			s.folderButtons[idx] = rowButton

			// Stack button and content
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
			rowWrapper.AddChild(rowContent)

			listContent.AddChild(rowWrapper)
		}
	}

	// Create scrollable container with border
	_, _, wrapper := style.ScrollableContainer(style.ScrollableOpts{
		Content:     listContent,
		BgColor:     style.Surface,
		BorderColor: style.Border,
		Spacing:     0,
		Padding:     2,
	})

	return wrapper
}

// onAddDirectoryClick handles adding a search directory
func (s *SettingsScreen) onAddDirectoryClick() {
	// Run dialog in goroutine to avoid blocking Ebiten's main thread
	go func() {
		path, err := dialog.Directory().
			Title("Select ROM Folder").
			Browse()
		if err != nil {
			return // User cancelled or error
		}
		s.library.AddScanDirectory(path, true) // recursive=true by default
		storage.SaveLibrary(s.library)
		// Trigger auto-scan after adding directory
		s.pendingScan = true
	}()
}

// OnEnter is called when entering the settings screen
func (s *SettingsScreen) OnEnter() {
	// Nothing to do
}

// OnExit is called when leaving the settings screen
func (s *SettingsScreen) OnExit() {
	// Nothing to clean up
}
