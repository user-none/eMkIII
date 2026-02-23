# Sega Master System Technical Reference

Technical reference for the Sega Master System hardware architecture. This
document covers system-level topics: CPU integration, memory map, I/O ports,
controllers, memory banking, interrupts, cartridge format, BIOS, and hardware
revisions. For component-specific details, refer to:

- [VDP.md](VDP.md) -- Video Display Processor (display, sprites, scrolling,
  timing, interrupts)
- [PSG.md](PSG.md) -- PSG integration (I/O mapping, clock, audio output,
  stereo extension)

This document does not cover Z80 CPU internals or SN76489 chip internals.
Those are documented in the source trees of their respective emulation modules.

## Table of Contents

- [System Overview](#system-overview)
- [Clock System](#clock-system)
- [CPU Integration](#cpu-integration)
- [Memory Map](#memory-map)
- [System RAM](#system-ram)
- [Sega Mapper](#sega-mapper)
- [Codemasters Mapper](#codemasters-mapper)
- [Cartridge RAM](#cartridge-ram)
- [I/O Port Map](#io-port-map)
- [Memory and I/O Control Register](#memory-and-io-control-register)
- [I/O Port Control Register](#io-port-control-register)
- [Controller Input](#controller-input)
- [Pause Button](#pause-button)
- [Reset Button](#reset-button)
- [BIOS](#bios)
- [ROM Header](#rom-header)
- [Cartridge Slot](#cartridge-slot)
- [Card Slot](#card-slot)
- [Hardware Revisions](#hardware-revisions)
- [Region Differences](#region-differences)
- [Game Gear Notes](#game-gear-notes)
- [Sources](#sources)

---

## System Overview

The Sega Master System is built around three primary ICs plus supporting
memory chips:

| IC   | Part Number | Function              | Package |
|------|-------------|-----------------------|---------|
| IC1  | Z0840004PSC | Zilog Z80A CPU        | 40-pin DIP |
| IC4  | 315-5216    | I/O controller        | 42-pin DIP |
| IC5  | 315-5124 or 315-5246 | VDP (with integrated PSG) | 64-pin DIP |

The I/O controller (315-5216) handles address decoding, controller port
multiplexing, the pause button NMI circuit, region detection logic, and
memory/I/O slot selection. Later models use the 315-5237 I/O controller which
is functionally equivalent.

The Mark III used discrete 74xx-series logic chips instead of a dedicated I/O
controller IC, with equivalent functionality.

Additional memory on the board:

| Component | Size | Purpose |
|-----------|------|---------|
| System RAM | 8 KB | Z80 work RAM at `$C000`-`$DFFF` (mirrored to `$FFFF`) |
| VRAM | 16 KB | Dedicated to VDP, not CPU-accessible via memory map |
| BIOS ROM | 8-32 KB | Boot ROM (export models; absent on Mark III) |

---

## Clock System

The SMS derives all internal clocks from a single master crystal. The CPU,
VDP, and PSG all receive the same clock signal, keeping them fully
synchronous.

### NTSC

| Clock          | Frequency       | Derivation |
|----------------|-----------------|------------|
| Master crystal | 53.693175 MHz   | Color burst x 15 |
| CPU / PSG      | 3,579,545 Hz    | Master / 15 |
| VDP pixel      | 10,738,635 Hz   | Master / 5 (3x CPU clock) |

The CPU clock is the NTSC chrominance subcarrier frequency (315/88 MHz),
which is standard for NTSC-based consoles.

### PAL

| Clock          | Frequency       | Derivation |
|----------------|-----------------|------------|
| Master crystal | 53.203424 MHz   | ~PAL subcarrier x 12 x 15 |
| CPU / PSG      | 3,546,893 Hz    | Master / 15 |
| VDP pixel      | 10,640,679 Hz   | Master / 5 (3x CPU clock) |

Because all clocks are derived from the same master crystal, there is no
clock domain crossing or drift between CPU execution, VDP rendering, and
PSG audio output.

---

## CPU Integration

The Z80 CPU is the sole bus master. There is no DMA controller. The Z80
manages all data transfers between ROM, RAM, VDP, PSG, and I/O ports.

### Interrupt Modes

The SMS always operates in Z80 **Interrupt Mode 1** (IM 1). In this mode:

- **INT (maskable):** Vectors to address `$0038`. Used by VDP frame interrupts
  and line interrupts. The INT line is active-low and level-triggered.
- **NMI (non-maskable):** Vectors to address `$0066`. Used by the pause
  button. The NMI is edge-triggered (falling edge).

The BIOS or game ROM initializes IM 1 early in the boot sequence. No
commercial software uses IM 0 or IM 2.

### Bus Interface

The Z80 accesses memory through a 16-bit address bus and 8-bit data bus. I/O
port access uses the lower 8 bits of the address bus (A0-A7) with partial
address decoding. See [I/O Port Map](#io-port-map) for details.

---

## Memory Map

The Z80 sees a 64 KB address space. The lower 48 KB is ROM (banked), and the
upper 16 KB is RAM (with bank control registers at the top).

```
$0000 +------------------+
      | ROM              |  First 1 KB always mapped to physical ROM bank 0
$0400 |                  |  Remainder of Slot 0 (bankable via $FFFD)
      |                  |
$4000 +------------------+
      | ROM Slot 1       |  16 KB, bankable via $FFFE
      |                  |
$8000 +------------------+
      | ROM Slot 2       |  16 KB, bankable via $FFFF
      | (or Cart RAM)    |  May be switched to cartridge RAM via $FFFC
$C000 +------------------+
      | System RAM       |  8 KB
$E000 +------------------+
      | RAM Mirror       |  Mirror of $C000-$DFFF
      |                  |
$FFFC | Bank Registers   |  $FFFC-$FFFF (writes go to both RAM and registers)
$FFFF +------------------+
```

### Address Ranges

| Range           | Size  | Content |
|-----------------|-------|---------|
| `$0000`-`$03FF` | 1 KB  | ROM bank 0 (never paged out) |
| `$0400`-`$3FFF` | 15 KB | ROM slot 0 (selectable bank) |
| `$4000`-`$7FFF` | 16 KB | ROM slot 1 (selectable bank) |
| `$8000`-`$BFFF` | 16 KB | ROM slot 2 (selectable bank) or cartridge RAM |
| `$C000`-`$DFFF` | 8 KB  | System RAM |
| `$E000`-`$FFFF` | 8 KB  | Mirror of system RAM |

The first 1 KB of ROM (`$0000`-`$03FF`) is pinned to bank 0 regardless of
the slot 0 bank register. This ensures the Z80 reset vector at `$0000`, the
interrupt vector at `$0038`, and the NMI vector at `$0066` are always
accessible.

Writes to `$FFFC`-`$FFFF` are written to both the underlying RAM and the bank
control registers simultaneously. Reads from these addresses return the RAM
contents, not the register values.

ROMs of 32 KB or smaller do not require banking and map linearly across
`$0000`-`$7FFF`.

---

## System RAM

- **Size:** 8 KB (8,192 bytes)
- **Location:** `$C000`-`$DFFF`
- **Mirror:** `$E000`-`$FFFF` (same 8 KB repeated)
- **Access:** Read/write by CPU

The RAM is implemented as a single 8 KB SRAM chip (or two 8 KB chips on early
boards, later consolidated to 32 KB PSRAM on cost-reduced revisions).

The Z80 stack pointer is typically initialized to point near the top of RAM
(just below the bank registers). The bank control registers at
`$FFFC`-`$FFFF` occupy the last 4 bytes of the mirror region and have dual
read/write behavior as described in the memory map section.

---

## Sega Mapper

The standard Sega mapper is implemented by a dedicated mapper chip on the
cartridge PCB (Sega part number 315-5235). This chip handles ROM banking for
cartridges larger than 48 KB and optionally provides cartridge RAM.

### Bank Registers

| Address | Register | Function |
|---------|----------|----------|
| `$FFFC` | RAM control | Controls cartridge RAM mapping |
| `$FFFD` | Slot 0 bank | Selects ROM bank for `$0400`-`$3FFF` |
| `$FFFE` | Slot 1 bank | Selects ROM bank for `$4000`-`$7FFF` |
| `$FFFF` | Slot 2 bank | Selects ROM bank for `$8000`-`$BFFF` |

Each bank register selects a 16 KB ROM bank. The bank number wraps based on
the actual ROM size (e.g., a 256 KB ROM has 16 banks numbered 0-15).

**Default values at power-on:**

| Register | Default |
|----------|---------|
| `$FFFC`  | `$00`   |
| `$FFFD`  | `$00` (bank 0) |
| `$FFFE`  | `$01` (bank 1) |
| `$FFFF`  | `$02` (bank 2) |

### RAM Control Register ($FFFC)

```
Bit 7: ROM write enable (development hardware; no effect on retail carts)
Bit 6: Unused
Bit 5: Unused
Bit 4: Cart RAM mapped to $C000-$FFFF (replaces system RAM mirror; use with
       port $3E to disable system RAM and avoid bus contention)
Bit 3: Cart RAM mapped to slot 2 ($8000-$BFFF), overriding ROM banking
Bit 2: Cart RAM bank select (0 = first 16 KB, 1 = second 16 KB)
Bit 1: Unused
Bit 0: Unused
```

Only bits 2 and 3 are used by commercial software. When bit 3 is set, slot 2
reads and writes access cartridge RAM instead of ROM. Bit 2 selects between
two 16 KB pages of cartridge RAM.

### Maximum Capacity

The 315-5235 mapper supports up to 32 ROM banks (512 KB / 4 Mbit) and up to
2 banks of cartridge RAM (32 KB).

---

## Codemasters Mapper

A small number of games published by Codemasters use an alternative mapper
that places bank registers at the start of each slot instead of at
`$FFFC`-`$FFFF`.

### Bank Registers

| Address  | Register | Function |
|----------|----------|----------|
| `$0000`  | Slot 0 bank | Selects ROM bank for `$0000`-`$3FFF` |
| `$4000`  | Slot 1 bank | Selects ROM bank for `$4000`-`$7FFF` |
| `$8000`  | Slot 2 bank | Selects ROM bank for `$8000`-`$BFFF` |

Unlike the Sega mapper, slot 0 is fully bankable -- the first 1 KB is **not**
pinned to bank 0. The system RAM region (`$C000`-`$FFFF`) has no bank
registers.

**Default values at power-on:**

| Register | Default |
|----------|---------|
| `$0000`  | `$00` (bank 0) |
| `$4000`  | `$01` (bank 1) |
| `$8000`  | `$00` (bank 0) |

The Codemasters mapper does not support cartridge RAM. No Codemasters title
uses battery-backed saves.

### Detection

Codemasters mapper cartridges are identified by CRC32 lookup against a known
database. There is no reliable heuristic to distinguish the mapper from ROM
contents alone.

---

## Cartridge RAM

Some cartridges include battery-backed SRAM for game saves. The Sega mapper
supports up to 32 KB of cartridge RAM, organized as two 16 KB banks.

### Access

Cartridge RAM is enabled by setting bit 3 of the RAM control register
(`$FFFC`). When enabled, reads and writes to `$8000`-`$BFFF` access
cartridge RAM instead of ROM. Bit 2 of `$FFFC` selects which 16 KB bank is
mapped.

### Games with Battery Backup

A small number of commercially released SMS games use battery-backed
cartridge RAM, including: Phantasy Star, Penguin Land, Ys: The Vanished
Omens, Golden Axe Warrior, Miracle Warriors, Monopoly, Ultima IV, and
Golfamania.

### Battery

Cartridges use a CR2032 coin cell battery to maintain SRAM contents when the
console is powered off.

---

## I/O Port Map

The SMS uses **partial address decoding** on bits A7, A6, and A0 of the Z80
I/O address bus. This means large ranges of port addresses map to the same
hardware.

### Complete Port Map

| Port Range          | Read                          | Write                         |
|---------------------|-------------------------------|-------------------------------|
| `$00`-`$3F` even   | (see note 1)                  | Memory/IO control (`$3E`)     |
| `$00`-`$3F` odd    | (see note 1)                  | I/O port control (`$3F`)      |
| `$40`-`$7F` even   | VDP V-counter                 | PSG data                      |
| `$40`-`$7F` odd    | VDP H-counter                 | PSG data                      |
| `$80`-`$BF` even   | VDP data port                 | VDP data port                 |
| `$80`-`$BF` odd    | VDP status register           | VDP control port              |
| `$C0`-`$FF` even   | I/O port A (controller 1)     | No effect                     |
| `$C0`-`$FF` odd    | I/O port B (controller 2)     | No effect                     |

**Note 1:** Reads from ports `$00`-`$3F` return the last value written to the
control registers on some hardware revisions, or `$FF` on others. Software
should not rely on reads from this range.

The canonical port addresses used in documentation are `$3E`/`$3F`,
`$7E`/`$7F`, `$BE`/`$BF`, and `$DC`/`$DD`, but any address within the
mirrored range behaves identically.

---

## Memory and I/O Control Register

Port `$3E` (write-only) controls which memory and I/O devices are active. It
is used by the BIOS to select between cartridge slot, card slot, expansion
port, and BIOS ROM.

```
Bit 7: Expansion slot disable (1 = disabled)
Bit 6: Cartridge slot disable (1 = disabled)
Bit 5: Card slot disable (1 = disabled)
Bit 4: System RAM disable (1 = disabled)
Bit 3: BIOS ROM disable (1 = disabled)
Bit 2: I/O chip disable (1 = disabled)
Bit 1: Unused
Bit 0: Unused
```

**Default value at power-on:** All devices enabled (`$00`). On models with a
BIOS, the BIOS ROM initially overlays the cartridge ROM. The BIOS writes to
port `$3E` to disable itself and enable the cartridge before jumping to the
game.

On models **without** a BIOS (Mark III, some SMS2 regions), the cartridge is
mapped directly at boot and port `$3E` has no practical effect since there
is no BIOS to disable.

When cartridge RAM is mapped to `$C000`-`$FFFF` (via `$FFFC` bit 4), system
RAM should be disabled via port `$3E` bit 4 to avoid bus contention.

---

## I/O Port Control Register

Port `$3F` (write-only) controls the TH and TR pin directions on the two
controller ports. It is also used for **region detection** (nationalization).

```
Bit 7: Port B TH output level (1 = high, 0 = low)
Bit 6: Port B TR output level (1 = high, 0 = low)
Bit 5: Port A TH output level (1 = high, 0 = low)
Bit 4: Port A TR output level (1 = high, 0 = low)
Bit 3: Port B TH direction (1 = input, 0 = output)
Bit 2: Port B TR direction (1 = input, 0 = output)
Bit 1: Port A TH direction (1 = input, 0 = output)
Bit 0: Port A TR direction (1 = input, 0 = output)
```

### Region Detection

The I/O controller echoes the TH output values back on port `$DD` bits 6 and
7, but the behavior differs by region:

- **Export (non-Japanese) consoles:** Bits 6 and 7 of port `$DD` reflect the
  TH output values directly.
- **Japanese consoles:** Bits 6 and 7 of port `$DD` reflect the TH output
  values **inverted** (complemented).

Games detect the region by writing `$F5` to port `$3F` (which sets both TH
pins as outputs with specific values), then reading port `$DD` bits 6-7. If
the read values match what was written, the console is export; if inverted,
it is Japanese.

Some games perform a second check by writing `$55` and reading again to
confirm.

---

## Controller Input

The SMS has two 9-pin D-sub controller ports (DE-9). Each controller has a
D-pad (up/down/left/right) and two action buttons.

### Controller Connector Pinout

| Pin | Function |
|-----|----------|
| 1   | Up       |
| 2   | Down     |
| 3   | Left     |
| 4   | Right    |
| 5   | +5V      |
| 6   | Button 1 (TL) |
| 7   | Unused (TH on Light Phaser) |
| 8   | Ground   |
| 9   | Button 2 (TR) |

### Port $DC -- I/O Port A

Reading port `$DC` (or any even address in `$C0`-`$FF`) returns:

```
Bit 7: Player 2 Down    (active low: 0 = pressed, 1 = released)
Bit 6: Player 2 Up      (active low)
Bit 5: Player 1 Button 2 (active low)
Bit 4: Player 1 Button 1 (active low)
Bit 3: Player 1 Right   (active low)
Bit 2: Player 1 Left    (active low)
Bit 1: Player 1 Down    (active low)
Bit 0: Player 1 Up      (active low)
```

All controller lines use **active-low** logic: a pressed button or direction
pulls the line low (reads as 0), and the released state is high (reads as 1).
The default value with no buttons pressed is `$FF`.

### Port $DD -- I/O Port B

Reading port `$DD` (or any odd address in `$C0`-`$FF`) returns:

```
Bit 7: Port B TH input  (Light Phaser / region detection)
Bit 6: Port A TH input  (Light Phaser / region detection)
Bit 5: Unused (always 1)
Bit 4: Reset button      (active low: 0 = pressed, 1 = not pressed)
Bit 3: Player 2 Button 2 (active low)
Bit 2: Player 2 Button 1 (active low)
Bit 1: Player 2 Right   (active low)
Bit 0: Player 2 Left    (active low)
```

**Note:** Player 2 direction and button inputs are split across port `$DC`
(bits 6-7 for up/down) and port `$DD` (bits 0-3 for left/right/buttons).
This split reflects the physical wiring of the I/O controller.

---

## Pause Button

The pause button is **not** a controller input. It is wired through the I/O
controller directly to the Z80 NMI (Non-Maskable Interrupt) line.

### Behavior

- Pressing the pause button triggers a **falling-edge NMI** on the Z80.
- The NMI causes an unconditional jump to address `$0066`.
- The NMI cannot be disabled or masked by software.
- The NMI is edge-triggered, so holding the button does not generate
  repeated interrupts. The button must be released and pressed again.

### Typical Software Handling

Most games set a flag in RAM inside the NMI handler and return immediately
with `RETN`. The main game loop checks this flag and toggles pause state at
a safe point (typically during VBlank processing).

```
; Typical NMI handler
$0066:  push af
        ld   a, (pause_flag)
        xor  1
        ld   (pause_flag), a
        pop  af
        retn
```

### Hardware Availability

| Model | Pause Button |
|-------|-------------|
| Mark III | On console (front panel) |
| SMS Model 1 | On console (front panel) |
| SMS Model 2 | On console (top) |
| Japanese SMS (MK-2000) | On console |

---

## Reset Button

The reset button is **not** connected to the Z80 hardware reset line. It is
routed through the I/O controller and mapped as a readable bit on port `$DD`.

### Behavior

- Reading port `$DD` bit 4: `0` = pressed, `1` = not pressed (active low).
- Software must poll this bit to detect a reset press.
- The hardware does **not** force a CPU reset. Software is free to ignore the
  button or perform any action in response.
- Some games jump to address `$0000` when the reset button is detected,
  simulating a soft reset.

### Hardware Availability

| Model | Reset Button |
|-------|-------------|
| Mark III | No |
| SMS Model 1 (export) | Yes |
| Japanese SMS (MK-2000) | No |
| SMS Model 2 | No |

On models without a reset button, port `$DD` bit 4 always reads as `1` (not
pressed).

---

## BIOS

Some SMS models include a BIOS ROM that executes before the game. The BIOS is
memory-mapped and overlays the cartridge ROM at `$0000` until it disables
itself via port `$3E`.

### BIOS Behavior

1. At power-on, the Z80 begins execution at `$0000`, which is BIOS ROM.
2. The BIOS performs initialization (display the Sega logo, optional built-in
   game, etc.).
3. The BIOS checks for a cartridge or card by testing for valid ROM data.
4. The BIOS writes to port `$3E` to disable BIOS ROM and enable the
   cartridge/card slot.
5. Execution jumps to `$0000` in the now-mapped cartridge ROM.

### BIOS Versions

| Version | Models | Size | Notes |
|---------|--------|------|-------|
| None | Mark III, some SMS2 | 0 KB | Cartridge maps directly at boot |
| v1.3 | SMS1 (early export) | 8 KB | Snail Maze easter egg |
| v2.1 | Japanese SMS (MK-2000) | 8 KB | Space Harrier theme at startup |
| v4.4 | SMS1 (later export) | 32 KB | Built-in game (e.g., Hang-On, Alex Kidd) |
| Various | SMS2 | 32 KB | Built-in game (Alex Kidd or Sonic) |

### Media Priority

When both a card and cartridge are inserted simultaneously:

- **Master System (export):** Card takes priority.
- **Mark III (Japan):** Cartridge takes priority.

### Emulation Note

For emulation of cartridge-only games, the BIOS is not required. The emulator
can map the cartridge ROM directly to `$0000` at power-on, equivalent to a
BIOS-less boot.

---

## ROM Header

SMS cartridge ROMs contain an optional header at offset `$7FF0` (the last
16 bytes of the first 32 KB). This header is used by the BIOS for
validation and by tools for identification.

### Header Location

The header is at `$7FF0` for standard ROMs. Some ROMs of 16 KB or smaller
may place the header at `$1FF0` or `$3FF0`.

### Header Format

| Offset | Size    | Content |
|--------|---------|---------|
| `$7FF0` | 8 bytes | `TMR SEGA` signature (ASCII) |
| `$7FF8` | 2 bytes | Reserved |
| `$7FFA` | 2 bytes | Checksum (little-endian) |
| `$7FFC` | 3 bytes | Product code (BCD, last nibble shared with version) |
| `$7FFF` | 1 byte  | Region and ROM size |

### Region / ROM Size Byte ($7FFF)

```
Bits 7-4: Region code
Bits 3-0: ROM size code
```

**Region codes:**

| Code | Region |
|------|--------|
| 3    | SMS Japan |
| 4    | SMS Export |
| 5    | GG Japan |
| 6    | GG Export |
| 7    | GG International |

**ROM size codes:**

| Code | Size |
|------|------|
| `$A` | 8 KB |
| `$B` | 16 KB |
| `$C` | 32 KB |
| `$D` | 48 KB |
| `$E` | 64 KB |
| `$F` | 128 KB |
| `$0` | 256 KB |
| `$1` | 512 KB |
| `$2` | 1 MB |

### Checksum

The checksum covers a region of the ROM that depends on the ROM size code.
It is a simple 16-bit sum of all bytes in the covered range, excluding the
header itself (`$7FF0`-`$7FFF`). The BIOS validates this checksum on models
that have a BIOS.

### Emulation Note

Not all ROMs have a valid header. The `TMR SEGA` signature is the primary
indicator. ROMs without the signature should still load and run; the header
is only required for BIOS validation. Mapper type and region are more
reliably determined via CRC32 database lookup than by parsing the header.

---

## Cartridge Slot

The SMS cartridge slot is a 50-pin edge connector. The slot carries the full
Z80 address bus (A0-A15), data bus (D0-D7), and control signals.

The export SMS cartridge connector uses a different physical form factor than
the Japanese Mark III / MK-2000 connector (44-pin). Export and Japanese
cartridges are not physically interchangeable without an adapter.

Korean and Chinese models use the Japanese 44-pin connector.

---

## Card Slot

The SMS Model 1 and Mark III include a card slot that accepts Sega Card (or
Sega My Card in Japan) media. Cards are credit-card-sized ROM boards
containing up to 32 KB of ROM.

### Characteristics

- Maximum card ROM size: 32 KB (no banking required).
- The card slot is selected via port `$3E`. The BIOS handles slot selection
  automatically.
- Cards use a simplified subset of the cartridge bus signals (no banking
  support).

### Availability

| Model | Card Slot |
|-------|-----------|
| Mark III | Yes |
| SMS Model 1 (export) | Yes |
| Japanese SMS (MK-2000) | Yes |
| SMS Model 2 | No |

The SMS Model 2 removed the card slot as a cost reduction.

---

## Hardware Revisions

### Mark III (1985, Japan)

- First SMS-compatible hardware, successor to SG-1000 II.
- VDP: 315-5124 (original; no extended height modes).
- I/O: Discrete 74xx logic (no dedicated I/O controller chip).
- No BIOS ROM.
- Expansion port (SG-1000 compatible) for FM Sound Unit and keyboard.
- Backward compatible with SG-1000 cartridges and cards.

### Master System Model 1 (1986, Export)

- Redesigned case for international markets.
- VDP: 315-5124 (early boards), 315-5246 (late VA3 boards).
- I/O: 315-5216 (early), 315-5237 (late VA3).
- BIOS ROM present (various versions).
- Card slot, expansion port, and reset button.

### Japanese Master System / MK-2000 (1987, Japan)

- Japanese redesign of the Mark III.
- Built-in FM Sound Unit (YM2413).
- Built-in 3D glasses adapter.
- BIOS v2.1 (Space Harrier theme).
- Uses Mark III cartridge connector (44-pin).
- No reset button.

### Master System II (1990, Export)

- Significant cost reduction.
- VDP: 315-5246.
- I/O: 315-5216 (NTSC) or 315-5237 (PAL).
- No card slot, no expansion port, no reset button.
- Built-in game in BIOS (Alex Kidd or Sonic, depending on region/revision).
- Composite video output removed on most models (RF only, except France).

---

## Region Differences

### NTSC vs PAL Timing

| Parameter | NTSC | PAL |
|-----------|------|-----|
| CPU clock | 3,579,545 Hz | 3,546,893 Hz |
| Scanlines per frame | 262 | 313 |
| Frame rate | ~60 Hz | ~50 Hz |
| Active display | 192 or 224 lines | 192 or 224 lines |
| VBlank lines | 70 or 38 | 121 or 89 |

PAL games run approximately 17% slower than NTSC due to the lower frame
rate. Some PAL-specific titles adjust game speed to compensate. Most do not.

### Region Detection

Games use the I/O port control register (`$3F`) and port `$DD` bits 6-7 to
detect whether the console is Japanese or export. See
[I/O Port Control Register](#io-port-control-register) for the detection
mechanism.

The ROM header region code at `$7FFF` is a secondary indicator but is less
reliable than the hardware-based detection.

### Cartridge Compatibility

Export and Japanese cartridges are physically incompatible due to different
connector sizes (50-pin vs 44-pin). Some games additionally check the
region via port `$3F` and refuse to run on the wrong region console.

---

## Game Gear Notes

The Game Gear is architecturally very close to the SMS. The following are
the key differences relevant to system-level emulation. VDP and PSG
differences are documented in their respective reference documents.

### CPU and Memory

- The Game Gear uses the same Z80 CPU at the same NTSC clock speed
  (3,579,545 Hz). There is no PAL Game Gear.
- The memory map is identical to the SMS (same Sega mapper).
- System RAM is 8 KB at `$C000`-`$DFFF`, same as SMS.

### I/O Differences

**Port `$00` (Game Gear only, read-only):**

```
Bit 7: Start button (active low: 0 = pressed, 1 = not pressed)
Bit 6: NJAP (0 = Japanese, 1 = export)
Bit 5: NNTS (0 = NTSC, 1 = PAL; always NTSC on real hardware)
Bits 4-0: Unused (active low, normally read as 1)
```

This port does not exist on the SMS. On the SMS, port `$00` is in the
memory/IO control register range.

**Start button vs Pause button:**

On the SMS, the pause button generates an NMI. On the Game Gear in native
mode, the Start button is a pollable input on port `$00` bit 7 and does
**not** generate an NMI. When the Game Gear runs SMS software (compatibility
mode), the Start button triggers an NMI like the SMS pause button.

### Controller

The Game Gear has a single built-in controller with the same button layout as
the SMS (D-pad, buttons 1 and 2) plus the Start button. Port `$DC` reads
identically to the SMS for player 1 inputs. Port `$DD` has no reset button
(bit 4 always reads `1`).

### No Reset Button

Port `$DD` bit 4 always returns `1` (not pressed) on the Game Gear.

### No Card Slot or Expansion Port

The Game Gear has neither a card slot nor an expansion port. Port `$3E` has
no practical effect.

---

## Sources

### Primary Technical References

- Richard Talbot-Watkins, "Sega Master System Technical Information"
  (smstech-20021112.txt)
  https://www.smspower.org/uploads/Development/smstech-20021112.txt

- Charles MacDonald, "Sega Master System VDP Documentation"
  (msvdp-20021112.txt)
  https://www.smspower.org/uploads/Development/msvdp-20021112.txt

### SMS Power! Development Resources

- Memory Map:
  https://www.smspower.org/Development/MemoryMap

- I/O Port Map:
  https://www.smspower.org/Development/IOPortMap

- Controllers:
  https://www.smspower.org/Development/Controllers

- Mappers:
  https://www.smspower.org/Development/Mappers

- Pause Button:
  https://www.smspower.org/Development/PauseButton

- Reset Button:
  https://www.smspower.org/Development/ResetButton

- Start Button (Game Gear):
  https://www.smspower.org/Development/StartButton

- ROM Header:
  https://www.smspower.org/Development/ROMHeader

- Card Slot:
  https://www.smspower.org/Development/CardSlot

- Cartridge Slot:
  https://www.smspower.org/Development/CartridgeSlot

- Pinouts:
  https://www.smspower.org/Development/Pinouts-Index

- Clock Rate:
  https://www.smspower.org/Development/ClockRate

- Development Documents Index:
  https://www.smspower.org/Development/Documents

### Additional Resources

- Rodrigo Copetti, "Master System Architecture":
  https://www.copetti.org/writings/consoles/master-system/

- ConsoleMods Wiki, "Master System Model Differences":
  https://consolemods.org/wiki/Master_System:Master_System_Model_Differences

- Sega Master System Service Manual
