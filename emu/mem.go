package emu

import "hash/crc32"

// MapperType identifies the memory mapper used by the cartridge
type MapperType int

const (
	MapperSega        MapperType = iota // Standard Sega mapper ($FFFC-$FFFF)
	MapperCodemasters                   // Codemasters mapper ($0000, $4000, $8000)
)

// Memory implements SMS memory map with support for multiple mappers
type Memory struct {
	rom        []uint8
	ram        [0x2000]uint8 // 8KB system RAM
	cartRAM    [0x8000]uint8 // 32KB cartridge RAM (for battery backup / work RAM)
	bankSlot   [3]uint8      // Bank numbers for slots 0, 1, 2
	ramControl uint8         // $FFFC: RAM mapping control (Sega mapper only)
	bankMask   uint8         // Mask for valid bank numbers (based on ROM size)
	mapper     MapperType    // Which mapper this ROM uses
}

func NewMemory(rom []byte) *Memory {
	m := &Memory{
		rom: make([]uint8, len(rom)),
	}
	copy(m.rom, rom)

	// Calculate bank mask based on ROM size (number of 16KB banks)
	// Round up to next power of 2 for proper wrapping
	bankCount := (len(rom) + 0x3FFF) / 0x4000
	if bankCount == 0 {
		bankCount = 1
	}
	// Find next power of 2
	pow2 := 1
	for pow2 < bankCount {
		pow2 <<= 1
	}
	m.bankMask = uint8(pow2 - 1)

	// Detect mapper type
	m.mapper = detectMapper(rom)

	// Default bank mapping depends on mapper type
	// Sega mapper: slots map to banks 0, 1, 2
	// Codemasters mapper: slots map to banks 0, 1, 0 (slot 2 starts at bank 0)
	m.bankSlot[0] = 0
	m.bankSlot[1] = 1
	if m.mapper == MapperCodemasters {
		m.bankSlot[2] = 0
	} else {
		m.bankSlot[2] = 2
	}

	return m
}

// detectMapper identifies the mapper type based on ROM CRC32.
func detectMapper(rom []byte) MapperType {
	crc := crc32.ChecksumIEEE(rom)
	if info, ok := romDatabase[crc]; ok {
		return info.Mapper
	}
	return MapperSega
}

// Get reads a byte from memory, dispatching to the appropriate mapper
func (m *Memory) Get(addr uint16) uint8 {
	switch m.mapper {
	case MapperCodemasters:
		return m.getCodemasters(addr)
	default:
		return m.getSegaMapper(addr)
	}
}

// Set writes a byte to memory, dispatching to the appropriate mapper
func (m *Memory) Set(addr uint16, val uint8) {
	switch m.mapper {
	case MapperCodemasters:
		m.setCodemasters(addr, val)
	default:
		m.setSegaMapper(addr, val)
	}
}

// ----------------------------------------------------------------------------
// Sega Mapper
// ----------------------------------------------------------------------------
// Memory map:
//   $0000-$03FF: ROM (first 1KB, always mapped to bank 0)
//   $0400-$3FFF: ROM slot 0 (selectable via $FFFD)
//   $4000-$7FFF: ROM slot 1 (selectable via $FFFE)
//   $8000-$BFFF: ROM slot 2 (selectable via $FFFF) or cartridge RAM
//   $C000-$DFFF: RAM (8KB)
//   $E000-$FFFF: RAM mirror + bank registers at $FFFC-$FFFF

func (m *Memory) getSegaMapper(addr uint16) uint8 {
	switch {
	case addr < 0x0400:
		// First 1KB always from ROM bank 0
		if int(addr) < len(m.rom) {
			return m.rom[addr]
		}
		return 0xFF

	case addr < 0x4000:
		// Slot 0: $0400-$3FFF (bankable)
		bank := uint32(m.bankSlot[0] & m.bankMask)
		romAddr := bank*0x4000 + uint32(addr)
		if romAddr < uint32(len(m.rom)) {
			return m.rom[romAddr]
		}
		return 0xFF

	case addr < 0x8000:
		// Slot 1: $4000-$7FFF
		bank := uint32(m.bankSlot[1] & m.bankMask)
		romAddr := bank*0x4000 + uint32(addr-0x4000)
		if romAddr < uint32(len(m.rom)) {
			return m.rom[romAddr]
		}
		return 0xFF

	case addr < 0xC000:
		// Slot 2: $8000-$BFFF
		// Check if cartridge RAM is enabled (bit 3 of $FFFC)
		if m.ramControl&0x08 != 0 {
			ramBank := uint32((m.ramControl >> 2) & 0x01)
			ramAddr := ramBank*0x4000 + uint32(addr-0x8000)
			return m.cartRAM[ramAddr]
		}
		// Normal ROM banking
		bank := uint32(m.bankSlot[2] & m.bankMask)
		romAddr := bank*0x4000 + uint32(addr-0x8000)
		if romAddr < uint32(len(m.rom)) {
			return m.rom[romAddr]
		}
		return 0xFF

	default:
		// $C000-$FFFF: RAM (8KB mirrored)
		return m.ram[addr&0x1FFF]
	}
}

func (m *Memory) setSegaMapper(addr uint16, val uint8) {
	switch {
	case addr < 0x8000:
		// ROM area - writes ignored

	case addr < 0xC000:
		// Slot 2: $8000-$BFFF - cartridge RAM if enabled
		if m.ramControl&0x08 != 0 {
			ramBank := uint32((m.ramControl >> 2) & 0x01)
			ramAddr := ramBank*0x4000 + uint32(addr-0x8000)
			m.cartRAM[ramAddr] = val
		}

	default:
		// $C000-$FFFF: RAM
		m.ram[addr&0x1FFF] = val

		// Bank control registers at $FFFC-$FFFF
		switch addr {
		case 0xFFFC:
			m.ramControl = val
		case 0xFFFD:
			m.bankSlot[0] = val
		case 0xFFFE:
			m.bankSlot[1] = val
		case 0xFFFF:
			m.bankSlot[2] = val
		}
	}
}

// ----------------------------------------------------------------------------
// Codemasters Mapper
// ----------------------------------------------------------------------------
// Memory map:
//   $0000-$3FFF: ROM slot 0 (entire 16KB bankable, selected via write to $0000)
//   $4000-$7FFF: ROM slot 1 (selected via write to $4000)
//   $8000-$BFFF: ROM slot 2 (selected via write to $8000)
//   $C000-$DFFF: RAM (8KB)
//   $E000-$FFFF: RAM mirror

func (m *Memory) getCodemasters(addr uint16) uint8 {
	switch {
	case addr < 0x4000:
		// Slot 0: $0000-$3FFF (entire slot is bankable)
		bank := uint32(m.bankSlot[0] & m.bankMask)
		romAddr := bank*0x4000 + uint32(addr)
		if romAddr < uint32(len(m.rom)) {
			return m.rom[romAddr]
		}
		return 0xFF

	case addr < 0x8000:
		// Slot 1: $4000-$7FFF
		bank := uint32(m.bankSlot[1] & m.bankMask)
		romAddr := bank*0x4000 + uint32(addr-0x4000)
		if romAddr < uint32(len(m.rom)) {
			return m.rom[romAddr]
		}
		return 0xFF

	case addr < 0xC000:
		// Slot 2: $8000-$BFFF
		bank := uint32(m.bankSlot[2] & m.bankMask)
		romAddr := bank*0x4000 + uint32(addr-0x8000)
		if romAddr < uint32(len(m.rom)) {
			return m.rom[romAddr]
		}
		return 0xFF

	default:
		// $C000-$FFFF: RAM (8KB mirrored)
		return m.ram[addr&0x1FFF]
	}
}

func (m *Memory) setCodemasters(addr uint16, val uint8) {
	switch {
	case addr < 0x4000:
		// Write to $0000 sets slot 0 bank
		if addr == 0x0000 {
			m.bankSlot[0] = val
		}

	case addr < 0x8000:
		// Write to $4000 sets slot 1 bank
		if addr == 0x4000 {
			m.bankSlot[1] = val
		}

	case addr < 0xC000:
		// Write to $8000 sets slot 2 bank
		if addr == 0x8000 {
			m.bankSlot[2] = val
		}

	default:
		// $C000-$FFFF: RAM (no bank registers here for Codemasters)
		m.ram[addr&0x1FFF] = val
	}
}

// GetBankSlot returns the bank number mapped to the given slot (0-2)
func (m *Memory) GetBankSlot(slot int) uint8 {
	return m.bankSlot[slot]
}

// GetRAMControl returns the RAM mapping control byte ($FFFC)
func (m *Memory) GetRAMControl() uint8 {
	return m.ramControl
}

// GetSystemRAM returns a pointer to the 8KB system RAM for external access.
// Used by libretro for RetroAchievements memory exposure.
func (m *Memory) GetSystemRAM() *[0x2000]uint8 {
	return &m.ram
}

// GetCartRAM returns a pointer to the 32KB cartridge RAM for external access.
// Used by libretro for battery-backed save RAM persistence.
func (m *Memory) GetCartRAM() *[0x8000]uint8 {
	return &m.cartRAM
}

// GetROMCRC32 returns the CRC32 checksum of the loaded ROM.
// Used for save state verification to ensure states are loaded with the correct ROM.
func (m *Memory) GetROMCRC32() uint32 {
	return crc32.ChecksumIEEE(m.rom)
}
