//go:build !libretro

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/user-none/emkiii/cli"
	"github.com/user-none/emkiii/emu"
	"github.com/user-none/emkiii/romloader"
	"github.com/user-none/emkiii/ui"
)

func main() {
	romPath := flag.String("rom", "", "path to ROM file (optional - opens UI if not provided)")
	regionFlag := flag.String("region", "auto", "region: auto, ntsc, or pal")
	cropBorder := flag.Bool("crop-border", false, "crop left border when blank")
	flag.Parse()

	// If no ROM provided, launch the UI
	if *romPath == "" {
		launchUI()
		return
	}

	// Direct emulator mode (existing behavior)
	romData, _, err := romloader.LoadROM(*romPath)
	if err != nil {
		log.Fatalf("Failed to load ROM: %v", err)
	}

	// Determine region
	var region emu.Region
	switch strings.ToLower(*regionFlag) {
	case "auto":
		region, _ = emu.DetectRegionFromROM(romData)
	case "ntsc":
		region = emu.RegionNTSC
	case "pal":
		region = emu.RegionPAL
	default:
		log.Fatalf("Invalid region: %s (use auto, ntsc, or pal)", *regionFlag)
	}

	timing := emu.GetTimingForRegion(region)
	e := emu.NewEmulator(romData, region)

	ebiten.SetWindowSize(emu.ScreenWidth*2, 192*2) // Default size for 192-line mode
	ebiten.SetWindowTitle("eMKIII")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(348, 348, -1, -1) // Min 348x348, no max
	ebiten.SetTPS(timing.FPS)

	runner := cli.NewRunner(e, *cropBorder)
	defer runner.Close()
	defer e.Close()

	if err := ebiten.RunGame(runner); err != nil {
		log.Fatal(err)
	}
}

// launchUI starts the standalone UI application
func launchUI() {
	app, err := ui.NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize: %v\n", err)
		os.Exit(1)
	}

	ebiten.SetWindowTitle("eMKIII")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(900, 650, -1, -1)

	// Set window size from saved config (before RunGame to avoid resize flash)
	// Fall back to minimum size if config doesn't have valid dimensions
	width, height, x, y := app.GetWindowConfig()
	if width < 900 {
		width = 900
	}
	if height < 650 {
		height = 650
	}
	ebiten.SetWindowSize(width, height)

	// Restore window position if previously saved
	if x != nil && y != nil {
		ebiten.SetWindowPosition(*x, *y)
	}

	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}

	// Save before exit
	app.SaveAndClose()
}
