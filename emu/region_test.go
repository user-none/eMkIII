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

// TestRegion_String verifies string representation
func TestRegion_String(t *testing.T) {
	if RegionNTSC.String() != "NTSC" {
		t.Errorf("RegionNTSC.String(): expected \"NTSC\", got %q", RegionNTSC.String())
	}
	if RegionPAL.String() != "PAL" {
		t.Errorf("RegionPAL.String(): expected \"PAL\", got %q", RegionPAL.String())
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
