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
	BaseScreen // Embedded for focus restoration

	callback        ScreenCallback
	library         *storage.Library
	config          *storage.Config
	selectedSection int
	selectedDirs    map[int]bool // Multi-select: indices of selected directories
	pendingScan     bool         // True when a directory was added and scan should start
}

// NewSettingsScreen creates a new settings screen
func NewSettingsScreen(callback ScreenCallback, library *storage.Library, config *storage.Config) *SettingsScreen {
	s := &SettingsScreen{
		callback:        callback,
		library:         library,
		config:          config,
		selectedSection: 0,
		selectedDirs:    make(map[int]bool),
	}
	s.InitBase()
	return s
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
	// Clear button references for fresh build
	s.ClearFocusButtons()

	// Use GridLayout for the root to properly constrain sizes
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Background)),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			// Row 0 (header) = fixed, Row 1 (main content) = stretch
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true}),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(style.DefaultPadding)),
			widget.GridLayoutOpts.Spacing(style.DefaultSpacing, style.DefaultSpacing),
		)),
	)

	// Header with back button and title
	header := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(style.DefaultSpacing),
		)),
	)

	backButton := style.TextButton("Back", style.ButtonPaddingSmall, func(args *widget.ButtonClickedEventArgs) {
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
			widget.GridLayoutOpts.Spacing(style.DefaultSpacing, 0),
		)),
	)

	// Sidebar
	sidebar := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Surface)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(style.SmallSpacing)),
			widget.RowLayoutOpts.Spacing(style.TinySpacing),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(style.SettingsSidebarMinWidth, 0),
		),
	)

	// Library section button
	libraryBtn := widget.NewButton(
		widget.ButtonOpts.Image(style.ActiveButtonImage(s.selectedSection == 0)),
		widget.ButtonOpts.Text("Library", style.FontFace(), &widget.ButtonTextColor{
			Idle:     style.Text,
			Disabled: style.TextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingSmall)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.selectedSection = 0
			s.SetPendingFocus("section-library")
			s.callback.RequestRebuild()
		}),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)
	s.RegisterFocusButton("section-library", libraryBtn)
	sidebar.AddChild(libraryBtn)

	// Appearance section button
	appearanceBtn := widget.NewButton(
		widget.ButtonOpts.Image(style.ActiveButtonImage(s.selectedSection == 1)),
		widget.ButtonOpts.Text("Appearance", style.FontFace(), &widget.ButtonTextColor{
			Idle:     style.Text,
			Disabled: style.TextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingSmall)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.selectedSection = 1
			s.SetPendingFocus("section-appearance")
			s.callback.RequestRebuild()
		}),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)
	s.RegisterFocusButton("section-appearance", appearanceBtn)
	sidebar.AddChild(appearanceBtn)

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
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(style.DefaultPadding)),
		)),
	)

	// Section content
	if s.selectedSection == 0 {
		contentArea.AddChild(s.buildLibrarySection())
	} else if s.selectedSection == 1 {
		contentArea.AddChild(s.buildAppearanceSection())
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
			widget.GridLayoutOpts.Spacing(0, style.ButtonPaddingMedium),
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
			widget.RowLayoutOpts.Spacing(style.ButtonPaddingMedium),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionCenter,
			}),
		),
	)

	// Add Folder button
	addDirBtn := style.TextButton("Add Folder...", style.ButtonPaddingMedium, func(args *widget.ButtonClickedEventArgs) {
		s.onAddDirectoryClick()
	})
	buttonRow.AddChild(addDirBtn)

	// Scan Library button
	scanBtn := style.PrimaryTextButton("Scan Library", style.ButtonPaddingMedium, func(args *widget.ButtonClickedEventArgs) {
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
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingMedium)),
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
	maxPathChars := 70

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
				widget.WidgetOpts.MinSize(0, style.SettingsFolderListMinHeight),
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
					widget.AnchorLayoutOpts.Padding(&widget.Insets{Left: style.ButtonPaddingMedium, Right: style.ButtonPaddingMedium}),
				)),
				widget.ContainerOpts.WidgetOpts(
					widget.WidgetOpts.MinSize(0, style.SettingsRowHeight),
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
					widget.WidgetOpts.MinSize(0, style.SettingsRowHeight),
				),
				widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
					// Toggle selection - click to select, click again to deselect
					if s.selectedDirs[idx] {
						delete(s.selectedDirs, idx)
					} else {
						s.selectedDirs[idx] = true
					}
					s.SetPendingFocus(fmt.Sprintf("folder-%d", idx))
					s.callback.RequestRebuild()
				}),
			)

			// Store button reference for focus restoration
			s.RegisterFocusButton(fmt.Sprintf("folder-%d", idx), rowButton)

			// Stack button and content
			rowWrapper := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewStackedLayout()),
				widget.ContainerOpts.WidgetOpts(
					widget.WidgetOpts.LayoutData(widget.RowLayoutData{
						Stretch: true,
					}),
					widget.WidgetOpts.MinSize(0, style.SettingsRowHeight),
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

// buildAppearanceSection creates the appearance settings section
func (s *SettingsScreen) buildAppearanceSection() *widget.Container {
	// Use GridLayout so the scrollable list can stretch
	section := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			// Row stretch: label=no, theme list=YES
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true}),
			widget.GridLayoutOpts.Spacing(0, style.DefaultSpacing),
		)),
	)

	// Theme label
	themeLabel := widget.NewText(
		widget.TextOpts.Text("Theme", style.FontFace(), style.Text),
	)
	section.AddChild(themeLabel)

	// Theme cards in scrollable list
	themeListContent := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(style.DefaultSpacing),
		)),
	)

	for _, theme := range style.AvailableThemes {
		themeListContent.AddChild(s.buildThemeCard(theme))
	}

	// Wrap in scrollable container using existing style helper
	scrollContainer, vSlider, scrollWrapper := style.ScrollableContainer(style.ScrollableOpts{
		Content:     themeListContent,
		BgColor:     style.Background,
		BorderColor: style.Border,
		Spacing:     0,
		Padding:     style.SmallSpacing,
	})
	s.SetScrollWidgets(scrollContainer, vSlider)
	// Restore scroll position after rebuild
	s.RestoreScrollPosition()
	section.AddChild(scrollWrapper)

	return section
}

// buildThemeCard creates a theme selection card with button and color preview
func (s *SettingsScreen) buildThemeCard(theme style.Theme) *widget.Container {
	themeName := theme.Name
	isActive := s.config.Theme == themeName
	focusKey := fmt.Sprintf("theme-%s", themeName)

	// Use grid layout so preview can stretch
	card := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{false, true}, []bool{true}),
			widget.GridLayoutOpts.Spacing(style.DefaultSpacing, 0),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)

	// Theme button
	themeBtn := widget.NewButton(
		widget.ButtonOpts.Image(style.ActiveButtonImage(isActive)),
		widget.ButtonOpts.Text(themeName, style.FontFace(), style.ButtonTextColor()),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingMedium)),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(80, 0),
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				VerticalPosition: widget.GridLayoutPositionCenter,
			}),
		),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.config.Theme = themeName
			style.ApplyThemeByName(themeName)
			storage.SaveConfig(s.config)
			s.SetPendingFocus(fmt.Sprintf("theme-%s", themeName))
			s.callback.RequestRebuild()
		}),
	)
	s.RegisterFocusButton(focusKey, themeBtn)
	card.AddChild(themeBtn)

	// Theme preview mockup
	card.AddChild(s.buildThemePreview(theme))

	return card
}

// buildThemePreview creates a mini UI mockup showing the theme applied
func (s *SettingsScreen) buildThemePreview(theme style.Theme) *widget.Container {
	const (
		previewHeight = 100
		sidebarWidth  = 70
		btnPadding    = 4
		itemHeight    = 22
	)

	// Outer container with theme's background color
	preview := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(theme.Background)),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{false, true}, []bool{true}),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(6)),
			widget.GridLayoutOpts.Spacing(6, 0),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(0, previewHeight),
		),
	)

	// Mini sidebar with surface color
	sidebar := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(theme.Surface)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(4)),
			widget.RowLayoutOpts.Spacing(2),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(sidebarWidth, 0),
		),
	)

	// Selected sidebar item (primary color)
	selectedItem := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(theme.Primary)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(2)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(0, itemHeight),
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)
	selectedItemText := widget.NewText(
		widget.TextOpts.Text("Library", style.FontFace(), theme.Text),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				VerticalPosition: widget.AnchorLayoutPositionCenter,
			}),
		),
	)
	selectedItem.AddChild(selectedItemText)
	sidebar.AddChild(selectedItem)

	// Unselected sidebar items
	for _, label := range []string{"Settings", "Help"} {
		item := widget.NewText(
			widget.TextOpts.Text(label, style.FontFace(), theme.TextSecondary),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(0, itemHeight),
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
			),
		)
		sidebar.AddChild(item)
	}

	preview.AddChild(sidebar)

	// Content area - surface panel
	contentPanel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(theme.Surface)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(6)),
			widget.RowLayoutOpts.Spacing(6),
		)),
	)

	// Header row with title
	title := widget.NewText(
		widget.TextOpts.Text("Game Title", style.FontFace(), theme.Text),
	)
	contentPanel.AddChild(title)

	// Info text
	info := widget.NewText(
		widget.TextOpts.Text("Developer: Studio Name", style.FontFace(), theme.TextSecondary),
	)
	contentPanel.AddChild(info)

	// Button row
	buttonRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(6),
		)),
	)

	// Primary button (Play)
	primaryBtn := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(theme.Primary)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(btnPadding)),
		)),
	)
	primaryBtnText := widget.NewText(
		widget.TextOpts.Text("Play", style.FontFace(), theme.Text),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	)
	primaryBtn.AddChild(primaryBtnText)
	buttonRow.AddChild(primaryBtn)

	// Secondary button (Options)
	secondaryBtn := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(theme.Background)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(btnPadding)),
		)),
	)
	secondaryBtnText := widget.NewText(
		widget.TextOpts.Text("Options", style.FontFace(), theme.Text),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	)
	secondaryBtn.AddChild(secondaryBtnText)
	buttonRow.AddChild(secondaryBtn)

	// Accent indicator (favorite star like in the UI)
	accentText := widget.NewText(
		widget.TextOpts.Text("*", style.FontFace(), theme.Accent),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)
	buttonRow.AddChild(accentText)

	contentPanel.AddChild(buttonRow)
	preview.AddChild(contentPanel)

	return preview
}

// OnEnter is called when entering the settings screen
func (s *SettingsScreen) OnEnter() {
	// Nothing to do
}

// EnsureFocusedVisible scrolls the theme list to keep the focused widget visible
func (s *SettingsScreen) EnsureFocusedVisible(focused widget.Focuser) {
	// Use the base implementation - all theme buttons should trigger scrolling
	s.BaseScreen.EnsureFocusedVisible(focused, nil)
}
