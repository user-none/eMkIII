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

// ADT (audio-driven timing) buffer thresholds in bytes.
// At 48kHz stereo 16-bit: 3200 bytes/frame at 60fps.
const (
	adtMinBuffer = 9600  // ~3 frames — speed up below this
	adtMaxBuffer = 19200 // ~6 frames — slow down above this
)

// GameplayManager handles all gameplay-related state and logic.
// This includes emulator control, input handling, save states,
// play time tracking, and the pause menu.
//
// The emulator runs on a dedicated goroutine with audio-driven timing (ADT).
// The Ebiten thread handles UI, input polling, and reads the shared framebuffer.
type GameplayManager struct {
	// Emulation state
	emulator    *emubridge.Emulator
	audioPlayer *AudioPlayer
	currentGame *storage.GameEntry
	cropBorder  bool

	// ADT goroutine control
	emuControl        *EmuControl
	sharedInput       *SharedInput
	sharedFramebuffer *SharedFramebuffer
	emuDone           chan struct{}

	// Cached auto-save state (written by emu goroutine, read by Ebiten thread)
	autoSaveState   []byte
	autoSaveStateMu sync.Mutex
	autoSaveReady   bool

	// Rewind
	rewindBuffer *RewindBuffer

	// Pause menu
	pauseMenu *PauseMenu

	// Achievement overlay
	achievementOverlay *AchievementOverlay

	// Play time tracking
	playTime PlayTimeTracker

	// Auto-save state
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

	// Create shared structures for ADT
	gm.sharedInput = &SharedInput{}
	gm.sharedFramebuffer = NewSharedFramebuffer()
	gm.emuControl = NewEmuControl()
	gm.emuDone = make(chan struct{})

	// Always create audio player for ADT timing.
	// When muted, volume is set to 0 so the player still drains
	// the buffer (driving timing) but produces no audible output.
	volume := gm.config.Audio.Volume
	if gm.config.Audio.Muted {
		volume = 0
	}
	player, err := NewAudioPlayer(volume)
	if err != nil {
		log.Printf("Failed to init audio: %v", err)
	} else {
		gm.audioPlayer = player
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

	// Set TPS to 60 for all regions — emu goroutine handles its own timing via ADT
	ebiten.SetTPS(60)

	// Start play time tracking
	gm.playTime.sessionSeconds = 0
	gm.playTime.trackStart = time.Now().Unix()
	gm.playTime.tracking = true

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

	// Start the emulation goroutine
	go gm.emulationLoop()

	return true
}

// emulationLoop runs on a dedicated goroutine. It executes emulator frames,
// queues audio, updates the shared framebuffer, and paces itself using
// audio-driven timing (ADT).
func (gm *GameplayManager) emulationLoop() {
	defer close(gm.emuDone)

	timing := gm.emulator.GetTiming()
	frameTime := time.Duration(float64(time.Second) / float64(timing.FPS))
	lastFrameTime := time.Now()
	autoSaveTimer := time.Now().Add(time.Second) // First serialize after 1 second

	for {
		// Check pause/stop
		if !gm.emuControl.CheckPause() {
			return
		}

		// Read input from shared state
		up, down, left, right, btn1, btn2, smsPause := gm.sharedInput.Read()
		gm.emulator.SetInput(up, down, left, right, btn1, btn2)
		if smsPause {
			gm.emulator.SetPause()
		}

		// Run one frame of emulation
		gm.emulator.RunFrame()

		// Process RetroAchievements (reads emulator memory — must be same goroutine)
		if gm.achievementManager != nil {
			gm.achievementManager.DoFrame()
		}

		// Queue audio samples
		if gm.audioPlayer != nil {
			gm.audioPlayer.QueueSamples(gm.emulator.GetAudioSamples())
		}

		// Update shared framebuffer for Draw thread
		gm.sharedFramebuffer.Update(
			gm.emulator.GetFramebuffer(),
			gm.emulator.GetFramebufferStride(),
			gm.emulator.GetActiveHeight(),
			gm.emulator.LeftColumnBlankEnabled(),
		)

		// Capture rewind state (only when not rewinding)
		if gm.rewindBuffer != nil {
			gm.rewindBuffer.Capture(gm.emulator)
		}

		// Periodic auto-save: serialize state and cache for Ebiten thread to write to disk
		now := time.Now()
		if now.After(autoSaveTimer) {
			state, err := gm.emulator.Serialize()
			if err == nil {
				gm.autoSaveStateMu.Lock()
				gm.autoSaveState = state
				gm.autoSaveReady = true
				gm.autoSaveStateMu.Unlock()
			}
			autoSaveTimer = now.Add(gm.autoSaveInterval)
		}

		// ADT sleep: wall-clock baseline ± adjustment from audio buffer level
		elapsed := time.Since(lastFrameTime)
		sleepTime := frameTime - elapsed

		if gm.audioPlayer != nil {
			bufferLevel := gm.audioPlayer.GetBufferLevel()
			if bufferLevel < adtMinBuffer {
				sleepTime = time.Duration(float64(sleepTime) * 0.9)
			} else if bufferLevel > adtMaxBuffer {
				sleepTime = time.Duration(float64(sleepTime) * 1.1)
			}
		}

		if sleepTime > time.Millisecond {
			time.Sleep(sleepTime)
		}

		lastFrameTime = time.Now()
	}
}

// Update handles the gameplay update loop. Returns true if pause menu was opened.
// This runs on the Ebiten thread — it polls input and manages UI state.
// The emulator itself runs on a separate goroutine.
func (gm *GameplayManager) Update() (pauseMenuOpened bool, err error) {
	if gm.emulator == nil {
		return false, nil
	}

	// Check for Tab key to toggle achievement overlay
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) && !gm.pauseMenu.IsVisible() {
		if gm.achievementOverlay.IsVisible() {
			gm.achievementOverlay.Hide()
			gm.emuControl.RequestResume()
			gm.playTime.trackStart = time.Now().Unix()
			gm.playTime.tracking = true
		} else if gm.achievementManager != nil && gm.achievementManager.IsGameLoaded() {
			gm.emuControl.RequestPause()
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
		// Pause emulation goroutine, then open pause menu
		gm.emuControl.RequestPause()
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

	// Poll input and write to shared state (emu goroutine reads it)
	gm.pollInputToShared()

	// Check rewind input (R key)
	if gm.rewindBuffer != nil {
		holdDuration := inpututil.KeyPressDuration(ebiten.KeyR)
		if holdDuration > 0 {
			items := rewindItemsForHoldDuration(holdDuration)
			if items > 0 {
				if !gm.rewindBuffer.IsRewinding() {
					// Pause goroutine for rewind — we access emulator directly
					gm.emuControl.RequestPause()
					gm.rewindBuffer.SetRewinding(true)
					if gm.audioPlayer != nil {
						gm.audioPlayer.ClearQueue()
					}
				}
				gm.rewindBuffer.Rewind(gm.emulator, items)
				// Update shared framebuffer after rewind step
				gm.sharedFramebuffer.Update(
					gm.emulator.GetFramebuffer(),
					gm.emulator.GetFramebufferStride(),
					gm.emulator.GetActiveHeight(),
					gm.emulator.LeftColumnBlankEnabled(),
				)
				return false, nil
			}
			// items == 0 means we're in a hold gap frame; skip normal execution
			return false, nil
		} else if gm.rewindBuffer.IsRewinding() {
			// R released - resume emulation goroutine
			gm.rewindBuffer.SetRewinding(false)
			gm.emuControl.RequestResume()
		}
	}

	// Handle save state keys (pauses goroutine as needed)
	gm.handleSaveStateKeys()

	// Check for cached auto-save state from emu goroutine → write to disk
	gm.autoSaveStateMu.Lock()
	if gm.autoSaveReady && !gm.autoSaving {
		state := gm.autoSaveState
		gm.autoSaveReady = false
		gm.autoSaveStateMu.Unlock()
		gm.writeAutoSave(state)
	} else {
		gm.autoSaveStateMu.Unlock()
	}

	return false, nil
}

// Draw renders the gameplay screen from the shared framebuffer.
func (gm *GameplayManager) Draw(screen *ebiten.Image) {
	if gm.emulator == nil || gm.sharedFramebuffer == nil {
		return
	}

	pixels, stride, height, leftColumnBlank := gm.sharedFramebuffer.Read()
	if height == 0 {
		return
	}
	gm.emulator.DrawCachedFramebuffer(screen, pixels, stride, height, leftColumnBlank, gm.cropBorder)

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
// Reads from the shared framebuffer rather than directly from the emulator.
func (gm *GameplayManager) DrawFramebuffer() *ebiten.Image {
	if gm.emulator == nil || gm.sharedFramebuffer == nil {
		return nil
	}
	pixels, stride, height, leftColumnBlank := gm.sharedFramebuffer.Read()
	if height == 0 {
		return nil
	}
	return gm.emulator.GetCachedFramebufferImage(pixels, stride, height, leftColumnBlank, gm.cropBorder)
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
	gm.emuControl.RequestResume()
	gm.playTime.trackStart = time.Now().Unix()
	gm.playTime.tracking = true
}

// Exit cleans up when exiting gameplay
func (gm *GameplayManager) Exit(saveResume bool) {
	if gm.emulator == nil {
		return
	}

	// Stop emulation goroutine and wait for it to exit
	if gm.emuControl != nil {
		gm.emuControl.Stop()
		<-gm.emuDone
	}

	// Wait for any pending auto-save disk write to complete (max 2 seconds)
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

	// Save SRAM (goroutine is stopped, safe to access emulator directly)
	if err := gm.saveStateManager.SaveSRAM(gm.emulator); err != nil {
		log.Printf("SRAM save failed: %v", err)
	}

	// Save resume state if requested
	if saveResume {
		if err := gm.saveStateManager.SaveResume(gm.emulator); err != nil {
			log.Printf("Resume save failed: %v", err)
		}
	}

	// Free shared state
	gm.rewindBuffer = nil
	gm.sharedInput = nil
	gm.sharedFramebuffer = nil
	gm.emuControl = nil
	gm.autoSaveState = nil
	gm.autoSaveReady = false

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

// pollInputToShared reads input and writes it to the shared input state
// for the emulation goroutine to consume.
func (gm *GameplayManager) pollInputToShared() {
	// Keyboard (WASD + JK)
	up := ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp)
	down := ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown)
	left := ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
	right := ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight)
	btn1 := ebiten.IsKeyPressed(ebiten.KeyJ)
	btn2 := ebiten.IsKeyPressed(ebiten.KeyK)

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

	gm.sharedInput.Set(up, down, left, right, btn1, btn2)

	// SMS Pause button (Enter key or Start button triggers NMI)
	smsPause := inpututil.IsKeyJustPressed(ebiten.KeyEnter)
	for _, id := range gamepadIDs {
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight) {
			smsPause = true
			break
		}
	}
	if smsPause {
		gm.sharedInput.SetPause()
	}
}

// handleSaveStateKeys handles F1/F2/F3 for save states.
// Pauses the emulation goroutine for Save/Load operations.
func (gm *GameplayManager) handleSaveStateKeys() {
	// F1 - Save to current slot
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		gm.emuControl.RequestPause()
		if err := gm.saveStateManager.Save(gm.emulator); err != nil {
			log.Printf("Save state failed: %v", err)
		}
		gm.emuControl.RequestResume()
	}

	// F2 - Next slot (Shift+F2 - Previous slot) — no pause needed
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			gm.saveStateManager.PreviousSlot()
		} else {
			gm.saveStateManager.NextSlot()
		}
	}

	// F3 - Load from current slot
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		gm.emuControl.RequestPause()
		if err := gm.saveStateManager.Load(gm.emulator); err != nil {
			log.Printf("Load state failed: %v", err)
		} else {
			if gm.rewindBuffer != nil {
				gm.rewindBuffer.Reset()
			}
			// Update shared framebuffer after load
			gm.sharedFramebuffer.Update(
				gm.emulator.GetFramebuffer(),
				gm.emulator.GetFramebufferStride(),
				gm.emulator.GetActiveHeight(),
				gm.emulator.LeftColumnBlankEnabled(),
			)
			if gm.audioPlayer != nil {
				gm.audioPlayer.ClearQueue()
			}
		}
		gm.emuControl.RequestResume()
	}
}

// triggerAutoSave saves the emulator state to disk.
// When the goroutine is paused, serializes fresh. Otherwise uses cached state.
func (gm *GameplayManager) triggerAutoSave() {
	if gm.emulator == nil || gm.currentGame == nil || gm.autoSaving {
		return
	}

	var state []byte
	if gm.emuControl != nil && gm.emuControl.IsPaused() {
		// Goroutine is paused — safe to serialize fresh and save SRAM
		var err error
		state, err = gm.emulator.Serialize()
		if err != nil {
			log.Printf("Auto-save serialize failed: %v", err)
			return
		}
		// Also save SRAM since goroutine is paused
		if err := gm.saveStateManager.SaveSRAM(gm.emulator); err != nil {
			log.Printf("SRAM save failed: %v", err)
		}
	} else {
		// Use cached state from emu goroutine (no SRAM save — goroutine running)
		gm.autoSaveStateMu.Lock()
		state = gm.autoSaveState
		gm.autoSaveStateMu.Unlock()
	}

	if state == nil {
		return
	}

	gm.writeAutoSave(state)
}

// writeAutoSave writes pre-serialized state data to disk asynchronously.
func (gm *GameplayManager) writeAutoSave(state []byte) {
	gm.autoSaving = true
	gm.autoSaveWg.Add(1)
	go func() {
		defer gm.autoSaveWg.Done()
		defer func() { gm.autoSaving = false }()

		if err := gm.saveStateManager.SaveResumeData(state); err != nil {
			log.Printf("Auto-save failed: %v", err)
		}

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
