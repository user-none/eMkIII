//go:build !libretro && !ios

package settings

import (
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/user-none/emkiii/ui/achievements"
	"github.com/user-none/emkiii/ui/storage"
	"github.com/user-none/emkiii/ui/style"
	"github.com/user-none/emkiii/ui/types"
)

// Max width for the settings content to keep toggles closer to labels
const settingsMaxWidth = 500

// RetroAchievementsSection manages RetroAchievements settings
type RetroAchievementsSection struct {
	callback     types.ScreenCallback
	config       *storage.Config
	achievements *achievements.Manager

	// Input handling
	textInputs    *style.TextInputGroup
	usernameInput *widget.TextInput
	passwordInput *widget.TextInput
	errorMessage  string
	loggingIn     bool
}

// NewRetroAchievementsSection creates a new RetroAchievements section
func NewRetroAchievementsSection(
	callback types.ScreenCallback,
	config *storage.Config,
	achievementMgr *achievements.Manager,
) *RetroAchievementsSection {
	return &RetroAchievementsSection{
		callback:     callback,
		config:       config,
		achievements: achievementMgr,
		textInputs:   style.NewTextInputGroup(),
	}
}

// SetConfig updates the config reference
func (r *RetroAchievementsSection) SetConfig(config *storage.Config) {
	r.config = config
}

// SetAchievements updates the achievement manager reference
func (r *RetroAchievementsSection) SetAchievements(mgr *achievements.Manager) {
	r.achievements = mgr
}

// Update handles keyboard shortcuts for text inputs (Ctrl+A, Ctrl+V, Ctrl+C)
func (r *RetroAchievementsSection) Update() {
	r.textInputs.Update()
}

// hasStoredCredentials returns true if the user has stored login credentials
func (r *RetroAchievementsSection) hasStoredCredentials() bool {
	return r.config.RetroAchievements.Username != "" && r.config.RetroAchievements.Token != ""
}

// isLoggedIn returns true if the user is logged in (live session or stored credentials)
func (r *RetroAchievementsSection) isLoggedIn() bool {
	return (r.achievements != nil && r.achievements.IsLoggedIn()) || r.hasStoredCredentials()
}

// Build creates the RetroAchievements section UI
func (r *RetroAchievementsSection) Build(focus types.FocusManager) *widget.Container {
	// Outer container that anchors content to top-left
	outer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// Inner container with fixed width
	section := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(style.SmallSpacing),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
			}),
			widget.WidgetOpts.MinSize(settingsMaxWidth, 0),
		),
	)

	// Account section
	section.AddChild(r.buildSectionHeader("Account"))
	if r.isLoggedIn() {
		section.AddChild(r.buildLoggedInSection(focus))
	} else {
		section.AddChild(r.buildLoginSection(focus))
	}

	// General section
	section.AddChild(r.buildSectionHeader("General"))
	section.AddChild(r.buildToggleRow(focus, "ra-enable", "Enable RetroAchievements", "",
		r.config.RetroAchievements.Enabled,
		func() {
			r.config.RetroAchievements.Enabled = !r.config.RetroAchievements.Enabled
			if r.achievements != nil {
				r.achievements.SetEnabled(r.config.RetroAchievements.Enabled)
			}
		}))

	// Options (only shown when enabled)
	if r.config.RetroAchievements.Enabled {
		// Notifications section
		section.AddChild(r.buildSectionHeader("Notifications"))
		section.AddChild(r.buildToggleRow(focus, "ra-sound", "Unlock Sound", "Play chime on achievement",
			r.config.RetroAchievements.UnlockSound,
			func() {
				r.config.RetroAchievements.UnlockSound = !r.config.RetroAchievements.UnlockSound
			}))
		section.AddChild(r.buildToggleRow(focus, "ra-screenshot", "Auto Screenshot", "Capture screen on unlock",
			r.config.RetroAchievements.AutoScreenshot,
			func() {
				r.config.RetroAchievements.AutoScreenshot = !r.config.RetroAchievements.AutoScreenshot
			}))
		section.AddChild(r.buildToggleRow(focus, "ra-suppress", "Suppress Hardcore Warning", "Hide 'Unknown Emulator' notice",
			r.config.RetroAchievements.SuppressHardcoreWarning,
			func() {
				r.config.RetroAchievements.SuppressHardcoreWarning = !r.config.RetroAchievements.SuppressHardcoreWarning
			}))

		// Advanced section
		section.AddChild(r.buildSectionHeader("Advanced"))
		section.AddChild(r.buildToggleRow(focus, "ra-encore", "Encore Mode", "Re-trigger unlocked achievements",
			r.config.RetroAchievements.EncoreMode,
			func() {
				r.config.RetroAchievements.EncoreMode = !r.config.RetroAchievements.EncoreMode
				if r.achievements != nil {
					r.achievements.SetEncoreMode(r.config.RetroAchievements.EncoreMode)
				}
			}))
	}

	r.setupNavigation(focus)

	outer.AddChild(section)
	return outer
}

// buildSectionHeader creates a section header label
func (r *RetroAchievementsSection) buildSectionHeader(title string) *widget.Container {
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

// buildToggleRow creates a toggle row with background, label, description, and right-aligned button
func (r *RetroAchievementsSection) buildToggleRow(focus types.FocusManager, key, label, description string, value bool, toggle func()) *widget.Container {
	// Outer container with background color
	row := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Surface)),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{true, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(style.DefaultSpacing, 0),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(style.SmallSpacing)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)

	// Info column (label + optional description)
	infoContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(style.TinySpacing),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				VerticalPosition: widget.GridLayoutPositionCenter,
			}),
		),
	)

	labelText := widget.NewText(
		widget.TextOpts.Text(label, style.FontFace(), style.Text),
	)
	infoContainer.AddChild(labelText)

	if description != "" {
		descText := widget.NewText(
			widget.TextOpts.Text(description, style.FontFace(), style.TextSecondary),
		)
		infoContainer.AddChild(descText)
	}

	row.AddChild(infoContainer)

	// Toggle button (right-aligned via grid)
	toggleBtn := widget.NewButton(
		widget.ButtonOpts.Image(style.ActiveButtonImage(value)),
		widget.ButtonOpts.Text(boolToOnOff(value), style.FontFace(), style.ButtonTextColor()),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingSmall)),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				VerticalPosition: widget.GridLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(50, 0),
		),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			toggle()
			storage.SaveConfig(r.config)
			focus.SetPendingFocus(key)
			r.callback.RequestRebuild()
		}),
	)
	focus.RegisterFocusButton(key, toggleBtn)
	row.AddChild(toggleBtn)

	return row
}

// setupNavigation registers navigation zones for the section
func (r *RetroAchievementsSection) setupNavigation(focus types.FocusManager) {
	keys := []string{}

	if r.isLoggedIn() {
		keys = append(keys, "ra-logout")
	} else {
		keys = append(keys, "ra-login")
	}

	keys = append(keys, "ra-enable")

	if r.config.RetroAchievements.Enabled {
		keys = append(keys, "ra-sound", "ra-screenshot", "ra-suppress", "ra-encore")
	}

	focus.RegisterNavZone("ra-settings", types.NavZoneVertical, keys, 0)
}

// buildLoggedInSection creates the logged-in status section
func (r *RetroAchievementsSection) buildLoggedInSection(focus types.FocusManager) *widget.Container {
	// Row with background
	row := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Surface)),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{true, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(style.DefaultSpacing, 0),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(style.SmallSpacing)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)

	// Get username from manager if available, otherwise from config
	username := r.config.RetroAchievements.Username
	if r.achievements != nil && r.achievements.IsLoggedIn() {
		username = r.achievements.GetUsername()
	}

	// Status text
	statusText := widget.NewText(
		widget.TextOpts.Text("Logged in as: "+username, style.FontFace(), style.Text),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				VerticalPosition: widget.GridLayoutPositionCenter,
			}),
		),
	)
	row.AddChild(statusText)

	// Logout button
	logoutBtn := widget.NewButton(
		widget.ButtonOpts.Image(style.ButtonImage()),
		widget.ButtonOpts.Text("Logout", style.FontFace(), style.ButtonTextColor()),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingSmall)),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				VerticalPosition: widget.GridLayoutPositionCenter,
			}),
		),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if r.achievements != nil {
				r.achievements.Logout()
			}
			// Clear stored credentials
			r.config.RetroAchievements.Username = ""
			r.config.RetroAchievements.Token = ""
			storage.SaveConfig(r.config)
			focus.SetPendingFocus("ra-enable")
			r.callback.RequestRebuild()
		}),
	)
	focus.RegisterFocusButton("ra-logout", logoutBtn)
	row.AddChild(logoutBtn)

	return row
}

// buildLoginSection creates the login form section
func (r *RetroAchievementsSection) buildLoginSection(focus types.FocusManager) *widget.Container {
	section := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(style.Surface)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(style.SmallSpacing),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(style.SmallSpacing)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)

	// Username row using grid for alignment
	usernameRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{false, true}, []bool{true}),
			widget.GridLayoutOpts.Spacing(style.DefaultSpacing, 0),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)

	usernameLabel := widget.NewText(
		widget.TextOpts.Text("Username", style.FontFace(), style.Text),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				VerticalPosition: widget.GridLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(80, 0),
		),
	)
	usernameRow.AddChild(usernameLabel)

	r.usernameInput = style.StyledTextInput("Enter username", false, 200)
	r.textInputs.Add(r.usernameInput)
	if r.config.RetroAchievements.Username != "" {
		r.usernameInput.SetText(r.config.RetroAchievements.Username)
	}
	usernameRow.AddChild(r.usernameInput)
	section.AddChild(usernameRow)

	// Password row using grid for alignment
	passwordRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{false, true}, []bool{true}),
			widget.GridLayoutOpts.Spacing(style.DefaultSpacing, 0),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)

	passwordLabel := widget.NewText(
		widget.TextOpts.Text("Password", style.FontFace(), style.Text),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				VerticalPosition: widget.GridLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(80, 0),
		),
	)
	passwordRow.AddChild(passwordLabel)

	r.passwordInput = style.StyledTextInput("Enter password", true, 200)
	r.textInputs.Add(r.passwordInput)
	passwordRow.AddChild(r.passwordInput)
	section.AddChild(passwordRow)

	// Error message (if any)
	if r.errorMessage != "" {
		errorText := widget.NewText(
			widget.TextOpts.Text(r.errorMessage, style.FontFace(), style.Accent),
		)
		section.AddChild(errorText)
	}

	// Login button
	loginText := "Login"
	if r.loggingIn {
		loginText = "Logging in..."
	}

	loginBtn := widget.NewButton(
		widget.ButtonOpts.Image(style.ButtonImage()),
		widget.ButtonOpts.Text(loginText, style.FontFace(), style.ButtonTextColor()),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingSmall)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if r.loggingIn || r.achievements == nil {
				return
			}

			username := r.usernameInput.GetText()
			password := r.passwordInput.GetText()

			if username == "" || password == "" {
				r.errorMessage = "Username and password required"
				r.callback.RequestRebuild()
				return
			}

			r.loggingIn = true
			r.errorMessage = ""
			r.callback.RequestRebuild()

			r.achievements.Login(username, password, func(success bool, token string, err error) {
				r.loggingIn = false
				if success {
					r.config.RetroAchievements.Username = username
					r.config.RetroAchievements.Token = token
					storage.SaveConfig(r.config)
					r.errorMessage = ""
				} else {
					if err != nil {
						r.errorMessage = err.Error()
					} else {
						r.errorMessage = "Login failed"
					}
				}
				focus.SetPendingFocus("ra-enable")
				r.callback.RequestRebuild()
			})
		}),
	)
	focus.RegisterFocusButton("ra-login", loginBtn)
	section.AddChild(loginBtn)

	return section
}
