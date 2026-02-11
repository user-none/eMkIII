//go:build !libretro

package ui

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	emubridge "github.com/user-none/emkiii/bridge/ebiten"
	"github.com/user-none/emkiii/emu"
	"github.com/user-none/emkiii/romloader"
	"github.com/user-none/emkiii/ui/achievements"
	"github.com/user-none/emkiii/ui/rdb"
	"github.com/user-none/emkiii/ui/storage"
	"github.com/user-none/emkiii/ui/style"
)

// GameplayManager handles all gameplay-related state and logic.
// This includes emulator control, input handling, save states,
// play time tracking, and the pause menu.
type GameplayManager struct {
	// Emulation state
	emulator    *emubridge.Emulator
	audioPlayer *AudioPlayer
	currentGame *storage.GameEntry
	cropBorder  bool

	// Rewind
	rewindBuffer *RewindBuffer

	// Pause menu
	pauseMenu *PauseMenu

	// Achievement overlay
	achievementOverlay *AchievementOverlay

	// Play time tracking
	playTime PlayTimeTracker

	// Auto-save state
	autoSaveTimer    time.Time
	autoSaveInterval time.Duration
	autoSaving       bool
	autoSaveWg       sync.WaitGroup

	// Achievement screenshot (set by callback, processed in Draw)
	achievementScreenshotPending bool
	achievementScreenshotMu      sync.Mutex

	// External dependencies (not owned by GameplayManager)
	saveStateManager   *SaveStateManager
	screenshotManager  *ScreenshotManager
	notification       *Notification
	library            *storage.Library
	config             *storage.Config
	achievementManager *achievements.Manager
	rdb                *rdb.RDB

	// Callbacks to App
	onExitToLibrary func()
	onExitApp       func()
}

// PlayTimeTracker tracks play time during gameplay
type PlayTimeTracker struct {
	sessionSeconds int64
	trackStart     int64
	tracking       bool
}

// NewGameplayManager creates a new gameplay manager
func NewGameplayManager(
	saveStateManager *SaveStateManager,
	screenshotManager *ScreenshotManager,
	notification *Notification,
	library *storage.Library,
	config *storage.Config,
	achievementManager *achievements.Manager,
	gameRDB *rdb.RDB,
	onExitToLibrary func(),
	onExitApp func(),
) *GameplayManager {
	gm := &GameplayManager{
		autoSaveInterval:   style.AutoSaveInterval,
		saveStateManager:   saveStateManager,
		screenshotManager:  screenshotManager,
		notification:       notification,
		library:            library,
		config:             config,
		achievementManager: achievementManager,
		rdb:                gameRDB,
		onExitToLibrary:    onExitToLibrary,
		onExitApp:          onExitApp,
	}

	// Initialize pause menu with callbacks
	gm.pauseMenu = NewPauseMenu(
		func() { // onResume
			gm.Resume()
		},
		func() { // onLibrary
			gm.Exit(true)
			if gm.onExitToLibrary != nil {
				gm.onExitToLibrary()
			}
		},
		func() { // onExit
			gm.Exit(true)
			if gm.onExitApp != nil {
				gm.onExitApp()
			}
		},
	)

	// Initialize achievement overlay
	gm.achievementOverlay = NewAchievementOverlay(achievementManager)

	return gm
}

// SetLibrary updates the library reference
func (gm *GameplayManager) SetLibrary(library *storage.Library) {
	gm.library = library
}

// SetConfig updates the config reference
func (gm *GameplayManager) SetConfig(config *storage.Config) {
	gm.config = config
}

// IsPlaying returns true if a game is currently being played
func (gm *GameplayManager) IsPlaying() bool {
	return gm.emulator != nil
}

// CurrentGameCRC returns the CRC of the currently loaded game, or empty string if none
func (gm *GameplayManager) CurrentGameCRC() string {
	if gm.currentGame != nil {
		return gm.currentGame.CRC32
	}
	return ""
}

// Launch starts the emulator with the specified game
func (gm *GameplayManager) Launch(gameCRC string, resume bool) bool {
	game := gm.library.GetGame(gameCRC)
	if game == nil {
		gm.notification.ShowDefault("Game not found")
		return false
	}

	// Load ROM
	romData, _, err := romloader.LoadROM(game.File)
	if err != nil {
		game.Missing = true
		storage.SaveLibrary(gm.library)
		gm.notification.ShowDefault("Failed to load ROM")
		return false
	}

	// Determine region
	region := gm.regionFromLibraryEntry(game)

	// Apply video settings from config
	gm.cropBorder = gm.config.Video.CropBorder

	// Create emulator
	gm.emulator = emubridge.NewEmulator(romData, region)
	gm.currentGame = game
	gm.saveStateManager.SetGame(gameCRC)

	// Create audio player for game audio (skip if muted)
	if !gm.config.Audio.Muted {
		player, err := NewAudioPlayer()
		if err != nil {
			log.Printf("Failed to init audio: %v", err)
		} else {
			gm.audioPlayer = player
		}
	}

	// Load SRAM if exists
	if err := gm.saveStateManager.LoadSRAM(gm.emulator); err != nil {
		log.Printf("Failed to load SRAM: %v", err)
	}

	// Load resume state if requested
	if resume {
		if err := gm.saveStateManager.LoadResume(gm.emulator); err != nil {
			gm.notification.ShowShort("Failed to resume, starting fresh")
		}
	}

	// Update library entry
	game.LastPlayed = time.Now().Unix()
	storage.SaveLibrary(gm.library)

	// Create rewind buffer if enabled
	if gm.config.Rewind.Enabled {
		gm.rewindBuffer = NewRewindBuffer(gm.config.Rewind.BufferSizeMB, gm.config.Rewind.FrameStep, emu.SerializeSize())
	}

	// Set TPS for region
	timing := emu.GetTimingForRegion(region)
	ebiten.SetTPS(timing.FPS)

	// Start play time tracking
	gm.playTime.sessionSeconds = 0
	gm.playTime.trackStart = time.Now().Unix()
	gm.playTime.tracking = true

	// Start auto-save timer (first save after 1 second)
	gm.autoSaveTimer = time.Now().Add(time.Second)

	// Initialize pause menu
	gm.pauseMenu.Hide()

	// Set up RetroAchievements if enabled and logged in
	if gm.achievementManager != nil && gm.achievementManager.IsEnabled() && gm.achievementManager.IsLoggedIn() {
		// Set up screenshot callback
		gm.achievementManager.SetScreenshotFunc(func() {
			gm.achievementScreenshotMu.Lock()
			gm.achievementScreenshotPending = true
			gm.achievementScreenshotMu.Unlock()
		})

		gm.achievementManager.SetEmulator(gm.emulator)
		// Look up MD5 from RDB for fast path (avoids re-hashing ROM)
		var md5Hash string
		if gm.rdb != nil {
			crc32, _ := strconv.ParseUint(game.CRC32, 16, 32)
			md5Hash = gm.rdb.GetMD5ByCRC32(uint32(crc32))
		}
		if err := gm.achievementManager.LoadGame(romData, game.File, md5Hash); err != nil {
			log.Printf("Failed to load achievements: %v", err)
		} else {
			// Initialize overlay with achievement data for this game
			gm.achievementOverlay.InitForGame()
		}
	}

	return true
}

// Update handles the gameplay update loop. Returns true if pause menu was opened.
func (gm *GameplayManager) Update() (pauseMenuOpened bool, err error) {
	if gm.emulator == nil {
		return false, nil
	}

	// Check for Tab key to toggle achievement overlay
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) && !gm.pauseMenu.IsVisible() {
		if gm.achievementOverlay.IsVisible() {
			gm.achievementOverlay.Hide()
			gm.playTime.trackStart = time.Now().Unix()
			gm.playTime.tracking = true
		} else if gm.achievementManager != nil && gm.achievementManager.IsGameLoaded() {
			gm.achievementOverlay.Show()
			gm.pausePlayTimeTracking()
		}
	}

	// Handle achievement overlay if visible
	if gm.achievementOverlay.IsVisible() {
		gm.achievementOverlay.Update()
		// Process achievement idle tasks while overlay is shown
		if gm.achievementManager != nil {
			gm.achievementManager.Idle()
		}
		return false, nil
	}

	// Check for pause menu toggle (ESC or Select button)
	openPauseMenu := inpututil.IsKeyJustPressed(ebiten.KeyEscape)

	// Check for Select button on gamepad
	gamepadIDs := ebiten.AppendGamepadIDs(nil)
	for _, id := range gamepadIDs {
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterLeft) {
			openPauseMenu = true
			break
		}
	}

	if openPauseMenu && !gm.pauseMenu.IsVisible() {
		// Open pause menu
		gm.triggerAutoSave()
		gm.pauseMenu.Show()
		gm.pausePlayTimeTracking()
		return true, nil
	}

	// Handle pause menu if visible
	if gm.pauseMenu.IsVisible() {
		gm.pauseMenu.Update()
		// Process achievement idle tasks while paused
		if gm.achievementManager != nil {
			gm.achievementManager.Idle()
		}
		return false, nil
	}

	// Poll and pass input to emulator
	gm.pollInput()

	// Check rewind input (R key)
	if gm.rewindBuffer != nil {
		holdDuration := inpututil.KeyPressDuration(ebiten.KeyR)
		if holdDuration > 0 {
			items := rewindItemsForHoldDuration(holdDuration)
			if items > 0 {
				if !gm.rewindBuffer.IsRewinding() {
					gm.rewindBuffer.SetRewinding(true)
					if gm.audioPlayer != nil {
						gm.audioPlayer.ClearQueue()
					}
				}
				gm.rewindBuffer.Rewind(gm.emulator, items)
				return false, nil
			}
			// items == 0 means we're in a hold gap frame; skip normal execution
			return false, nil
		} else if gm.rewindBuffer.IsRewinding() {
			// R released - resume normal play
			gm.rewindBuffer.SetRewinding(false)
		}
	}

	// Run one frame of emulation
	gm.emulator.RunFrame()

	// Process RetroAchievements
	if gm.achievementManager != nil {
		gm.achievementManager.DoFrame()
	}

	// Queue audio samples
	if gm.audioPlayer != nil {
		gm.audioPlayer.QueueSamples(gm.emulator.GetAudioSamples())
	}

	// Capture rewind state (after RunFrame, only when not rewinding)
	if gm.rewindBuffer != nil {
		gm.rewindBuffer.Capture(gm.emulator)
	}

	// Handle save state keys
	gm.handleSaveStateKeys()

	// Check auto-save timer
	if time.Now().After(gm.autoSaveTimer) && !gm.autoSaving {
		gm.triggerAutoSave()
		gm.autoSaveTimer = time.Now().Add(gm.autoSaveInterval)
	}

	return false, nil
}

// Draw renders the gameplay screen (scaled to fit)
func (gm *GameplayManager) Draw(screen *ebiten.Image) {
	if gm.emulator == nil {
		return
	}

	// Use emulator's DrawToScreen for rendering
	gm.emulator.DrawToScreen(screen, gm.cropBorder)

	// Check for pending achievement screenshot
	gm.achievementScreenshotMu.Lock()
	takeScreenshot := gm.achievementScreenshotPending
	gm.achievementScreenshotPending = false
	gm.achievementScreenshotMu.Unlock()

	if takeScreenshot && gm.screenshotManager != nil && gm.currentGame != nil {
		if err := gm.screenshotManager.TakeScreenshot(screen, gm.currentGame.CRC32); err != nil {
			log.Printf("Failed to take achievement screenshot: %v", err)
		}
	}
}

// DrawFramebuffer returns the native-resolution VDP framebuffer for xBR processing.
// When xBR is enabled, this provides the native image which xBR will upscale.
func (gm *GameplayManager) DrawFramebuffer() *ebiten.Image {
	if gm.emulator == nil {
		return nil
	}
	return gm.emulator.GetFramebufferImage(gm.cropBorder)
}

// DrawPauseMenu draws the pause menu overlay
func (gm *GameplayManager) DrawPauseMenu(screen *ebiten.Image) {
	gm.pauseMenu.Draw(screen)
}

// DrawAchievementOverlay draws the achievement overlay
func (gm *GameplayManager) DrawAchievementOverlay(screen *ebiten.Image) {
	gm.achievementOverlay.Draw(screen)
}

// IsPaused returns whether the pause menu is visible
func (gm *GameplayManager) IsPaused() bool {
	return gm.pauseMenu.IsVisible()
}

// Resume resumes gameplay after pause menu
func (gm *GameplayManager) Resume() {
	gm.pauseMenu.Hide()
	gm.playTime.trackStart = time.Now().Unix()
	gm.playTime.tracking = true
	gm.autoSaveTimer = time.Now().Add(gm.autoSaveInterval)
}

// Exit cleans up when exiting gameplay
func (gm *GameplayManager) Exit(saveResume bool) {
	if gm.emulator == nil {
		return
	}

	// Wait for any pending auto-save to complete (max 2 seconds)
	done := make(chan struct{})
	go func() {
		gm.autoSaveWg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// Auto-save completed
	case <-time.After(2 * time.Second):
		log.Printf("Warning: auto-save timed out on exit")
	}

	// Stop play time tracking and update
	gm.pausePlayTimeTracking()
	gm.updatePlayTime()

	// Save SRAM
	if err := gm.saveStateManager.SaveSRAM(gm.emulator); err != nil {
		log.Printf("SRAM save failed: %v", err)
	}

	// Save resume state if requested
	if saveResume {
		if err := gm.saveStateManager.SaveResume(gm.emulator); err != nil {
			log.Printf("Resume save failed: %v", err)
		}
	}

	// Free rewind buffer
	gm.rewindBuffer = nil

	// Close audio player
	if gm.audioPlayer != nil {
		gm.audioPlayer.Close()
		gm.audioPlayer = nil
	}

	// Reset achievement overlay and unload achievements
	gm.achievementOverlay.Reset()
	if gm.achievementManager != nil {
		gm.achievementManager.UnloadGame()
	}

	// Close emulator
	gm.emulator.Close()
	gm.emulator = nil
	gm.currentGame = nil

	// Reset TPS to 60 for UI
	ebiten.SetTPS(60)
}

// pollInput reads input and passes it to the emulator
func (gm *GameplayManager) pollInput() {
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

	gm.emulator.SetInput(up, down, left, right, btn1, btn2)

	// SMS Pause button (Enter key or Start button triggers NMI)
	smsPause := inpututil.IsKeyJustPressed(ebiten.KeyEnter)
	for _, id := range gamepadIDs {
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight) {
			smsPause = true
			break
		}
	}
	if smsPause {
		gm.emulator.SetPause()
	}
}

// handleSaveStateKeys handles F1/F2/F3 for save states
func (gm *GameplayManager) handleSaveStateKeys() {
	// F1 - Save to current slot
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		if err := gm.saveStateManager.Save(gm.emulator); err != nil {
			log.Printf("Save state failed: %v", err)
		}
	}

	// F2 - Next slot (Shift+F2 - Previous slot)
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			gm.saveStateManager.PreviousSlot()
		} else {
			gm.saveStateManager.NextSlot()
		}
	}

	// F3 - Load from current slot
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		if err := gm.saveStateManager.Load(gm.emulator); err != nil {
			log.Printf("Load state failed: %v", err)
		} else if gm.rewindBuffer != nil {
			gm.rewindBuffer.Reset()
		}
	}
}

// triggerAutoSave performs an auto-save
func (gm *GameplayManager) triggerAutoSave() {
	if gm.emulator == nil || gm.currentGame == nil || gm.autoSaving {
		return
	}

	gm.autoSaving = true
	gm.autoSaveWg.Add(1)
	go func() {
		defer gm.autoSaveWg.Done()
		defer func() { gm.autoSaving = false }()

		// Save resume state
		if err := gm.saveStateManager.SaveResume(gm.emulator); err != nil {
			log.Printf("Auto-save failed: %v", err)
		}

		// Save SRAM
		if err := gm.saveStateManager.SaveSRAM(gm.emulator); err != nil {
			log.Printf("SRAM save failed: %v", err)
		}

		// Update play time
		gm.updatePlayTime()
	}()
}

// pausePlayTimeTracking pauses the play time tracker
func (gm *GameplayManager) pausePlayTimeTracking() {
	if gm.playTime.tracking {
		elapsed := time.Now().Unix() - gm.playTime.trackStart
		gm.playTime.sessionSeconds += elapsed
		gm.playTime.tracking = false
	}
}

// updatePlayTime updates the play time in the library
func (gm *GameplayManager) updatePlayTime() {
	if gm.currentGame == nil {
		return
	}

	var totalSession int64
	if gm.playTime.tracking {
		elapsed := time.Now().Unix() - gm.playTime.trackStart
		totalSession = gm.playTime.sessionSeconds + elapsed
	} else {
		totalSession = gm.playTime.sessionSeconds
	}

	// Only update if there's actual play time
	if totalSession > 0 {
		gm.currentGame.PlayTimeSeconds += totalSession
		gm.playTime.sessionSeconds = 0
		if gm.playTime.tracking {
			gm.playTime.trackStart = time.Now().Unix()
		}
		storage.SaveLibrary(gm.library)
	}
}

// regionFromLibraryEntry determines the region from a library entry
func (gm *GameplayManager) regionFromLibraryEntry(game *storage.GameEntry) emu.Region {
	switch strings.ToLower(game.Region) {
	case "eu", "europe", "pal":
		return emu.RegionPAL
	default:
		return emu.RegionNTSC
	}
}
