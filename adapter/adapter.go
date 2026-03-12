package adapter

import (
	"github.com/user-none/eblitui/coreif"
	"github.com/user-none/emkiii/core"
)

// Factory implements CoreFactory for the SMS emulator.
type Factory struct{}

// SystemInfo returns system metadata for UI configuration.
func (f *Factory) SystemInfo() coreif.SystemInfo {
	return coreif.SystemInfo{
		Name:            "emkiii",
		ConsoleName:     "Sega Master System",
		Extensions:      []string{".sms"},
		ScreenWidth:     core.ScreenWidth,
		MaxScreenHeight: core.MaxScreenHeight,
		// NTSC pixel aspect ratio for SMS (8:7).
		// The SMS master clock is 10.738635 MHz. The pixel clock is
		// master/2 and 256 active pixels span the same active line time
		// as the Genesis (both VDPs share the same timing lineage).
		// SMS: 256 pixels at 5.369318 MHz = 47.68 us active time.
		// This is identical to Genesis H40 (2560 master clocks at
		// 53.693175 MHz = 47.68 us). Since SMS has 256 pixels in the
		// same active time that Genesis H40 has 320, each SMS pixel is
		// 320/256 = 5/4 wider:
		// PAR = (32/35) * (5/4) = 8/7
		// The PAL master clock differs by <1%, so this value is used
		// for both NTSC and PAL.
		PixelAspectRatio: 8.0 / 7.0,
		SampleRate:       48000,
		Buttons: []coreif.Button{
			{Name: "1", ID: 4, DefaultKey: "J", DefaultPad: "A"},
			{Name: "2", ID: 5, DefaultKey: "K", DefaultPad: "B"},
			{Name: "Start", ID: 7, DefaultKey: "Enter", DefaultPad: "Start"},
		},
		Players: 2,
		CoreOptions: []coreif.CoreOption{
			{
				Key:         "crop_border",
				Label:       "Crop Left Border",
				Description: "Crop blank left column when enabled by game",
				Type:        coreif.CoreOptionBool,
				Default:     "false",
				Category:    coreif.CoreOptionCategoryVideo,
			},
		},
		MetadataVariants: []coreif.MetadataVariant{
			{Name: "Master System", RDBName: "Sega - Master System - Mark III", ThumbnailRepo: "Sega_-_Master_System_-_Mark_III"},
		},
		DataDirName:   "emkiii",
		ConsoleID:     2,
		CoreName:      core.Name,
		CoreVersion:   core.Version,
		SerializeSize: core.SerializeSize(),
	}
}

// CreateEmulator creates a new emulator instance with the given ROM and region.
func (f *Factory) CreateEmulator(rom []byte, region coreif.Region) (coreif.Emulator, error) {
	e, err := core.NewEmulator(rom, region)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// DetectRegion auto-detects the region from ROM data.
// The bool return indicates whether the region was found in the database.
func (f *Factory) DetectRegion(rom []byte) (coreif.Region, bool) {
	return core.DetectRegionFromROM(rom)
}
