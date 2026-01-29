# eMKIII

A Sega Master System Mark-3 (SMS) emulator written in Go 

## Project Overview

Core functionality is implemented including CPU emulation with accurate cycle
timing, VDP rendering with multiple display modes, PSG audio, memory banking
with multiple mapper support, ROM loading from multiple archive formats, input
handling (keyboard and gamepad), and libretro core support.

The emulator targets officially licensed and released SMS games for US, EU,
and Japan. Region and mapper type are auto-detected via CRC32 database; unknown
ROMs default to Sega mapper with NTSC timing.

## Build and Run Commands

```bash
# Build the emulator (standalone Ebiten/SDL version)
go build

# Run with a ROM file (auto-detects region from 357-game database)
go run main.go -rom <path-to-rom>

# Override region detection manually
go run main.go -rom <path-to-rom> -region ntsc
go run main.go -rom <path-to-rom> -region pal

# Crop left border (hides 8-pixel blank column when enabled by game)
go run main.go -rom <path-to-rom> -crop-border

# Run tests
go test ./...

# After building:
./emkiii -rom <path-to-rom>
./emkiii -rom <path-to-rom> -region pal -crop-border

# Build libretro core (for RetroArch and other frontends)
go build -tags libretro -buildmode=c-shared -o emkiii_libretro.dylib ./libretro/
```

**Supported ROM formats:** `.sms`, `.zip`, `.7z`, `.gz`, `.tar.gz`, `.rar` (auto-detected)

## Controls

**Keyboard:**
- **Movement:** W (up), A (left), S (down), D (right)
- **Buttons:** J (Button 1), K (Button 2)

**Gamepad** (PlayStation, Xbox, and standard controllers):
- **Movement:** D-pad or left analog stick
- **Buttons:** A/Cross (Button 1), B/Circle (Button 2)

## Architecture

The emulator uses Ebiten for windowing/rendering, koron-go/z80 for CPU emulation, and SDL2 for audio output.

**Package structure:**
- `main.go` - Entry point, CLI flag handling (`-rom`, `-region`, `-crop-border`), Ebiten game loop initialization
- `emu/` - All emulation components:
  - `emulator.go` - Core `EmulatorBase` struct orchestrating CPU/VDP/PSG/Memory, frame timing, scanline execution
  - `emulator_ebiten.go` - Standalone build: Ebiten rendering, SDL2 audio, keyboard/gamepad input, resizable window
  - `emulator_libretro.go` - Libretro build: minimal wrapper exposing framebuffer and audio samples
  - `z80.go` - Cycle-accurate Z80 wrapper with full opcode timing tables (base, CB, DD, ED, FD prefixes) and conditional instruction handling
  - `vdp.go` - Video Display Processor with VRAM (16KB), CRAM (32 bytes), 16 registers; implements background/sprite rendering, scrolling, interrupts, collision detection, per-scanline scroll latching, 192/224-line display modes
  - `mem.go` - 64KB memory space with Sega mapper ($FFFC-$FFFF) and Codemasters mapper ($0000/$4000/$8000) support, 32KB cartridge RAM
  - `io.go` - I/O port handler implementing z80.IO interface; maps VDP, PSG, and controller ports with SMS partial address decoding
  - `psg.go` - SN76489 sound chip with 3 tone channels, 1 noise channel, 4-bit volume, 15-bit LFSR; 48kHz stereo output
  - `region.go` - NTSC/PAL timing constants (CPU clock, scanlines, FPS), region auto-detection via CRC32 lookup
  - `romdb.go` - Embedded ROM database (357 games) mapping CRC32 to mapper type and region
- `romloader/` - ROM loading with archive support:
  - `loader.go` - Main loader with magic byte format detection
  - `zip.go`, `gzip.go`, `sevenzip.go`, `rar.go` - Archive format handlers
- `libretro/` - Libretro core implementation:
  - `main.go` - Libretro API exports, core options (region, crop border), XRGB8888 video output
  - `libretro.h`, `cfuncs.h` - C headers for libretro API

**Execution flow:** `Update()` runs one frame by stepping the CPU through
scanlines (262 NTSC / 313 PAL, ~228 cycles each), updating V/H counters,
checking interrupts, rendering via VDP, and generating PSG samples. Audio
samples are batched per-frame and queued to SDL2. `Draw()` blits the VDP
framebuffer to screen.

**Display modes:**
- 256×192 (standard Mode 4) - default
- 256×224 (extended height Mode 4) - enabled when M1 and M2 bits set
- 248×192/224 (cropped) - optional left border crop when VDP blank column enabled
- Window is resizable with aspect ratio preservation (default 2x scale)

**Region timing:**

| Region | CPU Clock | Scanlines | FPS |
|--------|-----------|-----------|-----|
| NTSC | 3.579545 MHz | 262 | 60 |
| PAL | 3.546893 MHz | 313 | 50 |

## Dependencies

- `github.com/hajimehoshi/ebiten/v2` - Windowing, rendering, input
- `github.com/koron-go/z80` - Z80 CPU emulation
- `github.com/veandco/go-sdl2` - Audio output
- `github.com/bodgit/sevenzip` - 7z archive support
- `github.com/nwaples/rardecode/v2` - RAR archive support

## Implementation Status

| Component | Status | Notes |
|-----------|--------|-------|
| CPU | Complete | Z80 via koron-go/z80 with accurate per-instruction cycle timing via lookup tables |
| Memory | Complete | 64KB with Sega mapper (3 slots + cart RAM) and Codemasters mapper (CRC32 detection) |
| VDP | Complete | Tiles, sprites (8×8/8×16, zoom), scrolling, priority, interrupts, per-scanline latching, 192/224-line modes |
| PSG | Complete | Full SN76489 emulation (3 tone + 1 noise), 48kHz output |
| I/O | Complete | Controller ports, VDP/PSG port decoding, V/H counter reads with accurate H-counter table |
| ROM Loading | Complete | Supports .sms, .zip, .7z, .gz, .tar.gz, .rar with magic byte detection |
| Input | Complete | Keyboard (WASD + JK) and gamepad (D-pad/stick + A/B) for P1 controller |
| Region | Complete | Auto-detection via CRC32 database (357 games), manual override with `-region` flag |
| Libretro | Complete | Full core implementation with region/crop options, works with RetroArch |
| Tests | Complete | Unit tests for I/O, memory, VDP, PSG, region timing, ROM loading, and libretro |

## Unsupported Functionality

This is functionality of the original hardware that is not planned or intended to be supported
in the future.

- Non-officially licensed and released games (homebrew, unlicensed)
- Beta or prototype games
- Korean mappers
- SG-1000
- Peripherals
  - FM Sound Unit
  - Light Phaser
  - Card Catcher
  - Telecon Pack
  - SK-1100
  - SF-7000

## Libretro Core Reload Issue

The libretro core cannot be unloaded and reloaded within the same RetroArch
session. This is a fundamental limitation of Go shared libraries—the Go runtime
cannot be safely unloaded via `dlclose()` and reinitialized via `dlopen()`.
When RetroArch closes a game and attempts to load another, the Go runtime
enters an inconsistent state causing crashes or hangs. **Workaround:** Restart
RetroArch between games when using this core.
