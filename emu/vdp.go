package emu

import (
	"image"
	"image/color"
)

// VDP timing constants (in CPU cycles within a scanline)
const (
	// Cycle at which VBlank interrupt is triggered
	// Real hardware fires VBlank slightly after scanline start
	VBlankInterruptCycle = 4
	// Cycle at which line counter decrements and line interrupt may fire
	// On real hardware, this happens around cycle 8-10 into the scanline
	LineInterruptCycle = 8
	// Cycle at which CRAM is latched for rendering
	// After line interrupt (cycle 8) has time to run, giving handlers ~6 cycles to modify CRAM
	CRAMLatchCycle = 14
)

// hCounterTable maps CPU cycle offset (0-227) to H-counter value (0-255)
// The SMS VDP master clock is 10.738 MHz (3x CPU clock). Each scanline is 684 master clocks = 228 CPU cycles.
// The H-counter is a 9-bit internal counter, but only the upper 8 bits are exposed via port $7E/$7F.
// This creates non-linear behavior with a jump from $93 to $E9 at H-blank start.
//
// Hardware timing per scanline:
//   - Master clocks 0-255 (CPU 0-85): H-counter $00-$7F (active display left)
//   - Master clocks 256-511 (CPU 85-170): H-counter $80-$93 (active display right)
//   - Master clocks 512+ (CPU 170+): H-counter jumps to $E9, counts to $FF, wraps to $00-$08 (H-blank)
var hCounterTable = func() [228]uint8 {
	var table [228]uint8

	// The VDP master clock runs at 3x CPU clock speed
	// 1 CPU cycle = 3 master clocks

	for cycle := 0; cycle < 228; cycle++ {
		masterClock := cycle * 3

		var hValue int
		if masterClock < 256 {
			// Phase 1: Active display left half
			// Master clocks 0-255 map to H-counter $00-$7F
			// 256 master clocks / 128 H-values = 2 clocks per H-count
			hValue = masterClock / 2
		} else if masterClock < 512 {
			// Phase 2: Active display right half
			// Master clocks 256-511 map to H-counter $80-$93 (20 values)
			// 256 master clocks / 20 H-values = 12.8 clocks per H-count
			progress := masterClock - 256
			hValue = 0x80 + (progress * 20 / 256)
			if hValue > 0x93 {
				hValue = 0x93
			}
		} else {
			// Phase 3: H-blank
			// Master clocks 512-683 map to H-counter $E9-$FF then $00-$08
			// Jump from $93 to $E9 (skipping $94-$E8)
			// $E9-$FF = 23 values, $00-$08 = 9 values, total 32 values over 172 clocks
			progress := masterClock - 512
			hValue = 0xE9 + (progress * 32 / 172)
			if hValue > 0xFF {
				// Wrap around: $FF+1 = $00
				hValue = hValue - 0x100
			}
		}

		table[cycle] = uint8(hValue)
	}

	return table
}()

// GetHCounterForCycle returns the H-counter value for a given cycle offset within a scanline
func GetHCounterForCycle(cycle int) uint8 {
	if cycle < 0 {
		return 0
	}
	if cycle >= 228 {
		return hCounterTable[227]
	}
	return hCounterTable[cycle]
}

type VDP struct {
	vram           [0x4000]uint8 // 16KB VRAM
	cram           [0x20]uint8   // 32 bytes CRAM (palette)
	cramLatch      [0x20]uint8   // Latched CRAM for rendering (latched at CRAMLatchCycle)
	register       [16]uint8     // VDP registers
	addr           uint16        // Current VRAM/CRAM address
	addrLatch      uint8         // First byte of control write
	writeLatch     bool          // True if first byte written
	codeReg        uint8         // Command code (bits 6-7 of second write)
	readBuffer     uint8         // Read buffer for VRAM reads
	status         uint8         // Status register
	vCounter       uint16        // Current scanline (raw)
	hCounter       uint8         // Horizontal counter
	lineCounter    int16         // Line interrupt counter
	lineIntPending bool          // Line interrupt pending flag
	bgPriority     [256]bool     // Background priority flags for current scanline
	framebuffer    *image.RGBA
	// Per-scanline latched values
	hScrollLatch uint8 // Latched hScroll for current scanline (per-scanline)
	reg2Latch    uint8 // Latched register 2 (name table base) for current scanline
	reg7Latch    uint8 // Latched backdrop color index for current scanline
	// Per-frame latched values (latched once at start of frame during VBlank)
	vScrollLatch uint8 // Latched vScroll for entire frame (per-frame, NOT per-scanline)
	// Region info for V-counter calculation
	totalScanlines int // 262 for NTSC, 313 for PAL

	// Interrupt state tracking
	statusWasRead          bool // Set when status register is read (flags cleared)
	interruptCheckRequired bool // Set when reg0/reg1 written, requiring interrupt state update

	// Pre-allocated for sprite collision detection (avoids per-scanline allocation)
	spritePixels []bool
}

// Palette scale: 2-bit SMS color to 8-bit RGB
var paletteScale = []uint8{0, 85, 170, 255}

func NewVDP() *VDP {
	return &VDP{
		framebuffer:    image.NewRGBA(image.Rect(0, 0, ScreenWidth, MaxScreenHeight)),
		totalScanlines: 262, // Default to NTSC
		lineCounter:    255, // Prevent spurious interrupt on first scanline
		spritePixels:   make([]bool, ScreenWidth),
	}
}

// SetTotalScanlines configures the VDP for the correct region timing
func (v *VDP) SetTotalScanlines(scanlines int) {
	v.totalScanlines = scanlines
}

// ReadVCounter returns the V-counter value with proper non-linear behavior
// The SMS V-counter jumps during vblank to fit 262/313 scanlines in 8 bits
func (v *VDP) ReadVCounter() uint8 {
	line := int(v.vCounter)
	activeHeight := v.ActiveHeight()

	if v.totalScanlines == 313 {
		// PAL timing (313 scanlines)
		switch activeHeight {
		case 192:
			// 192-line mode: 0-242 normal, 243-312 maps to 186-255
			if line <= 242 {
				return uint8(line)
			}
			return uint8(line - 57) // 243->186, 312->255
		case 224:
			// 224-line mode: 0-258 normal, 259-312 maps to 202-255
			if line <= 258 {
				return uint8(line)
			}
			return uint8(line - 57) // 259->202, 312->255
		}
	} else {
		// NTSC timing (262 scanlines)
		switch activeHeight {
		case 192:
			// 192-line mode: 0-218 normal, 219-261 maps to 213-255
			if line <= 218 {
				return uint8(line)
			}
			return uint8(line - 6) // 219->213, 261->255
		case 224:
			// 224-line mode: 0-234 normal, 235-261 maps to 229-255
			if line <= 234 {
				return uint8(line)
			}
			return uint8(line - 6) // 235->229, 261->255
		}
	}
	return uint8(line)
}

// ReadHCounter returns the horizontal counter
func (v *VDP) ReadHCounter() uint8 {
	return v.hCounter
}

// SetHCounter updates the horizontal counter
func (v *VDP) SetHCounter(h uint8) {
	v.hCounter = h
}

// ActiveHeight returns the active display height based on mode
// 192 lines: standard Mode 4 (default)
// 224 lines: M2=1, M1=1
// Where: M2 = reg0 bit 1, M1 = reg1 bit 4
func (v *VDP) ActiveHeight() int {
	m2 := v.register[0]&0x02 != 0 // Register 0 bit 1 - extended mode enable
	m1 := v.register[1]&0x10 != 0 // Register 1 bit 4

	// Only enable 224-line mode when both M2 and M1 are set
	// 240-line mode (M2=1, M1=0) is not supported on SMS
	if m2 && m1 {
		return 224
	}
	return 192
}

// ReadControl returns the status register and clears flags
func (v *VDP) ReadControl() uint8 {
	status := v.status
	// Clear VBlank, overflow, collision flags on read
	v.status &^= 0xE0 // Clear bits 7, 6, 5
	v.lineIntPending = false
	v.writeLatch = false   // Clear address latch (matches real hardware)
	v.statusWasRead = true // Signal that interrupt state needs updating
	return status
}

// StatusWasRead returns and clears the status-read flag.
// Used by emulator to update interrupt state only when needed.
func (v *VDP) StatusWasRead() bool {
	if v.statusWasRead {
		v.statusWasRead = false
		return true
	}
	return false
}

// InterruptCheckRequired returns and clears the interrupt check flag.
// Set when reg0 or reg1 is written (interrupt enable bits may have changed).
// Used by emulator to update interrupt state after register writes.
func (v *VDP) InterruptCheckRequired() bool {
	if v.interruptCheckRequired {
		v.interruptCheckRequired = false
		return true
	}
	return false
}

// WriteControl handles the two-write control port sequence
func (v *VDP) WriteControl(value uint8) {
	if !v.writeLatch {
		// First write: store low byte of address
		v.addrLatch = value
		v.writeLatch = true
	} else {
		// Second write: high byte + command code
		v.writeLatch = false
		v.addr = uint16(v.addrLatch) | (uint16(value&0x3F) << 8)
		v.codeReg = (value >> 6) & 0x03

		switch v.codeReg {
		case 0: // VRAM read setup
			// Pre-fetch byte into read buffer and increment address
			v.readBuffer = v.vram[v.addr&0x3FFF]
			v.addr = (v.addr + 1) & 0x3FFF
		case 1: // VRAM write setup
			// Nothing special needed
		case 2: // Register write
			regNum := value & 0x0F
			if regNum < 16 {
				v.register[regNum] = v.addrLatch
				// Interrupt enable bits are in reg0 bit 4 (line) and reg1 bit 5 (frame)
				// Writing to these registers may require interrupt state update
				if regNum == 0 || regNum == 1 {
					v.interruptCheckRequired = true
				}
			}
		case 3: // CRAM write setup
			// CRAM write mode
		}
	}
}

// ReadData returns data from VRAM
func (v *VDP) ReadData() uint8 {
	// Data port access clears the control write latch (matches real hardware)
	v.writeLatch = false
	data := v.readBuffer
	v.readBuffer = v.vram[v.addr&0x3FFF]
	v.addr = (v.addr + 1) & 0x3FFF
	return data
}

// WriteData writes to VRAM or CRAM depending on code register
func (v *VDP) WriteData(value uint8) {
	// Data port access clears the control write latch (matches real hardware)
	v.writeLatch = false
	// Writing to the data port also loads the value into the read buffer
	v.readBuffer = value
	if v.codeReg == 3 {
		// CRAM write
		cramAddr := v.addr & 0x1F
		v.cram[cramAddr] = value
	} else {
		// VRAM write
		v.vram[v.addr&0x3FFF] = value
	}
	v.addr = (v.addr + 1) & 0x3FFF
}

// cramToColor converts a CRAM entry to RGBA using the latched CRAM values
func (v *VDP) cramToColor(index uint8) color.RGBA {
	c := v.cramLatch[index&0x1F]
	r := (c >> 0) & 0x03
	g := (c >> 2) & 0x03
	b := (c >> 4) & 0x03
	return color.RGBA{
		R: paletteScale[r],
		G: paletteScale[g],
		B: paletteScale[b],
		A: 255,
	}
}

// SetVBlank sets the VBlank flag in the status register
func (v *VDP) SetVBlank() {
	v.status |= 0x80
}

// InterruptPending returns true if VDP wants to trigger an interrupt
func (v *VDP) InterruptPending() bool {
	// Frame interrupt: status bit 7 (VBlank) AND register 1 bit 5 (frame IE)
	frameInt := (v.status&0x80 != 0) && (v.register[1]&0x20 != 0)
	// Line interrupt: lineIntPending AND register 0 bit 4 (line IE)
	lineInt := v.lineIntPending && (v.register[0]&0x10 != 0)
	return frameInt || lineInt
}

// SetVCounter updates the current scanline
// This is called at the START of each scanline, BEFORE the CPU runs
// NOTE: Per-scanline register latching (hScroll, reg2, reg7) is done separately
// via LatchPerLineRegisters() AFTER line interrupts have had a chance to modify them
func (v *VDP) SetVCounter(line uint16) {
	v.vCounter = line
}

// LatchVScrollForFrame latches the vertical scroll register once per frame
// Per SMS VDP hardware, vScroll is locked at VBlank and cannot be changed mid-frame
// Any writes to Register 9 during active display are buffered until next frame
func (v *VDP) LatchVScrollForFrame() {
	v.vScrollLatch = v.register[9]
}

// LatchCRAM latches the CRAM palette for rendering
// Called at CRAMLatchCycle into each scanline, after line interrupt handlers have had time to modify CRAM
func (v *VDP) LatchCRAM() {
	copy(v.cramLatch[:], v.cram[:])
}

// LatchPerLineRegisters latches per-scanline registers (hScroll, reg2, reg7)
// Called at CRAMLatchCycle, AFTER line interrupts have had a chance to modify registers
// This ensures that register changes made in line interrupt handlers take effect on the current line
func (v *VDP) LatchPerLineRegisters() {
	v.hScrollLatch = v.register[8]
	v.reg2Latch = v.register[2]
	v.reg7Latch = v.register[7]
}

// UpdateLineCounter updates the line interrupt counter
// Should be called once per scanline
func (v *VDP) UpdateLineCounter() {
	activeHeight := uint16(v.ActiveHeight())

	// Per SMS VDP hardware documentation:
	// - Counter decrements on lines 0 through activeHeight (inclusive)
	// - Counter reloads on lines (activeHeight+1) through end of frame
	// For 192-line mode: decrement on 0-192, reload on 193-261
	// For 224-line mode: decrement on 0-224, reload on 225-261
	if v.vCounter <= activeHeight {
		// Active display + first VBlank line - decrement counter
		v.lineCounter--
		if v.lineCounter < 0 {
			// Counter underflow - reload and set interrupt pending
			v.lineCounter = int16(v.register[10])
			v.lineIntPending = true
		}
	} else {
		// VBlank period (after first line) - reload counter from register 10
		// No line interrupts are generated during this period
		v.lineCounter = int16(v.register[10])
	}
}

// RenderScanline renders the current scanline to the framebuffer
func (v *VDP) RenderScanline() {
	line := v.vCounter
	activeHeight := v.ActiveHeight()

	if int(line) >= activeHeight {
		return
	}

	// Clear priority flags for this scanline
	for i := range v.bgPriority {
		v.bgPriority[i] = false
	}

	// Check if display is enabled (register 1, bit 6)
	if v.register[1]&0x40 == 0 {
		// Display disabled - fill with backdrop color (using latched reg7)
		bgColor := v.cramToColor(16 + (v.reg7Latch & 0x0F))
		for x := 0; x < ScreenWidth; x++ {
			v.framebuffer.SetRGBA(x, int(line), bgColor)
		}
		return
	}

	// Render background first, then sprites on top
	v.renderBackground(line)
	v.renderSprites(line)

	// Left column blank (register 0 bit 5) - mask first 8 pixels with backdrop
	if v.register[0]&0x20 != 0 {
		bgColor := v.cramToColor(16 + (v.reg7Latch & 0x0F))
		for x := 0; x < 8; x++ {
			v.framebuffer.SetRGBA(x, int(line), bgColor)
		}
	}
}

// renderBackground renders the background layer for a scanline
func (v *VDP) renderBackground(line uint16) {
	// Get name table base address from register 2 (using latched value)
	// The calculation differs based on display mode:
	// - 192-line mode: (Reg2 & 0x0E) << 10
	// - 224/240-line mode: ((Reg2 & 0x0C) << 10) | 0x0700
	// We use the latched value to prevent mid-frame register changes from causing artifacts
	var nameTableBase uint16
	activeHeight := v.ActiveHeight()
	reg2 := v.reg2Latch // Use latched value, not current register
	if activeHeight == 192 {
		nameTableBase = uint16(reg2&0x0E) << 10
	} else {
		// 224 or 240 line mode - bit 1 is ignored, OR with 0x0700
		nameTableBase = (uint16(reg2&0x0C) << 10) | 0x0700
	}

	// Get scroll values from latched registers
	// Values are latched at the start of each scanline, so changes made during
	// the scanline (via line interrupts) take effect on the NEXT scanline
	hScroll := v.hScrollLatch
	vScroll := v.vScrollLatch

	// Top row scroll lock (register 0 bit 6) - top 2 rows ignore horizontal scroll
	topRowLock := v.register[0]&0x40 != 0
	// Right column scroll lock (register 0 bit 7) - right 8 columns ignore vertical scroll
	rightColLock := v.register[0]&0x80 != 0

	// Render 256 pixels (32 tiles × 8 pixels)
	for x := 0; x < ScreenWidth; x++ {
		// Determine effective scroll values based on lock bits
		effectiveHScroll := hScroll
		effectiveVScroll := vScroll

		// Top 2 tile rows (lines 0-15) ignore horizontal scroll if locked
		if topRowLock && line < 16 {
			effectiveHScroll = 0
		}

		// Right 8 columns (pixels 192-255) ignore vertical scroll if locked
		if rightColLock && x >= 192 {
			effectiveVScroll = 0
		}

		// Calculate the effective Y position with vertical scroll
		var effectiveY uint16
		if activeHeight == 224 {
			// 224-line mode: 256 modulo via bitmask
			effectiveY = (uint16(line) + uint16(effectiveVScroll)) & 0xFF
		} else {
			// 192-line mode: 224 modulo via conditional subtraction
			// Max value is 191 + 255 = 446 < 2 * 224, so it wraps at most once
			effectiveY = uint16(line) + uint16(effectiveVScroll)
			if effectiveY >= 224 {
				effectiveY -= 224
			}
		}

		// Which row of tiles (0-27)
		tileRow := effectiveY / 8
		// Which line within the tile (0-7)
		tileLine := effectiveY % 8

		// Calculate effective X with horizontal scroll
		// Note: horizontal scroll scrolls the screen LEFT (subtracts)
		effectiveX := (uint16(x) - uint16(effectiveHScroll)) & 0xFF

		// Which column of tiles (0-31)
		tileCol := effectiveX / 8
		// Which pixel within the tile (0-7)
		tilePixel := effectiveX % 8

		// Calculate name table entry address
		// Name table is 32×28 tiles, 2 bytes per entry
		nameTableAddr := nameTableBase + (tileRow*32+tileCol)*2

		// Read the 2-byte name table entry
		entryLo := v.vram[nameTableAddr&0x3FFF]
		entryHi := v.vram[(nameTableAddr+1)&0x3FFF]

		// Parse name table entry:
		// Bits 0-8: pattern index (9 bits, 512 patterns)
		// Bit 9: horizontal flip
		// Bit 10: vertical flip
		// Bit 11: palette select (0 = CRAM 0-15, 1 = CRAM 16-31)
		// Bit 12: priority (background in front of sprites)
		patternIndex := uint16(entryLo) | (uint16(entryHi&0x01) << 8)
		hFlip := (entryHi & 0x02) != 0
		vFlip := (entryHi & 0x04) != 0
		paletteSelect := (entryHi & 0x08) >> 3
		priority := (entryHi & 0x10) != 0

		// Calculate which line of the pattern to use
		patternLine := tileLine
		if vFlip {
			patternLine = 7 - tileLine
		}

		// Calculate pixel position within tile
		pixelPos := tilePixel
		if hFlip {
			pixelPos = 7 - tilePixel
		}

		// Get pattern data
		// Each pattern is 32 bytes (8 lines × 4 bytes per line)
		// 4 bytes per line = 4 bitplanes for 4bpp
		patternAddr := patternIndex*32 + patternLine*4

		// Read 4 bitplanes
		bp0 := v.vram[patternAddr&0x3FFF]
		bp1 := v.vram[(patternAddr+1)&0x3FFF]
		bp2 := v.vram[(patternAddr+2)&0x3FFF]
		bp3 := v.vram[(patternAddr+3)&0x3FFF]

		// Extract the pixel color from bitplanes
		// Bit 7 is leftmost pixel, bit 0 is rightmost
		shift := 7 - pixelPos
		colorIndex := ((bp0 >> shift) & 1) |
			(((bp1 >> shift) & 1) << 1) |
			(((bp2 >> shift) & 1) << 2) |
			(((bp3 >> shift) & 1) << 3)

		// Get color from CRAM
		// paletteSelect=0: CRAM 0-15, paletteSelect=1: CRAM 16-31
		cramIndex := uint8(paletteSelect)*16 + colorIndex
		pixelColor := v.cramToColor(cramIndex)

		v.framebuffer.SetRGBA(x, int(line), pixelColor)

		// Track priority: background is in front of sprites if priority bit set
		// and pixel is not transparent (colorIndex != 0)
		if priority && colorIndex != 0 {
			v.bgPriority[x] = true
		}
	}
}

// renderSprites renders sprites for a scanline
func (v *VDP) renderSprites(line uint16) {
	// Sprite Attribute Table base from register 5
	// Bits 1-6 × $100, typically $3F00
	satBase := uint16(v.register[5]&0x7E) << 7

	// Sprite height: 8 or 16 pixels (register 1 bit 1)
	spriteHeight := 8
	if v.register[1]&0x02 != 0 {
		spriteHeight = 16
	}

	// Zoomed sprites are 2x size (register 1 bit 0)
	zoom := 1
	zoomShift := 0
	if v.register[1]&0x01 != 0 {
		zoom = 2
		zoomShift = 1
	}
	effectiveHeight := spriteHeight * zoom

	// Sprite pattern base from register 6 (bit 2 selects $0000 or $2000)
	patternBase := uint16(v.register[6]&0x04) << 11

	// Sprite left shift (register 0 bit 3) - shifts all sprites left by 8 pixels
	spriteShift := 0
	if v.register[0]&0x08 != 0 {
		spriteShift = 8
	}

	// Get active height to determine sprite terminator behavior
	activeHeight := v.ActiveHeight()

	// Collect sprites on this line (max 8)
	type spriteInfo struct {
		x       int
		pattern uint8
		line    int // Line within sprite
	}
	var sprites [8]spriteInfo
	spriteCount := 0

	// Scan sprite Y positions (first 64 bytes of SAT)
	for i := 0; i < 64; i++ {
		y := int(v.vram[(satBase+uint16(i))&0x3FFF])

		// Y = 208 ($D0) terminates sprite list ONLY in 192-line mode
		// In 224 and 240 line modes, there is no early terminator
		if activeHeight == 192 && y == 208 {
			break
		}

		// Sprite Y is actually displayed at Y+1
		spriteY := y + 1

		// Check if sprite intersects this scanline
		if int(line) >= spriteY && int(line) < spriteY+effectiveHeight {
			if spriteCount >= 8 {
				// Sprite overflow - set status bit 6
				v.status |= 0x40
				break
			}

			// Get X and pattern from second part of SAT
			satAddr2 := satBase + 0x80 + uint16(i)*2
			spriteX := int(v.vram[satAddr2&0x3FFF]) - spriteShift
			pattern := v.vram[(satAddr2+1)&0x3FFF]

			// For 8x16 sprites, bit 0 of pattern is ignored
			if spriteHeight == 16 {
				pattern &= 0xFE
			}

			// Calculate which line of the sprite we're on
			spriteLine := (int(line) - spriteY) >> zoomShift

			sprites[spriteCount] = spriteInfo{
				x:       spriteX,
				pattern: pattern,
				line:    spriteLine,
			}
			spriteCount++
		}
	}

	// Render sprites in reverse order (sprite 0 has highest priority)
	// This means we draw from last to first, so earlier sprites overwrite later ones
	// Clear pre-allocated sprite pixel tracking array
	for i := range v.spritePixels {
		v.spritePixels[i] = false
	}

	for i := spriteCount - 1; i >= 0; i-- {
		spr := sprites[i]

		// Determine which pattern to use (for 8x16, top or bottom half)
		pattern := uint16(spr.pattern)
		spriteLine := spr.line
		if spriteHeight == 16 && spriteLine >= 8 {
			pattern++
			spriteLine -= 8
		}

		// Get pattern address
		patternAddr := patternBase + pattern*32 + uint16(spriteLine)*4

		// Read 4 bitplanes
		bp0 := v.vram[patternAddr&0x3FFF]
		bp1 := v.vram[(patternAddr+1)&0x3FFF]
		bp2 := v.vram[(patternAddr+2)&0x3FFF]
		bp3 := v.vram[(patternAddr+3)&0x3FFF]

		// Render 8 pixels (or 16 if zoomed)
		for px := 0; px < 8*zoom; px++ {
			screenX := spr.x + px
			if screenX < 0 || screenX >= ScreenWidth {
				continue
			}

			// Get pixel from pattern (accounting for zoom)
			patternPx := px >> zoomShift
			shift := uint(7 - patternPx)
			colorIndex := ((bp0 >> shift) & 1) |
				(((bp1 >> shift) & 1) << 1) |
				(((bp2 >> shift) & 1) << 2) |
				(((bp3 >> shift) & 1) << 3)

			// Color 0 is transparent
			if colorIndex == 0 {
				continue
			}

			// Check for sprite collision
			if v.spritePixels[screenX] {
				v.status |= 0x20 // Set collision flag
			}
			v.spritePixels[screenX] = true

			// Skip if background has priority at this pixel
			if v.bgPriority[screenX] {
				continue
			}

			// Draw sprite pixel - sprites always use CRAM 16-31 in Mode 4
			pixelColor := v.cramToColor(colorIndex + 16)
			v.framebuffer.SetRGBA(screenX, int(line), pixelColor)
		}
	}
}

// Framebuffer returns the current framebuffer
func (v *VDP) Framebuffer() *image.RGBA {
	return v.framebuffer
}

// GetVRAM returns the VRAM contents
func (v *VDP) GetVRAM() []uint8 {
	return v.vram[:]
}

// GetCRAM returns the CRAM (palette) contents
func (v *VDP) GetCRAM() []uint8 {
	return v.cram[:]
}

// GetRegister returns the value of a VDP register (0-15)
func (v *VDP) GetRegister(n int) uint8 {
	if n < 0 || n >= len(v.register) {
		return 0
	}
	return v.register[n]
}

// GetAddress returns the current VRAM/CRAM address
func (v *VDP) GetAddress() uint16 {
	return v.addr
}

// GetCodeReg returns the code register (command type)
func (v *VDP) GetCodeReg() uint8 {
	return v.codeReg
}

// GetWriteLatch returns whether a control write is pending
func (v *VDP) GetWriteLatch() bool {
	return v.writeLatch
}

// GetStatus returns the status register without clearing flags
func (v *VDP) GetStatus() uint8 {
	return v.status
}

// GetLineCounter returns the line interrupt counter
func (v *VDP) GetLineCounter() int16 {
	return v.lineCounter
}

// GetLineIntPending returns the line interrupt pending flag
func (v *VDP) GetLineIntPending() bool {
	return v.lineIntPending
}

// LeftColumnBlankEnabled returns true if VDP register 0 bit 5 is set,
// indicating the leftmost 8 pixels are masked with backdrop color
func (v *VDP) LeftColumnBlankEnabled() bool {
	return v.register[0]&0x20 != 0
}
