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
# Launch standalone UI (game library, settings, save states)
go run ./cmd/standalone/main.go

# Direct emulator mode (bypasses UI, loads ROM directly)
go run ./cmd/standalone/main.go -rom <path-to-rom>

# Override region detection manually
go run ./cmd/standalone/main.go -rom <path-to-rom> -region ntsc
go run ./cmd/standalone/main.go -rom <path-to-rom> -region pal

# Crop left border (hides 8-pixel blank column when enabled by game)
go run ./cmd/standalone/main.go -rom <path-to-rom> -crop-border

# Run tests
go test ./...
```

## Prerequisites

No external native libraries are required.

## Makefile

The top-level Makefile provides targets for building distributable applications.

| Target | Description |
|--------|-------------|
| `make standalone` | Build standalone binary to `build/emkiii` |
| `make macos` | Build macOS .app bundle to `build/eMkIII.app` |
| `make libretro` | Build libretro core to `build/emkiii_libretro.dylib` |
| `make icons` | Generate icons for macOS and iOS from `assets/icon.png` |
| `make clean` | Remove build directory |

The macOS app bundle is code-signed for distribution.

```bash
# Build macOS app
make macos

# Build libretro core
make libretro
```

## User Interfaces

### Standalone UI

When launched without a `-rom` argument, the emulator opens a standalone UI:

- **Game Library:** Browse games in icon or list view with sorting (title, last played, play time), favorites filtering, and search filter (press `/` to filter by title)
- **ROM Scanning:** Add ROM folders, scan for games with automatic metadata lookup from libretro database
- **Game Details:** View box art, metadata (developer, publisher, genre, release date), Play/Resume options
- **Save States:** 10 manual slots per game (F1/F2/F3), auto-save every 5 seconds, resume support
- **Rewind:** Hold R to rewind gameplay with acceleration curve; configurable buffer size and frame step
- **Screenshots:** F12 captures to screenshots directory
- **Play Time Tracking:** Automatic per-game tracking
- **Window Persistence:** Window size and position restored on launch
- **Themes:** 8 color themes (Default, Dark, Light, Retro, Pink, Hot Pink, Green LCD, High Contrast)
- **Font Scaling:** Adjustable font size (10-32pt) with automatic UI scaling
- **Audio Mute:** Toggle game audio on/off in settings
- **Shader Effects:** 20 visual effects including CRT simulation, scanlines, NTSC artifacts, and pixel smoothing

#### Controls

**Keyboard (Gameplay):**
- **Movement:** WASD
- **Buttons:** J (Button 1), K (Button 2)
- **SMS Pause:** Enter (hardware pause button, triggers NMI)

**Gamepad (Gameplay):**
- **Movement:** D-pad or left analog stick
- **Buttons:** A/Cross (Button 1), B/Circle (Button 2)
- **SMS Pause:** Start

**Library Navigation:**
- **Search/Filter:** `/` (type to filter games by title)
- **Clear Filter:** ESC
- **Navigate:** Arrow keys or D-pad (deactivates filter input but keeps filter visible)
- **Open Settings:** Start (gamepad)

**System Controls:**
- **Pause Menu:** ESC or Select (gamepad)
- **Rewind:** R (hold to rewind, accelerates over time)
- **Achievements:** Tab (toggle overlay during gameplay)
- **Save State:** F1
- **Load State:** F3
- **Next Slot:** F2
- **Previous Slot:** Shift+F2
- **Screenshot:** F12

**Pause Menu Navigation:**
- **Keyboard:** Arrow Up/Down, Enter to select, ESC to resume
- **Gamepad:** D-pad, A/Cross to select, B/Circle or Start to resume
- **Mouse:** Click or hover to select

#### Shader Effects

The standalone UI includes a comprehensive shader system for authentic retro display effects:

Shaders can be configured separately for UI (menus) and gameplay. Multiple
shaders can be stacked and are applied in weighted order for correct visual
layering.

#### RetroAchievements

The standalone UI integrates with
[RetroAchievements](https://retroachievements.org) to track and unlock
achievements while playing. Features include unlock notifications with badges,
unlock sound, auto-screenshot, achievement overlay during gameplay (Tab),
progress tracking on the game detail screen, Spectator Mode, and Encore Mode.
Configure in Settings > RetroAchievements.

**Note:** This emulator is not officially recognized by RetroAchievements, so
achievements unlock in softcore mode only.

#### Data Location

| OS | Path |
|----|------|
| macOS | `~/Library/Application Support/emkiii/` |
| Linux | `~/.config/emkiii/` |
| Windows | `%APPDATA%\emkiii\` |

#### Directory Structure

```
{data}/
|-- config.json          # Application settings
|-- library.json         # Game library and metadata
|-- metadata/sms.rdb     # Downloaded game database
|-- saves/{crc32}/       # Per-game save states and SRAM
|-- artwork/{crc32}/     # Per-game box art
+-- screenshots/         # Screenshots
```

### CLI (Direct Mode)

When launched with the `-rom` flag, the emulator bypasses the UI and loads the
ROM directly. This mode is useful for quick testing or integration with
external launchers.

#### Controls

**Keyboard:**
- **Movement:** WASD
- **Buttons:** J (Button 1), K (Button 2)
- **SMS Pause:** Enter

**Gamepad** (PlayStation, Xbox, and standard controllers):
- **Movement:** D-pad or left analog stick
- **Buttons:** A/Cross (Button 1), B/Circle (Button 2)

### iOS App

The iOS app is a native Swift application that embeds the emulator.

#### Prerequisites

- Xcode 15+ with iOS SDK
- Go 1.21+

#### Build Instructions

1. **Generate icons (from project root):**
   ```bash
   make icons
   ```
   This copies the app icon to the iOS asset catalog.

2. **Generate the framework:**
   ```bash
   cd ios
   make framework
   ```
   This creates `ios/Frameworks/Emulator.xcframework`.

3. **Configure code signing:**
   ```bash
   cp ios/Signing.xcconfig.template ios/Signing.xcconfig
   # Edit Signing.xcconfig and set your DEVELOPMENT_TEAM
   ```

4. **Build and run:**
   - Open `ios/eMkIII.xcodeproj` in Xcode
   - Select your target device
   - Build and run (Cmd+R)

#### Features

- Touch controls with virtual D-pad and buttons
- Game library with box art
- Resume state and SRAM persistence
- Gamepad support
- Metal rendering

**Supported ROM formats:** `.sms`, `.zip`, `.7z`, `.gz`, `.tar.gz`, `.rar` (auto-detected)

#### Controls

**Touch:**
- Virtual D-pad on the left side of the screen
- Button 1 and Button 2 on the right side

**Gamepad**
- **Movement:** D-pad or left analog stick
- **Buttons:** A (Button 1), B (Button 2)

## Architecture

The emulator core (`emu/`) is framework-agnostic and handles all SMS emulation
logic. UI, audio, rendering, ROM loading, and platform integration are provided
by the external `eblitui` modules. The emkiii project connects its core to
eblitui through an adapter that implements the `emucore.CoreFactory` interface
from `eblitui/api`. Each front-end entry point registers this factory with its
respective eblitui module.

**Package structure:**
- `adapter/adapter.go` - Implements `emucore.CoreFactory` from `eblitui/api`, defining system metadata (name, extensions, screen dimensions), button mappings, and core-specific options (crop border)
- `cmd/standalone/main.go` - Standalone UI entry point; registers the adapter factory with `eblitui/standalone`
- `cmd/libretro/main.go` - Libretro core entry point; registers the adapter factory with `eblitui/libretro`
- `cmd/ios/ios.go` - iOS bridge entry point; re-exports `eblitui-ios` functions for Swift integration
- `emu/` - Core emulation components (framework-agnostic):
  - `emulator.go` - Core `EmulatorBase` struct orchestrating CPU/VDP/PSG/Memory, frame timing, scanline execution
  - `bus.go` - SMSBus adapter bridging Memory and SMSIO into the go-chip-z80 Bus interface
  - `vdp.go` - Video Display Processor with VRAM (16KB), CRAM (32 bytes), 16 registers; implements background/sprite rendering, scrolling, interrupts, collision detection, per-scanline scroll latching, 192/224-line display modes
  - `mem.go` - 64KB memory space with Sega mapper ($FFFC-$FFFF) and Codemasters mapper ($0000/$4000/$8000) support, 32KB cartridge RAM
  - `io.go` - I/O port handler; maps VDP, PSG, and controller ports with SMS partial address decoding
  - `region.go` - NTSC/PAL timing constants (CPU clock, scanlines, FPS), region auto-detection via CRC32 lookup
  - `romdb.go` - Embedded ROM database mapping CRC32 to mapper type and region
  - `version.go` - Version constant
- `ios/` - Native iOS app (Swift/Xcode):
  - `eMkIII/` - App source: views, models, Metal renderer, audio engine
  - `eMkIII.xcodeproj/` - Xcode project
  - `Makefile` - framework build script
- `docs/` - Hardware documentation (VDP, PSG, SMS, Game Gear)

**Execution flow:** The emulator core steps the CPU through scanlines
(262 NTSC / 313 PAL, ~228 cycles each), updating V/H counters, checking
interrupts, rendering via VDP, and generating PSG samples. Audio output,
frame pacing, input handling, and rendering are managed by the eblitui
front-end modules.

**Display modes:**
- 256x192 (standard Mode 4) - default
- 256x224 (extended height Mode 4) - enabled when M1 and M2 bits set
- 248x192/224 (cropped) - optional left border crop when VDP blank column enabled
- Window is resizable with aspect ratio preservation (default 2x scale)

**Region timing:**

| Region | CPU Clock | Scanlines | FPS |
|--------|-----------|-----------|-----|
| NTSC | 3.579545 MHz | 262 | 60 |
| PAL | 3.546893 MHz | 313 | 50 |

## Dependencies

- `github.com/user-none/eblitui/api` - Core interface (`emucore.CoreFactory`)
- `github.com/user-none/eblitui/standalone` - Standalone ebiten UI
- `github.com/user-none/eblitui/libretro` - Libretro core framework
- `github.com/user-none/eblitui-ios` - iOS framework
- `github.com/user-none/go-chip-z80` - Z80 CPU emulation
- `github.com/user-none/go-chip-sn76489` - SN76489 PSG emulation

## Implementation Status

| Component | Status | Notes |
|-----------|--------|-------|
| CPU | Complete | Z80 via go-chip-z80 with built-in cycle-accurate timing, EI delay, and interrupt handling |
| Memory | Complete | 64KB with Sega mapper (3 slots + cart RAM) and Codemasters mapper (CRC32 detection) |
| VDP | Complete | Tiles, sprites (8x8/8x16, zoom), scrolling, priority, interrupts, per-scanline latching, 192/224-line modes |
| PSG | Complete | SN76489 via go-chip-sn76489 (3 tone + 1 noise), 48kHz output |
| I/O | Complete | Controller ports, VDP/PSG port decoding, V/H counter reads with accurate H-counter table |
| ROM Loading | Complete | Supports .sms, .zip, .7z, .gz, .tar.gz, .rar with magic byte detection |
| Input | Complete | Keyboard (WASD/Arrows + JK) and gamepad (D-pad/stick + A/B) for P1 controller |
| Region | Complete | Auto-detection via CRC32 database (357 games), manual override with `-region` flag |
| Libretro | Complete | Core implementation via eblitui/libretro with region/crop options, works with RetroArch |
| Standalone UI | Complete | Via eblitui/standalone: library management, save states (10 slots + auto-save), rewind, screenshots, themes, achievements, play time tracking |
| iOS App | Complete | Native Swift app via eblitui-ios with touch controls, Metal rendering, gamepad support, save states |
| Tests | Complete | Unit tests in emu/ for I/O, memory, VDP, PSG, region timing |

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
session. This is a fundamental limitation of Go shared libraries - the Go runtime
cannot be safely unloaded via `dlclose()` and reinitialized via `dlopen()`.
When RetroArch closes a game and attempts to load another, the Go runtime
enters an inconsistent state causing crashes or hangs. **Workaround:** Restart
RetroArch between games when using this core.
