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
# Build the emulator
go build

# Launch standalone UI (game library, settings, save states)
go run main.go

# Direct emulator mode (bypasses UI, loads ROM directly)
go run main.go -rom <path-to-rom>

# Override region detection manually
go run main.go -rom <path-to-rom> -region ntsc
go run main.go -rom <path-to-rom> -region pal

# Crop left border (hides 8-pixel blank column when enabled by game)
go run main.go -rom <path-to-rom> -crop-border

# Run tests
go test ./...

# After building:
./emkiii                                        # Launch UI
./emkiii -rom <path-to-rom>                     # Direct mode
./emkiii -rom <path-to-rom> -region pal -crop-border

# Build libretro core (for RetroArch and other frontends)
go build -tags libretro -buildmode=c-shared -o emkiii_libretro.dylib ./bridge/libretro/
```

## iOS App

The iOS app is a native Swift application that embeds the emulator via gomobile.

### Prerequisites

- Xcode 15+ with iOS SDK
- Go 1.21+
- gomobile: `go install golang.org/x/mobile/cmd/gomobile@latest && gomobile init`

### Build Instructions

1. **Generate the gomobile framework:**
   ```bash
   cd ios
   make framework
   ```
   This creates `ios/Frameworks/Emulator.xcframework`.

2. **Configure code signing:**
   ```bash
   cp ios/Signing.xcconfig.template ios/Signing.xcconfig
   # Edit Signing.xcconfig and set your DEVELOPMENT_TEAM
   ```

3. **Build and run:**
   - Open `ios/eMkIII.xcodeproj` in Xcode
   - Select your target device
   - Build and run (Cmd+R)

### iOS Features

- Touch controls with virtual D-pad and buttons
- Game library with box art
- Resume state and SRAM persistence
- Gamepad support (MFi controllers)
- Metal rendering

**Supported ROM formats:** `.sms`, `.zip`, `.7z`, `.gz`, `.tar.gz`, `.rar` (auto-detected)

## Controls

### Direct Mode (`-rom` flag)

**Keyboard:**
- **Movement:** W (up), A (left), S (down), D (right)
- **Buttons:** J (Button 1), K (Button 2)

**Gamepad** (PlayStation, Xbox, and standard controllers):
- **Movement:** D-pad or left analog stick
- **Buttons:** A/Cross (Button 1), B/Circle (Button 2)

### Standalone UI

All direct mode controls plus:

**Gameplay:**
- **SMS Pause:** Enter (hardware pause button, triggers NMI)

**System Controls:**
- **Pause Menu:** ESC (during gameplay)
- **Save State:** F1
- **Load State:** F3
- **Next Slot:** F2
- **Previous Slot:** Shift+F2
- **Screenshot:** F12

**Pause Menu Navigation:**
- **Keyboard:** Arrow Up/Down, Enter to select, ESC to resume
- **Gamepad:** D-pad, A/Cross to select, B/Circle or Start to resume

## Standalone UI

When launched without a `-rom` argument, the emulator opens a standalone UI:

- **Game Library:** Browse games in icon or list view with sorting (title, last played, play time) and favorites filtering
- **ROM Scanning:** Add ROM folders, scan for games with automatic metadata lookup from libretro database
- **Game Details:** View box art, metadata (developer, publisher, genre, release date), Play/Resume options
- **Save States:** 10 manual slots per game (F1/F2/F3), auto-save every 5 seconds, resume support
- **Screenshots:** F12 captures to screenshots directory
- **Play Time Tracking:** Automatic per-game tracking
- **Window Persistence:** Window size and position restored on launch
- **Shader Effects:** 20 visual effects including CRT simulation, scanlines, NTSC artifacts, and pixel smoothing

### Shader Effects

The standalone UI includes a comprehensive shader system for authentic retro display effects:

| Category | Shaders |
|----------|---------|
| CRT Simulation | CRT (curved screen), Scanlines, Phosphor Glow, Halation, CRT Gamma |
| Analog Video | NTSC Artifacts, NTSC Rainbow, Color Bleed, Horizontal Softness, Vertical Blur |
| Display Types | LCD Grid, Dot Matrix, Interlace |
| Signal Degradation | VHS Distortion, Rolling Band, RF Noise |
| Color Effects | Monochrome, Sepia |
| Enhancement | Pixel Smoothing (xBR), Phosphor Persistence |

Shaders can be configured separately for UI (menus) and gameplay. Multiple shaders can be stacked and are applied in weighted order for correct visual layering.

### Data Location

| OS | Path |
|----|------|
| macOS | `~/Library/Application Support/emkiii/` |
| Linux | `~/.config/emkiii/` |
| Windows | `%APPDATA%\emkiii\` |

### Directory Structure

```
{data}/
├── config.json          # Application settings
├── library.json         # Game library and metadata
├── metadata/sms.rdb     # Downloaded game database
├── saves/{crc32}/       # Per-game save states and SRAM
├── artwork/{crc32}/     # Per-game box art
└── screenshots/         # Screenshots
```

## Architecture

The emulator uses Ebiten for windowing/rendering, koron-go/z80 for CPU emulation, SDL3 for audio output, and ebitenui for the standalone UI.

The standalone UI follows a manager pattern with clear separation of concerns:
- `App` - Main application, implements `ebiten.Game`, owns screens and managers
- `GameplayManager` - Emulator lifecycle, input handling, auto-save, play time tracking
- `InputManager` - UI navigation with gamepad repeat support
- `ScanManager` - ROM scanning orchestration and library updates

**Package structure:**
- `main.go` - Entry point; launches UI by default, or direct emulator with `-rom` flag
- `ui/` - Standalone UI application
  - `app.go` - Main application struct, screen management, Ebiten game loop
  - `state.go` - AppState enum (Library, Detail, Settings, etc.)
  - `gameplay.go` - GameplayManager: emulator lifecycle, input, auto-save, play time tracking
  - `input.go` - InputManager: UI navigation with gamepad repeat support
  - `scan_manager.go` - ScanManager: ROM scanning orchestration
  - `screens/` - Library, Detail, Settings, Scan Progress, Error screens
    - `base.go` - BaseScreen: common scroll/focus management for screens
  - `pausemenu.go` - In-game pause overlay with keyboard/gamepad navigation
  - `savestate.go` - Save state management (10 slots per game, auto-save)
  - `screenshot.go` - Screenshot capture (F12)
  - `notification.go` - On-screen notifications
  - `scanner.go` - ROM discovery and metadata lookup
  - `metadata.go` - RDB download and artwork fetching
  - `storage/` - Config and library JSON persistence
  - `style/` - Theme colors, widget builders, constants
  - `shader/` - Shader system with weight-based ordering
  - `rdb/` - RDB parser for game metadata lookup
  - `assets/` - Embedded placeholder images
- `cli/` - CLI runner for direct ROM launch mode:
  - `runner.go` - Ebiten game wrapper for direct emulation (bypasses UI)
- `emu/` - Core emulation components (framework-agnostic):
  - `emulator.go` - Core `EmulatorBase` struct orchestrating CPU/VDP/PSG/Memory, frame timing, scanline execution
  - `z80.go` - Cycle-accurate Z80 wrapper with full opcode timing tables (base, CB, DD, ED, FD prefixes) and conditional instruction handling
  - `vdp.go` - Video Display Processor with VRAM (16KB), CRAM (32 bytes), 16 registers; implements background/sprite rendering, scrolling, interrupts, collision detection, per-scanline scroll latching, 192/224-line display modes
  - `mem.go` - 64KB memory space with Sega mapper ($FFFC-$FFFF) and Codemasters mapper ($0000/$4000/$8000) support, 32KB cartridge RAM
  - `io.go` - I/O port handler implementing z80.IO interface; maps VDP, PSG, and controller ports with SMS partial address decoding
  - `psg.go` - SN76489 sound chip with 3 tone channels, 1 noise channel, 4-bit volume, 15-bit LFSR; 48kHz stereo output
  - `region.go` - NTSC/PAL timing constants (CPU clock, scanlines, FPS), region auto-detection via CRC32 lookup
  - `romdb.go` - Embedded ROM database (357 games) mapping CRC32 to mapper type and region
- `bridge/` - Platform-specific wrappers:
  - `ebiten/emulator.go` - Ebiten wrapper: rendering, SDL3 audio, keyboard/gamepad input, resizable window
  - `ios/ios.go` - gomobile bridge for iOS: exposes emulator API to Swift
  - `libretro/main.go` - Libretro core: API exports, core options (region, crop border), XRGB8888 video output
  - `libretro/libretro.h`, `cfuncs.h` - C headers for libretro API
- `romloader/` - ROM loading with archive support:
  - `loader.go` - Main loader with magic byte format detection
  - `zip.go`, `gzip.go`, `sevenzip.go`, `rar.go` - Archive format handlers
- `ios/` - Native iOS app (Swift/SwiftUI):
  - `eMkIII/` - App source: views, models, Metal renderer, audio engine
  - `eMkIII.xcodeproj/` - Xcode project
  - `Makefile` - gomobile framework build script

**Execution flow:** `Update()` runs one frame by stepping the CPU through
scanlines (262 NTSC / 313 PAL, ~228 cycles each), updating V/H counters,
checking interrupts, rendering via VDP, and generating PSG samples. Audio
samples are batched per-frame and queued to SDL3. `Draw()` blits the VDP
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
- `github.com/ebitenui/ebitenui` - Retained-mode UI widgets
- `github.com/sqweek/dialog` - Native OS file picker dialogs
- `github.com/koron-go/z80` - Z80 CPU emulation
- `github.com/Zyko0/go-sdl3` - Audio output (purego-based, no CGO)
- `github.com/bodgit/sevenzip` - 7z archive support
- `github.com/nwaples/rardecode/v2` - RAR archive support
- `golang.org/x/image` - Font rendering

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
| Standalone UI | Complete | Library management, save states (10 slots + auto-save), screenshots, play time tracking |
| iOS App | Complete | Native Swift app with touch controls, Metal rendering, gamepad support, save states |
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
