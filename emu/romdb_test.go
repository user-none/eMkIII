package emu

import (
	"hash/crc32"
	"testing"
)

// TestROMDatabase_KnownEntries tests that known game CRCs return correct info
func TestROMDatabase_KnownEntries(t *testing.T) {
	testCases := []struct {
		name       string
		crc        uint32
		wantMapper MapperType
		wantRegion Region
	}{
		// NTSC Sega games
		{"Sonic the Hedgehog", 0xb519e833, MapperSega, RegionNTSC},
		{"Alex Kidd in Miracle World", 0x50a8e8a7, MapperSega, RegionNTSC},
		{"Phantasy Star", 0x07301f83, MapperSega, RegionNTSC},

		// PAL Sega games
		{"Sonic the Hedgehog 2", 0x5b3b922c, MapperSega, RegionPAL},
		{"Streets of Rage", 0x4ab3790f, MapperSega, RegionPAL},
		{"Aladdin", 0xc8718d40, MapperSega, RegionPAL},

		// Codemasters games (PAL)
		{"Fantastic Dizzy", 0xb9664ae1, MapperCodemasters, RegionPAL},
		{"Cosmic Spacehead", 0x29822980, MapperCodemasters, RegionPAL},
		{"Micro Machines (PAL)", 0xa577ce46, MapperCodemasters, RegionPAL},

		// Codemasters (NTSC, not in CSV)
		{"Micro Machines (NTSC)", 0xa567a0c6, MapperCodemasters, RegionNTSC},
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
			if info.Region != tc.wantRegion {
				t.Errorf("Region: got %v, want %v", info.Region, tc.wantRegion)
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
		if info.Region != RegionNTSC && info.Region != RegionPAL {
			t.Errorf("CRC 0x%08x has invalid region: %v", crc, info.Region)
		}
	}
}

// TestDetectRegionFromROM_KnownROM tests detection of a known ROM
func TestDetectRegionFromROM_KnownROM(t *testing.T) {
	// Create a fake ROM that would have the CRC of Sonic the Hedgehog
	// We can't easily create a ROM with a specific CRC, so we test with actual lookup
	testCases := []struct {
		name       string
		crc        uint32
		wantRegion Region
		wantFound  bool
	}{
		{"Sonic the Hedgehog (NTSC)", 0xb519e833, RegionNTSC, true},
		{"Sonic the Hedgehog 2 (PAL)", 0x5b3b922c, RegionPAL, true},
		{"Fantastic Dizzy (PAL)", 0xb9664ae1, RegionPAL, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Since we can't create a ROM with a known CRC easily,
			// directly test the lookup mechanism
			info, ok := romDatabase[tc.crc]
			if !ok != !tc.wantFound {
				t.Errorf("Found=%v, want found=%v", ok, tc.wantFound)
			}
			if ok && info.Region != tc.wantRegion {
				t.Errorf("Region=%v, want %v", info.Region, tc.wantRegion)
			}
		})
	}
}

// TestDetectRegionFromROM_UnknownROM tests detection of an unknown ROM
func TestDetectRegionFromROM_UnknownROM(t *testing.T) {
	// Create a ROM that won't be in the database
	unknownROM := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x12, 0x34, 0x56, 0x78}

	region, found := DetectRegionFromROM(unknownROM)

	if found {
		t.Errorf("Unknown ROM should not be found in database")
	}
	if region != RegionNTSC {
		t.Errorf("Unknown ROM should default to NTSC, got %v", region)
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
	// Create a ROM that won't be in the database (unknown ROM)
	unknownROM := make([]byte, 0x8000) // 32KB
	for i := range unknownROM {
		unknownROM[i] = byte(i & 0xFF)
	}

	mem := NewMemory(unknownROM)
	if mem.mapper != MapperSega {
		t.Errorf("Unknown ROM should use Sega mapper, got %v", mem.mapper)
	}
}

// TestDetectRegionFromROM_Integration tests the full detection function
func TestDetectRegionFromROM_Integration(t *testing.T) {
	// Create some test data and verify the CRC calculation path works
	testData := []byte("test rom data for crc verification")
	expectedCRC := crc32.ChecksumIEEE(testData)

	// Verify CRC is computed correctly
	if expectedCRC == 0 {
		t.Error("CRC should not be zero for non-empty data")
	}

	// The ROM shouldn't be in our database
	region, found := DetectRegionFromROM(testData)
	if found {
		t.Error("Random test data should not be found in database")
	}
	if region != RegionNTSC {
		t.Errorf("Default region should be NTSC, got %v", region)
	}
}
