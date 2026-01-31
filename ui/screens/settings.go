//go:build !libretro

package screens

import (
	"fmt"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/sqweek/dialog"
	"github.com/user-none/emkiii/ui/storage"
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
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeBackground)),
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
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeSurface)),
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
		widget.ButtonOpts.Image(s.getSidebarButtonImage(s.selectedSection == 0)),
		widget.ButtonOpts.Text("Library", getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
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
	videoItem := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeBorder)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(8)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)
	videoItem.AddChild(widget.NewText(
		widget.TextOpts.Text("Video*", getFontFace(), themeTextSecondary),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	))
	sidebar.AddChild(videoItem)

	audioItem := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeBorder)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(8)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)
	audioItem.AddChild(widget.NewText(
		widget.TextOpts.Text("Audio*", getFontFace(), themeTextSecondary),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	))
	sidebar.AddChild(audioItem)

	inputItem := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeBorder)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(8)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)
	inputItem.AddChild(widget.NewText(
		widget.TextOpts.Text("Input*", getFontFace(), themeTextSecondary),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	))
	sidebar.AddChild(inputItem)

	// Future note
	futureNote := widget.NewText(
		widget.TextOpts.Text("* Coming soon", getFontFace(), themeTextSecondary),
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

// truncatePath truncates a path to fit within maxChars, adding "..." if needed
func truncatePath(path string, maxChars int) (truncated string, wasTruncated bool) {
	if len(path) <= maxChars {
		return path, false
	}
	// Keep the end of the path (most relevant part)
	return "..." + path[len(path)-maxChars+3:], true
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
		widget.TextOpts.Text("ROM Folders", getFontFace(), themeText),
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
	addDirBtn := widget.NewButton(
		widget.ButtonOpts.Image(newButtonImage()),
		widget.ButtonOpts.Text("Add Folder...", getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.onAddDirectoryClick()
		}),
	)
	buttonRow.AddChild(addDirBtn)

	// Scan Library button
	scanBtn := widget.NewButton(
		widget.ButtonOpts.Image(newPrimaryButtonImage()),
		widget.ButtonOpts.Text("Scan Library", getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.callback.SwitchToScanProgress(true)
		}),
	)
	buttonRow.AddChild(scanBtn)

	// Remove button - disabled when nothing selected, removes all selected folders
	removeButtonImage := newButtonImage()
	if len(s.selectedDirs) == 0 {
		removeButtonImage = newDisabledButtonImage()
	}
	removeBtn := widget.NewButton(
		widget.ButtonOpts.Image(removeButtonImage),
		widget.ButtonOpts.Text("Remove", getFontFace(), &widget.ButtonTextColor{
			Idle:     themeText,
			Disabled: themeTextSecondary,
		}),
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
		widget.TextOpts.Text(countText, getFontFace(), themeTextSecondary),
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
			widget.TextOpts.Text("No folders added", getFontFace(), themeTextSecondary),
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
			displayPath, wasTruncated := truncatePath(dirPath, maxPathChars)

			// Determine row background based on selection state
			var rowBg = themeBackground
			if s.selectedDirs[idx] {
				rowBg = themePrimary // Selected items show primary color
			} else if idx%2 == 1 {
				rowBg = themeSurface // Alternating colors for unselected
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
				tooltipContainer := widget.NewContainer(
					widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeBorder)),
					widget.ContainerOpts.Layout(widget.NewRowLayout(
						widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(8)),
					)),
				)
				tooltipLabel := widget.NewText(
					widget.TextOpts.Text(dirPath, getFontFace(), themeText),
				)
				tooltipContainer.AddChild(tooltipLabel)

				pathWidgetOpts = append(pathWidgetOpts, widget.WidgetOpts.ToolTip(
					widget.NewToolTip(
						widget.ToolTipOpts.Content(tooltipContainer),
					),
				))
			}

			pathLabel := widget.NewText(
				widget.TextOpts.Text(displayPath, getFontFace(), themeText),
				widget.TextOpts.WidgetOpts(pathWidgetOpts...),
			)
			rowContent.AddChild(pathLabel)

			// Wrap in a button for click handling (selection)
			rowButton := widget.NewButton(
				widget.ButtonOpts.Image(&widget.ButtonImage{
					Idle:    image.NewNineSliceColor(rowBg),
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

	// Create scroll container
	scrollContainer := widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(listContent),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
			Idle: image.NewNineSliceColor(themeSurface),
			Mask: image.NewNineSliceColor(themeSurface),
		}),
	)

	// Helper to check if scrolling is needed
	needsScroll := func() bool {
		contentHeight := listContent.GetWidget().Rect.Dy()
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
			contentHeight := listContent.GetWidget().Rect.Dy()
			viewHeight := scrollContainer.ViewRect().Dy()
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

	// Mouse wheel scroll support - only scroll if content exceeds view
	scrollContainer.GetWidget().ScrolledEvent.AddHandler(func(args interface{}) {
		if !needsScroll() {
			scrollContainer.ScrollTop = 0
			return
		}
		a := args.(*widget.WidgetScrolledEventArgs)
		p := scrollContainer.ScrollTop + (a.Y * 0.05)
		if p < 0 {
			p = 0
		} else if p > 1 {
			p = 1
		}
		scrollContainer.ScrollTop = p
		vSlider.Current = int(p * 1000)
	})

	// Outer wrapper with border - fills the grid cell
	listWrapper := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(themeBorder)),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{true, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(0, 0),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(2)), // Border width
		)),
	)

	listWrapper.AddChild(scrollContainer)
	listWrapper.AddChild(vSlider)

	return listWrapper
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

// getSidebarButtonImage returns the appropriate button image for sidebar items
func (s *SettingsScreen) getSidebarButtonImage(active bool) *widget.ButtonImage {
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

// OnEnter is called when entering the settings screen
func (s *SettingsScreen) OnEnter() {
	// Nothing to do
}

// OnExit is called when leaving the settings screen
func (s *SettingsScreen) OnExit() {
	// Nothing to clean up
}
