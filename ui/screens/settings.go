//go:build !libretro

package screens

import (
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/user-none/emkiii/ui/screens/settings"
	"github.com/user-none/emkiii/ui/storage"
	"github.com/user-none/emkiii/ui/style"
)

// SettingsScreen displays application settings
type SettingsScreen struct {
	BaseScreen // Embedded for focus restoration

	callback        ScreenCallback
	selectedSection int

	// Encapsulated sections
	library    *settings.LibrarySection
	appearance *settings.AppearanceSection
}

// NewSettingsScreen creates a new settings screen
func NewSettingsScreen(callback ScreenCallback, library *storage.Library, config *storage.Config) *SettingsScreen {
	s := &SettingsScreen{
		callback:        callback,
		selectedSection: 0,
		library:         settings.NewLibrarySection(callback, library),
		appearance:      settings.NewAppearanceSection(callback, config),
	}
	s.InitBase()
	return s
}

// HasPendingScan delegates to library section
func (s *SettingsScreen) HasPendingScan() bool {
	return s.library.HasPendingScan()
}

// ClearPendingScan delegates to library section
func (s *SettingsScreen) ClearPendingScan() {
	s.library.ClearPendingScan()
}

// SetLibrary updates the library reference in the library section
func (s *SettingsScreen) SetLibrary(library *storage.Library) {
	s.library.SetLibrary(library)
}

// SetConfig updates the config reference in the appearance section
func (s *SettingsScreen) SetConfig(config *storage.Config) {
	s.appearance.SetConfig(config)
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

	// Section content - delegate to encapsulated sections
	if s.selectedSection == 0 {
		contentArea.AddChild(s.library.Build(s))
	} else if s.selectedSection == 1 {
		contentArea.AddChild(s.appearance.Build(s))
	}

	mainContent.AddChild(contentArea)
	rootContainer.AddChild(mainContent)

	return rootContainer
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
