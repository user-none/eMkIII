package main

import (
	libretro "github.com/user-none/eblitui/libretro"
	"github.com/user-none/emkiii/adapter"
)

func init() {
	libretro.RegisterFactory(&adapter.Factory{}, []libretro.RetropadMapping{
		{RetroID: libretro.JoypadA, BitID: 4},     // Button 1
		{RetroID: libretro.JoypadB, BitID: 5},     // Button 2
		{RetroID: libretro.JoypadStart, BitID: 7}, // Pause/Start
	})
}

func main() {}
