# Sega Game Gear Technical Reference Addendum

This document covers hardware differences between the Sega Game Gear and the
Sega Master System. It is an addendum to the SMS reference documents and does
not repeat information already covered in:

- [SMS.md](SMS.md) -- System architecture, memory map, I/O ports, controllers,
  mappers, BIOS, interrupts (includes a Game Gear Notes section)
- [VDP.md](VDP.md) -- VDP reference (includes Game Gear VDP Notes: 12-bit
  CRAM, latch writes, 160x144 viewport)
- [PSG.md](PSG.md) -- PSG integration (includes Game Gear Stereo Extension:
  port `$06` panning register)

This document does not cover Z80 CPU internals or SN76489 chip internals.

## Table of Contents

- [Hardware Overview](#hardware-overview)
- [ASIC Variants and Board Revisions](#asic-variants-and-board-revisions)
- [BIOS](#bios)
- [SMS Compatibility Mode](#sms-compatibility-mode)
- [GG-Specific I/O Ports](#gg-specific-io-ports)
- [Link Port (EXT Connector)](#link-port-ext-connector)
- [Cartridge Slot](#cartridge-slot)
- [Port $DD Differences](#port-dd-differences)
- [Color Depth Reduction on Later Revisions](#color-depth-reduction-on-later-revisions)
- [LCD Display](#lcd-display)
- [Power Supply](#power-supply)
- [Sources](#sources)

---

## Hardware Overview

The Game Gear is architecturally a portable SMS2. It uses the same Z80 CPU at
the same NTSC clock frequency (3,579,545 Hz), the same 8 KB system RAM, the
same 16 KB VRAM, and the same Sega mapper. There is no PAL Game Gear.

The key hardware differences from the SMS are:

| Feature | SMS | Game Gear |
|---------|-----|-----------|
| VDP chip | 315-5124 or 315-5246 (discrete) | 315-5378 (discrete on VA0) or integrated into ASIC |
| CRAM | 32 bytes, 6-bit color | 64 bytes, 12-bit color (see [VDP.md](VDP.md#game-gear-vdp-notes)) |
| Display | 256x192 on CRT | 160x144 LCD viewport of 256x192 internal frame |
| Audio output | Mono only | Mono speaker + stereo headphone jack (see [PSG.md](PSG.md#game-gear-stereo-extension)) |
| Controller | External DE-9 ports (2) | Built-in D-pad + 2 buttons + Start |
| Pause/Start | Pause button triggers NMI | Start button polled via port `$00` (see [SMS.md](SMS.md#game-gear-notes)) |
| Link port | None | EXT connector (serial/parallel) |
| Card slot | Yes (SMS1) | No |
| Expansion port | Yes (SMS1, Mark III) | No |
| Reset button | Yes (SMS1 only) | No |
| Integration | 3 discrete ICs + logic | Progressed from 2-chip to single ASIC |

---

## ASIC Variants and Board Revisions

The Game Gear went through several board revisions, progressively integrating
discrete components into fewer ASICs.

### VA0 (1990-1993)

- **Two-chip design** with discrete Z80 CPU.
- IC2: **315-5377** -- I/O controller and serial/parallel link port logic.
- IC3: **315-5378** -- VDP with integrated PSG (Game Gear variant of
  315-5246, with 12-bit CRAM and stereo support).
- No BIOS ROM.
- Full 12-bit color output to LCD.

### VA1 (1992-1996)

- **Single-ASIC design.**
- IC1: **315-5535** -- Integrates Z80 CPU, VDP, PSG, I/O controller, and
  serial link logic onto one chip.
- BIOS ROM built into the ASIC. Initially disabled via jumper J1; enabled by
  default from 1993 onward (J1 left unpopulated).
- Full 12-bit color output to LCD.
- Uses Citizen UC-320 LCD panel.

### VA2 (~1992, Japan only)

- Japan-only variant of VA1 (board 837-8560).
- IC1: **315-5535** (same ASIC as VA1).

### VA4 (1993-1996, North America)

- IC1: **315-5682** -- Cost-reduced ASIC.
- Components relocated; uses JST PH connectors (incompatible with VA0/VA1
  power and sound boards).
- **Reduced to 9-bit color output** (3 bits per channel to the LCD) despite
  VDP CRAM still operating at 12-bit internally. See
  [Color Depth Reduction](#color-depth-reduction-on-later-revisions).
- Different LCD panel with three separate ribbon cables.
- BIOS present and active.

### VA5 (1994-2001, North America)

- IC1: **315-5682** (same ASIC as VA4).
- Audio preamp and VRAM moved to rear of board.
- Single LCD ribbon cable.
- Same 9-bit color output as VA4.
- BIOS present and active.

### Majesco Variant (2000-2001, North America)

- Board: 171-7923A.
- IC1: **315-5682** (same ASIC as VA4/VA5).
- Budget re-release at $29.99.
- Cosmetically different (monochrome lens, black shell).
- Incompatible with TV Tuner and Master Gear Converter.
- Unusual region configuration: Japanese region flag for SMS mode, but export
  behavior for GG mode.
- BIOS present and active.

---

## BIOS

The Game Gear BIOS is a 1 KB ROM mapped at `$0000`-`$03FF`. Its purpose is
trademark verification, not copy protection.

### Availability

| Revision | BIOS |
|----------|------|
| VA0 | None |
| VA1 (early, pre-1993) | Present in ASIC but disabled via jumper J1 |
| VA1 (1993+) | Enabled (J1 unpopulated) |
| VA4, VA5 | Present and active |
| Majesco | Present and active |

### Boot Sequence

1. The BIOS occupies `$0000`-`$03FF`; the cartridge maps to `$0400`-`$BFFF`.
2. The BIOS detects GG mode vs SMS mode by writing to and reading from port
   `$02` (the EXT direction register returns different values depending on
   mode) and selects the appropriate palette data.
3. The BIOS searches the cartridge ROM for the ASCII string `"TMR SEGA"` at
   three offsets: `$7FF0`, `$3FF0`, and `$1FF0`.
4. If found: displays a splash screen reading "PRODUCED BY OR UNDER LICENSE
   FROM SEGA ENTERPRISES, LTD." then boots the game.
5. If not found: leaves the screen off and the system locks up.
6. To boot the game, the BIOS disables itself by writing to port `$3E`
   (setting bit 3 to disable the BIOS ROM) and jumps to `$0000` in the
   now-mapped cartridge.

### Emulation Note

The BIOS is not required for emulation. The emulator can map the cartridge
ROM directly to `$0000` at power-on, bypassing the trademark check entirely.

---

## SMS Compatibility Mode

The Game Gear can run SMS software through hardware compatibility mode. The
mode is controlled by **pin 42** of the cartridge connector:

- **Pin 42 LOW (ground):** GG mode (default; GG cartridges leave this pin
  floating, pulled low internally).
- **Pin 42 HIGH (+5V):** SMS compatibility mode. Triggered by the Master Gear
  Converter or similar adapters that tie pin 42 to +5V.

### What Changes in SMS Mode

| Feature | GG Mode | SMS Mode |
|---------|---------|----------|
| CRAM | 64 bytes, 12-bit, two-byte latch writes | 32 bytes, 6-bit, single-byte writes |
| Display | 160x144 viewport | Full 256x192 (scaled to LCD) |
| Start button | Polled via port `$00` bit 7 | Triggers NMI to `$0066` (same as SMS Pause) |
| Ports `$00`-`$06` | GG-specific registers active | Revert to SMS port `$3E`/`$3F` decoding |
| Color palette | 4,096 possible colors | 64 possible colors |
| EXT port | Serial/parallel link | Can be used as 2nd controller via adapter |

### Mode Detection

Software can detect GG vs SMS mode by reading port `$00`. In GG mode, this
port returns the Start button state and region bits. In SMS mode, port `$00`
is in the memory/IO control register range and does not return GG-specific
data. The BIOS detects the mode by testing port `$02` (EXT direction
register) write/read behavior.

---

## GG-Specific I/O Ports

The Game Gear adds ports `$00`-`$06` that do not exist on the SMS. In GG
mode, these ports are decoded separately from the SMS partial address
decoding that normally maps the `$00`-`$3F` range.

### Port Summary

| Port | R/W | Function | Initial Value |
|------|-----|----------|---------------|
| `$00` | R | System register (Start, NJAP, NNTS) | `$C0` |
| `$01` | R/W | EXT parallel data (7-bit: PC6-PC0) | `$7F` |
| `$02` | R/W | EXT direction / NMI control | `$FF` |
| `$03` | R/W | Serial transmit data | `$00` |
| `$04` | R | Serial receive data | `$FF` |
| `$05` | R/W | Serial mode / status | `$00` |
| `$06` | W | Stereo panning (see [PSG.md](PSG.md#game-gear-stereo-extension)) | `$FF` |

Port `$00` is documented in [SMS.md](SMS.md#game-gear-notes). Port `$06` is
documented in [PSG.md](PSG.md#game-gear-stereo-extension). Ports `$01`-`$05`
are the link port registers, documented below.

### Port `$00` Region Bits

Bits 6 and 5 of port `$00` are hardwired to reflect the console region:

| Region | NJAP (bit 6) | NNTS (bit 5) |
|--------|-------------|-------------|
| Japan | 0 | 0 |
| North America | 1 | 0 |
| Europe | 1 | 1 |

Games use these bits for region lockout and language selection.

---

## Link Port (EXT Connector)

The Game Gear has a proprietary EXT connector for the Gear-to-Gear cable,
enabling two-player link play. The SMS has no equivalent.

### Physical Connector

The EXT port is a 10-pin proprietary connector. It exposes 7 I/O lines
(PC0-PC6) that can operate in parallel or serial mode.

| Pin | Signal |
|-----|--------|
| 1 | PC0 |
| 2 | PC1 |
| 3 | PC2 |
| 4 | PC3 |
| 5 | +5V |
| 6 | PC4 (serial TX) |
| 7 | PC6 |
| 8 | GND |
| 9 | PC5 (serial RX) |
| 10 | NC |

### Parallel Mode

Ports `$01` and `$02` provide 7-bit parallel I/O:

**Port `$01` -- Parallel Data (R/W):**

Bits 6-0 correspond to lines PC6-PC0. Read returns the current state of each
line. Write sets the output level for lines configured as outputs.

**Port `$02` -- Direction Control (R/W):**

Bits 6-0 set the direction for each line: `0` = output, `1` = input. The
default value `$FF` configures all lines as inputs.

When connecting two Game Gears, one unit's lines must be configured as outputs
while the other unit's corresponding lines are inputs.

### Serial Mode

Ports `$03`-`$05` provide UART serial communication:

**Port `$03` -- Serial Transmit Data (R/W):**

8-bit transmit data register. Writing a byte begins serial transmission on
PC4 (pin 6).

**Port `$04` -- Serial Receive Data (R only):**

8-bit receive data register. Read returns the last received byte from
PC5 (pin 9).

**Port `$05` -- Serial Mode / Status (R/W):**

```
Bit 7-6: Baud rate select
         00 = 4800 bps
         01 = 2400 bps
         10 = 1200 bps
         11 = 300 bps
Bit 5:   RON -- Receive enable (1 = enable, forces PC5 as input)
Bit 4:   TON -- Transmit enable (1 = enable, forces PC4 as output)
Bit 3:   INT -- NMI on receive (1 = generate NMI when data received)
Bit 2:   FRER -- Framing error flag (read: 1 = framing error detected)
Bit 1:   RXRD -- Receive data ready (read: 1 = data available in port $04)
Bit 0:   TXFL -- Transmit buffer full (read: 1 = cannot write yet)
```

**Serial protocol:** 5V TTL UART, 8 data bits, LSB first, no parity, 1 stop
bit.

### Emulation Note

The link port is not used by single-player games and does not need to be
emulated for standard game compatibility. Games that support link play
typically detect the absence of a connected partner and fall back to
single-player mode.

---

## Cartridge Slot

The Game Gear cartridge slot shares the same signal assignments as the export
SMS cartridge slot for address bus, data bus, and control lines. The physical
connector is smaller than the SMS connector, requiring an adapter (Master
Gear Converter) for SMS cartridges.

### Pin 42 -- Mode Select

Pin 42 determines the operating mode:

- **LOW (0V):** Game Gear mode. GG cartridges leave this pin unconnected
  (floating low via internal pull-down).
- **HIGH (+5V):** SMS compatibility mode. The Master Gear Converter ties this
  pin to +5V.

This is the only mechanism for switching between GG and SMS mode. There is no
software-accessible register to change modes.

---

## Port $DD Differences

The Game Gear returns different values for some bits of port `$DD` compared
to the SMS:

| Bit | SMS | Game Gear |
|-----|-----|-----------|
| 4 (Reset) | 0 when pressed, 1 when released (SMS1 only; always 1 on SMS2) | Always 1 (no reset button) |
| 5 (CONT) | 0 | 1 |

Bit 5 can be used as an additional method to distinguish GG from SMS hardware
in software.

---

## Color Depth Reduction on Later Revisions

Starting with the VA4 revision (ASIC 315-5682), the LCD interface was reduced
from 12-bit to **9-bit color** (3 bits per channel instead of 4).

### Impact

| Revision | ASIC | CRAM (internal) | LCD Output | Visible Colors |
|----------|------|-----------------|------------|----------------|
| VA0 | 315-5377/5378 | 12-bit (4 bpp) | 12-bit | 4,096 |
| VA1, VA2 | 315-5535 | 12-bit (4 bpp) | 12-bit | 4,096 |
| VA4, VA5 | 315-5682 | 12-bit (4 bpp) | 9-bit | 512 |
| Majesco | 315-5682 | 12-bit (4 bpp) | 9-bit | 512 |

The VDP CRAM still stores full 12-bit color values internally. The reduction
happens at the LCD interface, where the least significant bit of each channel
is dropped before being sent to the display. This means games designed on
VA0/VA1 hardware may show slightly different colors (banding in gradients) on
VA4/VA5 hardware.

### Emulation Note

For emulation purposes, the full 12-bit CRAM values should be used for
rendering. The 9-bit reduction is a hardware display limitation, not a change
in the VDP's behavior.

---

## LCD Display

### Specifications

| Parameter | Value |
|-----------|-------|
| Size | 3.2 inches (81 mm) diagonal |
| Resolution | 160 x 144 pixels |
| Technology | STN (Super Twisted Nematic) color LCD |
| Backlight | CCFL (Cold Cathode Fluorescent Lamp), ~35V |
| Frame rate | ~60 Hz (NTSC timing) |
| Pixel aspect | Non-square (wider than tall) |

### Viewport Offset

The 160x144 LCD displays the central region of the 256x192 internal frame:

- **Horizontal offset:** 48 pixels from the left edge (columns 48-207)
- **Vertical offset:** 24 pixels from the top edge (lines 24-167)

This is documented in detail in [VDP.md](VDP.md#game-gear-vdp-notes).

### LCD Characteristics

The original STN LCD has slow pixel response times (characteristic of early
1990s STN technology), causing visible ghosting and motion blur behind moving
objects. Viewing angles are limited. The CCFL backlight produces muted colors
compared to CRT displays.

These characteristics are properties of the physical display hardware and do
not affect emulation of the VDP or game logic.

---

## Power Supply

| Parameter | Value |
|-----------|-------|
| Batteries | 6 x AA (LR6) in series (~9V) |
| Battery life | 3-5 hours (alkaline) |
| AC adapter (US/NTSC) | DC 9V, 850 mA, EIAJ-03 tip-positive |
| AC adapter (EU/PAL) | DC 10V, 850 mA, tip-negative |
| AC adapter (Japan) | DC 9V, tip-negative |
| Internal regulation | +5V for logic, +35V for CCFL backlight |

The high power consumption relative to contemporary handhelds (e.g., Game
Boy at ~30 hours on 4 AA cells) was due to the color STN LCD backlight.

---

## Sources

### Primary Technical References

- Sega, "Sega Game Gear Hardware Reference Manual"
  https://segaretro.org/images/1/16/Sega_Game_Gear_Hardware_Reference_Manual.pdf

- Richard Talbot-Watkins, "Sega Master System Technical Information"
  (smstech-20021112.txt) -- Includes GG-specific port documentation.
  https://www.smspower.org/uploads/Development/smstech-20021112.txt

### SMS Power! Development Resources

- Start Button:
  https://www.smspower.org/Development/StartButton

- Gear to Gear Cable:
  https://www.smspower.org/Development/GearToGearCable

- GG VDP:
  https://www.smspower.org/Development/GGVDP

- Palette:
  https://www.smspower.org/Development/Palette

- BIOSes:
  https://www.smspower.org/Development/BIOSes

- Pinouts:
  https://www.smspower.org/maxim/Documents/Pinouts

- Development Documents Index:
  https://www.smspower.org/Development/Documents

### Hardware Reference

- ConsoleMods Wiki, "Game Gear Model Differences":
  https://consolemods.org/wiki/Game_Gear:Game_Gear_Model_Differences

- RetroSix Wiki, "Model Versions - Game Gear":
  https://www.retrosix.wiki/model-versions-game-gear

- RetroSix Wiki, "VA0/VA1 LCD Interface":
  https://www.retrosix.wiki/va0va1-lcd-interface-game-gear

- RetroSix Wiki, "VA4/VA5 LCD Interface":
  https://www.retrosix.wiki/va4-va5-lcd-interface-game-gear

- Sega Game Gear - Sega Retro:
  https://segaretro.org/Sega_Game_Gear

### Additional Resources

- nicole.express, "Converting from the Game Gear to the Master System":
  https://nicole.express/2022/sega-8-bit-conversion-kit.html
