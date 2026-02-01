//go:build libretro

package emu

// Emulator wraps EmulatorBase with libretro-specific functionality
type Emulator struct {
	EmulatorBase
}

// NewEmulatorForLibretro creates an emulator instance for libretro use (no SDL/Ebiten)
func NewEmulatorForLibretro(rom []byte, region Region) *Emulator {
	base := initEmulatorBase(rom, region)

	return &Emulator{
		EmulatorBase: base,
	}
}
