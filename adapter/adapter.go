package adapter

import (
	emucore "github.com/user-none/eblitui/api"
	"github.com/user-none/emkiii/emu"
)

// Compile-time interface check.
var _ emucore.CoreFactory = (*Factory)(nil)

// Factory implements emucore.CoreFactory for the SMS emulator.
type Factory struct{}

// SystemInfo returns system metadata for UI configuration.
func (f *Factory) SystemInfo() emucore.SystemInfo {
	return emucore.SystemInfo{
		Name:            "emkiii",
		ConsoleName:     "Sega Master System",
		Extensions:      []string{".sms"},
		ScreenWidth:     emu.ScreenWidth,
		MaxScreenHeight: emu.MaxScreenHeight,
		AspectRatio:     256.0 / 192.0,
		SampleRate:      48000,
		Buttons: []emucore.Button{
			{Name: "1", ID: 4, DefaultKey: "J", DefaultPad: "A"},
			{Name: "2", ID: 5, DefaultKey: "K", DefaultPad: "B"},
			{Name: "Start", ID: 7, DefaultKey: "Enter", DefaultPad: "Start"},
		},
		Players: 2,
		CoreOptions: []emucore.CoreOption{
			{
				Key:         "crop_border",
				Label:       "Crop Left Border",
				Description: "Crop blank left column when enabled by game",
				Type:        emucore.CoreOptionBool,
				Default:     "false",
				Category:    emucore.CoreOptionCategoryVideo,
			},
		},
		RDBName:       "Sega - Master System - Mark III",
		ThumbnailRepo: "Sega_-_Master_System_-_Mark_III",
		DataDirName:   "emkiii",
		ConsoleID:     2,
		CoreName:      emu.Name,
		CoreVersion:   emu.Version,
		SerializeSize: emu.SerializeSize(),
	}
}

// CreateEmulator creates a new emulator instance with the given ROM and region.
func (f *Factory) CreateEmulator(rom []byte, region emucore.Region) (emucore.Emulator, error) {
	e, err := emu.NewEmulator(rom, region)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// DetectRegion auto-detects the region from ROM data.
// The bool return indicates whether the region was found in the database.
func (f *Factory) DetectRegion(rom []byte) (emucore.Region, bool) {
	return emu.DetectRegionFromROM(rom)
}
