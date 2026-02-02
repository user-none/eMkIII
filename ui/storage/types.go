package storage

// Config represents the application configuration stored in config.json
type Config struct {
	Version int          `json:"version"`
	Theme   string       `json:"theme"` // Theme name: "Default", "Dark", "Light", "Retro"
	Video   VideoConfig  `json:"video"`
	Audio   AudioConfig  `json:"audio"`
	Window  WindowConfig `json:"window"`
	Library LibraryView  `json:"library"`
}

// VideoConfig contains video-related settings
type VideoConfig struct {
	CropBorder bool `json:"cropBorder"`
}

// AudioConfig contains audio-related settings
type AudioConfig struct {
	Volume float64 `json:"volume"`
	Muted  bool    `json:"muted"`
}

// WindowConfig contains window position and size
type WindowConfig struct {
	Width  int  `json:"width"`
	Height int  `json:"height"`
	X      *int `json:"x,omitempty"` // nil = OS decides position
	Y      *int `json:"y,omitempty"`
}

// LibraryView contains library display preferences
type LibraryView struct {
	ViewMode        string `json:"viewMode"`        // "icon" or "list"
	SortBy          string `json:"sortBy"`          // "title", "lastPlayed", "playTime"
	FavoritesFilter bool   `json:"favoritesFilter"` // Show only favorites
}

// Library represents the game library stored in library.json
type Library struct {
	Version         int                   `json:"version"`
	ScanDirectories []ScanDirectory       `json:"scanDirectories"`
	ExcludedPaths   []string              `json:"excludedPaths"`
	Games           map[string]*GameEntry `json:"games"` // CRC32 hex string -> entry
}

// ScanDirectory represents a directory to scan for ROMs
type ScanDirectory struct {
	Path      string `json:"path"`
	Recursive bool   `json:"recursive"`
}

// GameEntry represents a single game in the library
type GameEntry struct {
	CRC32           string       `json:"crc32"`
	File            string       `json:"file"`        // Path to ROM file or archive on disk
	Name            string       `json:"name"`        // Full No-Intro name from RDB
	DisplayName     string       `json:"displayName"` // Cleaned name for display (region info removed)
	Region          string       `json:"region"`      // "us", "eu", "jp" (from RDB)
	Developer       string       `json:"developer,omitempty"`
	Publisher       string       `json:"publisher,omitempty"`
	Genre           string       `json:"genre,omitempty"`
	Franchise       string       `json:"franchise,omitempty"`
	ESRBRating      string       `json:"esrbRating,omitempty"`
	ReleaseDate     string       `json:"releaseDate,omitempty"` // "Month / Year" format
	Favorite        bool         `json:"favorite"`              // User marked as favorite
	Missing         bool         `json:"missing"`               // true if ROM file not found
	PlayTimeSeconds int64        `json:"playTimeSeconds"`       // Total play time
	LastPlayed      int64        `json:"lastPlayed"`            // Unix timestamp
	Added           int64        `json:"added"`                 // Unix timestamp when added to library
	Settings        GameSettings `json:"settings"`              // Per-game settings
}

// GameSettings contains per-game configuration overrides
type GameSettings struct {
	RegionOverride string `json:"regionOverride,omitempty"` // "", "ntsc", "pal"
	CropBorder     *bool  `json:"cropBorder,omitempty"`     // nil = use global setting
	SaveSlot       int    `json:"saveSlot,omitempty"`       // Last-used save state slot (0-9)
}

// DefaultConfig returns a new Config with default values
func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		Theme:   "Default",
		Video: VideoConfig{
			CropBorder: false,
		},
		Audio: AudioConfig{
			Volume: 1.0,
			Muted:  false,
		},
		Window: WindowConfig{
			Width:  900,
			Height: 650,
			X:      nil,
			Y:      nil,
		},
		Library: LibraryView{
			ViewMode:        "icon",
			SortBy:          "title",
			FavoritesFilter: false,
		},
	}
}

// DefaultLibrary returns a new Library with default values
func DefaultLibrary() *Library {
	return &Library{
		Version:         1,
		ScanDirectories: []ScanDirectory{},
		ExcludedPaths:   []string{},
		Games:           make(map[string]*GameEntry),
	}
}
