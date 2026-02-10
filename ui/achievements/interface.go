//go:build !libretro && !ios

package achievements

// EmulatorInterface defines the interface for emulator memory access.
// This decouples the achievement manager from the concrete emulator type.
type EmulatorInterface interface {
	GetSystemRAM() *[0x2000]uint8
	GetCartRAM() *[0x8000]uint8
}
