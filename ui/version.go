//go:build !libretro

package ui

import "github.com/user-none/emkiii/emu"

// Standalone UI version
const (
	Name    = emu.Name + "-Standalone"
	Version = "1.0.0"
)
