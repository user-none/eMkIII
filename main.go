//go:build !libretro

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/user-none/emkiii/emu"
	"github.com/user-none/emkiii/romloader"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	romPath := flag.String("rom", "", "path to ROM file")
	regionFlag := flag.String("region", "auto", "region: auto, ntsc, or pal")
	cropBorder := flag.Bool("crop-border", false, "crop left border when blank")
	flag.Parse()

	if *romPath == "" {
		fmt.Println("Usage: go run main.go -rom <romfile> [-region auto|ntsc|pal] [-crop-border]")
		os.Exit(1)
	}

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
	e := emu.NewEmulator(romData, region, *cropBorder)

	ebiten.SetWindowSize(emu.ScreenWidth*2, 192*2) // Default size for 192-line mode
	ebiten.SetWindowTitle("eMKIII")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(348, 348, -1, -1) // Min 348x348, no max
	ebiten.SetTPS(timing.FPS)

	if err := ebiten.RunGame(e); err != nil {
		log.Fatal(err)
	}
}
