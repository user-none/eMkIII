// Package emuios provides a gomobile-compatible interface to the emulator.
package emuios

import (
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"

	"github.com/user-none/emkiii/emu"
	"github.com/user-none/emkiii/romloader"
)

// ExtractResult contains the result of ROM extraction
type ExtractResult struct {
	Crc32    string // Hex string, e.g., "AABBCCDD"
	Filename string // Original filename from archive, e.g., "Sonic The Hedgehog (USA).sms"
}

// currentEmu holds the emulator state (unexported)
var currentEmu *emulatorState

type emulatorState struct {
	base      emu.EmulatorBase
	frameData []byte
	audioData []byte
	stateData []byte
	sramData  []byte
}

// InitFromPath creates an emulator from a ROM file path.
// Automatically extracts from ZIP/7z/gzip/RAR if needed.
// regionCode: 0=NTSC, 1=PAL
// Returns true on success, false on error.
func InitFromPath(path string, regionCode int) bool {
	rom, _, err := romloader.LoadROM(path)
	if err != nil {
		return false
	}

	region := emu.RegionNTSC
	if regionCode == 1 {
		region = emu.RegionPAL
	}
	currentEmu = &emulatorState{}
	currentEmu.base = emu.InitEmulatorBase(rom, region)
	return true
}

// Close releases the emulator.
func Close() {
	currentEmu = nil
}

// RunFrame executes one frame of emulation.
func RunFrame() {
	if currentEmu == nil {
		return
	}
	currentEmu.base.RunFrame()

	// Cache frame buffer - only the active display area (192 or 224 lines)
	// The full VDP framebuffer is always 224 lines, but we only return the active portion
	fullBuffer := currentEmu.base.GetFramebuffer()
	activeHeight := currentEmu.base.GetActiveHeight()
	stride := currentEmu.base.GetFramebufferStride()
	activeBytes := stride * activeHeight
	currentEmu.frameData = fullBuffer[:activeBytes]

	// Convert audio samples to bytes
	samples := currentEmu.base.GetAudioSamples()
	if len(samples) > 0 {
		currentEmu.audioData = make([]byte, len(samples)*2)
		for i, s := range samples {
			currentEmu.audioData[i*2] = byte(s)
			currentEmu.audioData[i*2+1] = byte(s >> 8)
		}
	} else {
		currentEmu.audioData = nil
	}
}

// FrameWidth returns the display width (always 256).
func FrameWidth() int {
	return 256
}

// FrameHeight returns the active display height (192 or 224).
func FrameHeight() int {
	if currentEmu == nil {
		return 192
	}
	return currentEmu.base.GetActiveHeight()
}

// GetFrameData returns the frame buffer for the active display area only.
// The buffer contains only activeHeight rows (192 or 224), not the full 224-line VDP buffer.
func GetFrameData() []byte {
	if currentEmu == nil {
		return nil
	}
	return currentEmu.frameData
}

// GetAudioData returns the entire audio buffer.
func GetAudioData() []byte {
	if currentEmu == nil {
		return nil
	}
	return currentEmu.audioData
}

// SetInput sets Player 1 controller state.
func SetInput(up, down, left, right, btn1, btn2 bool) {
	if currentEmu != nil {
		currentEmu.base.SetInput(up, down, left, right, btn1, btn2)
	}
}

// SetPause triggers the SMS pause button (NMI).
func SetPause() {
	if currentEmu != nil {
		currentEmu.base.SetPause()
	}
}

// Region returns the current region (0=NTSC, 1=PAL).
func Region() int {
	if currentEmu == nil {
		return 0
	}
	if currentEmu.base.GetRegion() == emu.RegionPAL {
		return 1
	}
	return 0
}

// LeftBlank returns whether VDP left column blanking is enabled.
func LeftBlank() bool {
	if currentEmu == nil {
		return false
	}
	return currentEmu.base.LeftColumnBlankEnabled()
}

// SaveState creates a save state. Returns true on success.
func SaveState() bool {
	if currentEmu == nil {
		return false
	}
	data, err := currentEmu.base.Serialize()
	if err != nil {
		currentEmu.stateData = nil
		return false
	}
	currentEmu.stateData = data
	return true
}

// StateLen returns the length of the last saved state.
func StateLen() int {
	if currentEmu == nil {
		return 0
	}
	return len(currentEmu.stateData)
}

// StateByte returns a single byte from the saved state at index i.
func StateByte(i int) int {
	if currentEmu == nil || i < 0 || i >= len(currentEmu.stateData) {
		return 0
	}
	return int(currentEmu.stateData[i])
}

// LoadState loads a save state. Returns true on success.
func LoadState(data []byte) bool {
	if currentEmu == nil {
		return false
	}
	return currentEmu.base.Deserialize(data) == nil
}

// PrepareSRAM copies SRAM to internal buffer.
func PrepareSRAM() {
	if currentEmu == nil {
		return
	}
	ram := currentEmu.base.GetCartRAM()
	currentEmu.sramData = make([]byte, len(ram))
	copy(currentEmu.sramData, ram[:])
}

// SRAMLen returns the SRAM length (32768).
func SRAMLen() int {
	if currentEmu == nil {
		return 0
	}
	return len(currentEmu.sramData)
}

// SRAMByte returns a single byte from SRAM at index i.
func SRAMByte(i int) int {
	if currentEmu == nil || i < 0 || i >= len(currentEmu.sramData) {
		return 0
	}
	return int(currentEmu.sramData[i])
}

// LoadSRAM loads 32KB cartridge RAM.
func LoadSRAM(data []byte) {
	if currentEmu == nil || len(data) != 0x8000 {
		return
	}
	ram := currentEmu.base.GetCartRAM()
	copy(ram[:], data)
}

// DetectRegionFromPath returns the detected region for a ROM file (0=NTSC, 1=PAL).
// Automatically extracts from ZIP/7z/gzip/RAR if needed.
func DetectRegionFromPath(path string) int {
	rom, _, err := romloader.LoadROM(path)
	if err != nil {
		return 0 // Default to NTSC on error
	}

	region, _ := emu.DetectRegionFromROM(rom)
	if region == emu.RegionPAL {
		return 1
	}
	return 0
}

// GetFPS returns the target FPS for a region code.
func GetFPS(regionCode int) int {
	if regionCode == 1 {
		return 50
	}
	return 60
}

// GetCRC32FromPath calculates the CRC32 checksum of a ROM file.
// Automatically extracts from ZIP/7z/gzip/RAR if needed.
// Returns -1 on error.
func GetCRC32FromPath(path string) int64 {
	rom, _, err := romloader.LoadROM(path)
	if err != nil {
		return -1
	}

	return int64(crc32.ChecksumIEEE(rom))
}

// ExtractAndStoreROM extracts a ROM from an archive (or copies a raw ROM),
// calculates its CRC32, and stores it as {destDir}/{CRC32}.sms.
// If a file with the same CRC32 already exists, it skips writing.
// Returns the CRC32 and original filename on success, or an error.
func ExtractAndStoreROM(srcPath, destDir string) (*ExtractResult, error) {
	// Extract ROM (handles zip, 7z, gzip, rar, or raw .sms)
	rom, filename, err := romloader.LoadROM(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load ROM: %w", err)
	}

	// Calculate CRC32
	crc := crc32.ChecksumIEEE(rom)
	crcHex := fmt.Sprintf("%08X", crc)

	// Build destination path
	destPath := filepath.Join(destDir, crcHex+".sms")

	// Skip write if file already exists (same CRC = same content)
	if _, err := os.Stat(destPath); err == nil {
		return &ExtractResult{Crc32: crcHex, Filename: filename}, nil
	}

	// Write extracted ROM
	if err := os.WriteFile(destPath, rom, 0644); err != nil {
		return nil, fmt.Errorf("failed to write ROM: %w", err)
	}

	return &ExtractResult{Crc32: crcHex, Filename: filename}, nil
}
