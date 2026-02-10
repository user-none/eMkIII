//go:build !libretro && !ios

package achievements

import (
	"fmt"
	"image"
	_ "image/png" // PNG decoder for badge images
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/user-none/go-rcheevos"
)

// Notification interface for showing achievement popups
type Notification interface {
	ShowDefault(message string)
	ShowAchievementWithBadge(title, description string, badge *ebiten.Image)
	SetBadge(badge *ebiten.Image) // Update badge after async fetch
	PlaySound(soundData []byte)   // Play sound through notification audio stream
}

// ScreenshotFunc is called when an achievement triggers to capture a screenshot
type ScreenshotFunc func()

// Manager wraps the rcheevos client for RetroAchievements integration
type Manager struct {
	client       *rcheevos.Client
	httpClient   *http.Client
	notification Notification
	userAgent    string // Cached User-Agent string

	// State
	mu         sync.Mutex
	emulator   EmulatorInterface
	loggedIn   bool
	username   string
	token      string
	gameLoaded bool
	enabled    bool

	// Callbacks for unlock events
	screenshotFunc   ScreenshotFunc
	screenshotEnable bool

	// Unlock sound
	unlockSoundData   []byte
	unlockSoundEnable bool

	// Suppress hardcore warning notification
	suppressHardcoreWarning bool

	// Badge cache (gameID<<32 | achievementID -> image)
	badgeCache map[uint64]*ebiten.Image
	// Game image cache (gameID -> image)
	gameImageCache map[uint32]*ebiten.Image
}

// NewManager creates a new achievement manager
func NewManager(notification Notification, appName, appVersion string) *Manager {
	m := &Manager{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		notification:    notification,
		unlockSoundData: generateUnlockSound(),
		badgeCache:      make(map[uint64]*ebiten.Image),
		gameImageCache:  make(map[uint32]*ebiten.Image),
	}

	// Create rcheevos client with memory and server callbacks
	m.client = rcheevos.NewClient(m.readMemory, m.serverCall)

	// Build User-Agent string: "AppName/Version rcheevos/X.X.X"
	rcheevosUA := m.client.GetUserAgentClause()
	m.userAgent = fmt.Sprintf("%s/%s %s", appName, appVersion, rcheevosUA)

	// Set up event handler
	m.client.SetEventHandler(m.handleEvent)

	return m
}

// IsLoggedIn returns whether a user is currently logged in
func (m *Manager) IsLoggedIn() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loggedIn
}

// IsEnabled returns whether achievements are enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// SetEnabled enables or disables achievement processing
// Called from settings UI on main thread, read during gameplay on main thread
func (m *Manager) SetEnabled(enabled bool) {
	m.enabled = enabled
}

// SetScreenshotFunc sets the callback for taking screenshots on achievement unlock
// Must be called before game loop starts (configure then run pattern)
func (m *Manager) SetScreenshotFunc(fn ScreenshotFunc) {
	m.screenshotFunc = fn
}

// SetScreenshotEnabled enables or disables auto-screenshot on unlock
// Must be called before game loop starts (configure then run pattern)
func (m *Manager) SetScreenshotEnabled(enabled bool) {
	m.screenshotEnable = enabled
}

// SetUnlockSoundEnabled enables or disables unlock sound
// Must be called before game loop starts (configure then run pattern)
func (m *Manager) SetUnlockSoundEnabled(enabled bool) {
	m.unlockSoundEnable = enabled
}

// SetSuppressHardcoreWarning enables or disables suppression of the hardcore warning
// Must be called before game loop starts (configure then run pattern)
func (m *Manager) SetSuppressHardcoreWarning(suppress bool) {
	m.suppressHardcoreWarning = suppress
}

// GetUsername returns the logged in username
func (m *Manager) GetUsername() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.username
}

// GetToken returns the stored auth token
func (m *Manager) GetToken() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.token
}

// Login authenticates with RetroAchievements using username and password
func (m *Manager) Login(username, password string, callback func(success bool, token string, err error)) {
	m.client.LoginWithPassword(username, password, func(result int, errorMessage string) {
		if result != rcheevos.OK {
			callback(false, "", fmt.Errorf("login failed: %s", errorMessage))
			return
		}

		user := m.client.GetUser()
		if user == nil {
			callback(false, "", fmt.Errorf("login succeeded but user info unavailable"))
			return
		}

		m.mu.Lock()
		m.loggedIn = true
		m.username = user.Username
		m.token = user.Token
		m.mu.Unlock()

		callback(true, user.Token, nil)
	})
}

// LoginWithToken authenticates with RetroAchievements using a stored token
func (m *Manager) LoginWithToken(username, token string, callback func(success bool, err error)) {
	m.client.LoginWithToken(username, token, func(result int, errorMessage string) {
		if result != rcheevos.OK {
			callback(false, fmt.Errorf("token login failed: %s", errorMessage))
			return
		}

		user := m.client.GetUser()
		if user == nil {
			callback(false, fmt.Errorf("login succeeded but user info unavailable"))
			return
		}

		m.mu.Lock()
		m.loggedIn = true
		m.username = user.Username
		m.token = user.Token
		m.mu.Unlock()

		callback(true, nil)
	})
}

// Logout logs out the current user
func (m *Manager) Logout() {
	m.client.Logout()

	m.mu.Lock()
	m.loggedIn = false
	m.username = ""
	m.token = ""
	m.mu.Unlock()
}

// SetEmulator sets the emulator for memory access
// Must be called before game loop starts (configure then run pattern)
func (m *Manager) SetEmulator(emu EmulatorInterface) {
	m.emulator = emu
}

// SetEncoreMode enables or disables encore mode (re-triggering unlocked achievements)
func (m *Manager) SetEncoreMode(enabled bool) {
	m.client.SetEncoreModeEnabled(enabled)
}

// LoadGame identifies and loads a game for achievement tracking
func (m *Manager) LoadGame(romData []byte, filePath string) error {
	m.mu.Lock()
	if !m.loggedIn {
		m.mu.Unlock()
		return fmt.Errorf("not logged in")
	}
	m.mu.Unlock()

	// Use a channel to capture the async result
	done := make(chan error, 1)

	m.client.IdentifyAndLoadGame(rcheevos.ConsoleMasterSystem, filePath, romData, func(result int, errorMessage string) {
		if result != rcheevos.OK {
			done <- fmt.Errorf("failed to load game: %s", errorMessage)
			return
		}

		m.mu.Lock()
		m.gameLoaded = true
		m.mu.Unlock()

		done <- nil
	})

	// Wait for the callback with a timeout
	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		return fmt.Errorf("game load timed out")
	}
}

// badgeCacheKey creates a composite cache key from game ID and achievement ID
func badgeCacheKey(gameID, achievementID uint32) uint64 {
	return (uint64(gameID) << 32) | uint64(achievementID)
}

// getBadge returns a cached badge or fetches it on-demand
func (m *Manager) getBadge(achievementID uint32, url string) *ebiten.Image {
	if url == "" {
		return nil
	}

	game := m.client.GetGame()
	if game == nil {
		return nil
	}
	cacheKey := badgeCacheKey(game.ID, achievementID)

	// Check cache first
	m.mu.Lock()
	if img, ok := m.badgeCache[cacheKey]; ok {
		m.mu.Unlock()
		return img
	}
	m.mu.Unlock()

	// Fetch on-demand
	img := m.fetchImage(url)
	if img != nil {
		m.mu.Lock()
		m.badgeCache[cacheKey] = img
		m.mu.Unlock()
	}
	return img
}

// getGameImage returns the cached game image or fetches it on-demand
func (m *Manager) getGameImage() *ebiten.Image {
	game := m.client.GetGame()
	if game == nil {
		return nil
	}

	// Check cache first
	m.mu.Lock()
	if img, ok := m.gameImageCache[game.ID]; ok {
		m.mu.Unlock()
		return img
	}
	m.mu.Unlock()

	url := m.client.GetGameImageURL()
	if url == "" {
		return nil
	}

	img := m.fetchImage(url)
	if img != nil {
		m.mu.Lock()
		m.gameImageCache[game.ID] = img
		m.mu.Unlock()
	}
	return img
}

// fetchImage downloads an image from a URL and returns an ebiten.Image
func (m *Manager) fetchImage(url string) *ebiten.Image {
	resp, err := m.httpClient.Get(url)
	if err != nil {
		log.Printf("[RetroAchievements] Failed to fetch image: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[RetroAchievements] Image fetch returned status %d", resp.StatusCode)
		return nil
	}

	// Limit read to 1MB
	limitedReader := io.LimitReader(resp.Body, 1024*1024)
	img, _, err := image.Decode(limitedReader)
	if err != nil {
		log.Printf("[RetroAchievements] Failed to decode image: %v", err)
		return nil
	}

	return ebiten.NewImageFromImage(img)
}

// DoFrame processes achievements for the current frame
func (m *Manager) DoFrame() {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	loggedIn := m.loggedIn
	gameLoaded := m.gameLoaded
	m.mu.Unlock()

	if !loggedIn || !gameLoaded {
		return
	}

	m.client.DoFrame()
}

// Idle processes periodic tasks when paused
func (m *Manager) Idle() {
	m.mu.Lock()
	loggedIn := m.loggedIn
	m.mu.Unlock()

	if !loggedIn {
		return
	}

	m.client.Idle()
}

// UnloadGame unloads the current game
func (m *Manager) UnloadGame() {
	m.mu.Lock()
	wasLoaded := m.gameLoaded
	m.gameLoaded = false
	m.emulator = nil
	m.mu.Unlock()

	if wasLoaded {
		m.client.UnloadGame()
	}
}

// Destroy cleans up the client resources
func (m *Manager) Destroy() {
	m.client.Destroy()
}

// readMemory is the memory callback for rcheevos
// SMS memory map for RetroAchievements:
// $0000-$1FFF: System RAM (8KB)
// $2000-$9FFF: Cart RAM (32KB)
// Note: emulator is set before game loop starts and doesn't change during gameplay
func (m *Manager) readMemory(address uint32, buffer []byte) uint32 {
	if m.emulator == nil {
		return 0
	}

	bytesRead := uint32(0)
	for i := range buffer {
		addr := address + uint32(i)
		var value uint8

		if addr < 0x2000 {
			// System RAM (8KB)
			ram := m.emulator.GetSystemRAM()
			if ram != nil {
				value = ram[addr]
			}
		} else if addr < 0xA000 {
			// Cart RAM (32KB) at offset 0x2000
			cartRAM := m.emulator.GetCartRAM()
			if cartRAM != nil {
				offset := addr - 0x2000
				if offset < 0x8000 {
					value = cartRAM[offset]
				}
			}
		} else {
			// Out of mapped range
			value = 0
		}

		buffer[i] = value
		bytesRead++
	}

	return bytesRead
}

// serverCall handles HTTP requests to the RetroAchievements API
func (m *Manager) serverCall(request *rcheevos.ServerRequest) {
	go func() {
		var resp *http.Response
		var err error

		// Create request with User-Agent header
		var req *http.Request
		if request.PostData != "" {
			// POST request
			req, err = http.NewRequest("POST", request.URL, strings.NewReader(request.PostData))
			if err == nil {
				req.Header.Set("Content-Type", request.ContentType)
			}
		} else {
			// GET request
			req, err = http.NewRequest("GET", request.URL, nil)
		}

		if err != nil {
			log.Printf("[RetroAchievements] Failed to create request: %v", err)
			request.Respond(nil, 0)
			return
		}

		req.Header.Set("User-Agent", m.userAgent)
		resp, err = m.httpClient.Do(req)

		if err != nil {
			log.Printf("[RetroAchievements] HTTP error: %v", err)
			request.Respond(nil, 0)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[RetroAchievements] Read error: %v", err)
			request.Respond(nil, resp.StatusCode)
			return
		}

		request.Respond(body, resp.StatusCode)
	}()
}

// handleEvent processes achievement events
func (m *Manager) handleEvent(event *rcheevos.Event) {
	switch event.Type {
	case rcheevos.EventAchievementTriggered:
		if event.Achievement == nil || m.notification == nil {
			return
		}

		// Copy values we need (event may become invalid after handler returns)
		title := event.Achievement.Title
		description := event.Achievement.Description
		achievementID := event.Achievement.ID
		badgeURL := m.client.GetAchievementImageURL(event.Achievement, rcheevos.AchievementStateUnlocked)

		// Check if this is the hardcore warning and should be suppressed
		isHardcoreWarning := strings.Contains(title, "Unknown Emulator") ||
			strings.Contains(description, "Hardcore unlocks cannot be earned")
		if m.suppressHardcoreWarning && isHardcoreWarning {
			return
		}

		// Get cached badge
		game := m.client.GetGame()
		var cachedBadge *ebiten.Image
		if game != nil {
			m.mu.Lock()
			cachedBadge = m.badgeCache[badgeCacheKey(game.ID, achievementID)]
			m.mu.Unlock()
		}

		// Play unlock sound
		if m.unlockSoundEnable && len(m.unlockSoundData) > 0 {
			m.notification.PlaySound(m.unlockSoundData)
		}

		// Take screenshot
		if m.screenshotEnable && m.screenshotFunc != nil {
			m.screenshotFunc()
		}

		// Show notification
		if cachedBadge != nil {
			m.notification.ShowAchievementWithBadge(title, description, cachedBadge)
		} else {
			// Show notification immediately without badge, fetch async
			m.notification.ShowAchievementWithBadge(title, description, nil)

			// Fetch badge in background and update notification
			go func() {
				badge := m.getBadge(achievementID, badgeURL)
				if badge != nil {
					m.notification.SetBadge(badge)
				}
			}()
		}
	case rcheevos.EventGameCompleted:
		if m.notification != nil {
			// Check cache first
			game := m.client.GetGame()
			var cachedImg *ebiten.Image
			if game != nil {
				m.mu.Lock()
				cachedImg = m.gameImageCache[game.ID]
				m.mu.Unlock()
			}

			if cachedImg != nil {
				m.notification.ShowAchievementWithBadge("Game Mastered!", "All achievements unlocked", cachedImg)
			} else {
				// Show immediately, fetch image async
				m.notification.ShowAchievementWithBadge("Game Mastered!", "All achievements unlocked", nil)
				go func() {
					gameImg := m.getGameImage()
					if gameImg != nil {
						m.notification.SetBadge(gameImg)
					}
				}()
			}
		}
	case rcheevos.EventServerError:
		if event.ServerError != nil {
			log.Printf("[RetroAchievements] Server error: %s", event.ServerError.ErrorMessage)
		}
	}
}
