package core

import "hash/crc32"

// VideoStandard represents the video standard (NTSC or PAL).
type VideoStandard int

const (
	VideoNTSC VideoStandard = iota
	VideoPAL
)

// VideoTiming holds timing constants for a specific video standard.
type VideoTiming struct {
	CPUClockHz int // Z80 clock frequency
	Scanlines  int // Total scanlines per frame
	FPS        int // Frames per second
}

// NTSCTiming: 3.579545 MHz, 262 scanlines, 60 Hz
var NTSCTiming = VideoTiming{
	CPUClockHz: 3579545,
	Scanlines:  262,
	FPS:        60,
}

// PALTiming: 3.546893 MHz, 313 scanlines, 50 Hz
var PALTiming = VideoTiming{
	CPUClockHz: 3546893,
	Scanlines:  313,
	FPS:        50,
}

// GetVideoTiming returns the appropriate timing constants.
func GetVideoTiming(v VideoStandard) VideoTiming {
	if v == VideoPAL {
		return PALTiming
	}
	return NTSCTiming
}

// DetectVideoStandardFromROM returns the video standard for a ROM based on
// CRC32 lookup. Returns (detected standard, true) if found in database,
// (VideoNTSC, false) if not found.
func DetectVideoStandardFromROM(rom []byte) (VideoStandard, bool) {
	crc := crc32.ChecksumIEEE(rom)
	if info, ok := romDatabase[crc]; ok {
		return info.VideoStd, true
	}
	return VideoNTSC, false
}

// Nationality represents the console nationality (Japanese or Export).
// This is orthogonal to video standard (NTSC/PAL): Japanese is always
// NTSC, but Export can be either NTSC (Americas) or PAL (Europe).
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
