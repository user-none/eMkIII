package emu

import "hash/crc32"

// Region represents the console region (NTSC or PAL)
type Region int

const (
	RegionNTSC Region = iota
	RegionPAL
)

func (r Region) String() string {
	switch r {
	case RegionNTSC:
		return "NTSC"
	case RegionPAL:
		return "PAL"
	default:
		return "Unknown"
	}
}

// RegionTiming holds timing constants for a specific region
type RegionTiming struct {
	CPUClockHz int // Z80 clock frequency
	Scanlines  int // Total scanlines per frame
	FPS        int // Frames per second
}

// NTSC timing: 3.579545 MHz, 262 scanlines, 60 Hz
var NTSCTiming = RegionTiming{
	CPUClockHz: 3579545,
	Scanlines:  262,
	FPS:        60,
}

// PAL timing: 3.546893 MHz, 313 scanlines, 50 Hz
var PALTiming = RegionTiming{
	CPUClockHz: 3546893,
	Scanlines:  313,
	FPS:        50,
}

// GetTimingForRegion returns the appropriate timing constants
func GetTimingForRegion(r Region) RegionTiming {
	if r == RegionPAL {
		return PALTiming
	}
	return NTSCTiming
}

// DefaultRegion returns the default region (NTSC).
// SMS ROM headers don't distinguish PAL from NTSC for export regions,
// so use the --region flag to specify PAL games.
func DefaultRegion() Region {
	return RegionNTSC
}

// DetectRegionFromROM returns the region for a ROM based on CRC32 lookup.
// Returns (detected region, true) if found in database, (NTSC, false) if not found.
func DetectRegionFromROM(rom []byte) (Region, bool) {
	crc := crc32.ChecksumIEEE(rom)
	if info, ok := romDatabase[crc]; ok {
		return info.Region, true
	}
	return RegionNTSC, false
}

// Nationality represents the console nationality (Japanese or Export).
// This is orthogonal to Region (NTSC/PAL): Japanese is always NTSC,
// but Export can be either NTSC (Americas) or PAL (Europe).
type Nationality int

const (
	NationalityExport Nationality = iota // Default
	NationalityJapanese
)

func (n Nationality) String() string {
	switch n {
	case NationalityExport:
		return "Export"
	case NationalityJapanese:
		return "Japanese"
	default:
		return "Unknown"
	}
}

// DetectNationalityFromROM reads the ROM header to determine nationality.
// Returns Export if the header is missing or unrecognizable.
func DetectNationalityFromROM(rom []byte) Nationality {
	// Header is at $7FF0; need at least $8000 bytes
	if len(rom) < 0x8000 {
		return NationalityExport
	}

	// Check for "TMR SEGA" signature at $7FF0
	if string(rom[0x7FF0:0x7FF8]) != "TMR SEGA" {
		return NationalityExport
	}

	// Region code is upper nibble of $7FFF
	regionCode := rom[0x7FFF] >> 4
	if regionCode == 3 { // SMS Japan
		return NationalityJapanese
	}
	return NationalityExport
}
