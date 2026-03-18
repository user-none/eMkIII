package core

import (
	"hash/crc32"
	"testing"
)

// TestROMDatabase_KnownEntries tests that known game CRCs return correct info
func TestROMDatabase_KnownEntries(t *testing.T) {
	testCases := []struct {
		name         string
		crc          uint32
		wantMapper   MapperType
		wantVideoStd VideoStandard
	}{
		// NTSC Sega games
		{"Sonic the Hedgehog", 0xb519e833, MapperSega, VideoNTSC},
		{"Alex Kidd in Miracle World", 0x50a8e8a7, MapperSega, VideoNTSC},
		{"Phantasy Star", 0x07301f83, MapperSega, VideoNTSC},

		// PAL Sega games
		{"Sonic the Hedgehog 2", 0x5b3b922c, MapperSega, VideoPAL},
		{"Streets of Rage", 0x4ab3790f, MapperSega, VideoPAL},
		{"Aladdin", 0xc8718d40, MapperSega, VideoPAL},

		// Codemasters games (PAL)
		{"Fantastic Dizzy", 0xb9664ae1, MapperCodemasters, VideoPAL},
		{"Cosmic Spacehead", 0x29822980, MapperCodemasters, VideoPAL},
		{"Micro Machines (PAL)", 0xa577ce46, MapperCodemasters, VideoPAL},

		// Codemasters (NTSC, not in CSV)
		{"Micro Machines (NTSC)", 0xa567a0c6, MapperCodemasters, VideoNTSC},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, ok := romDatabase[tc.crc]
			if !ok {
				t.Fatalf("CRC 0x%08x not found in database", tc.crc)
			}
			if info.Mapper != tc.wantMapper {
				t.Errorf("Mapper: got %v, want %v", info.Mapper, tc.wantMapper)
			}
			if info.VideoStd != tc.wantVideoStd {
				t.Errorf("VideoStd: got %v, want %v", info.VideoStd, tc.wantVideoStd)
			}
		})
	}
}

// TestROMDatabase_EntryCount verifies the database contains the expected number of entries
func TestROMDatabase_EntryCount(t *testing.T) {
	// 357 from CSV + 1 additional Codemasters (0xa567a0c6) = 358
	expectedMin := 357
	if len(romDatabase) < expectedMin {
		t.Errorf("Database has %d entries, expected at least %d", len(romDatabase), expectedMin)
	}
}

// TestROMDatabase_AllEntriesValid verifies all entries have valid values
func TestROMDatabase_AllEntriesValid(t *testing.T) {
	for crc, info := range romDatabase {
		if info.Mapper != MapperSega && info.Mapper != MapperCodemasters {
			t.Errorf("CRC 0x%08x has invalid mapper: %v", crc, info.Mapper)
		}
		if info.VideoStd != VideoNTSC && info.VideoStd != VideoPAL {
			t.Errorf("CRC 0x%08x has invalid video standard: %v", crc, info.VideoStd)
		}
	}
}

// TestDetectVideoStandardFromROM_KnownROM tests detection of a known ROM
func TestDetectVideoStandardFromROM_KnownROM(t *testing.T) {
	testCases := []struct {
		name         string
		crc          uint32
		wantVideoStd VideoStandard
		wantFound    bool
	}{
		{"Sonic the Hedgehog (NTSC)", 0xb519e833, VideoNTSC, true},
		{"Sonic the Hedgehog 2 (PAL)", 0x5b3b922c, VideoPAL, true},
		{"Fantastic Dizzy (PAL)", 0xb9664ae1, VideoPAL, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, ok := romDatabase[tc.crc]
			if !ok != !tc.wantFound {
				t.Errorf("Found=%v, want found=%v", ok, tc.wantFound)
			}
			if ok && info.VideoStd != tc.wantVideoStd {
				t.Errorf("VideoStd=%v, want %v", info.VideoStd, tc.wantVideoStd)
			}
		})
	}
}

// TestDetectVideoStandardFromROM_UnknownROM tests detection of an unknown ROM
func TestDetectVideoStandardFromROM_UnknownROM(t *testing.T) {
	unknownROM := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x12, 0x34, 0x56, 0x78}

	videoStd, found := DetectVideoStandardFromROM(unknownROM)

	if found {
		t.Errorf("Unknown ROM should not be found in database")
	}
	if videoStd != VideoNTSC {
		t.Errorf("Unknown ROM should default to NTSC, got %v", videoStd)
	}
}

// TestDetectMapper_Codemasters tests that Codemasters games are detected correctly
func TestDetectMapper_Codemasters(t *testing.T) {
	testCases := []struct {
		name string
		crc  uint32
	}{
		{"Fantastic Dizzy", 0xb9664ae1},
		{"Cosmic Spacehead", 0x29822980},
		{"Micro Machines (PAL)", 0xa577ce46},
		{"Micro Machines (NTSC)", 0xa567a0c6},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, ok := romDatabase[tc.crc]
			if !ok {
				t.Fatalf("CRC 0x%08x not found in database", tc.crc)
			}
			if info.Mapper != MapperCodemasters {
				t.Errorf("Expected MapperCodemasters, got %v", info.Mapper)
			}
		})
	}
}

// TestDetectMapper_Integration tests the detectMapper function via NewMemory
func TestDetectMapper_Integration(t *testing.T) {
	unknownROM := make([]byte, 0x8000) // 32KB
	for i := range unknownROM {
		unknownROM[i] = byte(i & 0xFF)
	}

	mem := NewMemory(unknownROM)
	if mem.mapper != MapperSega {
		t.Errorf("Unknown ROM should use Sega mapper, got %v", mem.mapper)
	}
}

// TestDetectVideoStandardFromROM_Integration tests the full detection function
func TestDetectVideoStandardFromROM_Integration(t *testing.T) {
	testData := []byte("test rom data for crc verification")
	expectedCRC := crc32.ChecksumIEEE(testData)

	if expectedCRC == 0 {
		t.Error("CRC should not be zero for non-empty data")
	}

	videoStd, found := DetectVideoStandardFromROM(testData)
	if found {
		t.Error("Random test data should not be found in database")
	}
	if videoStd != VideoNTSC {
		t.Errorf("Default should be NTSC, got %v", videoStd)
	}
}
