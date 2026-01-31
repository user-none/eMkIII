//go:build !libretro

package ui

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ebitenui/ebitenui"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/user-none/emkiii/emu"
	"github.com/user-none/emkiii/romloader"
	"github.com/user-none/emkiii/ui/screens"
	"github.com/user-none/emkiii/ui/storage"
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

	// Emulation
	emulator    *emu.Emulator
	currentGame *storage.GameEntry
	cropBorder  bool

	// Gameplay managers
	pauseMenu         *PauseMenu
	notification      *Notification
	saveStateManager  *SaveStateManager
	screenshotManager *ScreenshotManager
	playTimeTracker   *PlayTimeTracker

	// Auto-save
	autoSaveTimer    time.Time
	autoSaveInterval time.Duration
	autoSaving       bool

	// Scanner
	activeScanner *Scanner

	// Error state
	errorFile string
	errorPath string

	// Window tracking for persistence and responsive layouts
	windowX, windowY int
	windowWidth      int
	lastBuildWidth   int // Track width used for last UI build

	// Screenshot pending flag (set in Update, processed in Draw)
	screenshotPending bool
}

// PlayTimeTracker tracks play time during gameplay
type PlayTimeTracker struct {
	sessionSeconds int64
	trackStart     int64
	tracking       bool
}

// NewApp creates and initializes the application
func NewApp() (*App, error) {
	app := &App{
		state:            StateLibrary,
		autoSaveInterval: 5 * time.Second,
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

	// Initialize managers
	app.notification = NewNotification()
	app.saveStateManager = NewSaveStateManager(app.notification)
	app.screenshotManager = NewScreenshotManager(app.notification)
	app.playTimeTracker = &PlayTimeTracker{}

	// Initialize pause menu with callbacks
	app.pauseMenu = NewPauseMenu(
		func() { // onResume
			app.resumeFromPause()
		},
		func() { // onLibrary
			app.exitGame(true)
			app.SwitchToLibrary()
		},
		func() { // onExit
			app.exitGame(true)
			app.Exit()
		},
	)

	// Load config
	config, err := storage.LoadConfig()
	if err != nil {
		// JSON parse error - show error screen
		configPath, _ := storage.GetConfigPath()
		app.state = StateError
		app.errorFile = "config.json"
		app.errorPath = configPath
		app.config = storage.DefaultConfig()
		app.library = storage.DefaultLibrary()
		app.initScreens()
		app.buildErrorScreen()
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
		app.buildErrorScreen()
		return app, nil
	}
	app.library = library

	// Set library on save state manager for slot persistence
	app.saveStateManager.SetLibrary(library)

	// Initialize screens
	app.initScreens()
	app.rebuildCurrentScreen()

	// Restore window state from config
	app.restoreWindowState()

	return app, nil
}

// restoreWindowState restores window position and size from config
func (a *App) restoreWindowState() {
	// Restore window size
	if a.config.Window.Width > 0 && a.config.Window.Height > 0 {
		ebiten.SetWindowSize(a.config.Window.Width, a.config.Window.Height)
	}

	// Restore window position (only if explicitly set)
	if a.config.Window.X != nil && a.config.Window.Y != nil {
		ebiten.SetWindowPosition(*a.config.Window.X, *a.config.Window.Y)
	}
}

// saveWindowState saves current window position and size to config
func (a *App) saveWindowState() {
	// Get current window state
	w, h := ebiten.WindowSize()
	x, y := ebiten.WindowPosition()

	// Update config
	a.config.Window.Width = w
	a.config.Window.Height = h
	a.config.Window.X = &x
	a.config.Window.Y = &y

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
		return a.updatePlaying()
	case StateScanProgress:
		return a.updateScanProgress()
	case StateSettings:
		a.ui.Update()
		// Check if settings screen triggered a scan (after adding directory)
		if a.settingsScreen.HasPendingScan() {
			a.settingsScreen.ClearPendingScan()
			a.SwitchToScanProgress(false)
		}
	default:
		a.ui.Update()
	}
	return nil
}

// abs returns the absolute value of an int
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Draw implements ebiten.Game
func (a *App) Draw(screen *ebiten.Image) {
	switch a.state {
	case StatePlaying:
		a.drawPlaying(screen)
	default:
		a.ui.Draw(screen)
	}

	// Draw notification overlay (all screens)
	a.notification.Draw(screen)

	// Take screenshot if pending (after everything is drawn)
	if a.screenshotPending {
		a.screenshotPending = false
		var gameCRC string
		if a.state == StatePlaying && a.currentGame != nil {
			gameCRC = a.currentGame.CRC32
		}
		if err := a.screenshotManager.TakeScreenshot(screen, gameCRC); err != nil {
			log.Printf("Screenshot failed: %v", err)
		}
	}
}

// Layout implements ebiten.Game
func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Store width for responsive layout calculations
	a.windowWidth = outsideWidth
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
	game := a.library.GetGame(gameCRC)
	if game == nil {
		a.notification.ShowDefault("Game not found")
		return
	}

	// Load ROM
	romData, _, err := romloader.LoadROM(game.File)
	if err != nil {
		game.Missing = true
		storage.SaveLibrary(a.library)
		a.notification.ShowDefault("Failed to load ROM")
		return
	}

	// Determine region
	region := a.regionFromLibraryEntry(game)

	// Apply video settings from config
	a.cropBorder = a.config.Video.CropBorder

	// Create emulator
	a.emulator = emu.NewEmulator(romData, region)
	a.currentGame = game
	a.saveStateManager.SetGame(gameCRC)

	// Load SRAM if exists
	if err := a.saveStateManager.LoadSRAM(a.emulator); err != nil {
		log.Printf("Failed to load SRAM: %v", err)
	}

	// Load resume state if requested
	if resume {
		if err := a.saveStateManager.LoadResume(a.emulator); err != nil {
			a.notification.ShowShort("Failed to resume, starting fresh")
		}
	}

	// Update library entry
	game.LastPlayed = time.Now().Unix()
	storage.SaveLibrary(a.library)

	// Set TPS for region
	timing := emu.GetTimingForRegion(region)
	ebiten.SetTPS(timing.FPS)

	// Start play time tracking
	a.playTimeTracker.sessionSeconds = 0
	a.playTimeTracker.trackStart = time.Now().Unix()
	a.playTimeTracker.tracking = true

	// Start auto-save timer (first save after 1 second)
	a.autoSaveTimer = time.Now().Add(time.Second)

	// Initialize pause menu
	a.pauseMenu.Hide()

	// Change state
	a.previousState = a.state
	a.state = StatePlaying
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
func (a *App) RequestRebuild() {
	a.rebuildCurrentScreen()
}

// regionFromLibraryEntry determines the region from a library entry
func (a *App) regionFromLibraryEntry(game *storage.GameEntry) emu.Region {
	switch strings.ToLower(game.Region) {
	case "eu", "europe", "pal":
		return emu.RegionPAL
	default:
		return emu.RegionNTSC
	}
}

// updatePlaying handles the gameplay update loop
func (a *App) updatePlaying() error {
	// Check for pause menu toggle (ESC) - only open, not close
	// (closing is handled by the menu's ESC key detection calling onResume)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && !a.pauseMenu.IsVisible() {
		// Open pause menu
		a.triggerAutoSave()
		a.pauseMenu.Show()
		a.pausePlayTimeTracking()
		return nil
	}

	// Handle pause menu if visible (input handled via callbacks)
	if a.pauseMenu.IsVisible() {
		a.pauseMenu.Update()
		return nil
	}

	// Poll and pass input to emulator
	a.pollGameplayInput()

	// Run one frame of emulation
	a.emulator.RunFrame()

	// Queue audio samples to SDL
	a.emulator.QueueAudio()

	// Handle save state keys
	a.handleSaveStateKeys()

	// Check auto-save timer
	if time.Now().After(a.autoSaveTimer) && !a.autoSaving {
		a.triggerAutoSave()
		a.autoSaveTimer = time.Now().Add(a.autoSaveInterval)
	}

	return nil
}

// pollGameplayInput reads input and passes it to the emulator
func (a *App) pollGameplayInput() {
	// Keyboard (WASD + JK)
	up := ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp)
	down := ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown)
	left := ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
	right := ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight)
	btn1 := ebiten.IsKeyPressed(ebiten.KeyJ) || ebiten.IsKeyPressed(ebiten.KeyZ)
	btn2 := ebiten.IsKeyPressed(ebiten.KeyK) || ebiten.IsKeyPressed(ebiten.KeyX)

	// Gamepad support
	gamepadIDs := ebiten.AppendGamepadIDs(nil)
	for _, id := range gamepadIDs {
		// D-pad
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			up = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			down = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
			left = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight) {
			right = true
		}
		// Buttons
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightBottom) {
			btn1 = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightRight) {
			btn2 = true
		}
		// Analog stick
		axisX := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		axisY := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
		if axisX < -0.25 {
			left = true
		}
		if axisX > 0.25 {
			right = true
		}
		if axisY < -0.25 {
			up = true
		}
		if axisY > 0.25 {
			down = true
		}
	}

	a.emulator.SetInput(up, down, left, right, btn1, btn2)

	// SMS Pause button (Enter key triggers NMI)
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		a.emulator.SetPause()
	}
}

// handleSaveStateKeys handles F1/F2/F3 for save states
func (a *App) handleSaveStateKeys() {
	// F1 - Save to current slot
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		if err := a.saveStateManager.Save(a.emulator); err != nil {
			log.Printf("Save state failed: %v", err)
		}
	}

	// F2 - Next slot (Shift+F2 - Previous slot)
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			a.saveStateManager.PreviousSlot()
		} else {
			a.saveStateManager.NextSlot()
		}
	}

	// F3 - Load from current slot
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		if err := a.saveStateManager.Load(a.emulator); err != nil {
			log.Printf("Load state failed: %v", err)
		}
	}
}

// drawPlaying renders the gameplay screen
func (a *App) drawPlaying(screen *ebiten.Image) {
	if a.emulator == nil {
		return
	}

	// Use emulator's DrawToScreen for rendering (same implementation as command-line mode)
	a.emulator.DrawToScreen(screen, a.cropBorder)

	// Draw pause menu overlay if visible
	if a.pauseMenu.IsVisible() {
		a.pauseMenu.Draw(screen)
	}
}

// triggerAutoSave performs an auto-save
func (a *App) triggerAutoSave() {
	if a.emulator == nil || a.currentGame == nil || a.autoSaving {
		return
	}

	a.autoSaving = true
	go func() {
		defer func() { a.autoSaving = false }()

		// Save resume state
		if err := a.saveStateManager.SaveResume(a.emulator); err != nil {
			log.Printf("Auto-save failed: %v", err)
		}

		// Save SRAM
		if err := a.saveStateManager.SaveSRAM(a.emulator); err != nil {
			log.Printf("SRAM save failed: %v", err)
		}

		// Update play time
		a.updatePlayTime()
	}()
}

// resumeFromPause resumes gameplay after pause menu
func (a *App) resumeFromPause() {
	a.pauseMenu.Hide()
	a.playTimeTracker.trackStart = time.Now().Unix()
	a.playTimeTracker.tracking = true
	a.autoSaveTimer = time.Now().Add(a.autoSaveInterval)
}

// pausePlayTimeTracking pauses the play time tracker
func (a *App) pausePlayTimeTracking() {
	if a.playTimeTracker.tracking {
		elapsed := time.Now().Unix() - a.playTimeTracker.trackStart
		a.playTimeTracker.sessionSeconds += elapsed
		a.playTimeTracker.tracking = false
	}
}

// updatePlayTime updates the play time in the library
func (a *App) updatePlayTime() {
	if a.currentGame == nil {
		return
	}

	var totalSession int64
	if a.playTimeTracker.tracking {
		elapsed := time.Now().Unix() - a.playTimeTracker.trackStart
		totalSession = a.playTimeTracker.sessionSeconds + elapsed
	} else {
		totalSession = a.playTimeTracker.sessionSeconds
	}

	// Only update if there's actual play time
	if totalSession > 0 {
		a.currentGame.PlayTimeSeconds += totalSession
		a.playTimeTracker.sessionSeconds = 0
		if a.playTimeTracker.tracking {
			a.playTimeTracker.trackStart = time.Now().Unix()
		}
		storage.SaveLibrary(a.library)
	}
}

// exitGame cleans up when exiting gameplay
func (a *App) exitGame(saveResume bool) {
	if a.emulator == nil {
		return
	}

	// Stop play time tracking and update
	a.pausePlayTimeTracking()
	a.updatePlayTime()

	// Save SRAM
	if err := a.saveStateManager.SaveSRAM(a.emulator); err != nil {
		log.Printf("SRAM save failed: %v", err)
	}

	// Save resume state if requested
	if saveResume {
		if err := a.saveStateManager.SaveResume(a.emulator); err != nil {
			log.Printf("Resume save failed: %v", err)
		}
	}

	// Close emulator
	a.emulator.Close()
	a.emulator = nil
	a.currentGame = nil

	// Reset TPS to 60 for UI
	ebiten.SetTPS(60)
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

// buildErrorScreen creates the error screen UI
func (a *App) buildErrorScreen() {
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(eimage.NewNineSliceColor(Theme.Background)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
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
		widget.TextOpts.Text("Configuration Error", GetFontFace(), Theme.Text),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(titleLabel)

	msgText := fmt.Sprintf("The file \"%s\" is invalid or corrupted.", a.errorFile)
	msgLabel := widget.NewText(
		widget.TextOpts.Text(msgText, GetFontFace(), Theme.Text),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(msgLabel)

	helpLabel := widget.NewText(
		widget.TextOpts.Text("You can delete the file and start fresh, or exit to manually fix the file.", GetFontFace(), Theme.TextSecondary),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	centerContent.AddChild(helpLabel)

	// Buttons container
	buttonsContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(16),
		)),
	)

	deleteButton := widget.NewButton(
		widget.ButtonOpts.Image(NewButtonImage()),
		widget.ButtonOpts.Text("Delete and Continue", GetFontFace(), &widget.ButtonTextColor{
			Idle:     Theme.Text,
			Disabled: Theme.TextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			a.handleDeleteAndContinue()
		}),
	)
	buttonsContainer.AddChild(deleteButton)

	exitButton := widget.NewButton(
		widget.ButtonOpts.Image(NewButtonImage()),
		widget.ButtonOpts.Text("Exit", GetFontFace(), &widget.ButtonTextColor{
			Idle:     Theme.Text,
			Disabled: Theme.TextSecondary,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(12)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			log.Fatalf("Configuration error in %s", a.errorPath)
		}),
	)
	buttonsContainer.AddChild(exitButton)

	centerContent.AddChild(buttonsContainer)
	rootContainer.AddChild(centerContent)

	a.ui = &ebitenui.UI{Container: rootContainer}
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
			a.buildErrorScreen()
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

	// Reinitialize screens with fresh data
	a.initScreens()

	// Proceed to library screen
	a.state = StateLibrary
	a.rebuildCurrentScreen()
}

// SaveAndClose saves config and library before exit
func (a *App) SaveAndClose() {
	// Save window position if we had one
	// Note: Ebiten doesn't provide a way to get window position easily
	// This would be implemented with platform-specific code

	if err := storage.SaveConfig(a.config); err != nil {
		log.Printf("Failed to save config: %v", err)
	}
	if err := storage.SaveLibrary(a.library); err != nil {
		log.Printf("Failed to save library: %v", err)
	}
}
