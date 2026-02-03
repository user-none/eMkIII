package emu

import (
	"image/color"
	"testing"
)

// TestVDP_RenderScanline_DisplayDisabled tests rendering with display disabled
func TestVDP_RenderScanline_DisplayDisabled(t *testing.T) {
	vdp := NewVDP()

	// Set backdrop color (register 7, bits 0-3 select color from sprite palette)
	// Set backdrop to palette entry 17 (index 1 in sprite palette)
	vdp.WriteControl(0x01)
	vdp.WriteControl(0x87) // Register 7 = 0x01

	// Set CRAM entry 17 (16 + 1) to a specific color: RGB(170, 85, 0)
	// SMS color: R=2, G=1, B=0 = 0x06
	vdp.WriteControl(17)
	vdp.WriteControl(0xC0) // CRAM write at index 17
	vdp.WriteData(0x06)    // Color value

	// Display is disabled by default (register 1 bit 6 = 0)
	// Make sure it's disabled
	vdp.WriteControl(0x00) // Value with bit 6 clear
	vdp.WriteControl(0x81) // Register 1

	// Set scanline
	vdp.SetVCounter(0)
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()

	// Render
	vdp.RenderScanline()

	// Check that all pixels are backdrop color
	fb := vdp.Framebuffer()
	expectedColor := color.RGBA{R: 170, G: 85, B: 0, A: 255}

	for x := 0; x < ScreenWidth; x++ {
		c := fb.RGBAAt(x, 0)
		if c != expectedColor {
			t.Errorf("Pixel (%d, 0): expected %v, got %v", x, expectedColor, c)
			break
		}
	}
}

// TestVDP_RenderScanline_BeyondActiveHeight tests that scanlines beyond active area don't render
func TestVDP_RenderScanline_BeyondActiveHeight(t *testing.T) {
	vdp := NewVDP()

	// Enable display
	vdp.WriteControl(0x40) // Bit 6 set
	vdp.WriteControl(0x81) // Register 1

	// Set scanline beyond active area
	vdp.SetVCounter(192)

	// Clear framebuffer area (should remain unchanged)
	fb := vdp.Framebuffer()
	for x := 0; x < ScreenWidth; x++ {
		fb.SetRGBA(x, 192, color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	}

	// Render should be a no-op
	vdp.RenderScanline()

	// Verify pixels unchanged
	for x := 0; x < ScreenWidth; x++ {
		c := fb.RGBAAt(x, 192)
		expected := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
		if c != expected {
			t.Errorf("Pixel (%d, 192) changed when it shouldn't", x)
			break
		}
	}
}

// TestVDP_RenderScanline_LeftColumnBlank tests left column masking
func TestVDP_RenderScanline_LeftColumnBlank(t *testing.T) {
	vdp := NewVDP()

	// Enable display
	vdp.WriteControl(0x40)
	vdp.WriteControl(0x81)

	// Set backdrop color
	vdp.WriteControl(0x01) // backdrop = color 1
	vdp.WriteControl(0x87)

	// Set CRAM entry 17 to red
	vdp.WriteControl(17)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x03) // Pure red

	// Enable left column blank (register 0 bit 5)
	vdp.WriteControl(0x20)
	vdp.WriteControl(0x80)

	vdp.SetVCounter(0)
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	// First 8 pixels should be backdrop color
	fb := vdp.Framebuffer()
	backdropColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	for x := 0; x < 8; x++ {
		c := fb.RGBAAt(x, 0)
		if c != backdropColor {
			t.Errorf("Left column pixel (%d, 0): expected backdrop %v, got %v", x, backdropColor, c)
		}
	}
}

// TestVDP_cramToColor tests CRAM to RGBA conversion
func TestVDP_cramToColor(t *testing.T) {
	vdp := NewVDP()

	testCases := []struct {
		cramVal  uint8
		expected color.RGBA
	}{
		{0x00, color.RGBA{R: 0, G: 0, B: 0, A: 255}},       // Black
		{0x03, color.RGBA{R: 255, G: 0, B: 0, A: 255}},     // Pure red
		{0x0C, color.RGBA{R: 0, G: 255, B: 0, A: 255}},     // Pure green
		{0x30, color.RGBA{R: 0, G: 0, B: 255, A: 255}},     // Pure blue
		{0x3F, color.RGBA{R: 255, G: 255, B: 255, A: 255}}, // White
		{0x15, color.RGBA{R: 85, G: 85, B: 85, A: 255}},    // Gray (1,1,1)
		{0x2A, color.RGBA{R: 170, G: 170, B: 170, A: 255}}, // Light gray (2,2,2)
	}

	for _, tc := range testCases {
		// Write to CRAM entry 0
		vdp.WriteControl(0x00)
		vdp.WriteControl(0xC0)
		vdp.WriteData(tc.cramVal)

		// Latch CRAM before reading (cramToColor uses latched values)
		vdp.LatchCRAM()
		vdp.LatchPerLineRegisters()

		c := vdp.cramToColor(0)
		if c != tc.expected {
			t.Errorf("cramToColor(0x%02X): expected %v, got %v", tc.cramVal, tc.expected, c)
		}
	}
}

// TestVDP_RenderBackground_BasicTile tests rendering a simple tile
func TestVDP_RenderBackground_BasicTile(t *testing.T) {
	vdp := NewVDP()

	// Enable display
	vdp.WriteControl(0x40)
	vdp.WriteControl(0x81)

	// Set name table base (register 2)
	// Default value 0x0E gives base at $3800
	vdp.WriteControl(0x0E)
	vdp.WriteControl(0x82)

	// Set up a simple tile pattern at index 0 (address $0000)
	// Pattern is 8x8, 4 bytes per line (4bpp)
	// Let's make a solid color tile (all pixels = color 1)
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40) // VRAM write at 0x0000

	for line := 0; line < 8; line++ {
		vdp.WriteData(0xFF) // Bitplane 0: all 1s
		vdp.WriteData(0x00) // Bitplane 1: all 0s
		vdp.WriteData(0x00) // Bitplane 2: all 0s
		vdp.WriteData(0x00) // Bitplane 3: all 0s
		// This gives color index 1 for all pixels
	}

	// Set up name table entry at $3800 (first tile)
	// Entry: tile 0, no flip, palette 0, no priority
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x78) // VRAM write at 0x3800
	vdp.WriteData(0x00)    // Low byte: tile index 0
	vdp.WriteData(0x00)    // High byte: no flags

	// Set up CRAM entry 1 (palette 0, color 1) to green
	vdp.WriteControl(0x01)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x0C) // Green

	// Set up CRAM entry 0 for background (transparent)
	vdp.WriteControl(0x00)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x00) // Black

	vdp.SetVCounter(0)
	vdp.LatchVScrollForFrame()
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	// First 8 pixels should be green (from our tile)
	fb := vdp.Framebuffer()
	greenColor := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	for x := 0; x < 8; x++ {
		c := fb.RGBAAt(x, 0)
		if c != greenColor {
			t.Errorf("Tile pixel (%d, 0): expected %v, got %v", x, greenColor, c)
		}
	}
}

// TestVDP_RenderBackground_HorizontalFlip tests horizontal tile flipping
func TestVDP_RenderBackground_HorizontalFlip(t *testing.T) {
	vdp := NewVDP()

	// Enable display
	vdp.WriteControl(0x40)
	vdp.WriteControl(0x81)

	// Set name table base
	vdp.WriteControl(0x0E)
	vdp.WriteControl(0x82)

	// Create a gradient tile: left pixels are color 1, right pixels are color 2
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40)

	for line := 0; line < 8; line++ {
		// Left 4 pixels = color 1 (0001), right 4 pixels = color 2 (0010)
		vdp.WriteData(0xF0) // BP0: 11110000
		vdp.WriteData(0x0F) // BP1: 00001111
		vdp.WriteData(0x00) // BP2: 00000000
		vdp.WriteData(0x00) // BP3: 00000000
	}

	// Set up name table entry with horizontal flip (bit 9 = 1)
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x78)
	vdp.WriteData(0x00) // Low byte: tile 0
	vdp.WriteData(0x02) // High byte: hFlip

	// Set colors
	vdp.WriteControl(0x01)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x03) // Color 1 = red

	vdp.WriteControl(0x02)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x0C) // Color 2 = green

	vdp.SetVCounter(0)
	vdp.LatchVScrollForFrame()
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	fb := vdp.Framebuffer()

	// With horizontal flip, right pixels (color 2) become left pixels
	greenColor := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	redColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	// First 4 pixels should be green (originally right side)
	for x := 0; x < 4; x++ {
		c := fb.RGBAAt(x, 0)
		if c != greenColor {
			t.Errorf("Flipped tile left pixel (%d): expected green, got %v", x, c)
		}
	}

	// Next 4 pixels should be red (originally left side)
	for x := 4; x < 8; x++ {
		c := fb.RGBAAt(x, 0)
		if c != redColor {
			t.Errorf("Flipped tile right pixel (%d): expected red, got %v", x, c)
		}
	}
}

// TestVDP_RenderBackground_VerticalFlip tests vertical tile flipping
func TestVDP_RenderBackground_VerticalFlip(t *testing.T) {
	vdp := NewVDP()

	// Enable display
	vdp.WriteControl(0x40)
	vdp.WriteControl(0x81)

	vdp.WriteControl(0x0E)
	vdp.WriteControl(0x82)

	// Create a tile where line 0 is color 1, line 7 is color 2
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40)

	// Line 0: color 1
	vdp.WriteData(0xFF)
	vdp.WriteData(0x00)
	vdp.WriteData(0x00)
	vdp.WriteData(0x00)

	// Lines 1-6: color 0
	for line := 1; line < 7; line++ {
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
	}

	// Line 7: color 2
	vdp.WriteData(0x00)
	vdp.WriteData(0xFF)
	vdp.WriteData(0x00)
	vdp.WriteData(0x00)

	// Set up name table with vertical flip
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x78)
	vdp.WriteData(0x00)
	vdp.WriteData(0x04) // vFlip

	// Set colors
	vdp.WriteControl(0x01)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x03) // red

	vdp.WriteControl(0x02)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x0C) // green

	// Render line 0 - with vFlip, should show line 7's pattern (color 2)
	vdp.SetVCounter(0)
	vdp.LatchVScrollForFrame()
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	fb := vdp.Framebuffer()
	greenColor := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	// Line 0 should show color 2 (green) because vFlip maps line 0 to tile line 7
	for x := 0; x < 8; x++ {
		c := fb.RGBAAt(x, 0)
		if c != greenColor {
			t.Errorf("VFlipped tile pixel (%d, 0): expected green, got %v", x, c)
		}
	}
}

// TestVDP_RenderBackground_PaletteSelect tests sprite palette selection
func TestVDP_RenderBackground_PaletteSelect(t *testing.T) {
	vdp := NewVDP()

	// Enable display
	vdp.WriteControl(0x40)
	vdp.WriteControl(0x81)

	vdp.WriteControl(0x0E)
	vdp.WriteControl(0x82)

	// Create solid color 1 tile
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40)
	for line := 0; line < 8; line++ {
		vdp.WriteData(0xFF)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
	}

	// Set up name table with palette select (bit 11 = 1)
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x78)
	vdp.WriteData(0x00)
	vdp.WriteData(0x08) // palette 1 (sprite palette)

	// Set color 1 in BG palette (CRAM 1) to red
	vdp.WriteControl(0x01)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x03)

	// Set color 1 in sprite palette (CRAM 17) to blue
	vdp.WriteControl(17)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x30)

	vdp.SetVCounter(0)
	vdp.LatchVScrollForFrame()
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	fb := vdp.Framebuffer()
	blueColor := color.RGBA{R: 0, G: 0, B: 255, A: 255}

	// Should use sprite palette (blue)
	for x := 0; x < 8; x++ {
		c := fb.RGBAAt(x, 0)
		if c != blueColor {
			t.Errorf("Palette select pixel (%d): expected blue, got %v", x, c)
		}
	}
}

// TestVDP_RenderSprites_BasicSprite tests basic sprite rendering
func TestVDP_RenderSprites_BasicSprite(t *testing.T) {
	vdp := NewVDP()

	// Enable display
	vdp.WriteControl(0x40)
	vdp.WriteControl(0x81)

	// Set SAT base (register 5)
	// 0x7E gives SAT at $3F00
	vdp.WriteControl(0x7E)
	vdp.WriteControl(0x85)

	// Set sprite pattern base (register 6)
	// Bit 2 = 0: patterns at $0000
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x86)

	// Create sprite pattern at $0000 (solid color 1)
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40)
	for line := 0; line < 8; line++ {
		vdp.WriteData(0xFF)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
	}

	// Set up SAT Y positions (first 64 bytes at $3F00)
	// Sprite Y value is display_line - 1, so Y=9 displays at lines 10-17
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x7F) // VRAM write at $3F00
	vdp.WriteData(0x09)    // Y = 9 (displayed at Y=10 through Y=17)

	// Terminate rest with $D0 (only in 192-line mode)
	vdp.WriteData(0xD0) // Terminator

	// Set up SAT X and pattern (second 128 bytes at $3F80)
	vdp.WriteControl(0x80)
	vdp.WriteControl(0x7F) // VRAM write at $3F80
	vdp.WriteData(0x10)    // X = 16
	vdp.WriteData(0x00)    // Pattern 0

	// Set sprite color 1 (CRAM 17) to red
	vdp.WriteControl(17)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x03)

	// Set background to black
	vdp.WriteControl(0x00)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x00)

	// Render line 10 (sprite starts at Y=10)
	vdp.SetVCounter(10)
	vdp.LatchVScrollForFrame()
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	fb := vdp.Framebuffer()
	redColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	// Sprite should be at X=16, 8 pixels wide
	for x := 16; x < 24; x++ {
		c := fb.RGBAAt(x, 10)
		if c != redColor {
			t.Errorf("Sprite pixel (%d, 10): expected red, got %v", x, c)
		}
	}
}

// TestVDP_RenderSprites_Collision tests sprite collision detection
func TestVDP_RenderSprites_Collision(t *testing.T) {
	vdp := NewVDP()

	// Enable display
	vdp.WriteControl(0x40)
	vdp.WriteControl(0x81)

	vdp.WriteControl(0x7E)
	vdp.WriteControl(0x85)

	vdp.WriteControl(0x00)
	vdp.WriteControl(0x86)

	// Create solid sprite pattern
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40)
	for line := 0; line < 8; line++ {
		vdp.WriteData(0xFF)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
	}

	// Set up two overlapping sprites at line 10
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x7F)
	vdp.WriteData(0x09) // Sprite 0: Y = 9 (line 10)
	vdp.WriteData(0x09) // Sprite 1: Y = 9 (line 10)
	vdp.WriteData(0xD0) // Terminator

	// Both sprites at similar X position (overlapping)
	vdp.WriteControl(0x80)
	vdp.WriteControl(0x7F)
	vdp.WriteData(0x10) // Sprite 0: X = 16
	vdp.WriteData(0x00) // Pattern 0
	vdp.WriteData(0x14) // Sprite 1: X = 20 (overlaps with sprite 0)
	vdp.WriteData(0x00) // Pattern 0

	// Set sprite color
	vdp.WriteControl(17)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x03)

	// Clear collision flag
	vdp.ReadControl()

	vdp.SetVCounter(10)
	vdp.LatchVScrollForFrame()
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	// Check collision flag (status bit 5)
	status := vdp.GetStatus()
	if status&0x20 == 0 {
		t.Error("Sprite collision flag should be set")
	}
}

// TestVDP_RenderSprites_Overflow tests sprite overflow detection
func TestVDP_RenderSprites_Overflow(t *testing.T) {
	vdp := NewVDP()

	// Enable display
	vdp.WriteControl(0x40)
	vdp.WriteControl(0x81)

	vdp.WriteControl(0x7E)
	vdp.WriteControl(0x85)

	vdp.WriteControl(0x00)
	vdp.WriteControl(0x86)

	// Create sprite pattern
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40)
	for line := 0; line < 8; line++ {
		vdp.WriteData(0xFF)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
	}

	// Set up 9 sprites on line 10 (max is 8)
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x7F)
	for i := 0; i < 9; i++ {
		vdp.WriteData(0x09) // Y = 9 (line 10)
	}
	vdp.WriteData(0xD0) // Terminator

	// Set X positions for all sprites
	vdp.WriteControl(0x80)
	vdp.WriteControl(0x7F)
	for i := 0; i < 9; i++ {
		vdp.WriteData(uint8(i * 16)) // Spread across screen
		vdp.WriteData(0x00)
	}

	// Set sprite color
	vdp.WriteControl(17)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x03)

	// Clear flags
	vdp.ReadControl()

	vdp.SetVCounter(10)
	vdp.LatchVScrollForFrame()
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	// Check overflow flag (status bit 6)
	status := vdp.GetStatus()
	if status&0x40 == 0 {
		t.Error("Sprite overflow flag should be set")
	}
}

// TestVDP_RenderSprites_Height16 tests 8x16 sprite mode
func TestVDP_RenderSprites_Height16(t *testing.T) {
	vdp := NewVDP()

	// Enable display and 8x16 sprites (register 1 bit 1)
	vdp.WriteControl(0x42) // Display + 8x16 sprites
	vdp.WriteControl(0x81)

	vdp.WriteControl(0x7E)
	vdp.WriteControl(0x85)

	vdp.WriteControl(0x00)
	vdp.WriteControl(0x86)

	// Create pattern 0 (top half of sprite)
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40)
	for line := 0; line < 8; line++ {
		vdp.WriteData(0xFF) // Color 1
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
	}

	// Create pattern 1 (bottom half of sprite)
	for line := 0; line < 8; line++ {
		vdp.WriteData(0x00) // Color 2
		vdp.WriteData(0xFF)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
	}

	// Set up sprite at Y=9 (displays at lines 10-25 for 8x16)
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x7F)
	vdp.WriteData(0x09) // Y = 9 (line 10)
	vdp.WriteData(0xD0)

	vdp.WriteControl(0x80)
	vdp.WriteControl(0x7F)
	vdp.WriteData(0x10)
	vdp.WriteData(0x00) // Pattern 0 (bit 0 ignored in 8x16 mode)

	// Set colors
	vdp.WriteControl(17)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x03) // Color 1 = red

	vdp.WriteControl(18)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x0C) // Color 2 = green

	// Render line 10 (top of sprite)
	vdp.SetVCounter(10)
	vdp.LatchVScrollForFrame()
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	fb := vdp.Framebuffer()
	redColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	for x := 16; x < 24; x++ {
		c := fb.RGBAAt(x, 10)
		if c != redColor {
			t.Errorf("8x16 sprite top pixel (%d, 10): expected red, got %v", x, c)
		}
	}

	// Render line 18 (bottom half of sprite, line 8 of the 16-line sprite)
	vdp.SetVCounter(18)
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	greenColor := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	for x := 16; x < 24; x++ {
		c := fb.RGBAAt(x, 18)
		if c != greenColor {
			t.Errorf("8x16 sprite bottom pixel (%d, 18): expected green, got %v", x, c)
		}
	}
}

// TestVDP_RenderSprites_Terminator tests sprite list termination
func TestVDP_RenderSprites_Terminator(t *testing.T) {
	vdp := NewVDP()

	// Enable display
	vdp.WriteControl(0x40)
	vdp.WriteControl(0x81)

	vdp.WriteControl(0x7E)
	vdp.WriteControl(0x85)

	vdp.WriteControl(0x00)
	vdp.WriteControl(0x86)

	// Create sprite pattern
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40)
	for line := 0; line < 8; line++ {
		vdp.WriteData(0xFF)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
	}

	// Set up sprites with terminator at line 10
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x7F)
	vdp.WriteData(0x09) // Sprite 0 visible (line 10)
	vdp.WriteData(0xD0) // Terminator - should stop processing
	vdp.WriteData(0x09) // Sprite 2 should NOT be rendered

	vdp.WriteControl(0x80)
	vdp.WriteControl(0x7F)
	vdp.WriteData(0x10) // Sprite 0: X = 16
	vdp.WriteData(0x00)
	vdp.WriteData(0x00) // Sprite 1: (terminator)
	vdp.WriteData(0x00)
	vdp.WriteData(0x30) // Sprite 2: X = 48 (should not render)
	vdp.WriteData(0x00)

	vdp.WriteControl(17)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x03)

	vdp.SetVCounter(10)
	vdp.LatchVScrollForFrame()
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	fb := vdp.Framebuffer()
	redColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	blackColor := color.RGBA{R: 0, G: 0, B: 0, A: 255}

	// Sprite 0 should render
	for x := 16; x < 24; x++ {
		c := fb.RGBAAt(x, 10)
		if c != redColor {
			t.Errorf("Sprite 0 pixel (%d): expected red, got %v", x, c)
		}
	}

	// Sprite 2 should NOT render (X=48-55)
	for x := 48; x < 56; x++ {
		c := fb.RGBAAt(x, 10)
		if c != blackColor {
			t.Errorf("After terminator pixel (%d): expected black (no sprite), got %v", x, c)
		}
	}
}

// TestVDP_RenderSprites_Zoom tests zoomed sprites (2x size)
func TestVDP_RenderSprites_Zoom(t *testing.T) {
	vdp := NewVDP()

	// Enable display and sprite zoom (register 1 bit 0)
	vdp.WriteControl(0x41) // Display + zoom
	vdp.WriteControl(0x81)

	vdp.WriteControl(0x7E)
	vdp.WriteControl(0x85)

	vdp.WriteControl(0x00)
	vdp.WriteControl(0x86)

	// Create sprite pattern (only left pixel is color 1)
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40)
	for line := 0; line < 8; line++ {
		vdp.WriteData(0x80) // Only leftmost pixel
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
	}

	vdp.WriteControl(0x00)
	vdp.WriteControl(0x7F)
	vdp.WriteData(0x09) // Y = 9 (line 10)
	vdp.WriteData(0xD0)

	vdp.WriteControl(0x80)
	vdp.WriteControl(0x7F)
	vdp.WriteData(0x10)
	vdp.WriteData(0x00)

	vdp.WriteControl(17)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x03)

	vdp.SetVCounter(10)
	vdp.LatchVScrollForFrame()
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	fb := vdp.Framebuffer()
	redColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	// With zoom, single pixel becomes 2 pixels wide
	// X=16 and X=17 should both be red
	for x := 16; x < 18; x++ {
		c := fb.RGBAAt(x, 10)
		if c != redColor {
			t.Errorf("Zoomed sprite pixel (%d): expected red, got %v", x, c)
		}
	}
}

// TestVDP_RenderSprites_SpriteShift tests sprite left shift (register 0 bit 3)
func TestVDP_RenderSprites_SpriteShift(t *testing.T) {
	vdp := NewVDP()

	// Enable display and sprite shift (register 0 bit 3)
	vdp.WriteControl(0x08)
	vdp.WriteControl(0x80)

	vdp.WriteControl(0x40)
	vdp.WriteControl(0x81)

	vdp.WriteControl(0x7E)
	vdp.WriteControl(0x85)

	vdp.WriteControl(0x00)
	vdp.WriteControl(0x86)

	// Create sprite pattern
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40)
	for line := 0; line < 8; line++ {
		vdp.WriteData(0xFF)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
	}

	vdp.WriteControl(0x00)
	vdp.WriteControl(0x7F)
	vdp.WriteData(0x09) // Y = 9 (line 10)
	vdp.WriteData(0xD0)

	vdp.WriteControl(0x80)
	vdp.WriteControl(0x7F)
	vdp.WriteData(0x10) // X = 16, but shifted left by 8 = displays at X=8
	vdp.WriteData(0x00)

	vdp.WriteControl(17)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x03)

	vdp.SetVCounter(10)
	vdp.LatchVScrollForFrame()
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	fb := vdp.Framebuffer()
	redColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	// Sprite should be at X=8 (16-8 shift)
	for x := 8; x < 16; x++ {
		c := fb.RGBAAt(x, 10)
		if c != redColor {
			t.Errorf("Shifted sprite pixel (%d): expected red, got %v", x, c)
		}
	}
}

// TestVDP_RenderBackground_Priority tests background priority over sprites
func TestVDP_RenderBackground_Priority(t *testing.T) {
	vdp := NewVDP()

	// Enable display
	vdp.WriteControl(0x40)
	vdp.WriteControl(0x81)

	vdp.WriteControl(0x0E)
	vdp.WriteControl(0x82)

	vdp.WriteControl(0x7E)
	vdp.WriteControl(0x85)

	vdp.WriteControl(0x00)
	vdp.WriteControl(0x86)

	// Create BG tile pattern (color 1)
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x40)
	for line := 0; line < 8; line++ {
		vdp.WriteData(0xFF)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
		vdp.WriteData(0x00)
	}

	// Create sprite pattern (color 1 in sprite palette)
	// Same pattern, different color

	// Set up name table with priority bit (bit 12 = 1)
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x78)
	vdp.WriteData(0x00)
	vdp.WriteData(0x10) // Priority bit set

	// Set up sprite at same position
	vdp.WriteControl(0x00)
	vdp.WriteControl(0x7F)
	vdp.WriteData(0xFF)
	vdp.WriteData(0xD0)

	vdp.WriteControl(0x80)
	vdp.WriteControl(0x7F)
	vdp.WriteData(0x00) // X = 0 (same as BG tile)
	vdp.WriteData(0x00)

	// BG color 1 = green
	vdp.WriteControl(0x01)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x0C)

	// Sprite color 1 = red
	vdp.WriteControl(17)
	vdp.WriteControl(0xC0)
	vdp.WriteData(0x03)

	vdp.SetVCounter(0)
	vdp.LatchVScrollForFrame()
	vdp.LatchCRAM()
	vdp.LatchPerLineRegisters()
	vdp.RenderScanline()

	fb := vdp.Framebuffer()
	greenColor := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	// BG has priority, should show green (not red from sprite)
	for x := 0; x < 8; x++ {
		c := fb.RGBAAt(x, 0)
		if c != greenColor {
			t.Errorf("Priority pixel (%d): expected green (BG priority), got %v", x, c)
		}
	}
}

// TestVDP_ActiveHeight_224Mode tests 224-line display mode
func TestVDP_ActiveHeight_224Mode(t *testing.T) {
	vdp := NewVDP()

	// Set 224-line mode: M2=1 (reg0 bit1), M1=1 (reg1 bit4)
	vdp.WriteControl(0x02)
	vdp.WriteControl(0x80)

	vdp.WriteControl(0x50) // bit 4 + bit 6 (display enable)
	vdp.WriteControl(0x81)

	if got := vdp.ActiveHeight(); got != 224 {
		t.Errorf("224-line mode: expected 224, got %d", got)
	}

	// Line 223 should render
	vdp.SetVCounter(223)
	// Should not panic and should be within range
}

// TestVDP_GetHCounterForCycle tests H-counter table lookup
func TestVDP_GetHCounterForCycle(t *testing.T) {
	// Test boundary conditions
	testCases := []struct {
		cycle    int
		expected uint8
	}{
		{0, hCounterTable[0]},     // Start
		{227, hCounterTable[227]}, // End
		{-1, 0},                   // Negative (clamp)
		{228, hCounterTable[227]}, // Beyond range (clamp)
	}

	for _, tc := range testCases {
		got := GetHCounterForCycle(tc.cycle)
		if got != tc.expected {
			t.Errorf("GetHCounterForCycle(%d): expected 0x%02X, got 0x%02X", tc.cycle, tc.expected, got)
		}
	}

	// Verify table has correct structure
	// Phase 1 (0-85): H-counter $00-$7F
	if hCounterTable[0] != 0x00 {
		t.Errorf("hCounterTable[0]: expected 0x00, got 0x%02X", hCounterTable[0])
	}

	// Check for jump to $E9 in H-blank region
	foundE9 := false
	for i := 170; i < 228; i++ {
		if hCounterTable[i] >= 0xE9 || hCounterTable[i] <= 0x08 {
			foundE9 = true
			break
		}
	}
	if !foundE9 {
		t.Error("H-counter should include $E9+ values in H-blank region")
	}
}
