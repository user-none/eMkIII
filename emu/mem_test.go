package emu

import "testing"

// TestMemory_RAMReadWrite tests basic RAM operations at $C000-$DFFF
func TestMemory_RAMReadWrite(t *testing.T) {
	rom := createTestROM(4)
	mem := NewMemory(rom)

	// Write various values to RAM
	testCases := []struct {
		addr uint16
		val  uint8
	}{
		{0xC000, 0x42},
		{0xC001, 0xFF},
		{0xCFFF, 0xAB},
		{0xD000, 0xCD},
		{0xDFFF, 0x12},
	}

	for _, tc := range testCases {
		mem.Set(tc.addr, tc.val)
		got := mem.Get(tc.addr)
		if got != tc.val {
			t.Errorf("RAM[0x%04X]: expected 0x%02X, got 0x%02X", tc.addr, tc.val, got)
		}
	}
}

// TestMemory_RAMMirroring tests that $E000-$FFFF mirrors $C000-$DFFF
func TestMemory_RAMMirroring(t *testing.T) {
	rom := createTestROM(4)
	mem := NewMemory(rom)

	// Write to $C000-$DFFF, read from mirror at $E000-$FFFF
	testCases := []struct {
		writeAddr uint16
		readAddr  uint16
		val       uint8
	}{
		{0xC000, 0xE000, 0x42},
		{0xC100, 0xE100, 0xAB},
		{0xDFFF, 0xFFFF, 0xCD}, // Note: $FFFF is bank slot register but also mirrors RAM
	}

	for _, tc := range testCases {
		mem.Set(tc.writeAddr, tc.val)
		got := mem.Get(tc.readAddr)
		if got != tc.val {
			t.Errorf("Mirror test: wrote 0x%02X to 0x%04X, read from 0x%04X: expected 0x%02X, got 0x%02X",
				tc.val, tc.writeAddr, tc.readAddr, tc.val, got)
		}
	}

	// Also test reverse: write to mirror, read from base
	mem.Set(0xE500, 0x99)
	if got := mem.Get(0xC500); got != 0x99 {
		t.Errorf("Reverse mirror: wrote to 0xE500, read from 0xC500: expected 0x99, got 0x%02X", got)
	}
}

// TestMemory_Slot0Banking tests bank switching via $FFFD (slot 0, $0400-$3FFF)
func TestMemory_Slot0Banking(t *testing.T) {
	rom := createTestROM(8) // 8 banks = 128KB
	mem := NewMemory(rom)

	// Initially slot 0 should map to bank 0
	if got := mem.Get(0x0400); got != 0x00 {
		t.Errorf("Initial slot 0 value at 0x0400: expected 0x00, got 0x%02X", got)
	}

	// Switch slot 0 to bank 5
	mem.Set(0xFFFD, 5)
	if got := mem.GetBankSlot(0); got != 5 {
		t.Errorf("Bank slot 0 after switch: expected 5, got %d", got)
	}

	// Verify data comes from bank 5
	if got := mem.Get(0x0400); got != 0x05 {
		t.Errorf("Slot 0 after switch to bank 5: expected 0x05, got 0x%02X", got)
	}

	// End of slot 0 region
	if got := mem.Get(0x3FFF); got != 0x05 {
		t.Errorf("Slot 0 at 0x3FFF after switch to bank 5: expected 0x05, got 0x%02X", got)
	}
}

// TestMemory_Slot1Banking tests bank switching via $FFFE (slot 1, $4000-$7FFF)
func TestMemory_Slot1Banking(t *testing.T) {
	rom := createTestROM(8)
	mem := NewMemory(rom)

	// Initially slot 1 should map to bank 1
	if got := mem.Get(0x4000); got != 0x01 {
		t.Errorf("Initial slot 1 value at 0x4000: expected 0x01, got 0x%02X", got)
	}

	// Switch slot 1 to bank 3
	mem.Set(0xFFFE, 3)
	if got := mem.GetBankSlot(1); got != 3 {
		t.Errorf("Bank slot 1 after switch: expected 3, got %d", got)
	}

	// Verify data comes from bank 3
	if got := mem.Get(0x4000); got != 0x03 {
		t.Errorf("Slot 1 after switch to bank 3: expected 0x03, got 0x%02X", got)
	}

	if got := mem.Get(0x7FFF); got != 0x03 {
		t.Errorf("Slot 1 at 0x7FFF after switch to bank 3: expected 0x03, got 0x%02X", got)
	}
}

// TestMemory_Slot2Banking tests bank switching via $FFFF (slot 2, $8000-$BFFF)
func TestMemory_Slot2Banking(t *testing.T) {
	rom := createTestROM(8)
	mem := NewMemory(rom)

	// Initially slot 2 should map to bank 2
	if got := mem.Get(0x8000); got != 0x02 {
		t.Errorf("Initial slot 2 value at 0x8000: expected 0x02, got 0x%02X", got)
	}

	// Switch slot 2 to bank 7
	mem.Set(0xFFFF, 7)
	if got := mem.GetBankSlot(2); got != 7 {
		t.Errorf("Bank slot 2 after switch: expected 7, got %d", got)
	}

	// Verify data comes from bank 7
	if got := mem.Get(0x8000); got != 0x07 {
		t.Errorf("Slot 2 after switch to bank 7: expected 0x07, got 0x%02X", got)
	}

	if got := mem.Get(0xBFFF); got != 0x07 {
		t.Errorf("Slot 2 at 0xBFFF after switch to bank 7: expected 0x07, got 0x%02X", got)
	}
}

// TestMemory_First1KBFixed tests that $0000-$03FF always reads from bank 0
func TestMemory_First1KBFixed(t *testing.T) {
	rom := createTestROM(8)
	mem := NewMemory(rom)

	// First 1KB should always return bank 0 data
	if got := mem.Get(0x0000); got != 0x00 {
		t.Errorf("Address 0x0000: expected 0x00, got 0x%02X", got)
	}
	if got := mem.Get(0x03FF); got != 0x00 {
		t.Errorf("Address 0x03FF: expected 0x00, got 0x%02X", got)
	}

	// Switch slot 0 to a different bank
	mem.Set(0xFFFD, 5)

	// First 1KB should STILL return bank 0 data
	if got := mem.Get(0x0000); got != 0x00 {
		t.Errorf("Address 0x0000 after bank switch: expected 0x00, got 0x%02X", got)
	}
	if got := mem.Get(0x03FF); got != 0x00 {
		t.Errorf("Address 0x03FF after bank switch: expected 0x00, got 0x%02X", got)
	}

	// But $0400 onwards should come from the new bank
	if got := mem.Get(0x0400); got != 0x05 {
		t.Errorf("Address 0x0400 after bank switch: expected 0x05, got 0x%02X", got)
	}
}

// TestMemory_CartRAMEnable tests that $FFFC bit 3 enables cart RAM at $8000-$BFFF
func TestMemory_CartRAMEnable(t *testing.T) {
	rom := createTestROM(4)
	mem := NewMemory(rom)

	// Initially, $8000 should read from ROM (bank 2)
	if got := mem.Get(0x8000); got != 0x02 {
		t.Errorf("Initial $8000: expected ROM data 0x02, got 0x%02X", got)
	}

	// Enable cart RAM via bit 3 of $FFFC
	mem.Set(0xFFFC, 0x08)
	if got := mem.GetRAMControl(); got != 0x08 {
		t.Errorf("RAM control after enable: expected 0x08, got 0x%02X", got)
	}

	// Write to cart RAM
	mem.Set(0x8000, 0xAB)
	mem.Set(0xBFFF, 0xCD)

	// Read back
	if got := mem.Get(0x8000); got != 0xAB {
		t.Errorf("Cart RAM at $8000: expected 0xAB, got 0x%02X", got)
	}
	if got := mem.Get(0xBFFF); got != 0xCD {
		t.Errorf("Cart RAM at $BFFF: expected 0xCD, got 0x%02X", got)
	}

	// Disable cart RAM
	mem.Set(0xFFFC, 0x00)

	// Should read from ROM again
	if got := mem.Get(0x8000); got != 0x02 {
		t.Errorf("After disabling cart RAM, $8000: expected ROM data 0x02, got 0x%02X", got)
	}
}

// TestMemory_CartRAMBankSelect tests that $FFFC bit 2 selects cart RAM bank
func TestMemory_CartRAMBankSelect(t *testing.T) {
	rom := createTestROM(4)
	mem := NewMemory(rom)

	// Enable cart RAM (bit 3) with bank 0 selected (bit 2 = 0)
	mem.Set(0xFFFC, 0x08)

	// Write to bank 0
	mem.Set(0x8000, 0x11)

	// Switch to bank 1 (set bit 2)
	mem.Set(0xFFFC, 0x0C) // bits 3 and 2 set

	// Write to bank 1
	mem.Set(0x8000, 0x22)

	// Verify bank 1 has new value
	if got := mem.Get(0x8000); got != 0x22 {
		t.Errorf("Cart RAM bank 1 at $8000: expected 0x22, got 0x%02X", got)
	}

	// Switch back to bank 0
	mem.Set(0xFFFC, 0x08) // only bit 3 set

	// Verify bank 0 still has its value
	if got := mem.Get(0x8000); got != 0x11 {
		t.Errorf("Cart RAM bank 0 at $8000: expected 0x11, got 0x%02X", got)
	}
}

// TestMemory_ROMWriteIgnored tests that writes to ROM area have no effect
func TestMemory_ROMWriteIgnored(t *testing.T) {
	rom := createTestROM(4)
	mem := NewMemory(rom)

	// Get original values
	orig0000 := mem.Get(0x0000)
	orig4000 := mem.Get(0x4000)

	// Attempt to write to ROM
	mem.Set(0x0000, 0xFF)
	mem.Set(0x4000, 0xFF)

	// Values should be unchanged
	if got := mem.Get(0x0000); got != orig0000 {
		t.Errorf("ROM write at $0000: value changed from 0x%02X to 0x%02X", orig0000, got)
	}
	if got := mem.Get(0x4000); got != orig4000 {
		t.Errorf("ROM write at $4000: value changed from 0x%02X to 0x%02X", orig4000, got)
	}
}

// TestMemory_BankWrapping tests that bank numbers wrap via masking when exceeding ROM size
func TestMemory_BankWrapping(t *testing.T) {
	rom := createTestROM(2) // Only 2 banks = 32KB, so bankMask = 1
	mem := NewMemory(rom)

	// Initially, slot 2 maps to bank 2, but we only have 2 banks
	// Bank 2 & mask 1 = 0, so it wraps to bank 0
	if got := mem.Get(0x8000); got != 0x00 {
		t.Errorf("Bank wrap at $8000 (bank 2 -> 0): expected 0x00, got 0x%02X", got)
	}

	// Switch slot 0 to bank 10 (non-existent)
	// Bank 10 & mask 1 = 0, so it wraps to bank 0
	mem.Set(0xFFFD, 10)
	if got := mem.Get(0x0400); got != 0x00 {
		t.Errorf("Bank wrap at $0400 (bank 10 -> 0): expected 0x00, got 0x%02X", got)
	}

	// Switch slot 1 to bank 5
	// Bank 5 & mask 1 = 1, so it wraps to bank 1
	mem.Set(0xFFFE, 5)
	if got := mem.Get(0x4000); got != 0x01 {
		t.Errorf("Bank wrap at $4000 (bank 5 -> 1): expected 0x01, got 0x%02X", got)
	}

	// Test with 4-bank ROM (mask = 3)
	rom4 := createTestROM(4)
	mem4 := NewMemory(rom4)

	// Bank 7 & mask 3 = 3
	mem4.Set(0xFFFF, 7)
	if got := mem4.Get(0x8000); got != 0x03 {
		t.Errorf("Bank wrap at $8000 (bank 7 -> 3): expected 0x03, got 0x%02X", got)
	}

	// Bank 12 & mask 3 = 0
	mem4.Set(0xFFFD, 12)
	if got := mem4.Get(0x0400); got != 0x00 {
		t.Errorf("Bank wrap at $0400 (bank 12 -> 0): expected 0x00, got 0x%02X", got)
	}
}

// TestMemory_InitialBankState tests that banks are correctly initialized
func TestMemory_InitialBankState(t *testing.T) {
	rom := createTestROM(4)
	mem := NewMemory(rom)

	if got := mem.GetBankSlot(0); got != 0 {
		t.Errorf("Initial bank slot 0: expected 0, got %d", got)
	}
	if got := mem.GetBankSlot(1); got != 1 {
		t.Errorf("Initial bank slot 1: expected 1, got %d", got)
	}
	if got := mem.GetBankSlot(2); got != 2 {
		t.Errorf("Initial bank slot 2: expected 2, got %d", got)
	}
	if got := mem.GetRAMControl(); got != 0 {
		t.Errorf("Initial RAM control: expected 0, got %d", got)
	}
}

// ----------------------------------------------------------------------------
// Codemasters Mapper Tests
// ----------------------------------------------------------------------------

// createCodemastersTestROM creates a ROM with Codemasters CRC32 signature
func createCodemastersTestROM(banks int) []byte {
	// Use the CRC32 of "Excellent Dizzy Collection" to trigger Codemasters mapper
	// But since we can't easily create a ROM with a specific CRC32, we'll test
	// the mapper functions directly by creating the ROM and setting the mapper type
	rom := make([]byte, banks*0x4000)
	for b := 0; b < banks; b++ {
		for i := 0; i < 0x4000; i++ {
			rom[b*0x4000+i] = byte(b)
		}
	}
	return rom
}

// TestMemory_CodemastersSlot0Banking tests Codemasters slot 0 switching via $0000
func TestMemory_CodemastersSlot0Banking(t *testing.T) {
	rom := createCodemastersTestROM(8)
	mem := NewMemory(rom)
	mem.mapper = MapperCodemasters // Force Codemasters mapper for testing

	// Initially slot 0 maps to bank 0
	if got := mem.Get(0x0000); got != 0x00 {
		t.Errorf("Initial Codemasters slot 0 at $0000: expected 0x00, got 0x%02X", got)
	}
	if got := mem.Get(0x3FFF); got != 0x00 {
		t.Errorf("Initial Codemasters slot 0 at $3FFF: expected 0x00, got 0x%02X", got)
	}

	// Switch slot 0 to bank 5 via write to $0000
	mem.Set(0x0000, 5)

	// Verify slot 0 now reads from bank 5
	if got := mem.Get(0x0000); got != 0x05 {
		t.Errorf("Codemasters slot 0 after bank switch at $0000: expected 0x05, got 0x%02X", got)
	}
	if got := mem.Get(0x3FFF); got != 0x05 {
		t.Errorf("Codemasters slot 0 after bank switch at $3FFF: expected 0x05, got 0x%02X", got)
	}

	// Note: In Codemasters mapper, $0000-$3FFF is fully bankable (no fixed first 1KB)
}

// TestMemory_CodemastersSlot1Banking tests Codemasters slot 1 switching via $4000
func TestMemory_CodemastersSlot1Banking(t *testing.T) {
	rom := createCodemastersTestROM(8)
	mem := NewMemory(rom)
	mem.mapper = MapperCodemasters

	// Initially slot 1 maps to bank 1
	if got := mem.Get(0x4000); got != 0x01 {
		t.Errorf("Initial Codemasters slot 1 at $4000: expected 0x01, got 0x%02X", got)
	}

	// Switch slot 1 to bank 3 via write to $4000
	mem.Set(0x4000, 3)

	// Verify slot 1 now reads from bank 3
	if got := mem.Get(0x4000); got != 0x03 {
		t.Errorf("Codemasters slot 1 after bank switch: expected 0x03, got 0x%02X", got)
	}
	if got := mem.Get(0x7FFF); got != 0x03 {
		t.Errorf("Codemasters slot 1 at $7FFF after switch: expected 0x03, got 0x%02X", got)
	}
}

// TestMemory_CodemastersSlot2Banking tests Codemasters slot 2 switching via $8000
func TestMemory_CodemastersSlot2Banking(t *testing.T) {
	rom := createCodemastersTestROM(8)
	mem := NewMemory(rom)
	mem.mapper = MapperCodemasters

	// Initially slot 2 maps to bank 2
	if got := mem.Get(0x8000); got != 0x02 {
		t.Errorf("Initial Codemasters slot 2 at $8000: expected 0x02, got 0x%02X", got)
	}

	// Switch slot 2 to bank 7 via write to $8000
	mem.Set(0x8000, 7)

	// Verify slot 2 now reads from bank 7
	if got := mem.Get(0x8000); got != 0x07 {
		t.Errorf("Codemasters slot 2 after bank switch: expected 0x07, got 0x%02X", got)
	}
	if got := mem.Get(0xBFFF); got != 0x07 {
		t.Errorf("Codemasters slot 2 at $BFFF after switch: expected 0x07, got 0x%02X", got)
	}
}

// TestMemory_CodemastersRAM tests Codemasters RAM at $C000-$FFFF
func TestMemory_CodemastersRAM(t *testing.T) {
	rom := createCodemastersTestROM(4)
	mem := NewMemory(rom)
	mem.mapper = MapperCodemasters

	// Write to RAM
	mem.Set(0xC000, 0x42)
	mem.Set(0xDFFF, 0xAB)

	// Read back
	if got := mem.Get(0xC000); got != 0x42 {
		t.Errorf("Codemasters RAM at $C000: expected 0x42, got 0x%02X", got)
	}
	if got := mem.Get(0xDFFF); got != 0xAB {
		t.Errorf("Codemasters RAM at $DFFF: expected 0xAB, got 0x%02X", got)
	}

	// Test RAM mirroring at $E000-$FFFF
	if got := mem.Get(0xE000); got != 0x42 {
		t.Errorf("Codemasters RAM mirror at $E000: expected 0x42, got 0x%02X", got)
	}
}

// TestMemory_CodemastersNoRAMRegisters tests that $FFFC-$FFFF don't affect banking
func TestMemory_CodemastersNoRAMRegisters(t *testing.T) {
	rom := createCodemastersTestROM(8)
	mem := NewMemory(rom)
	mem.mapper = MapperCodemasters

	// In Sega mapper, $FFFD controls slot 0
	// In Codemasters, it's just RAM

	// Record initial bank state
	initialSlot0 := mem.GetBankSlot(0)

	// Write to $FFFD (would change slot 0 in Sega mapper)
	mem.Set(0xFFFD, 5)

	// In Codemasters, this should NOT change the bank
	// (it should just write to RAM)
	if got := mem.GetBankSlot(0); got != initialSlot0 {
		t.Errorf("Codemasters: write to $FFFD changed slot 0 (Sega behavior). Expected %d, got %d",
			initialSlot0, got)
	}

	// But the RAM at that address should be updated
	if got := mem.Get(0xFFFD); got != 5 {
		t.Errorf("Codemasters RAM at $FFFD: expected 5, got 0x%02X", got)
	}
}

// TestMemory_CodemastersFullyBankableSlot0 tests that slot 0 is fully bankable
func TestMemory_CodemastersFullyBankableSlot0(t *testing.T) {
	rom := createCodemastersTestROM(8)
	mem := NewMemory(rom)
	mem.mapper = MapperCodemasters

	// In Sega mapper, $0000-$03FF is fixed to bank 0
	// In Codemasters, entire $0000-$3FFF is bankable

	// Switch slot 0 to bank 5
	mem.Set(0x0000, 5)

	// Address $0000 (which is fixed in Sega) should read from bank 5
	if got := mem.Get(0x0000); got != 0x05 {
		t.Errorf("Codemasters $0000 after bank switch: expected 0x05, got 0x%02X", got)
	}

	// Address $0300 should also read from bank 5
	if got := mem.Get(0x0300); got != 0x05 {
		t.Errorf("Codemasters $0300 after bank switch: expected 0x05, got 0x%02X", got)
	}
}

// TestMemory_CodemastersBankWrapping tests bank number wrapping
func TestMemory_CodemastersBankWrapping(t *testing.T) {
	rom := createCodemastersTestROM(4) // 4 banks, mask = 3
	mem := NewMemory(rom)
	mem.mapper = MapperCodemasters

	// Switch to bank 7 (should wrap to bank 3 with mask 3)
	mem.Set(0x0000, 7)

	// Should read from bank 3 (7 & 3 = 3)
	if got := mem.Get(0x0000); got != 0x03 {
		t.Errorf("Codemasters bank wrap: bank 7 should wrap to 3, got data 0x%02X", got)
	}
}

// TestMemory_CodemastersWriteOnlyBankRegisters tests that only $0000/$4000/$8000 affect banks
func TestMemory_CodemastersWriteOnlyBankRegisters(t *testing.T) {
	rom := createCodemastersTestROM(8)
	mem := NewMemory(rom)
	mem.mapper = MapperCodemasters

	// Write to $0001 should NOT change bank (only $0000 does)
	mem.Set(0x0001, 5)
	if got := mem.Get(0x0000); got != 0x00 {
		t.Errorf("Write to $0001 should not change slot 0 bank")
	}

	// Write to $4001 should NOT change bank (only $4000 does)
	mem.Set(0x4001, 5)
	if got := mem.Get(0x4000); got != 0x01 {
		t.Errorf("Write to $4001 should not change slot 1 bank")
	}

	// Write to $8001 should NOT change bank (only $8000 does)
	mem.Set(0x8001, 5)
	if got := mem.Get(0x8000); got != 0x02 {
		t.Errorf("Write to $8001 should not change slot 2 bank")
	}
}

// TestMemory_MapperDetectionSega tests that standard ROMs use Sega mapper
func TestMemory_MapperDetectionSega(t *testing.T) {
	rom := createTestROM(4)
	mem := NewMemory(rom)

	// Standard ROM should use Sega mapper
	if mem.mapper != MapperSega {
		t.Errorf("Standard ROM should use Sega mapper, got %v", mem.mapper)
	}
}

// TestMemory_CodemastersOutOfBoundsRead tests reading beyond ROM size
func TestMemory_CodemastersOutOfBoundsRead(t *testing.T) {
	rom := createCodemastersTestROM(2) // Only 2 banks (32KB)
	mem := NewMemory(rom)
	mem.mapper = MapperCodemasters

	// Switch to a bank that would be out of bounds
	mem.Set(0x0000, 10) // Bank 10 doesn't exist, should wrap

	// Should not crash, should return wrapped bank data
	got := mem.Get(0x0000)
	// Bank 10 & mask 1 = 0
	if got != 0x00 {
		t.Errorf("Out of bounds read: expected 0x00 (wrapped), got 0x%02X", got)
	}
}
