//go:build !libretro && !ios

package main

import (
	"flag"
	"log"

	"github.com/user-none/eblitui/desktop"
	"github.com/user-none/emkiii/adapter"
)

func main() {
	romPath := flag.String("rom", "", "path to ROM file (opens UI if not provided)")
	regionFlag := flag.String("region", "auto", "video standard: auto, ntsc, or pal")
	cropBorder := flag.Bool("crop-border", false, "crop blank left column when enabled by game")
	flag.Parse()

	factory := &adapter.Factory{}

	if *romPath != "" {
		options := map[string]string{
			"video_standard": *regionFlag,
		}
		if *cropBorder {
			options["crop_border"] = "true"
		}
		if err := desktop.RunDirect(factory, *romPath, options, nil); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := desktop.Run(factory); err != nil {
		log.Fatal(err)
	}
}
