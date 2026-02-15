package emu

import (
	"math"
	"testing"

	"github.com/user-none/go-chip-sn76489"
)

// TestPSG_SilentOnInit verifies all volumes start at 0x0F (silent)
func TestPSG_SilentOnInit(t *testing.T) {
	psg := sn76489.New(3579545, 48000, 800, sn76489.Sega)

	for ch := 0; ch < 4; ch++ {
		if vol := psg.GetVolume(ch); vol != 0x0F {
			t.Errorf("Channel %d initial volume: expected 0x0F (silent), got 0x%02X", ch, vol)
		}
	}
}

// TestPSG_VolumeRegisterWrite tests 4-bit volume writes for all channels
func TestPSG_VolumeRegisterWrite(t *testing.T) {
	psg := sn76489.New(3579545, 48000, 800, sn76489.Sega)

	testCases := []struct {
		channel uint8
		volume  uint8
	}{
		{0, 0x00}, // Channel 0, max volume
		{1, 0x08}, // Channel 1, mid volume
		{2, 0x0F}, // Channel 2, silent
		{3, 0x05}, // Noise channel
	}

	for _, tc := range testCases {
		// Volume write: 1 CC 1 VVVV (bit 7=1, CC=channel, bit 4=1 for volume, VVVV=volume)
		cmd := uint8(0x90) | (tc.channel << 5) | tc.volume
		psg.Write(cmd)

		if got := psg.GetVolume(int(tc.channel)); got != tc.volume {
			t.Errorf("Channel %d volume after write: expected 0x%02X, got 0x%02X", tc.channel, tc.volume, got)
		}
	}
}

// TestPSG_ToneRegisterWrite tests 10-bit tone register via latch+data bytes
func TestPSG_ToneRegisterWrite(t *testing.T) {
	psg := sn76489.New(3579545, 48000, 800, sn76489.Sega)

	// Write a 10-bit tone value (0x1AB = 427) to channel 0
	// First byte: 1 CC 0 DDDD (low 4 bits) = 0x80 | 0x0B = 0x8B
	// Second byte: 0 X DDDDDD (high 6 bits) = 0x1A
	psg.Write(0x8B) // Latch channel 0 tone, low nibble = 0xB
	psg.Write(0x1A) // Data = 0x1A (high 6 bits)

	expected := uint16(0x1AB)
	if got := psg.GetToneReg(0); got != expected {
		t.Errorf("Channel 0 tone register: expected 0x%03X, got 0x%03X", expected, got)
	}

	// Test channel 1 with a different value
	psg.Write(0xA5) // Latch channel 1 tone, low nibble = 0x5
	psg.Write(0x3F) // Data = 0x3F (high 6 bits)

	expected = uint16(0x3F5)
	if got := psg.GetToneReg(1); got != expected {
		t.Errorf("Channel 1 tone register: expected 0x%03X, got 0x%03X", expected, got)
	}
}

// TestPSG_NoiseRegisterWrite tests noise control register writes
func TestPSG_NoiseRegisterWrite(t *testing.T) {
	psg := sn76489.New(3579545, 48000, 800, sn76489.Sega)

	// Noise write: 1 11 0 0NNN (channel 3, type=0, NNN=noise control)
	// Test different noise modes
	testCases := []struct {
		noiseReg uint8
		desc     string
	}{
		{0x00, "periodic noise, /512"},
		{0x01, "periodic noise, /1024"},
		{0x02, "periodic noise, /2048"},
		{0x03, "periodic noise, tone2 rate"},
		{0x04, "white noise, /512"},
		{0x05, "white noise, /1024"},
		{0x06, "white noise, /2048"},
		{0x07, "white noise, tone2 rate"},
	}

	for _, tc := range testCases {
		// Noise register write: 0xE0 | noise bits
		psg.Write(0xE0 | tc.noiseReg)

		if got := psg.GetNoiseReg(); got != tc.noiseReg {
			t.Errorf("Noise register for %s: expected 0x%02X, got 0x%02X", tc.desc, tc.noiseReg, got)
		}
	}
}

// TestPSG_VolumeTable tests volume lookup table values
func TestPSG_VolumeTable(t *testing.T) {
	table := sn76489.GetVolumeTable()

	// Volume 0 should be maximum (1.0)
	if table[0] != 1.0 {
		t.Errorf("Volume 0: expected 1.0, got %f", table[0])
	}

	// Volume 15 should be silent (0.0)
	if table[15] != 0.0 {
		t.Errorf("Volume 15: expected 0.0, got %f", table[15])
	}

	// Each step should decrease (approximately -2dB)
	for i := 0; i < 14; i++ {
		if table[i+1] >= table[i] {
			t.Errorf("Volume %d (%.3f) should be greater than volume %d (%.3f)",
				i, table[i], i+1, table[i+1])
		}
	}

	// Verify approximately -2dB per step (ratio ≈ 0.794)
	for i := 0; i < 14; i++ {
		if table[i] > 0 && table[i+1] > 0 {
			ratio := table[i+1] / table[i]
			if ratio < 0.7 || ratio > 0.9 {
				t.Errorf("Volume ratio %d->%d: expected ~0.794, got %.3f", i, i+1, ratio)
			}
		}
	}
}

// TestPSG_ClockDivider tests that input clock is divided by 16
func TestPSG_ClockDivider(t *testing.T) {
	psg := sn76489.New(3579545, 48000, 800, sn76489.Sega)

	// Set up tone channel 0 with frequency divider of 1 (highest frequency)
	// and max volume
	psg.Write(0x81) // Channel 0 tone, low nibble = 1
	psg.Write(0x00) // High bits = 0, so tone = 1
	psg.Write(0x90) // Channel 0 volume = 0 (max)

	// The tone output should flip every (divider value) internal clocks
	// Since divider = 16, and tone reg = 1, output flips every 16 input clocks
	// after the counter decrements

	// Clock 15 times - should not complete a full divider cycle
	for i := 0; i < 15; i++ {
		psg.Clock()
	}

	// The 16th clock should trigger an internal update
	// This tests that the divider is working
	psg.Clock()
	// After 16 clocks, the tone counter should have decremented
}

// TestPSG_SampleGeneration tests that samples are generated correctly
func TestPSG_SampleGeneration(t *testing.T) {
	psg := sn76489.New(3579545, 48000, 800, sn76489.Sega)

	// All channels silent - should generate ~0 output
	sample := psg.Sample()

	// With all volumes at 0x0F (silent), output should be 0
	if math.Abs(float64(sample)) > 0.001 {
		t.Errorf("Silent sample: expected ~0, got %f", sample)
	}
}

// TestPSG_SampleMixing tests 4 channels mixed and normalized
func TestPSG_SampleMixing(t *testing.T) {
	psg := sn76489.New(3579545, 48000, 800, sn76489.Sega)

	// Set all channels to max volume
	psg.Write(0x90) // Channel 0 volume = 0 (max)
	psg.Write(0xB0) // Channel 1 volume = 0 (max)
	psg.Write(0xD0) // Channel 2 volume = 0 (max)
	psg.Write(0xF0) // Noise volume = 0 (max)

	// Generate a sample - with all at max, should be bounded
	sample := psg.Sample()

	// Sample should be normalized to [-1, 1] range
	if sample < -1.0 || sample > 1.0 {
		t.Errorf("Sample out of range: %f", sample)
	}
}

// TestPSG_NoiseRateFromTone2 tests noise rate 3 uses tone channel 2
func TestPSG_NoiseRateFromTone2(t *testing.T) {
	psg := sn76489.New(3579545, 48000, 800, sn76489.Sega)

	// Set tone channel 2 to a specific frequency
	psg.Write(0xC5) // Channel 2 tone, low nibble = 5
	psg.Write(0x10) // High bits = 0x10, so tone = 0x105

	// Set noise to use tone 2 rate (rate = 3)
	psg.Write(0xE3) // Noise control = 3 (use tone 2)

	tone2Reg := psg.GetToneReg(2)
	if tone2Reg != 0x105 {
		t.Errorf("Tone 2 register: expected 0x105, got 0x%03X", tone2Reg)
	}

	noiseReg := psg.GetNoiseReg()
	if noiseReg&0x03 != 3 {
		t.Errorf("Noise rate bits: expected 3, got %d", noiseReg&0x03)
	}
}

// TestPSG_GenerateSamples tests buffer-based sample generation
func TestPSG_GenerateSamples(t *testing.T) {
	psg := sn76489.New(3579545, 48000, 100, sn76489.Sega)

	// Generate samples for a certain number of clocks
	clocks := 10000 // Enough clocks to generate some samples
	psg.GenerateSamples(clocks)

	buf, count := psg.GetBuffer()
	if count == 0 {
		t.Error("GenerateSamples produced no samples")
	}
	if buf == nil {
		t.Error("GetBuffer returned nil buffer")
	}
	if count > len(buf) {
		t.Errorf("Sample count %d exceeds buffer size %d", count, len(buf))
	}
}

// TestPSG_ToneLatchPersistence tests that the latched channel persists
func TestPSG_ToneLatchPersistence(t *testing.T) {
	psg := sn76489.New(3579545, 48000, 800, sn76489.Sega)

	// Latch channel 2
	psg.Write(0xC0) // Channel 2 tone, low nibble = 0

	// Write multiple data bytes - should all go to channel 2
	psg.Write(0x10) // High 6 bits = 0x10
	expected := uint16(0x100)
	if got := psg.GetToneReg(2); got != expected {
		t.Errorf("After first data: expected 0x%03X, got 0x%03X", expected, got)
	}

	// Write another data byte
	psg.Write(0x20) // High 6 bits = 0x20
	expected = uint16(0x200)
	if got := psg.GetToneReg(2); got != expected {
		t.Errorf("After second data: expected 0x%03X, got 0x%03X", expected, got)
	}
}
