package storage

import (
	"errors"
	"os"
)

// LoadConfig loads the configuration from config.json.
// If the file doesn't exist, it returns default configuration.
// If the file is corrupted, it returns an error.
func LoadConfig() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if file exists
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		// File doesn't exist, return defaults
		return DefaultConfig(), nil
	}

	// Load and parse the file
	config := &Config{}
	if err := ReadJSON(path, config); err != nil {
		return nil, err
	}

	// Apply any migration for older config versions
	config = migrateConfig(config)

	return config, nil
}

// SaveConfig saves the configuration to config.json atomically
func SaveConfig(config *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	return AtomicWriteJSON(path, config)
}

// CreateConfigIfMissing creates a default config.json if it doesn't exist
func CreateConfigIfMissing() error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Check if file exists
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		// Create default config
		return SaveConfig(DefaultConfig())
	}

	return nil
}

// DeleteConfig removes the config.json file
func DeleteConfig() error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}

// migrateConfig handles any necessary migrations from older config versions
func migrateConfig(config *Config) *Config {
	// Currently at version 1, no migrations needed
	if config.Version == 0 {
		config.Version = 1
	}

	// Ensure defaults for any missing fields
	if config.Audio.Volume == 0 {
		config.Audio.Volume = 1.0
	}
	if config.Window.Width == 0 {
		config.Window.Width = 900
	}
	if config.Window.Height == 0 {
		config.Window.Height = 650
	}
	if config.Library.ViewMode == "" {
		config.Library.ViewMode = "icon"
	}
	if config.Library.SortBy == "" {
		config.Library.SortBy = "title"
	}
	if config.Theme == "" {
		config.Theme = "Default"
	}

	// Rewind defaults for existing configs
	if config.Rewind.BufferSizeMB == 0 {
		config.Rewind.BufferSizeMB = 40
	}
	if config.Rewind.FrameStep == 0 {
		config.Rewind.FrameStep = 1
	}

	return config
}
