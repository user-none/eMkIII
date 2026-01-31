//go:build !libretro

package ui

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/user-none/emkiii/ui/screens"
	"github.com/user-none/emkiii/ui/storage"
	"github.com/user-none/emkiii/ui/style"
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

	// Gameplay (emulation, input, save states, pause menu)
	gameplay *GameplayManager

	// UI managers
	notification      *Notification
	saveStateManager  *SaveStateManager
	screenshotManager *ScreenshotManager

	// Scanner
	activeScanner *Scanner

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

	// Gamepad navigation state for UI navigation
	gamepadNav gamepadNavState
}

// gamepadNavState tracks continuous navigation state for gamepad input.
// This handles repeat navigation when holding a direction.
type gamepadNavState struct {
	direction    int           // 0=none, 1=prev, 2=next
	startTime    time.Time     // When direction was first pressed
	lastMove     time.Time     // When last move occurred
	repeatDelay  time.Duration // Current repeat interval
	stickMoved   bool          // Analog stick state for debouncing
	focusChanged bool          // Track if focus changed this frame (for scroll after layout)
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
		app.library = storage.DefaultLibrary()
		app.initScreens()
		app.rebuildCurrentScreen()
		return app, nil
	}
	app.config = config

	// Load library
	library, err := storage.LoadLibrary()
	if err != nil {
		// JSON parse error - show error screen
		libraryPath, _ := storage.GetLibraryPath()
		app.state = StateError
		app.errorFile = "library.json"
		app.errorPath = libraryPath
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
		app.notification,
		app.library,
		app.config,
		func() { app.SwitchToLibrary() }, // onExitToLibrary
		func() { app.Exit() },            // onExitApp
	)

	// Initialize screens and build initial UI
	// Window dimensions are set by main before RunGame, so they're already correct
	app.initScreens()
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
	a.detailScreen = screens.NewDetailScreen(a, a.library, a.config)
	a.settingsScreen = screens.NewSettingsScreen(a, a.library, a.config)
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

	// Handle screenshot globally (F12 works everywhere)
	// Set flag here, actual screenshot taken in Draw() where we have screen access
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		a.screenshotPending = true
	}

	// Check for window resize that needs UI rebuild (for responsive layouts)
	// Rebuild when width changes in icon mode (or if never built with real width)
	if a.state == StateLibrary && a.config.Library.ViewMode == "icon" {
		if a.windowWidth > 0 && a.windowWidth != a.lastBuildWidth {
			a.rebuildCurrentScreen()
		}
	}

	switch a.state {
	case StatePlaying:
		_, err := a.gameplay.Update()
		return err
	case StateScanProgress:
		a.handleGamepadUI()
		err := a.updateScanProgress()
		// Clear focus changed flag - no scroll containers in scan screen
		a.gamepadNav.focusChanged = false
		return err
	case StateSettings:
		a.handleGamepadUI()
		a.ui.Update()
		// Check if state changed during ui.Update (e.g., user navigated away)
		if a.state != StateSettings {
			return nil
		}
		a.gamepadNav.focusChanged = false // Settings has its own scroll handling
		a.restorePendingFocus(a.settingsScreen)
		// Check if settings screen triggered a scan (after adding directory)
		if a.settingsScreen.HasPendingScan() {
			a.settingsScreen.ClearPendingScan()
			a.SwitchToScanProgress(false)
		}
	case StateLibrary:
		a.handleGamepadUI()
		a.ui.Update()
		// Check if state changed during ui.Update (e.g., user clicked a game)
		if a.state != StateLibrary {
			return nil
		}
		a.restorePendingFocus(a.libraryScreen)
		a.handleFocusScroll()
	default:
		// StateDetail, StateError
		a.handleGamepadUI()
		prevState := a.state
		a.ui.Update()
		// Check if state changed during ui.Update (e.g., user clicked Back)
		if a.state != prevState {
			return nil
		}
		a.handleFocusScroll()
	}
	return nil
}

// handleFocusScroll scrolls to keep focused widget visible after gamepad navigation
func (a *App) handleFocusScroll() {
	if a.gamepadNav.focusChanged {
		a.ensureFocusedVisible()
		a.gamepadNav.focusChanged = false
	}
}

// restorePendingFocus restores focus to a pending button if one exists
func (a *App) restorePendingFocus(screen screens.FocusRestorer) {
	btn := screen.GetPendingFocusButton()
	if btn != nil {
		btn.Focus(true)
		screen.ClearPendingFocus()
	}
}

// handleGamepadUI processes gamepad input for UI navigation
// This leverages ebitenui's built-in focus system
func (a *App) handleGamepadUI() {
	if a.ui == nil {
		return
	}

	gamepadIDs := ebiten.AppendGamepadIDs(nil)
	if len(gamepadIDs) == 0 {
		return
	}

	// Use first connected gamepad
	id := gamepadIDs[0]

	// Navigation timing uses constants from style package

	// Determine current navigation direction from D-pad and analog stick
	navPrev := false
	navNext := false

	// D-pad
	if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop) ||
		ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
		navPrev = true
	}
	if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom) ||
		ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight) {
		navNext = true
	}

	// Analog stick
	axisY := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
	axisX := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
	if axisY < -0.5 || axisX < -0.5 {
		navPrev = true
	}
	if axisY > 0.5 || axisX > 0.5 {
		navNext = true
	}

	// Determine desired direction (prev takes priority if both pressed)
	desiredDir := 0
	if navPrev {
		desiredDir = 1
	} else if navNext {
		desiredDir = 2
	}

	now := time.Now()
	focusChanged := false

	if desiredDir == 0 {
		// No direction pressed - reset state
		a.gamepadNav.direction = 0
		a.gamepadNav.repeatDelay = style.NavStartInterval
	} else if desiredDir != a.gamepadNav.direction {
		// Direction changed - move immediately and start tracking
		a.gamepadNav.direction = desiredDir
		a.gamepadNav.startTime = now
		a.gamepadNav.lastMove = now
		a.gamepadNav.repeatDelay = style.NavStartInterval

		if desiredDir == 1 {
			a.ui.ChangeFocus(widget.FOCUS_PREVIOUS)
		} else {
			a.ui.ChangeFocus(widget.FOCUS_NEXT)
		}
		focusChanged = true
	} else {
		// Same direction held - check for repeat
		holdDuration := now.Sub(a.gamepadNav.startTime)
		timeSinceLastMove := now.Sub(a.gamepadNav.lastMove)

		if holdDuration >= style.NavInitialDelay && timeSinceLastMove >= a.gamepadNav.repeatDelay {
			// Time to repeat
			if desiredDir == 1 {
				a.ui.ChangeFocus(widget.FOCUS_PREVIOUS)
			} else {
				a.ui.ChangeFocus(widget.FOCUS_NEXT)
			}
			focusChanged = true
			a.gamepadNav.lastMove = now

			// Accelerate (decrease interval)
			a.gamepadNav.repeatDelay -= style.NavAcceleration
			if a.gamepadNav.repeatDelay < style.NavMinInterval {
				a.gamepadNav.repeatDelay = style.NavMinInterval
			}
		}
	}

	// Mark that focus changed - scroll check happens after ui.Update()
	if focusChanged {
		a.gamepadNav.focusChanged = true
	}

	// A/Cross button activates focused widget
	if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom) {
		if focused := a.ui.GetFocusedWidget(); focused != nil {
			if btn, ok := focused.(*widget.Button); ok {
				btn.Click()
			}
		}
	}

	// B/Circle button for back navigation
	if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightRight) {
		a.handleGamepadBack()
	}

	// Start button opens settings from library
	if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight) {
		if a.state == StateLibrary {
			a.SwitchToSettings()
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
		if a.activeScanner != nil {
			a.activeScanner.Cancel()
		}
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
		// Other screens can be added here as needed
	}
}

// Draw implements ebiten.Game
func (a *App) Draw(screen *ebiten.Image) {
	switch a.state {
	case StatePlaying:
		a.gameplay.Draw(screen)
	default:
		a.ui.Draw(screen)
	}

	// Draw notification overlay (all screens)
	a.notification.Draw(screen)

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

	// Create and start scanner
	a.activeScanner = NewScanner(
		a.library.ScanDirectories,
		a.library.ExcludedPaths,
		a.library.Games,
		rescanAll,
	)

	a.scanScreen.SetScanner(a.activeScanner)
	a.scanScreen.OnEnter()
	a.rebuildCurrentScreen()

	// Start scanner in background
	go a.activeScanner.Run()
}

// LaunchGame starts the emulator with the specified game
func (a *App) LaunchGame(gameCRC string, resume bool) {
	if a.gameplay.Launch(gameCRC, resume) {
		a.previousState = a.state
		a.state = StatePlaying
	}
}

// Exit closes the application
func (a *App) Exit() {
	// Save window state before exiting
	a.saveWindowState()

	// Clean exit using os.Exit to avoid log.Fatal's stack trace
	os.Exit(0)
}

// GetWindowWidth returns the current window width for responsive layouts
func (a *App) GetWindowWidth() int {
	return a.windowWidth
}

// RequestRebuild triggers a UI rebuild for the current screen
// Focus restoration is handled in the Update loop after ui.Update()
func (a *App) RequestRebuild() {
	a.rebuildCurrentScreen()
}

// GetPlaceholderImageData returns the raw embedded placeholder image data
func (a *App) GetPlaceholderImageData() []byte {
	return placeholderImageData
}

// updateScanProgress handles the scan progress screen updates
func (a *App) updateScanProgress() error {
	a.ui.Update()

	if a.activeScanner == nil {
		return nil
	}

	// Non-blocking read from progress channel
	select {
	case progress := <-a.activeScanner.Progress():
		// Convert ui.ScanProgress to screens.ScanProgress
		a.scanScreen.UpdateProgress(screens.ScanProgress{
			Phase:           int(progress.Phase),
			Progress:        progress.Progress,
			GamesFound:      progress.GamesFound,
			ArtworkTotal:    progress.ArtworkTotal,
			ArtworkComplete: progress.ArtworkComplete,
			StatusText:      progress.StatusText,
		})
		// Rebuild UI to reflect progress changes
		a.rebuildCurrentScreen()
	default:
		// No update this frame
	}

	// Check for completion
	select {
	case result := <-a.activeScanner.Done():
		a.handleScanComplete(result)
	default:
		// Still running
	}

	// Handle cancel - scanner.Cancel is already called by the screen's button handler

	return nil
}

// handleScanComplete processes scan results
func (a *App) handleScanComplete(result ScanResult) {
	// Merge discovered games into library
	for crc, game := range a.activeScanner.Games() {
		a.library.Games[crc] = game
	}

	// Save updated library
	if err := storage.SaveLibrary(a.library); err != nil {
		log.Printf("Failed to save library: %v", err)
	}

	// Prepare notification message
	var msg string
	if result.Cancelled {
		msg = "" // No notification on cancel
	} else if len(result.Errors) > 0 {
		msg = result.Errors[0].Error()
	} else if result.NewGames > 0 {
		msg = fmt.Sprintf("Found %d new games", result.NewGames)
	} else {
		msg = "Library up to date"
	}

	// Return to settings with notification
	a.activeScanner = nil
	a.state = StateSettings
	a.rebuildCurrentScreen()
	if msg != "" {
		a.notification.ShowDefault(msg)
	}
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
			a.notification,
			a.library,
			a.config,
			func() { a.SwitchToLibrary() },
			func() { a.Exit() },
		)
	} else {
		a.gameplay.SetLibrary(a.library)
		a.gameplay.SetConfig(a.config)
	}

	// Reinitialize screens with fresh data
	a.initScreens()

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
