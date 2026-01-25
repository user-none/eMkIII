package emu

// createTestROM creates a test ROM with the given number of 16KB banks.
// Each bank is filled with its bank number (0, 1, 2, etc.) to allow
// easy verification of which bank is mapped.
func createTestROM(banks int) []byte {
	rom := make([]byte, banks*0x4000)
	for b := 0; b < banks; b++ {
		for i := 0; i < 0x4000; i++ {
			rom[b*0x4000+i] = byte(b)
		}
	}
	return rom
}

// createTestROMWithPattern creates a test ROM where each byte contains
// a value derived from both the bank number and the offset within the bank.
// This allows verifying that both bank and offset are correct.
func createTestROMWithPattern(banks int) []byte {
	rom := make([]byte, banks*0x4000)
	for b := 0; b < banks; b++ {
		for i := 0; i < 0x4000; i++ {
			// High nibble = bank, low nibble = (offset / 0x400) & 0x0F
			rom[b*0x4000+i] = byte((b << 4) | ((i >> 10) & 0x0F))
		}
	}
	return rom
}
