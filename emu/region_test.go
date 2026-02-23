package emu

import "testing"

// TestRegion_NTSCTiming verifies NTSC timing constants
func TestRegion_NTSCTiming(t *testing.T) {
	if NTSCTiming.CPUClockHz != 3579545 {
		t.Errorf("NTSC CPU clock: expected 3579545, got %d", NTSCTiming.CPUClockHz)
	}
	if NTSCTiming.Scanlines != 262 {
		t.Errorf("NTSC scanlines: expected 262, got %d", NTSCTiming.Scanlines)
	}
	if NTSCTiming.FPS != 60 {
		t.Errorf("NTSC FPS: expected 60, got %d", NTSCTiming.FPS)
	}
}

// TestRegion_PALTiming verifies PAL timing constants
func TestRegion_PALTiming(t *testing.T) {
	if PALTiming.CPUClockHz != 3546893 {
		t.Errorf("PAL CPU clock: expected 3546893, got %d", PALTiming.CPUClockHz)
	}
	if PALTiming.Scanlines != 313 {
		t.Errorf("PAL scanlines: expected 313, got %d", PALTiming.Scanlines)
	}
	if PALTiming.FPS != 50 {
		t.Errorf("PAL FPS: expected 50, got %d", PALTiming.FPS)
	}
}

// TestRegion_GetTimingForRegion verifies correct timing is returned per region
func TestRegion_GetTimingForRegion(t *testing.T) {
	ntsc := GetTimingForRegion(RegionNTSC)
	if ntsc.CPUClockHz != NTSCTiming.CPUClockHz {
		t.Errorf("GetTimingForRegion(NTSC): expected NTSC timing, got CPUClockHz=%d", ntsc.CPUClockHz)
	}

	pal := GetTimingForRegion(RegionPAL)
	if pal.CPUClockHz != PALTiming.CPUClockHz {
		t.Errorf("GetTimingForRegion(PAL): expected PAL timing, got CPUClockHz=%d", pal.CPUClockHz)
	}
}

// TestRegion_DefaultRegion verifies default is NTSC
func TestRegion_DefaultRegion(t *testing.T) {
	if DefaultRegion() != RegionNTSC {
		t.Errorf("DefaultRegion: expected NTSC, got %v", DefaultRegion())
	}
}

// TestRegion_ScanlineTimingConsistency verifies timing relationships
func TestRegion_ScanlineTimingConsistency(t *testing.T) {
	// NTSC: ~3.58MHz / 262 scanlines / 60fps ≈ 228 cycles per scanline
	ntscCyclesPerScanline := float64(NTSCTiming.CPUClockHz) / float64(NTSCTiming.Scanlines) / float64(NTSCTiming.FPS)
	if ntscCyclesPerScanline < 220 || ntscCyclesPerScanline > 240 {
		t.Errorf("NTSC cycles per scanline: expected ~228, got %.1f", ntscCyclesPerScanline)
	}

	// PAL: ~3.55MHz / 313 scanlines / 50fps ≈ 227 cycles per scanline
	palCyclesPerScanline := float64(PALTiming.CPUClockHz) / float64(PALTiming.Scanlines) / float64(PALTiming.FPS)
	if palCyclesPerScanline < 220 || palCyclesPerScanline > 240 {
		t.Errorf("PAL cycles per scanline: expected ~227, got %.1f", palCyclesPerScanline)
	}
}

// TestNationality_String verifies string representation
func TestNationality_String(t *testing.T) {
	if NationalityExport.String() != "Export" {
		t.Errorf("NationalityExport.String(): expected \"Export\", got %q", NationalityExport.String())
	}
	if NationalityJapanese.String() != "Japanese" {
		t.Errorf("NationalityJapanese.String(): expected \"Japanese\", got %q", NationalityJapanese.String())
	}
}

// TestDetectNationalityFromROM_ValidHeader tests detection with valid TMR SEGA headers
func TestDetectNationalityFromROM_ValidHeader(t *testing.T) {
	testCases := []struct {
		name       string
		regionCode uint8
		expected   Nationality
	}{
		{"SMS Japan (code 3)", 0x3, NationalityJapanese},
		{"SMS Export (code 4)", 0x4, NationalityExport},
		{"GG Japan (code 5)", 0x5, NationalityExport},
		{"GG Export (code 6)", 0x6, NationalityExport},
		{"GG International (code 7)", 0x7, NationalityExport},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			copy(rom[0x7FF0:], "TMR SEGA")
			// Region code in upper nibble of $7FFF, ROM size in lower nibble
			rom[0x7FFF] = tc.regionCode<<4 | 0x0C // 0x0C = 32KB ROM size

			got := DetectNationalityFromROM(rom)
			if got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

// TestDetectNationalityFromROM_NoSignature tests that missing TMR SEGA defaults to Export
func TestDetectNationalityFromROM_NoSignature(t *testing.T) {
	rom := make([]byte, 0x8000)
	// No TMR SEGA signature written

	got := DetectNationalityFromROM(rom)
	if got != NationalityExport {
		t.Errorf("expected Export for missing signature, got %v", got)
	}
}

// TestDetectNationalityFromROM_TooSmall tests that small ROMs default to Export
func TestDetectNationalityFromROM_TooSmall(t *testing.T) {
	rom := make([]byte, 0x4000) // 16KB, too small for header at $7FF0

	got := DetectNationalityFromROM(rom)
	if got != NationalityExport {
		t.Errorf("expected Export for small ROM, got %v", got)
	}
}
