# Sega Master System VDP Technical Reference

Technical reference for the Video Display Processor (VDP) used in the Sega Master
System. This document focuses on the SMS hardware. Game Gear notes are included
where relevant since the GG VDP is a close derivative.

## Table of Contents

- [VDP Chip Variants](#vdp-chip-variants)
- [I/O Port Mapping](#io-port-mapping)
- [Control Port](#control-port)
- [Data Port](#data-port)
- [Status Register](#status-register)
- [VDP Registers](#vdp-registers)
- [Video Memory (VRAM)](#video-memory-vram)
- [Color RAM (CRAM)](#color-ram-cram)
- [Name Table](#name-table)
- [Tile Pattern Format](#tile-pattern-format)
- [Sprite System](#sprite-system)
- [Scrolling](#scrolling)
- [Display Modes](#display-modes)
- [Display Timing](#display-timing)
- [V-Counter](#v-counter)
- [H-Counter](#h-counter)
- [Interrupts](#interrupts)
- [Register Latching](#register-latching)
- [SMS1 vs SMS2 VDP Differences](#sms1-vs-sms2-vdp-differences)
- [240-Line Mode Analysis](#240-line-mode-analysis)
- [Game Gear VDP Notes](#game-gear-vdp-notes)
- [Sources](#sources)

---

## VDP Chip Variants

The Sega 8-bit family used several VDP revisions. The Genesis/Mega Drive VDP
(315-5313) is a completely different chip and is **not** covered here.

| Chip     | System                             | Notes                                     |
|----------|------------------------------------|-----------------------------------------|
| 315-5124 | Mark III, Master System (SMS1)     | Original VDP; TMS9918 derivative; no extended height modes |
| 315-5246 | Master System II (SMS2), late SMS1 | Fixes sprite zoom bug; adds 224/240-line modes |
| 315-5378 | Game Gear                          | Functionally identical to 315-5246 + 12-bit CRAM, 160x144 viewport |

All SMS/GG VDP chips include an integrated SN76489-compatible PSG (Programmable
Sound Generator) accessible through the same I/O port range.

---

## I/O Port Mapping

The VDP is accessed through Z80 I/O ports. The SMS uses **partial address
decoding** on bits A7, A6, and A0. This means multiple port addresses map to the
same VDP function.

| Port Range   | Read                 | Write                |
|-------------|----------------------|----------------------|
| `$40`-`$7F` even | V-counter       | SN76489 PSG data    |
| `$40`-`$7F` odd  | H-counter       | SN76489 PSG data    |
| `$80`-`$BF` even | VDP data port   | VDP data port       |
| `$80`-`$BF` odd  | VDP control port (status register) | VDP control port (command word) |

The canonical port addresses are `$7E`/`$7F` for counters/PSG and `$BE`/`$BF`
for VDP data/control, but any address in the mirrored range works identically.

---

## Control Port

### Writing: Two-Byte Command Word

The VDP control port uses a **two-byte write sequence**. An internal flag tracks
whether the next write is the first or second byte.

```
Byte 1 (first write):   A07 A06 A05 A04 A03 A02 A01 A00
Byte 2 (second write):  CD1 CD0 A13 A12 A11 A10 A09 A08
```

- Bits A13-A00: 14-bit VRAM address
- Bits CD1-CD0: 2-bit code register (command type)

**Code register values:**

| Code | Operation        | Effect                                                    |
|------|-----------------|-----------------------------------------------------------|
| `00` | VRAM read       | Pre-fetches byte at address into read buffer; increments address |
| `01` | VRAM write      | Sets address for subsequent data port writes               |
| `10` | Register write  | Byte 1 = data, low 4 bits of byte 2 = register number (0-15) |
| `11` | CRAM write      | Sets address for subsequent CRAM writes via data port      |

**Register write format:**

```
Byte 1:  D07 D06 D05 D04 D03 D02 D01 D00   (register data)
Byte 2:  1   0   x   x   R03 R02 R01 R00   (register number)
```

### Write Latch Reset

The internal first/second byte flag is reset by any of the following:

- Completing the two-byte sequence (second byte written)
- Reading the control port (status register read)
- Reading the data port
- Writing to the data port

This means any data port access or status read will cause the next control port
write to be treated as the first byte.

### Reading: Status Register

Reading the control port returns the status register. See
[Status Register](#status-register) for details.

---

## Data Port

### VRAM Read (Code = `00`)

VRAM reads use a **buffered mechanism** with a one-byte lag:

1. Setting the address via the command word immediately pre-fetches the byte at
   that address into an internal read buffer and increments the address register.
2. Reading the data port returns the **contents of the read buffer** (not the
   byte at the current address), then loads the byte at the current address into
   the buffer and increments the address.

This means the first read after setting an address returns the pre-fetched value
from the address that was set, while subsequent reads are offset by one.

### VRAM Write (Code = `01`)

1. Writing to the data port stores the value in VRAM at the current address.
2. The written value is also placed into the read buffer (side effect).
3. The address register increments by 1.
4. Address wraps from `$3FFF` to `$0000` (14-bit address space).

### CRAM Write (Code = `11`)

1. Writing to the data port stores the value in CRAM at `address & $1F`.
2. The address register increments by 1 (wraps at `$3FFF`).
3. CRAM writes take effect immediately. Writing to CRAM during active display can
   cause visual artifacts because the VDP has no write FIFO.

### Address Auto-Increment

All data port reads and writes auto-increment the 14-bit address register.
The address wraps from `$3FFF` back to `$0000`.

---

## Status Register

Reading the control port returns the status register and **clears all flag bits**
(7, 6, 5). It also clears the internal line interrupt pending flag and resets the
control port write latch.

```
Bit 7: INT  - Frame interrupt pending (set at VBlank)
Bit 6: OVR  - Sprite overflow (more than 8 sprites on a scanline)
Bit 5: COL  - Sprite collision (opaque pixels of two sprites overlap)
Bits 4-0: Not used on SMS/SMS2 (return 0). In TMS9918 modes, bits 4-0
           contain the index of the first overflowed sprite.
```

All three flags are **sticky**: once set, they remain set until the status
register is read. Reading the status register clears all three flags
simultaneously.

---

## VDP Registers

Registers 0-10 (`$00`-`$0A`) are functional. Writes to registers 11-15
(`$0B`-`$0F`) have no effect.

### Register $00 -- Mode Control No. 1

| Bit | Name | Function |
|-----|------|----------|
| 7 | VS  | Vertical scroll lock: columns 24-31 (pixels 192-255) ignore vertical scroll |
| 6 | HS  | Horizontal scroll lock: rows 0-1 (lines 0-15) ignore horizontal scroll |
| 5 | LCB | Left column blank: mask leftmost 8 pixels with overscan color from Register $07 |
| 4 | IE1 | Line interrupt enable |
| 3 | EC  | Shift all sprites left by 8 pixels |
| 2 | M4  | Mode 4 enable (must be set for SMS display mode) |
| 1 | M2  | Extended height enable (used with M1/M3 for 224/240-line modes; SMS2/GG only) |
| 0 | ES  | External sync / monochrome. On SMS1 VDP (315-5124), setting this causes gradual display brightness/color fade and eventual sync loss. No effect on SMS2/GG. |

### Register $01 -- Mode Control No. 2

| Bit | Name | Function |
|-----|------|----------|
| 7 | --  | No effect in Mode 4 |
| 6 | BLK | Display enable: 1 = display visible, 0 = display blanked (backdrop color fills screen) |
| 5 | IE0 | Frame interrupt (VBlank) enable |
| 4 | M1  | 224-line mode select (when M2=1) |
| 3 | M3  | 240-line mode select (when M2=1) |
| 2 | --  | No effect |
| 1 | SZ  | Sprite size: 0 = 8x8, 1 = 8x16 (Mode 4); In TMS9918 modes: 16x16 |
| 0 | MAG | Sprite zoom/doubling: each sprite pixel is doubled (8x8 becomes 16x16, 8x16 becomes 16x32) |

### Register $02 -- Name Table Base Address

Determines the base address of the name table (tilemap) in VRAM.

**192-line mode:**
```
Base address = (Reg2 & $0E) << 10
```
Bits 3-1 select from 8 possible base addresses in `$0800` increments.

**224-line mode (SMS2/GG only):**
```
Base address = ((Reg2 & $0C) << 10) | $0700
```
Only bits 3-2 are used. Possible bases: `$0700`, `$1700`, `$2700`, `$3700`.

**SMS1 VDP quirk:** Bit 0 acts as an AND mask on bit 10 of the name table row
address. When bit 0 is cleared, the lower 8 rows of the name table mirror the
upper 16 rows. The Japanese version of Ys is the only known game that relies on
this behavior.

### Register $03 -- Color Table Base Address

Not used in Mode 4. Should be set to `$FF` for normal operation on SMS1 VDP
(unused bits can affect VRAM addressing in TMS9918 legacy modes).

### Register $04 -- Background Pattern Generator Base Address

Not used in Mode 4. Lower 3 bits should be set to 1 for normal operation on
SMS1 VDP.

### Register $05 -- Sprite Attribute Table (SAT) Base Address

```
SAT base = (Reg5 & $7E) << 7
```
Bits 6-1 define bits 13-8 of the SAT base address. Bit 0 is unused on SMS2/GG.

**SMS1 VDP quirk:** Bit 0 controls whether sprite data is fetched from the upper
or lower 128 bytes of the SAT range.

### Register $06 -- Sprite Pattern Generator Base Address

```
Sprite pattern base = (Reg6 & $04) << 11
```
Bit 2 selects the sprite pattern base: 0 = `$0000`, 1 = `$2000`.

**SMS1 VDP quirk:** Bits 1-0 act as AND masks over bits 8 and 6 of the sprite
tile index when cleared. This has no effect on SMS2/GG.

### Register $07 -- Overscan/Backdrop Color

Bits 3-0 select a color index from the **sprite palette** (CRAM 16-31) as the
backdrop/overscan color. The full CRAM index is `16 + (Reg7 & $0F)`.

This color fills:
- The border area around the active display
- The masked left column when LCB (Register $00, bit 5) is enabled
- The entire active display when the display is disabled (Register $01 bit 6 = 0)

Note: Background tile color index 0 is **not** replaced by the backdrop. It
renders as a normal palette lookup (`CRAM[paletteSelect*16 + 0]`). Color index 0
is only "transparent" in the sense that sprites show through it regardless of
the tile's priority bit.

### Register $08 -- Background X Scroll (Horizontal Scroll)

All 8 bits define the horizontal scroll offset (0-255 pixels). Scrolling moves
the background to the **right** (the viewport shifts left).

- Upper 5 bits: coarse scroll (starting column = `(32 - (Reg8 >> 3)) & 31`)
- Lower 3 bits: fine scroll (0-7 pixel sub-tile offset)

When fine scroll is non-zero, the leftmost 1-7 pixels show garbage from the
wrapped column edge. Games typically enable left column blank (Register $00,
bit 5) to hide this seam.

### Register $09 -- Background Y Scroll (Vertical Scroll)

All 8 bits define the vertical scroll offset.

- Upper 5 bits: coarse scroll (starting row)
- Lower 3 bits: fine scroll (0-7 pixel sub-tile offset)

**Vertical scroll is latched once per frame** during VBlank. Changes to
Register $09 during active display do NOT take effect until the next frame.
This is a hardware design characteristic.

**Scroll wrapping:**
- 192-line mode: wraps at 224 (28 rows x 8 pixels)
- 224-line mode: wraps at 256 (32 rows x 8 pixels)

Values above the wrap point wrap around to the top of the name table. For
192-line mode, this means values 224-255 wrap to rows 0-3.

### Register $0A -- Line Counter

All 8 bits define the reload value for the line interrupt counter. See
[Interrupts](#interrupts) for detailed counter behavior.

---

## Video Memory (VRAM)

### Size and Addressing

- **16 KB** total (`$0000`-`$3FFF`)
- 14-bit address register, wraps at `$3FFF`
- Accessible only through the VDP data port (not memory-mapped)

### Typical VRAM Layout (Mode 4)

| Address Range     | Content                    | Size        |
|------------------|---------------------------|-------------|
| `$0000`-`$1FFF` | Tile patterns 0-255       | 8,192 bytes |
| `$2000`-`$37FF` | Tile patterns 256-447     | 6,144 bytes |
| `$3800`-`$3EFF` | Name table (tilemap)      | 1,792 bytes |
| `$3F00`-`$3FFF` | Sprite Attribute Table    | 256 bytes   |

This canonical layout requires Register $02 = `$FF` (name table at `$3800`),
Register $05 = `$FF` (SAT at `$3F00`), Register $06 = `$FB` (sprite patterns
at `$0000`).

Alternatively, Register $06 = `$FF` places sprite patterns at `$2000`, allowing
up to 192 unique sprite tile patterns separate from background patterns.

### Total Tile Capacity

512 tiles maximum (`$0000`-`$3FFF` / 32 bytes per tile = 512). In practice,
the name table and SAT consume some of this space, leaving approximately 448
tiles available with the canonical layout above.

---

## Color RAM (CRAM)

### SMS Palette

- **32 bytes** of dedicated color memory (not part of VRAM)
- Organized as two 16-color palettes:
  - Palette 0 (CRAM bytes 0-15): background palette
  - Palette 1 (CRAM bytes 16-31): sprite palette (also selectable for background tiles)
- Write-only on real hardware (reads return garbage)

**Color format:** `--BBGGRR` (6-bit color)

| Bits | Channel | Range |
|------|---------|-------|
| 1-0  | Red     | 0-3   |
| 3-2  | Green   | 0-3   |
| 5-4  | Blue    | 0-3   |

**2-bit to 8-bit scaling:** `{0, 85, 170, 255}` (0x00, 0x55, 0xAA, 0xFF)

**Palette capacity:** 64 possible colors, 32 on screen simultaneously.

### Color Index 0 and Transparency

- In background tiles, color index 0 is a normal palette lookup
  (`CRAM[paletteSelect*16 + 0]`). It is "transparent" only for priority
  purposes: sprites show through it even when the tile's priority bit is set.
- In sprites, color index 0 is transparent (the pixel is not drawn and the
  background shows through).

### CRAM Address Wrapping

CRAM addresses wrap at 32 bytes (`address & $1F`). Writing with CRAM code
selected at addresses beyond 31 wraps back around.

---

## Name Table

### Structure

The name table is a grid of 16-bit (2-byte, little-endian) tile entries:

- **192-line mode:** 32 columns x 28 rows = 1,792 bytes
- **224-line mode:** 32 columns x 32 rows = 2,048 bytes

### Entry Format

```
  F   E   D   C   B   A   9   8   7   6   5   4   3   2   1   0
+---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+
| - | - | - | p | c | v | h |          pattern index            |
+---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+
```

| Bits  | Field          | Description                                          |
|-------|----------------|------------------------------------------------------|
| 15-13 | Unused         | Can be used by software for collision flags etc.      |
| 12    | Priority (p)   | 1 = background draws OVER sprites (when pixel is non-transparent) |
| 11    | Palette (c)    | 0 = palette 0 (CRAM 0-15), 1 = palette 1 (CRAM 16-31) |
| 10    | V-flip (v)     | Vertical flip                                        |
| 9     | H-flip (h)     | Horizontal flip                                      |
| 8-0   | Pattern index  | 9-bit index into tile pattern table (0-511)          |

### Priority Behavior

The SMS uses **per-tile** priority (not per-sprite like the NES):

- Priority = 0: sprites draw over the background tile
- Priority = 1 AND background pixel color index is non-zero: background draws over sprites
- Priority = 1 BUT background pixel color index is 0 (transparent): sprites show through

---

## Tile Pattern Format

Each tile is **8x8 pixels, 4 bits per pixel (4bpp)**, consuming **32 bytes**:

```
8 pixels x 8 rows x 4 bits per pixel / 8 bits per byte = 32 bytes per tile
```

### Byte Layout

Each row is stored as 4 consecutive bytes, one per bitplane:

```
Bytes  0-3:  Row 0 -- bitplane 0, bitplane 1, bitplane 2, bitplane 3
Bytes  4-7:  Row 1 -- bitplane 0, bitplane 1, bitplane 2, bitplane 3
  ...
Bytes 28-31: Row 7 -- bitplane 0, bitplane 1, bitplane 2, bitplane 3
```

### Pixel Color Extraction

Within each byte, **bit 7 is the leftmost pixel** and **bit 0 is the rightmost**.
The 4-bit color index for pixel position `p` (0=left, 7=right) is assembled:

```
shift = 7 - p
color = ((bp0 >> shift) & 1)       |
        ((bp1 >> shift) & 1) << 1  |
        ((bp2 >> shift) & 1) << 2  |
        ((bp3 >> shift) & 1) << 3
```

This interleaved bitplane format is unique to the SMS. It differs from the
Genesis (packed 4-bit nibbles per byte) and NES (sequential full bitplanes
per tile).

---

## Sprite System

### Sprite Attribute Table (SAT)

The SAT is a 256-byte table in VRAM (typically at `$3F00`):

| Byte Range   | Content                                            |
|-------------|---------------------------------------------------|
| `$00`-`$3F` | Y coordinates for sprites 0-63 (1 byte each)     |
| `$40`-`$7F` | Unused (available as extra tile storage -- 2 tiles worth) |
| `$80`-`$FF` | X coordinate + pattern index pairs for sprites 0-63 (2 bytes each) |

For sprite `i`:
- Y coordinate: `SAT_BASE + i`
- X coordinate: `SAT_BASE + $80 + (i * 2)`
- Pattern index: `SAT_BASE + $80 + (i * 2) + 1`

### Sprite Positioning

**Y coordinate:** The sprite is displayed at **Y + 1**. A stored value of 0
places the sprite starting at scanline 1.

**X coordinate:** Directly specifies the horizontal pixel position (0-255).
Sprites that extend past the right screen edge are clipped (they do not wrap).

**Sprite left shift:** When Register $00 bit 3 (EC) is set, all sprite X
positions are shifted left by 8 pixels (X = stored_X - 8). This allows sprites
to smoothly enter the screen from the left edge, since X coordinates are
unsigned and cannot be negative otherwise.

### Sprite Sizes

Controlled by Register $01 bits 1 and 0:

| Reg $01 Bit 1 (SZ) | Reg $01 Bit 0 (MAG) | Base Size | Effective Pixel Size |
|-------|-------|---------|------------|
| 0     | 0     | 8x8     | 8x8        |
| 0     | 1     | 8x8     | 16x16 (zoomed) |
| 1     | 0     | 8x16    | 8x16       |
| 1     | 1     | 8x16    | 16x32 (zoomed) |

When MAG (zoom) is set, each pixel of the sprite pattern is doubled in both
dimensions.

For **8x16 sprites**, bit 0 of the pattern index is forced to 0. The top
half uses the even-numbered pattern and the bottom half uses the next
(odd-numbered) pattern.

### `$D0` Terminator

In **192-line mode only**, a Y coordinate value of `$D0` (208) terminates
sprite processing. The sprite with Y = `$D0` and all subsequent sprites are
not displayed and not checked.

In **224-line mode**, `$D0` has **no special meaning** and is
treated as a normal Y coordinate.

### Sprite Pattern Base

Register $06 bit 2 acts as a global bit 8 for all sprite pattern indices:

- Bit 2 = 0: sprite patterns at `$0000`-`$1FFF` (indices 0-255)
- Bit 2 = 1: sprite patterns at `$2000`-`$3FFF` (indices 256-511)

### Per-Scanline Rendering

1. The VDP scans all 64 Y coordinates in the SAT.
2. For each sprite whose Y range intersects the current scanline, it is added
   to an 8-entry internal display buffer.
3. Processing stops when: all 64 sprites are checked, 8 buffer slots are
   filled, or `$D0` is encountered (192-line mode only).
4. Sprites are rendered with **sprite 0 having highest priority** (drawn on top
   of all others). Implementation typically renders in reverse order so that
   lower-numbered sprites overwrite higher-numbered ones.

### Sprite Palette

Sprites **always** use palette 1 (CRAM 16-31). Color index 0 is transparent.

### Sprite Overflow (Status Bit 6)

Set when more than 8 sprites have Y ranges that intersect a single scanline.
The 9th and subsequent sprites are not rendered. This flag is sticky and
persists until the status register is read.

### Sprite Collision (Status Bit 5)

Set when any two sprites have **opaque (non-transparent, color index > 0)
pixels** that overlap at the same screen position. The VDP does not indicate
which sprites collided. This flag is sticky and persists until the status
register is read.

Games typically use this flag as a coarse trigger for more precise
software-based collision detection.

### SMS1 VDP Sprite Zoom Bug

On the SMS1 VDP (315-5124), when sprite zoom (MAG) is enabled, **only the
first 4 of the 8 sprites** in the per-scanline buffer are zoomed both
horizontally and vertically. The remaining 4 are zoomed vertically only
(horizontal pixels are not doubled). The SMS2 VDP (315-5246) and Game Gear
VDP fix this, correctly zooming all 8 sprites.

---

## Scrolling

### Horizontal Scrolling (Register $08)

- Full 8-bit range: 0-255 pixels
- Scrolls the background to the **right** (viewport shifts left)
- Upper 5 bits = coarse scroll (column offset)
- Lower 3 bits = fine scroll (pixel offset within a tile)

**Scroll seam:** When fine scroll is non-zero, the leftmost 1-7 pixels are
filled with the backdrop color rather than tile data from the wrapped column.
Games typically enable left column blank (Register $00 bit 5) to hide this.

**Per-scanline capability:** Horizontal scroll can be changed each scanline
using line interrupts, enabling split-screen, wavy distortion, and parallax
scrolling effects.

### Vertical Scrolling (Register $09)

- Full 8-bit range: 0-255 pixels
- **Latched once per frame** during VBlank. Changes during active display are
  buffered until the next frame.
- Wraps at 224 pixels in 192-line mode, 256 pixels in 224-line mode

### Scroll Locking

Two register bits allow parts of the screen to be exempt from scrolling, useful
for status bars:

- **Register $00 bit 6 (HS lock):** Rows 0-1 (lines 0-15) always display with
  horizontal scroll = 0, regardless of Register $08. Used for top status bars.
- **Register $00 bit 7 (VS lock):** Columns 24-31 (pixels 192-255) always
  display with vertical scroll = 0, regardless of Register $09. Used for right
  side status bars.

---

## Display Modes

### Mode Selection

Display mode is controlled by 4 bits across two registers:

| Bit | Register | Name |
|-----|----------|------|
| M4  | Reg $00 bit 2 | Mode 4 enable |
| M2  | Reg $00 bit 1 | Extended height enable |
| M1  | Reg $01 bit 4 | Height select |
| M3  | Reg $01 bit 3 | Height select |

### Mode 4 (Primary SMS Mode)

When M4 = 1, the VDP operates in Mode 4. The resolution depends on the M2, M1,
and M3 bits:

| M4 | M2 | M1 | M3 | Resolution | Notes |
|----|----|----|-----|-----------|-------|
| 1  | 0  | x  | x  | 256x192   | Standard display (default) |
| 1  | 1  | 1  | 0  | 256x224   | Extended height (SMS2/GG only) |
| 1  | 1  | 0  | 1  | 256x240   | Extended height (SMS2/GG only; see [240-line analysis](#240-line-mode-analysis)) |
| 1  | 1  | 1  | 1  | 256x192   | Both M1+M3 set = falls back to 192-line |

On the SMS1 VDP (315-5124), M2/M1/M3 are ignored and Mode 4 always produces
192-line output.

### Legacy TMS9918 Modes (Modes 0-3)

When M4 = 0, the VDP falls back to TMS9918A-compatible display modes:

| M4 | M2 | M1 | M3 | Mode | Description |
|----|----|----|-----|------|-------------|
| 0  | 0  | 0  | 0  | Graphics I | 32x24 tiles, 2 colors per 8-character group, 32 sprites |
| 0  | 0  | 1  | 0  | Text | 40x24 characters, 2 colors screen-wide, no sprites |
| 0  | 1  | 0  | 0  | Graphics II | Like Graphics I, 2 colors per character row, 32 sprites |
| 0  | 0  | 0  | 1  | Multicolor | 64x48 blocks of 4x4 pixels, 32 sprites |

These modes use a fixed 15-color palette inherited from the TMS9918. On SMS
hardware, these colors are approximated through CRAM initialization and appear
dimmer than on original TMS9918 hardware.

No commercially released SMS game uses TMS9918 legacy modes. They exist only
for backward compatibility with SG-1000/SC-3000 software.

---

## Display Timing

### Scanline Structure

Each scanline is **228 CPU cycles** (684 master clocks). The master clock runs
at 10.738635 MHz, which is 3x the CPU clock.

| Region | CPU Clock       | Master Clock    | Scanlines/Frame | FPS |
|--------|----------------|-----------------|-----------------|-----|
| NTSC   | 3.579545 MHz   | 10.738635 MHz   | 262             | ~60 |
| PAL    | 3.546893 MHz   | 10.640679 MHz   | 313             | ~50 |

### Frame Structure (NTSC, 192-Line Mode)

| Scanlines   | Count | Content          |
|------------|-------|------------------|
| 0-191      | 192   | Active display   |
| 192-215    | 24    | Bottom border    |
| 216-218    | 3     | Bottom blanking  |
| 219-221    | 3     | Vertical sync    |
| 222-234    | 13    | Top blanking     |
| 235-261    | 27    | Top border       |
| **Total**  | **262** |                |

### Frame Structure (NTSC, 224-Line Mode)

| Scanlines   | Count | Content          |
|------------|-------|------------------|
| 0-223      | 224   | Active display   |
| 224-231    | 8     | Bottom border    |
| 232-234    | 3     | Bottom blanking  |
| 235-237    | 3     | Vertical sync    |
| 238-250    | 13    | Top blanking     |
| 251-261    | 11    | Top border       |
| **Total**  | **262** |                |

### Frame Structure (PAL, 192-Line Mode)

| Scanlines   | Count | Content          |
|------------|-------|------------------|
| 0-191      | 192   | Active display   |
| 192-239    | 48    | Bottom border    |
| 240-242    | 3     | Bottom blanking  |
| 243-245    | 3     | Vertical sync    |
| 246-258    | 13    | Top blanking     |
| 259-312    | 54    | Top border       |
| **Total**  | **313** |                |

### Frame Structure (PAL, 224-Line Mode)

| Scanlines   | Count | Content          |
|------------|-------|------------------|
| 0-223      | 224   | Active display   |
| 224-255    | 32    | Bottom border    |
| 256-258    | 3     | Bottom blanking  |
| 259-261    | 3     | Vertical sync    |
| 262-274    | 13    | Top blanking     |
| 275-312    | 38    | Top border       |
| **Total**  | **313** |                |

### Scanline-Level Timing

Within each 228-cycle scanline, key events occur at specific cycle offsets:

| Cycle | Event |
|-------|-------|
| ~4    | VBlank interrupt check (on VBlank scanline only) |
| ~8    | Line counter decrement; line interrupt may fire |
| ~14   | CRAM and per-scanline registers latched for rendering |
| 0-~170 | Active display pixel rendering |
| ~170-228 | H-blank period |

The gap between line interrupt (cycle ~8) and register latching (cycle ~14)
gives interrupt handlers approximately 6 CPU cycles to modify registers before
they are captured for the current scanline's rendering.

---

## V-Counter

The V-counter is a free-running counter readable at any time via port `$7E`.
It counts linearly through the active display and then **jumps** during VBlank
to fit the full frame into an 8-bit value.

### NTSC V-Counter Tables

**192-line mode (262 scanlines):**
```
Lines   0-218: V-counter $00-$DA (linear)
Lines 219-261: V-counter $D5-$FF (jump from $DA to $D5)
```

**224-line mode (262 scanlines):**
```
Lines   0-234: V-counter $00-$EA (linear)
Lines 235-261: V-counter $E5-$FF (jump from $EA to $E5)
```

### PAL V-Counter Tables

**192-line mode (313 scanlines):**
```
Lines   0-242: V-counter $00-$F2 (linear)
Lines 243-312: V-counter $BA-$FF (jump from $F2 to $BA)
```

**224-line mode (313 scanlines):**
```
Lines   0-258: V-counter $00-$FF, $00-$02 (wraps through zero)
Lines 259-312: V-counter $CA-$FF (jump from $02 to $CA)
```

---

## H-Counter

The H-counter is a 9-bit internal counter. Only the upper 8 bits are exposed
via port `$7F`. Each scanline spans 342 pixels (684 master clocks = 228 CPU
cycles).

### H-Counter Regions

| H-Counter | Pixels   | Region |
|-----------|---------|--------|
| `$00`-`$7F` | 0-255   | Active display (256 pixels) |
| `$80`-`$87` | 256-270 | Right border |
| `$87`-`$8B` | 271-278 | Right blanking |
| `$8B`-`$93` | 279-304 | Horizontal sync |
| `$93` jumps to `$E9` | -- | Non-linear jump |
| `$E9`-`$ED` | 305-306 | Left blanking |
| `$EE`-`$F5` | 307-320 | Color burst |
| `$F5`-`$F9` | 321-328 | Left blanking |
| `$F9`-`$FF` | 329-341 | Left border |

The jump from `$93` to `$E9` (skipping `$94`-`$E8`) is how the VDP maps the
full scanline into the 8-bit counter range.

### H-Counter Latching (Light Phaser)

On real hardware, the H-counter value is latched when the TH pin of either
controller port transitions from high to low. This mechanism was designed for
Light Phaser (light gun) support. Without TH triggering, the value returned by
reading port `$7F` is the last latched value.

---

## Interrupts

The VDP generates two types of maskable interrupts, both delivered via the Z80's
INT line (active low, level-triggered). The Z80 typically operates in IM 1 mode,
vectoring to address `$0038` on interrupt.

### Frame Interrupt (VBlank)

- **Trigger:** Status register bit 7 (INT) is set at a specific scanline:
  - 192-line mode: scanline 193 (V-counter `$C1`)
  - 224-line mode: scanline 225 (V-counter `$E1`)
- **Enable:** Register $01 bit 5 (IE0)
- **Assert condition:** INT flag set AND IE0 bit set
- **Clear:** Reading the status register (clears INT flag)

### Line Interrupt (Raster Interrupt)

The VDP maintains an 8-bit line counter with the following behavior:

**During active display (scanlines 0 through activeHeight, inclusive):**
- Counter decrements by 1 each scanline
- When the counter underflows from `$00` to `$FF`:
  - Counter is reloaded from Register $0A
  - Internal line interrupt pending flag is set

**During VBlank (scanlines activeHeight+1 through end of frame):**
- Counter is reloaded from Register $0A each scanline
- No line interrupts are generated

For 192-line NTSC mode:
- Lines 0-192: counter decrements (193 scanlines, including line 192)
- Lines 193-261: counter reloads from Register $0A

- **Enable:** Register $00 bit 4 (IE1)
- **Assert condition:** line interrupt pending flag set AND IE1 bit set
- **Clear:** Reading the status register (clears pending flag)

### Interrupt De-assertion

Both interrupt sources are ORed together. The VDP asserts the Z80 INT line when
either enabled interrupt condition is true. The INT line is **level-triggered**,
meaning:

- If a pending interrupt flag is cleared (by reading the status register), the
  INT line de-asserts immediately.
- If an interrupt enable bit is cleared (by writing to Register $00 or $01),
  the INT line de-asserts immediately.
- Conversely, setting an enable bit when a flag is already pending will
  immediately assert the INT line.

---

## Register Latching

Several VDP registers are latched at specific points in the frame to prevent
mid-operation changes from causing rendering artifacts.

### Per-Frame Latching

At the start of each frame (scanline 0):

- **Register $09 (vScroll)** is latched. The latched value is used for the
  entire frame. Changes to Register $09 during active display are ignored
  until the next frame.

### Per-Scanline Latching

At approximately cycle 14 of each scanline (after line interrupts have had
time to execute):

- **Register $08 (hScroll)** is latched
- **Register $02 (name table base)** is latched
- **Register $07 (backdrop color)** is latched
- **CRAM contents** are latched

The delay between line interrupt firing (~cycle 8) and register latching
(~cycle 14) is intentional. It gives line interrupt handlers time to modify
scroll registers, palette, or name table base before the values are captured
for rendering the current scanline. This is the mechanism that enables
per-scanline raster effects.

---

## SMS1 vs SMS2 VDP Differences

| Feature | SMS1 VDP (315-5124) | SMS2 VDP (315-5246) |
|---------|---------------------|---------------------|
| Extended height modes | Not supported (224/240-line modes ignored) | Supported |
| Sprite zoom | First 4 of 8 per-line sprites zoom correctly; remaining 4 zoom vertically only | All 8 sprites zoom correctly |
| Register $02 bit 0 | AND mask on name table row bit 10 (causes mirroring) | Ignored |
| Register $06 bits 1-0 | AND masks on sprite tile index bits 8 and 6 | Ignored |
| Register $05 bit 0 | Affects SAT fetch location | Ignored |
| Register $00 bit 0 (ES) | Causes gradual display fade and sync loss | No effect |
| PAL video encoding | Hardware bug causing improper PAL CVBS encoding | Fixed |

For emulation of officially released SMS games, the SMS2 behavior is the
standard target. The SMS1 quirks were used by very few games (notably the
Japanese version of Ys for Register $02 bit 0 mirroring).

---

## 240-Line Mode Analysis

Many technical documents and emulators reference a 240-line display mode for the
SMS VDP. The mode bits exist in the registers (M4=1, M2=1, M1=0, M3=1), and
the SMS2 VDP (315-5246) does respond to them. However, the practical reality is
more nuanced.

### NTSC Behavior

On NTSC systems (262 scanlines per frame), enabling 240-line mode results in
**240 active lines + 22 remaining scanlines**, which is insufficient for proper
borders, blanking, and vertical sync. The result is a display with **no vertical
blanking interval** that **rolls continuously** on standard televisions. While
the image could theoretically be stabilized using the vertical hold adjustment
on older CRT sets, the video signal is not standard-compliant.

Charles MacDonald's VDP documentation states plainly that the V-counter for
NTSC 240-line mode is `$00`-`$FF`, `$00`-`$06` -- it wraps naturally with no
jump, confirming there is no blanking interval.

### PAL Behavior

On PAL systems (313 scanlines per frame), 240-line mode does technically
produce a valid signal. There are enough scanlines (313 - 240 = 73) for
borders, blanking, and sync. The V-counter mapping is
`$00`-`$FF`, `$00`-`$0A`, then jumps to `$D2`-`$FF`.

### Game Gear Behavior

On the Game Gear, enabling 240-line mode causes a **system freeze** requiring
power cycling.

### Commercial Usage

**No commercially released game** for any platform (SMS, SMS2, or Game Gear)
ever used 240-line mode. The only games that used extended height modes are a
handful of Codemasters titles (Cosmic Spacehead, Fantastic Dizzy, Micro
Machines) that use **224-line mode**.

### SMS1 VDP

The SMS1 VDP (315-5124) ignores the M2/M1/M3 mode bits entirely and always
operates in 192-line mode when M4 is set.

### Emulation Recommendation

For an emulator targeting officially released SMS games, 240-line mode does not
need to be implemented. The mode is not used by any commercial software and
does not produce valid output on NTSC hardware. The only extended mode that
matters is 224-line mode on the SMS2/GG VDP.

---

## Game Gear VDP Notes

The Game Gear VDP (315-5378) is functionally identical to the SMS2 VDP
(315-5246) with the following differences:

### 12-Bit Color CRAM

- **64 bytes** of CRAM (32 x 16-bit entries)
- Format: `----BBBBGGGGRRRR` (4 bits per channel, 12 bits total)
- **4,096 possible colors**, 32 on screen simultaneously

**Latch write mechanism:** CRAM writes use a two-stage latch:
- Writing to an **even** CRAM address stores the value in an internal latch but
  does not modify CRAM.
- Writing to an **odd** CRAM address commits both the latched byte (to the even
  address) and the current byte (to the odd address) to CRAM atomically.

This ensures 16-bit palette entries are written as a unit, preventing
half-updated color values from appearing on screen.

### 160x144 Viewport

The GG LCD displays only the central **160x144 pixels** of the internal
256x192 display. The VDP still generates the full 256-pixel-wide frame
internally. This means:

- Horizontal scroll lock (top 2 rows) falls outside the visible GG viewport
- Vertical scroll lock (right 8 columns) is mostly off-screen
- Left column blank is entirely off-screen
- Off-screen sprites **still count** toward the 8-per-scanline limit

### Other Differences

- The GG always operates with NTSC timing (262 scanlines, 60 Hz)
- TMS9918 legacy modes use colors 16-31 and require manual CRAM setup
  (no automatic palette initialization like SMS)
- 240-line mode causes a system freeze

---

## Sources

### Primary Technical References

- Charles MacDonald, "Sega Master System VDP Documentation" (msvdp-20021112.txt)
  -- The definitive SMS VDP hardware reference. Available at:
  https://www.smspower.org/uploads/Development/msvdp-20021112.txt

- Texas Instruments, "TMS9918A/TMS9928A/TMS9929A Video Display Processors
  Data Manual" -- Original VDP datasheet for the chip the SMS VDP is derived
  from. Available at:
  https://www.smspower.org/uploads/Development/TMS9918.pdf

### SMS Power! Development Resources

- VDP Register Reference: https://www.smspower.org/Development/VDPRegisters
- Display Modes: https://www.smspower.org/Development/Modes
- Sprite System: https://www.smspower.org/Development/Sprites
- Scanline Counter: https://www.smspower.org/Development/ScanlineCounter
- VRAM Memory Map: https://www.smspower.org/Development/VRAMMemoryMap
- Palette: https://www.smspower.org/Development/Palette
- Tilemap Mirroring: https://www.smspower.org/Development/TilemapMirroring
- SMS1/SMS2 VDP Differences: https://www.smspower.org/Development/SMS1SMS2VDPs
- Development Documents Index: https://www.smspower.org/Development/Documents

### Additional Resources

- SMS/GG Emulation Resources Collection (franckverrot/EmulationResources):
  https://github.com/franckverrot/EmulationResources/tree/master/consoles/sms-gg
- Rodrigo Copetti, "Master System Architecture":
  https://www.copetti.org/writings/consoles/master-system/
