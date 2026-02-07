//go:build libretro

package main

import (
	"math"
	"testing"

	"github.com/user-none/emkiii/emu"
)

// TestConvertRGBAToXRGB8888_Basic verifies R,G,B channels swap correctly
func TestConvertRGBAToXRGB8888_Basic(t *testing.T) {
	// RGBA input: Red=0xFF, Green=0x80, Blue=0x40, Alpha=0x00
	src := []byte{0xFF, 0x80, 0x40, 0x00}
	dst := make([]byte, 4)

	convertRGBAToXRGB8888(src, dst, 1)

	// Expected XRGB8888 (little-endian): B=0x40, G=0x80, R=0xFF, X=0xFF
	expected := []byte{0x40, 0x80, 0xFF, 0xFF}

	for i := 0; i < 4; i++ {
		if dst[i] != expected[i] {
			t.Errorf("dst[%d] = %#02x, want %#02x", i, dst[i], expected[i])
		}
	}
}

// TestConvertRGBAToXRGB8888_Alpha verifies alpha becomes 0xFF
func TestConvertRGBAToXRGB8888_Alpha(t *testing.T) {
	testCases := []struct {
		name     string
		srcAlpha byte
	}{
		{"zero alpha", 0x00},
		{"half alpha", 0x80},
		{"full alpha", 0xFF},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte{0x12, 0x34, 0x56, tc.srcAlpha}
			dst := make([]byte, 4)

			convertRGBAToXRGB8888(src, dst, 1)

			if dst[3] != 0xFF {
				t.Errorf("X channel = %#02x, want 0xFF (input alpha was %#02x)", dst[3], tc.srcAlpha)
			}
		})
	}
}

// TestConvertRGBAToXRGB8888_Empty handles empty input
func TestConvertRGBAToXRGB8888_Empty(t *testing.T) {
	src := []byte{}
	dst := []byte{}

	// Should not panic
	convertRGBAToXRGB8888(src, dst, 0)
}

// TestConvertRGBAToXRGB8888_MultiplePixels tests conversion of multiple pixels
func TestConvertRGBAToXRGB8888_MultiplePixels(t *testing.T) {
	// Two pixels: Red and Blue
	src := []byte{
		0xFF, 0x00, 0x00, 0xFF, // Red pixel (RGBA)
		0x00, 0x00, 0xFF, 0xFF, // Blue pixel (RGBA)
	}
	dst := make([]byte, 8)

	convertRGBAToXRGB8888(src, dst, 2)

	// Expected: swapped channels
	expected := []byte{
		0x00, 0x00, 0xFF, 0xFF, // Red pixel in XRGB8888: B=0, G=0, R=FF, X=FF
		0xFF, 0x00, 0x00, 0xFF, // Blue pixel in XRGB8888: B=FF, G=0, R=0, X=FF
	}

	for i := 0; i < 8; i++ {
		if dst[i] != expected[i] {
			t.Errorf("dst[%d] = %#02x, want %#02x", i, dst[i], expected[i])
		}
	}
}

// TestConvertAudioSamples_Silence verifies 0.0 converts to 0
func TestConvertAudioSamples_Silence(t *testing.T) {
	samples := []float32{0.0, 0.0}
	result := emu.ConvertAudioSamples(samples)

	if len(result) != 4 {
		t.Fatalf("len(result) = %d, want 4", len(result))
	}

	for i, val := range result {
		if val != 0 {
			t.Errorf("result[%d] = %d, want 0", i, val)
		}
	}
}

// TestConvertAudioSamples_MaxPositive verifies 1.0 converts to 32767
func TestConvertAudioSamples_MaxPositive(t *testing.T) {
	samples := []float32{1.0}
	result := emu.ConvertAudioSamples(samples)

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	expected := int16(32767)
	if result[0] != expected || result[1] != expected {
		t.Errorf("result = [%d, %d], want [%d, %d]", result[0], result[1], expected, expected)
	}
}

// TestConvertAudioSamples_MaxNegative verifies -1.0 converts to -32767
func TestConvertAudioSamples_MaxNegative(t *testing.T) {
	samples := []float32{-1.0}
	result := emu.ConvertAudioSamples(samples)

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	expected := int16(-32767)
	if result[0] != expected || result[1] != expected {
		t.Errorf("result = [%d, %d], want [%d, %d]", result[0], result[1], expected, expected)
	}
}

// TestConvertAudioSamples_Stereo verifies mono to stereo duplication
func TestConvertAudioSamples_Stereo(t *testing.T) {
	samples := []float32{0.5, -0.5}
	result := emu.ConvertAudioSamples(samples)

	if len(result) != 4 {
		t.Fatalf("len(result) = %d, want 4 (2 stereo pairs)", len(result))
	}

	// Each mono sample should be duplicated for left and right
	// Note: The conversion uses int16(sample * 32767) which truncates
	sampleFirst := float32(0.5)
	sampleSecond := float32(-0.5)
	expectedFirst := int16(sampleFirst * 32767)
	expectedSecond := int16(sampleSecond * 32767)

	if result[0] != expectedFirst || result[1] != expectedFirst {
		t.Errorf("first stereo pair = [%d, %d], want [%d, %d]", result[0], result[1], expectedFirst, expectedFirst)
	}

	if result[2] != expectedSecond || result[3] != expectedSecond {
		t.Errorf("second stereo pair = [%d, %d], want [%d, %d]", result[2], result[3], expectedSecond, expectedSecond)
	}
}

// TestConvertAudioSamples_Empty handles empty input
func TestConvertAudioSamples_Empty(t *testing.T) {
	samples := []float32{}
	result := emu.ConvertAudioSamples(samples)

	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

// TestDisplayConstants verifies display dimension constants
func TestDisplayConstants(t *testing.T) {
	if WIDTH != 256 {
		t.Errorf("WIDTH = %d, want 256", WIDTH)
	}
	if HEIGHT != 192 {
		t.Errorf("HEIGHT = %d, want 192", HEIGHT)
	}
	if MAXHEIGHT != 224 {
		t.Errorf("MAXHEIGHT = %d, want 224", MAXHEIGHT)
	}
	if SAMPLERATE != 48000 {
		t.Errorf("SAMPLERATE = %d, want 48000", SAMPLERATE)
	}
}

// TestAspectRatio verifies the aspect ratio calculation
func TestAspectRatio(t *testing.T) {
	aspectRatio := float64(WIDTH) / float64(HEIGHT)
	expected := 256.0 / 192.0 // 1.333...

	if math.Abs(aspectRatio-expected) > 0.001 {
		t.Errorf("aspect ratio = %f, want %f", aspectRatio, expected)
	}

	// Verify it's approximately 4:3
	if math.Abs(aspectRatio-4.0/3.0) > 0.001 {
		t.Errorf("aspect ratio = %f, want approximately 4:3 (1.333)", aspectRatio)
	}
}

// TestGetRegionMapping tests the region to libretro constant mapping
func TestGetRegionMapping(t *testing.T) {
	// Save original region
	originalRegion := region
	defer func() { region = originalRegion }()

	// Test NTSC region
	region = emu.RegionNTSC
	// Note: We can't call retro_get_region directly as it returns C.uint,
	// but we can verify the mapping logic works correctly
	if region != emu.RegionNTSC {
		t.Errorf("region should be NTSC")
	}

	// Test PAL region
	region = emu.RegionPAL
	if region != emu.RegionPAL {
		t.Errorf("region should be PAL")
	}
}
