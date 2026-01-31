package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Version != 1 {
		t.Errorf("expected version 1, got %d", config.Version)
	}
	if config.Window.Width != 800 {
		t.Errorf("expected window width 800, got %d", config.Window.Width)
	}
	if config.Window.Height != 600 {
		t.Errorf("expected window height 600, got %d", config.Window.Height)
	}
	if config.Audio.Volume != 1.0 {
		t.Errorf("expected volume 1.0, got %f", config.Audio.Volume)
	}
	if config.Library.ViewMode != "icon" {
		t.Errorf("expected view mode 'icon', got '%s'", config.Library.ViewMode)
	}
}

func TestDefaultLibrary(t *testing.T) {
	lib := DefaultLibrary()

	if lib.Version != 1 {
		t.Errorf("expected version 1, got %d", lib.Version)
	}
	if len(lib.Games) != 0 {
		t.Errorf("expected empty games map, got %d entries", len(lib.Games))
	}
	if len(lib.ScanDirectories) != 0 {
		t.Errorf("expected empty scan directories, got %d entries", len(lib.ScanDirectories))
	}
}

func TestAtomicWriteJSON(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "test.json")

	data := struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}{
		Name:  "test",
		Value: 42,
	}

	// Write file
	if err := AtomicWriteJSON(path, data); err != nil {
		t.Fatalf("AtomicWriteJSON failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// Read back
	var result struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	if err := ReadJSON(path, &result); err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}

	if result.Name != data.Name || result.Value != data.Value {
		t.Errorf("data mismatch: expected %+v, got %+v", data, result)
	}

	// Verify temp file is cleaned up
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file was not cleaned up")
	}
}

func TestLibraryAddGetRemoveGame(t *testing.T) {
	lib := DefaultLibrary()

	game := &GameEntry{
		CRC32:       "12345678",
		File:        "/path/to/game.sms",
		DisplayName: "Test Game",
		Region:      "us",
	}

	// Add game
	lib.AddGame(game)

	if lib.GameCount() != 1 {
		t.Errorf("expected 1 game, got %d", lib.GameCount())
	}

	// Get game
	retrieved := lib.GetGame("12345678")
	if retrieved == nil {
		t.Fatal("game not found")
	}
	if retrieved.DisplayName != "Test Game" {
		t.Errorf("expected 'Test Game', got '%s'", retrieved.DisplayName)
	}

	// Remove game
	lib.RemoveGame("12345678")
	if lib.GameCount() != 0 {
		t.Errorf("expected 0 games after removal, got %d", lib.GameCount())
	}
}

func TestLibrarySorting(t *testing.T) {
	lib := DefaultLibrary()

	lib.AddGame(&GameEntry{CRC32: "1", DisplayName: "Zelda", PlayTimeSeconds: 100, LastPlayed: 1000})
	lib.AddGame(&GameEntry{CRC32: "2", DisplayName: "Alex Kidd", PlayTimeSeconds: 500, LastPlayed: 500})
	lib.AddGame(&GameEntry{CRC32: "3", DisplayName: "Sonic", PlayTimeSeconds: 300, LastPlayed: 2000})

	// Sort by title
	games := lib.GetGamesSorted("title", false)
	if len(games) != 3 {
		t.Fatalf("expected 3 games, got %d", len(games))
	}
	if games[0].DisplayName != "Alex Kidd" {
		t.Errorf("expected first game 'Alex Kidd', got '%s'", games[0].DisplayName)
	}
	if games[2].DisplayName != "Zelda" {
		t.Errorf("expected last game 'Zelda', got '%s'", games[2].DisplayName)
	}

	// Sort by play time
	games = lib.GetGamesSorted("playTime", false)
	if games[0].DisplayName != "Alex Kidd" { // Most played (500s)
		t.Errorf("expected most played 'Alex Kidd', got '%s'", games[0].DisplayName)
	}

	// Sort by last played
	games = lib.GetGamesSorted("lastPlayed", false)
	if games[0].DisplayName != "Sonic" { // Most recent (2000)
		t.Errorf("expected most recent 'Sonic', got '%s'", games[0].DisplayName)
	}
}

func TestLibrarySortingStability(t *testing.T) {
	// Test that sorting is stable when primary sort values are equal.
	// Games with the same display name should be sorted by region, then by
	// No-Intro name (to distinguish revisions), then by CRC32.
	lib := DefaultLibrary()

	// Add games with same display name but different regions and revisions
	lib.AddGame(&GameEntry{
		CRC32:       "C",
		DisplayName: "Zillion",
		Name:        "Zillion (Japan) (Rev 2)",
		Region:      "jp",
	})
	lib.AddGame(&GameEntry{
		CRC32:       "A",
		DisplayName: "Zillion",
		Name:        "Zillion (USA)",
		Region:      "us",
	})
	lib.AddGame(&GameEntry{
		CRC32:       "B",
		DisplayName: "Zillion",
		Name:        "Zillion (Europe)",
		Region:      "eu",
	})
	lib.AddGame(&GameEntry{
		CRC32:       "D",
		DisplayName: "Zillion",
		Name:        "Zillion (Japan) (Rev 1)",
		Region:      "jp",
	})
	lib.AddGame(&GameEntry{
		CRC32:       "E",
		DisplayName: "Alex Kidd",
		Name:        "Alex Kidd (USA)",
		Region:      "us",
	})

	// Sort by title multiple times and verify order is consistent
	for i := 0; i < 5; i++ {
		games := lib.GetGamesSorted("title", false)
		if len(games) != 5 {
			t.Fatalf("expected 5 games, got %d", len(games))
		}
		// Alex Kidd should be first (alphabetically)
		if games[0].DisplayName != "Alex Kidd" {
			t.Errorf("iteration %d: expected first game 'Alex Kidd', got '%s'", i, games[0].DisplayName)
		}
		// Zillion games should be sorted by region (eu, jp, us), then by Name
		// EU version
		if games[1].Region != "eu" {
			t.Errorf("iteration %d: expected second game region 'eu', got '%s'", i, games[1].Region)
		}
		// JP versions (Rev 1 before Rev 2 alphabetically)
		if games[2].Name != "Zillion (Japan) (Rev 1)" {
			t.Errorf("iteration %d: expected third game 'Zillion (Japan) (Rev 1)', got '%s'", i, games[2].Name)
		}
		if games[3].Name != "Zillion (Japan) (Rev 2)" {
			t.Errorf("iteration %d: expected fourth game 'Zillion (Japan) (Rev 2)', got '%s'", i, games[3].Name)
		}
		// US version
		if games[4].Region != "us" {
			t.Errorf("iteration %d: expected fifth game region 'us', got '%s'", i, games[4].Region)
		}
	}

	// Test with lastPlayed - games with same timestamp should have stable order
	lib2 := DefaultLibrary()
	lib2.AddGame(&GameEntry{CRC32: "C", DisplayName: "Game C", Name: "Game C (JP)", Region: "jp", LastPlayed: 1000})
	lib2.AddGame(&GameEntry{CRC32: "A", DisplayName: "Game A", Name: "Game A (US)", Region: "us", LastPlayed: 1000})
	lib2.AddGame(&GameEntry{CRC32: "B", DisplayName: "Game B", Name: "Game B (EU)", Region: "eu", LastPlayed: 1000})

	for i := 0; i < 5; i++ {
		games := lib2.GetGamesSorted("lastPlayed", false)
		// With equal lastPlayed, should fall back to title order (alphabetical)
		if games[0].DisplayName != "Game A" {
			t.Errorf("lastPlayed iteration %d: expected first 'Game A', got '%s'", i, games[0].DisplayName)
		}
		if games[1].DisplayName != "Game B" {
			t.Errorf("lastPlayed iteration %d: expected second 'Game B', got '%s'", i, games[1].DisplayName)
		}
		if games[2].DisplayName != "Game C" {
			t.Errorf("lastPlayed iteration %d: expected third 'Game C', got '%s'", i, games[2].DisplayName)
		}
	}

	// Test with playTime - games with same play time should have stable order
	lib3 := DefaultLibrary()
	lib3.AddGame(&GameEntry{CRC32: "C", DisplayName: "Game C", Name: "Game C (JP)", Region: "jp", PlayTimeSeconds: 100})
	lib3.AddGame(&GameEntry{CRC32: "A", DisplayName: "Game A", Name: "Game A (US)", Region: "us", PlayTimeSeconds: 100})
	lib3.AddGame(&GameEntry{CRC32: "B", DisplayName: "Game B", Name: "Game B (EU)", Region: "eu", PlayTimeSeconds: 100})

	for i := 0; i < 5; i++ {
		games := lib3.GetGamesSorted("playTime", false)
		// With equal playTime, should fall back to title order (alphabetical)
		if games[0].DisplayName != "Game A" {
			t.Errorf("playTime iteration %d: expected first 'Game A', got '%s'", i, games[0].DisplayName)
		}
		if games[1].DisplayName != "Game B" {
			t.Errorf("playTime iteration %d: expected second 'Game B', got '%s'", i, games[1].DisplayName)
		}
		if games[2].DisplayName != "Game C" {
			t.Errorf("playTime iteration %d: expected third 'Game C', got '%s'", i, games[2].DisplayName)
		}
	}
}

func TestLibraryFavoritesFilter(t *testing.T) {
	lib := DefaultLibrary()

	lib.AddGame(&GameEntry{CRC32: "1", DisplayName: "Game1", Favorite: true})
	lib.AddGame(&GameEntry{CRC32: "2", DisplayName: "Game2", Favorite: false})
	lib.AddGame(&GameEntry{CRC32: "3", DisplayName: "Game3", Favorite: true})

	// All games
	all := lib.GetGamesSorted("title", false)
	if len(all) != 3 {
		t.Errorf("expected 3 games, got %d", len(all))
	}

	// Favorites only
	favorites := lib.GetGamesSorted("title", true)
	if len(favorites) != 2 {
		t.Errorf("expected 2 favorites, got %d", len(favorites))
	}
}

func TestLibraryScanDirectories(t *testing.T) {
	lib := DefaultLibrary()

	lib.AddScanDirectory("/path/to/roms", true)
	lib.AddScanDirectory("/path/to/more", false)

	if len(lib.ScanDirectories) != 2 {
		t.Errorf("expected 2 directories, got %d", len(lib.ScanDirectories))
	}

	// Add duplicate (should be ignored)
	lib.AddScanDirectory("/path/to/roms", false)
	if len(lib.ScanDirectories) != 2 {
		t.Errorf("duplicate should be ignored, got %d directories", len(lib.ScanDirectories))
	}

	// Remove directory
	lib.RemoveScanDirectory("/path/to/roms")
	if len(lib.ScanDirectories) != 1 {
		t.Errorf("expected 1 directory after removal, got %d", len(lib.ScanDirectories))
	}
}

func TestLibraryExcludedPaths(t *testing.T) {
	lib := DefaultLibrary()

	lib.AddExcludedPath("/path/to/exclude")
	lib.AddExcludedPath("/path/to/file.sms")

	if len(lib.ExcludedPaths) != 2 {
		t.Errorf("expected 2 excluded paths, got %d", len(lib.ExcludedPaths))
	}

	// Test path exclusion
	if !lib.IsPathExcluded("/path/to/exclude") {
		t.Error("directory should be excluded")
	}
	if !lib.IsPathExcluded("/path/to/exclude/subdir/file.sms") {
		t.Error("subdirectory should be excluded")
	}
	if lib.IsPathExcluded("/path/to/other") {
		t.Error("/path/to/other should not be excluded")
	}

	// Remove excluded path
	lib.RemoveExcludedPath("/path/to/exclude")
	if len(lib.ExcludedPaths) != 1 {
		t.Errorf("expected 1 excluded path after removal, got %d", len(lib.ExcludedPaths))
	}
}

func TestConfigMigration(t *testing.T) {
	// Test migration from version 0
	config := &Config{
		Version: 0,
		Audio:   AudioConfig{Volume: 0}, // Will be set to 1.0
		Window:  WindowConfig{},
		Library: LibraryView{},
	}

	migrated := migrateConfig(config)

	if migrated.Version != 1 {
		t.Errorf("expected version 1 after migration, got %d", migrated.Version)
	}
	if migrated.Audio.Volume != 1.0 {
		t.Errorf("expected volume 1.0 after migration, got %f", migrated.Audio.Volume)
	}
	if migrated.Window.Width != 800 {
		t.Errorf("expected width 800 after migration, got %d", migrated.Window.Width)
	}
	if migrated.Library.ViewMode != "icon" {
		t.Errorf("expected view mode 'icon' after migration, got '%s'", migrated.Library.ViewMode)
	}
}

func TestUpdatePlayTime(t *testing.T) {
	lib := DefaultLibrary()

	lib.AddGame(&GameEntry{
		CRC32:           "12345678",
		DisplayName:     "Test Game",
		PlayTimeSeconds: 100,
	})

	lib.UpdatePlayTime("12345678", 50)

	game := lib.GetGame("12345678")
	if game.PlayTimeSeconds != 150 {
		t.Errorf("expected 150 seconds, got %d", game.PlayTimeSeconds)
	}
	if game.LastPlayed == 0 {
		t.Error("LastPlayed should be updated")
	}
}
