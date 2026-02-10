//go:build !libretro

package ui

import (
	"fmt"
	"log"
	"os"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/user-none/emkiii/ui/achievements"
	"github.com/user-none/emkiii/ui/rdb"
	"github.com/user-none/emkiii/ui/screens"
	"github.com/user-none/emkiii/ui/shader"
	"github.com/user-none/emkiii/ui/storage"
	"github.com/user-none/emkiii/ui/style"
	"github.com/user-none/emkiii/ui/types"
)

// App is the main application struct that implements ebiten.Game
type App struct {
	ui *ebitenui.UI

	// State management
	state         AppState
	previousState AppState

	// Data
	config  *storage.Config
	library *storage.Library

	// Screens
	libraryScreen  *screens.LibraryScreen
	detailScreen   *screens.DetailScreen
	settingsScreen *screens.SettingsScreen
	scanScreen     *screens.ScanProgressScreen
	errorScreen    *screens.ErrorScreen

	// Metadata for RDB lookups
	metadata *MetadataManager

	// Gameplay (emulation, input, save states, pause menu)
	gameplay *GameplayManager

	// UI managers
	notification      *Notification
	saveStateManager  *SaveStateManager
	screenshotManager *ScreenshotManager
	searchOverlay     *SearchOverlay

	// Scan manager
	scanManager *ScanManager

	// Error state
	errorFile        string
	errorPath        string
	configLoadFailed bool // True if config.json failed to load (don't overwrite on exit)

	// Window tracking for persistence and responsive layouts
	windowX, windowY int
	windowWidth      int
	windowHeight     int
	lastBuildWidth   int // Track width used for last UI build

	// Screenshot pending flag (set in Update, processed in Draw)
	screenshotPending bool

	// Input manager for UI navigation
	inputManager *InputManager

	// Shader manager for visual effects
	shaderManager *shader.Manager
	shaderBuffer  *ebiten.Image // Intermediate buffer for shader rendering

	// Achievement manager for RetroAchievements integration
	achievementManager *achievements.Manager

	// Rebuild pending flag (set from goroutines, processed on main thread)
	rebuildPending bool
}

// NewApp creates and initializes the application
func NewApp() (*App, error) {
	app := &App{
		state: StateLibrary,
	}

	// Ensure directory structure exists
	if err := storage.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	// Create config/library files if missing
	if err := storage.CreateConfigIfMissing(); err != nil {
		log.Printf("Warning: failed to create config: %v", err)
	}
	if err := storage.CreateLibraryIfMissing(); err != nil {
		log.Printf("Warning: failed to create library: %v", err)
	}

	// Start RDB download in background if missing (non-blocking)
	if !RDBExists() {
		go func() {
			metadata := NewMetadataManager()
			if err := metadata.DownloadRDB(); err != nil {
				log.Printf("Background RDB download failed: %v", err)
			}
		}()
	}

	// Initialize UI managers
	app.notification = NewNotification()
	app.saveStateManager = NewSaveStateManager(app.notification)
	app.screenshotManager = NewScreenshotManager(app.notification)
	app.inputManager = NewInputManager()
	app.shaderManager = shader.NewManager()
	app.searchOverlay = NewSearchOverlay(func(text string) {
		if app.state == StateLibrary {
			app.libraryScreen.SetSearchText(text)
			app.rebuildCurrentScreen()
		}
	})

	// Initialize metadata manager and load RDB (for achievement MD5 lookups)
	app.metadata = NewMetadataManager()
	if err := app.metadata.LoadRDB(); err != nil {
		log.Printf("Failed to load RDB: %v", err)
	}

	// Load config
	config, err := storage.LoadConfig()
	if err != nil {
		// JSON parse error - show error screen
		configPath, _ := storage.GetConfigPath()
		app.state = StateError
		app.errorFile = "config.json"
		app.errorPath = configPath
		app.configLoadFailed = true // Don't overwrite the file on exit
		app.config = storage.DefaultConfig()
		app.achievementManager = achievements.NewManager(app.notification, app.config, Name, Version)
		app.library = storage.DefaultLibrary()
		app.preloadConfiguredShaders()
		app.initScreens()
		app.rebuildCurrentScreen()
		return app, nil
	}
	app.config = config

	// Create achievement manager with config
	app.achievementManager = achievements.NewManager(app.notification, app.config, Name, Version)

	// Validate and apply theme
	if !style.IsValidThemeName(app.config.Theme) {
		app.config.Theme = "Default"
	}
	style.ApplyThemeByName(app.config.Theme)

	// Apply font size
	style.ApplyFontSize(storage.ValidFontSize(app.config.FontSize))

	// Load library
	library, err := storage.LoadLibrary()
	if err != nil {
		// JSON parse error - show error screen
		libraryPath, _ := storage.GetLibraryPath()
		app.state = StateError
		app.errorFile = "library.json"
		app.errorPath = libraryPath
		app.preloadConfiguredShaders()
		app.initScreens()
		app.rebuildCurrentScreen()
		return app, nil
	}
	app.library = library

	// Set library on save state manager for slot persistence
	app.saveStateManager.SetLibrary(library)

	// Initialize gameplay manager with callbacks
	app.gameplay = NewGameplayManager(
		app.saveStateManager,
		app.screenshotManager,
		app.notification,
		app.library,
		app.config,
		app.achievementManager,
		app.metadata.GetRDB(),
		func() { app.SwitchToLibrary() }, // onExitToLibrary
		func() { app.Exit() },            // onExitApp
	)

	// Auto-login with stored token if available
	if app.config.RetroAchievements.Enabled &&
		app.config.RetroAchievements.Username != "" &&
		app.config.RetroAchievements.Token != "" {
		go func() {
			app.achievementManager.LoginWithToken(
				app.config.RetroAchievements.Username,
				app.config.RetroAchievements.Token,
				func(success bool, err error) {
					if !success {
						log.Printf("RetroAchievements auto-login failed: %v", err)
						// Clear invalid token
						app.config.RetroAchievements.Token = ""
						storage.SaveConfig(app.config)
					}
				},
			)
		}()
	}

	// Preload configured shaders
	app.preloadConfiguredShaders()

	// Initialize screens and build initial UI
	// Window dimensions are set by main before RunGame, so they're already correct
	app.initScreens()

	// Initialize scan manager (needs scanScreen reference)
	app.scanManager = NewScanManager(
		app.library,
		app.scanScreen,
		func() { app.rebuildCurrentScreen() }, // onProgress
		func(msg string) { // onComplete
			app.libraryScreen.ClearArtworkCache()
			app.state = StateSettings
			app.rebuildCurrentScreen()
			if msg != "" {
				app.notification.ShowDefault(msg)
			}
		},
	)

	// Call OnEnter for initial screen (sets default focus)
	app.libraryScreen.OnEnter()
	app.rebuildCurrentScreen()

	return app, nil
}

// GetWindowConfig returns the saved window dimensions and position from config.
// This should be called before RunGame to set the initial window size.
func (a *App) GetWindowConfig() (width, height int, x, y *int) {
	return a.config.Window.Width, a.config.Window.Height, a.config.Window.X, a.config.Window.Y
}

// saveWindowState saves current window position and size to config
func (a *App) saveWindowState() {
	// Don't overwrite config if it failed to load (user may want to fix it manually)
	if a.configLoadFailed {
		return
	}

	// Don't save if we never got valid window dimensions
	// (window size is tracked in Layout(), position in Update())
	if a.windowWidth == 0 || a.windowHeight == 0 {
		return
	}

	// Use tracked values (ebiten functions don't work after game loop ends)
	a.config.Window.Width = a.windowWidth
	a.config.Window.Height = a.windowHeight
	a.config.Window.X = &a.windowX
	a.config.Window.Y = &a.windowY

	// Save to disk
	storage.SaveConfig(a.config)
}

// initScreens creates all screen instances
func (a *App) initScreens() {
	a.libraryScreen = screens.NewLibraryScreen(a, a.library, a.config)
	a.detailScreen = screens.NewDetailScreen(a, a.library, a.config, a.achievementManager)
	a.settingsScreen = screens.NewSettingsScreen(a, a.library, a.config, a.achievementManager)
	a.scanScreen = screens.NewScanProgressScreen(a)
	a.errorScreen = screens.NewErrorScreen(a, a.errorFile, a.errorPath, a.handleDeleteAndContinue)
}

// rebuildCurrentScreen rebuilds the UI for the current state
func (a *App) rebuildCurrentScreen() {
	var container *widget.Container

	switch a.state {
	case StateLibrary:
		// Save scroll position before rebuilding
		a.libraryScreen.SaveScrollPosition()
		a.libraryScreen.SetLibrary(a.library)
		a.libraryScreen.SetConfig(a.config)
		container = a.libraryScreen.Build()
	case StateDetail:
		container = a.detailScreen.Build()
	case StateSettings:
		// Save scroll position before rebuilding
		a.settingsScreen.SaveScrollPosition()
		container = a.settingsScreen.Build()
	case StateScanProgress:
		container = a.scanScreen.Build()
	case StateError:
		container = a.errorScreen.Build()
	default:
		// For StatePlaying, no UI container needed
		return
	}

	a.ui = &ebitenui.UI{Container: container}
	a.lastBuildWidth = a.windowWidth // Track width for responsive rebuild detection
}

// Update implements ebiten.Game
func (a *App) Update() error {
	// Track window position while game is running (for save on exit)
	// Layout() handles width/height, but position must be queried here
	a.windowX, a.windowY = ebiten.WindowPosition()

	// Process any pending rebuild request (set from goroutines)
	if a.rebuildPending {
		a.rebuildPending = false
		a.rebuildCurrentScreen()
	}

	// Poll input manager for global keys (F12 screenshot)
	if a.inputManager.Update() {
		a.screenshotPending = true
	}

	// Check for window resize that needs UI rebuild (for responsive layouts)
	// Rebuild when width changes for screens with responsive layout
	needsResizeRebuild := false
	if a.state == StateLibrary {
		needsResizeRebuild = true
	}
	if a.state == StateDetail {
		needsResizeRebuild = true
	}
	if a.state == StateSettings {
		needsResizeRebuild = true
	}
	if needsResizeRebuild && a.windowWidth > 0 && a.windowWidth != a.lastBuildWidth {
		a.rebuildCurrentScreen()
	}

	switch a.state {
	case StatePlaying:
		_, err := a.gameplay.Update()
		return err
	case StateScanProgress:
		nav := a.processUIInput()
		a.ui.Update()
		a.scanManager.Update()
		// No scroll containers in scan screen
		_ = nav
		return nil
	case StateSettings:
		nav := a.processUIInput()
		a.settingsScreen.Update() // Handle section-specific updates (e.g., clipboard shortcuts)
		a.ui.Update()
		// Check if state changed during ui.Update (e.g., user navigated away)
		if a.state != StateSettings {
			return nil
		}
		a.restorePendingFocus(a.settingsScreen)
		if nav.FocusChanged {
			a.ensureFocusedVisible()
		}
		// Check if settings screen triggered a scan (after adding directory)
		if a.settingsScreen.HasPendingScan() {
			a.settingsScreen.ClearPendingScan()
			a.SwitchToScanProgress(false)
		}
	case StateLibrary:
		// Handle search overlay input first
		if a.searchOverlay.IsActive() {
			a.searchOverlay.HandleInput()
		}

		// Check for '/' to activate search (when not already active)
		if inpututil.IsKeyJustPressed(ebiten.KeySlash) && !a.searchOverlay.IsActive() {
			a.searchOverlay.Activate()
		}

		// ESC clears search if visible or active (before normal back handling)
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && (a.searchOverlay.IsVisible() || a.searchOverlay.IsActive()) {
			a.searchOverlay.Clear()
			return nil // Don't process as normal back
		}

		// Skip normal UI input if search is capturing
		var nav UINavigation
		if !a.searchOverlay.IsActive() {
			nav = a.processUIInput()
		}
		a.ui.Update()
		// Check if state changed during ui.Update (e.g., user clicked a game)
		if a.state != StateLibrary {
			return nil
		}
		a.restorePendingFocus(a.libraryScreen)
		if nav.FocusChanged {
			a.ensureFocusedVisible()
		}
	case StateDetail:
		nav := a.processUIInput()
		a.ui.Update()
		if a.state != StateDetail {
			return nil
		}
		a.restorePendingFocus(a.detailScreen)
		if nav.FocusChanged {
			a.ensureFocusedVisible()
		}
	default:
		// StateError only
		nav := a.processUIInput()
		prevState := a.state
		a.ui.Update()
		if a.state != prevState {
			return nil
		}
		if nav.FocusChanged {
			a.ensureFocusedVisible()
		}
	}
	return nil
}

// restorePendingFocus restores focus to a pending button if one exists
func (a *App) restorePendingFocus(screen screens.FocusRestorer) {
	btn := screen.GetPendingFocusButton()
	if btn != nil {
		btn.Focus(true)
		screen.ClearPendingFocus()
	}
}

// processUIInput polls gamepad input via InputManager and applies UI actions.
// Returns the navigation result for focus scroll handling.
func (a *App) processUIInput() UINavigation {
	if a.ui == nil {
		return UINavigation{}
	}

	nav := a.inputManager.GetUINavigation()

	// Apply navigation direction using spatial navigation if supported
	if nav.Direction != types.DirNone {
		a.applySpatialNavigation(nav.Direction)
	}

	// A/Cross button activates focused widget
	if nav.Activate {
		if focused := a.ui.GetFocusedWidget(); focused != nil {
			if btn, ok := focused.(*widget.Button); ok {
				btn.Click()
			}
		}
	}

	// B/Circle button for back navigation
	if nav.Back {
		a.handleGamepadBack()
	}

	// Start button opens settings from library
	if nav.OpenSettings && a.state == StateLibrary {
		a.SwitchToSettings()
	}

	return nav
}

// applySpatialNavigation uses 2D spatial navigation to find the next focus target.
// Falls back to linear navigation for screens that don't support spatial nav.
func (a *App) applySpatialNavigation(direction int) {
	// Get the current focused widget
	focused := a.ui.GetFocusedWidget()

	// Try spatial navigation on the current screen
	var nextBtn *widget.Button

	switch a.state {
	case StateLibrary:
		nextBtn = a.libraryScreen.FindFocusInDirection(focused, direction)
	case StateDetail:
		nextBtn = a.detailScreen.FindFocusInDirection(focused, direction)
	case StateSettings:
		nextBtn = a.settingsScreen.FindFocusInDirection(focused, direction)
		// StateError and StateScanProgress use linear navigation (simple layouts)
	}

	if nextBtn != nil {
		// Spatial navigation found a target - unfocus current first
		if focused != nil {
			focused.Focus(false)
		}
		nextBtn.Focus(true)
	} else {
		// Fallback to linear navigation
		if direction == types.DirUp || direction == types.DirLeft {
			a.ui.ChangeFocus(widget.FOCUS_PREVIOUS)
		} else {
			a.ui.ChangeFocus(widget.FOCUS_NEXT)
		}
	}
}

// handleGamepadBack handles B button press for back navigation
func (a *App) handleGamepadBack() {
	switch a.state {
	case StateDetail:
		a.SwitchToLibrary()
	case StateSettings:
		a.SwitchToLibrary()
	case StateScanProgress:
		// Cancel scan and return to settings
		a.scanManager.Cancel()
		// StateLibrary and StateError have no back action
	}
}

// ensureFocusedVisible scrolls the current screen to keep the focused widget visible
func (a *App) ensureFocusedVisible() {
	focused := a.ui.GetFocusedWidget()
	if focused == nil {
		return
	}

	// Call the appropriate screen's scroll method
	switch a.state {
	case StateLibrary:
		a.libraryScreen.EnsureFocusedVisible(focused)
	case StateSettings:
		a.settingsScreen.EnsureFocusedVisible(focused)
	}
}

// Draw implements ebiten.Game
func (a *App) Draw(screen *ebiten.Image) {
	// Advance frame counter for animated shaders
	a.shaderManager.IncrementFrame()

	// Determine which shaders to apply based on state and application mode
	shaderIDs := a.getActiveShaders()

	if len(shaderIDs) == 0 {
		// No shaders/effects - direct draw
		switch a.state {
		case StatePlaying:
			a.gameplay.Draw(screen)
			a.gameplay.DrawPauseMenu(screen)
			a.gameplay.DrawAchievementOverlay(screen)
		default:
			a.ui.Draw(screen)
		}
		a.notification.Draw(screen)
		if a.state == StateLibrary {
			a.searchOverlay.Draw(screen)
		}
	} else {
		// With effects/shaders - use preprocessing pipeline
		sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
		buffer := a.getOrCreateShaderBuffer(sw, sh)
		buffer.Clear()

		// Determine input for preprocessing based on xBR and state
		var preprocessInput *ebiten.Image
		hasXBR := shader.HasXBR(shaderIDs)

		switch a.state {
		case StatePlaying:
			if hasXBR {
				// xBR path: pass native framebuffer to preprocessing
				preprocessInput = a.gameplay.DrawFramebuffer()
			} else {
				// Non-xBR path: draw scaled game to buffer
				a.gameplay.Draw(buffer)
				preprocessInput = buffer
			}
		default:
			// UI: draw to buffer (xBR has no effect on UI)
			a.ui.Draw(buffer)
			preprocessInput = buffer
		}

		// Apply preprocessing effects (xBR, ghosting)
		// xBR scales native -> screen size; ghosting operates at screen size
		// Returns processed image (screen-sized) and remaining shader IDs
		processed, remainingShaders := a.shaderManager.ApplyPreprocessEffects(
			preprocessInput, shaderIDs, sw, sh)

		// For StatePlaying: draw pause menu and achievement overlay after effects (so shaders apply to them)
		if a.state == StatePlaying {
			a.gameplay.DrawPauseMenu(processed)
			a.gameplay.DrawAchievementOverlay(processed)
		}

		// Notification drawn after effects, before shaders
		a.notification.Draw(processed)
		if a.state == StateLibrary {
			a.searchOverlay.Draw(processed)
		}

		// Apply shader chain to final screen
		a.shaderManager.ApplyShaders(screen, processed, remainingShaders)
	}

	// Take screenshot if pending (after everything is drawn)
	if a.screenshotPending {
		a.screenshotPending = false
		gameCRC := a.gameplay.CurrentGameCRC()
		if err := a.screenshotManager.TakeScreenshot(screen, gameCRC); err != nil {
			log.Printf("Screenshot failed: %v", err)
		}
	}
}

// Layout implements ebiten.Game
func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Store dimensions for responsive layout calculations and persistence
	a.windowWidth = outsideWidth
	a.windowHeight = outsideHeight
	return outsideWidth, outsideHeight
}

// ScreenCallback implementations

// SwitchToLibrary transitions to the library screen
func (a *App) SwitchToLibrary() {
	a.notification.Clear()
	a.previousState = a.state
	a.state = StateLibrary
	a.libraryScreen.OnEnter()
	a.rebuildCurrentScreen()
	// Focus restoration is handled by the Update loop on the next frame
}

// SwitchToDetail transitions to the detail screen
func (a *App) SwitchToDetail(gameCRC string) {
	a.notification.Clear()
	a.previousState = a.state
	a.state = StateDetail
	a.detailScreen.SetGame(gameCRC)
	a.detailScreen.OnEnter()
	a.rebuildCurrentScreen()
}

// SwitchToSettings transitions to the settings screen
func (a *App) SwitchToSettings() {
	a.notification.Clear()
	a.previousState = a.state
	a.state = StateSettings
	a.settingsScreen.OnEnter()
	a.rebuildCurrentScreen()
}

// SwitchToScanProgress transitions to the scan progress screen
func (a *App) SwitchToScanProgress(rescanAll bool) {
	a.notification.Clear()
	a.previousState = a.state
	a.state = StateScanProgress

	// Start scan via manager
	a.scanManager.Start(rescanAll)
	a.scanScreen.OnEnter()
	a.rebuildCurrentScreen()
}

// LaunchGame starts the emulator with the specified game
func (a *App) LaunchGame(gameCRC string, resume bool) {
	// Reset shader buffers to avoid stale data from previous game
	a.shaderManager.ResetBuffers()

	if a.gameplay.Launch(gameCRC, resume) {
		a.previousState = a.state
		a.state = StatePlaying
	}
}

// Exit closes the application
func (a *App) Exit() {
	// Save window state before exiting
	a.saveWindowState()

	// Clean up achievement manager resources
	if a.achievementManager != nil {
		a.achievementManager.Destroy()
	}

	// Clean exit using os.Exit to avoid log.Fatal's stack trace
	os.Exit(0)
}

// GetWindowWidth returns the current window width for responsive layouts
func (a *App) GetWindowWidth() int {
	return a.windowWidth
}

// RequestRebuild triggers a UI rebuild for the current screen.
// This is safe to call from goroutines - the rebuild happens on the main thread.
// Focus restoration is handled in the Update loop after ui.Update()
func (a *App) RequestRebuild() {
	a.rebuildPending = true
}

// GetPlaceholderImageData returns the raw embedded placeholder image data
func (a *App) GetPlaceholderImageData() []byte {
	return placeholderImageData
}

// GetRDB returns the RDB for metadata lookups
func (a *App) GetRDB() *rdb.RDB {
	if a.metadata == nil {
		return nil
	}
	return a.metadata.GetRDB()
}

// handleDeleteAndContinue handles the delete and continue button
func (a *App) handleDeleteAndContinue() {
	var err error

	if a.errorFile == "config.json" {
		if err = storage.DeleteConfig(); err != nil {
			log.Printf("Failed to delete config: %v", err)
		}
		a.config = storage.DefaultConfig()
		if err = storage.SaveConfig(a.config); err != nil {
			log.Printf("Failed to save config: %v", err)
		}

		// Now try loading library
		library, err := storage.LoadLibrary()
		if err != nil {
			// Library is also corrupt
			libraryPath, _ := storage.GetLibraryPath()
			a.errorFile = "library.json"
			a.errorPath = libraryPath
			a.errorScreen.SetError("library.json", libraryPath)
			a.rebuildCurrentScreen()
			return
		}
		a.library = library
	} else if a.errorFile == "library.json" {
		if err = storage.DeleteLibrary(); err != nil {
			log.Printf("Failed to delete library: %v", err)
		}
		a.library = storage.DefaultLibrary()
		if err = storage.SaveLibrary(a.library); err != nil {
			log.Printf("Failed to save library: %v", err)
		}
	}

	// Update save state manager with new library
	a.saveStateManager.SetLibrary(a.library)

	// Initialize or update gameplay manager
	if a.gameplay == nil {
		a.gameplay = NewGameplayManager(
			a.saveStateManager,
			a.screenshotManager,
			a.notification,
			a.library,
			a.config,
			a.achievementManager,
			a.metadata.GetRDB(),
			func() { a.SwitchToLibrary() },
			func() { a.Exit() },
		)
	} else {
		a.gameplay.SetLibrary(a.library)
		a.gameplay.SetConfig(a.config)
	}

	// Reinitialize screens with fresh data
	a.initScreens()

	// Initialize or update scan manager (needs scanScreen reference)
	if a.scanManager == nil {
		a.scanManager = NewScanManager(
			a.library,
			a.scanScreen,
			func() { a.rebuildCurrentScreen() },
			func(msg string) {
				a.state = StateSettings
				a.rebuildCurrentScreen()
				if msg != "" {
					a.notification.ShowDefault(msg)
				}
			},
		)
	} else {
		a.scanManager.SetLibrary(a.library)
		a.scanManager.SetScanScreen(a.scanScreen)
	}

	// Proceed to library screen
	a.state = StateLibrary
	a.rebuildCurrentScreen()
}

// SaveAndClose saves config and library before exit
func (a *App) SaveAndClose() {
	// Capture current window state before saving
	a.saveWindowState()

	if err := storage.SaveLibrary(a.library); err != nil {
		log.Printf("Failed to save library: %v", err)
	}
}

// preloadConfiguredShaders loads all shaders referenced in config
func (a *App) preloadConfiguredShaders() {
	allShaders := make(map[string]bool)
	for _, id := range a.config.Shaders.UIShaders {
		allShaders[id] = true
	}
	for _, id := range a.config.Shaders.GameShaders {
		allShaders[id] = true
	}

	ids := make([]string, 0, len(allShaders))
	for id := range allShaders {
		ids = append(ids, id)
	}
	a.shaderManager.PreloadShaders(ids)
}

// getActiveShaders returns the shader IDs to apply for the current state
func (a *App) getActiveShaders() []string {
	switch a.state {
	case StatePlaying:
		return a.config.Shaders.GameShaders
	default:
		return a.config.Shaders.UIShaders
	}
}

// getOrCreateShaderBuffer returns a buffer matching the given dimensions
func (a *App) getOrCreateShaderBuffer(width, height int) *ebiten.Image {
	if a.shaderBuffer != nil {
		bw, bh := a.shaderBuffer.Bounds().Dx(), a.shaderBuffer.Bounds().Dy()
		if bw == width && bh == height {
			return a.shaderBuffer
		}
		a.shaderBuffer.Deallocate()
	}
	a.shaderBuffer = ebiten.NewImage(width, height)
	return a.shaderBuffer
}
