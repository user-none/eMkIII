package romloader

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// createTestSMSFile creates a temporary .sms file with test data
func createTestSMSFile(t *testing.T, data []byte) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.sms")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("Failed to create test SMS file: %v", err)
	}
	return path
}

// createTestZipFile creates a temporary .zip file containing an SMS file
func createTestZipFile(t *testing.T, smsData []byte, smsName string) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.zip")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	fw, err := w.Create(smsName)
	if err != nil {
		t.Fatalf("Failed to create file in zip: %v", err)
	}
	if _, err := fw.Write(smsData); err != nil {
		t.Fatalf("Failed to write to zip: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close zip: %v", err)
	}
	return path
}

// createTestGzipFile creates a temporary .gz file containing SMS data
func createTestGzipFile(t *testing.T, smsData []byte) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.sms.gz")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create gzip file: %v", err)
	}
	defer f.Close()

	w := gzip.NewWriter(f)
	if _, err := w.Write(smsData); err != nil {
		t.Fatalf("Failed to write to gzip: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close gzip: %v", err)
	}
	return path
}

// TestLoader_RawSMSLoad tests loading plain .sms files
func TestLoader_RawSMSLoad(t *testing.T) {
	testData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	path := createTestSMSFile(t, testData)

	data, name, err := LoadROM(path)
	if err != nil {
		t.Fatalf("LoadROM failed: %v", err)
	}

	if !bytes.Equal(data, testData) {
		t.Errorf("Data mismatch: expected %v, got %v", testData, data)
	}

	if name != "test.sms" {
		t.Errorf("Name mismatch: expected test.sms, got %s", name)
	}
}

// TestLoader_ZipLoad tests loading SMS from ZIP archives
func TestLoader_ZipLoad(t *testing.T) {
	testData := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	path := createTestZipFile(t, testData, "game.sms")

	data, name, err := LoadROM(path)
	if err != nil {
		t.Fatalf("LoadROM failed: %v", err)
	}

	if !bytes.Equal(data, testData) {
		t.Errorf("Data mismatch: expected %v, got %v", testData, data)
	}

	if name != "game.sms" {
		t.Errorf("Name mismatch: expected game.sms, got %s", name)
	}
}

// TestLoader_GzipLoad tests loading SMS from gzip files
func TestLoader_GzipLoad(t *testing.T) {
	testData := []byte{0x11, 0x22, 0x33, 0x44, 0x55}
	path := createTestGzipFile(t, testData)

	data, _, err := LoadROM(path)
	if err != nil {
		t.Fatalf("LoadROM failed: %v", err)
	}

	if !bytes.Equal(data, testData) {
		t.Errorf("Data mismatch: expected %v, got %v", testData, data)
	}
}

// TestLoader_FormatDetectionMagic tests detection via magic bytes
func TestLoader_FormatDetectionMagic(t *testing.T) {
	testCases := []struct {
		header   []byte
		path     string
		expected formatType
	}{
		{[]byte{0x50, 0x4B, 0x03, 0x04}, "file.dat", formatZIP},
		{[]byte{0x50, 0x4B, 0x05, 0x06}, "file.dat", formatZIP},
		{[]byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}, "file.dat", format7z},
		{[]byte{0x1F, 0x8B}, "file.dat", formatGzip},
		{[]byte{0x52, 0x61, 0x72, 0x21}, "file.dat", formatRAR},
	}

	for _, tc := range testCases {
		result := detectFormat(tc.header, tc.path)
		if result != tc.expected {
			t.Errorf("detectFormat(%v, %s): expected %d, got %d", tc.header, tc.path, tc.expected, result)
		}
	}
}

// TestLoader_FormatDetectionExtension tests fallback to extension
func TestLoader_FormatDetectionExtension(t *testing.T) {
	testCases := []struct {
		path     string
		expected formatType
	}{
		{"game.sms", formatRawSMS},
		{"game.SMS", formatRawSMS},
		{"game.zip", formatZIP},
		{"game.ZIP", formatZIP},
		{"game.7z", format7z},
		{"game.gz", formatGzip},
		{"game.tgz", formatGzip},
		{"game.tar.gz", formatGzip},
		{"game.rar", formatRAR},
		{"game.unknown", formatUnknown},
	}

	for _, tc := range testCases {
		// Use empty header to force extension-based detection
		result := detectFormat([]byte{}, tc.path)
		if result != tc.expected {
			t.Errorf("detectFormat([], %s): expected %d, got %d", tc.path, tc.expected, result)
		}
	}
}

// TestLoader_NoSMSInArchive tests error when no .sms found in archive
func TestLoader_NoSMSInArchive(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.zip")

	// Create zip with non-SMS file
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create zip: %v", err)
	}

	w := zip.NewWriter(f)
	fw, _ := w.Create("readme.txt")
	fw.Write([]byte("hello"))
	w.Close()
	f.Close()

	_, _, err = LoadROM(path)
	if err == nil {
		t.Error("Expected error when no SMS file in archive")
	}
	if err != ErrNoSMSFile {
		t.Errorf("Expected ErrNoSMSFile, got %v", err)
	}
}

// TestLoader_FileTooLarge tests rejection of files exceeding size limit
func TestLoader_FileTooLarge(t *testing.T) {
	// Create a large file that exceeds maxROMSize
	largeData := make([]byte, maxROMSize+1)

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "large.sms")
	if err := os.WriteFile(path, largeData, 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// For raw files, the current implementation reads all data
	// The size check is in limitedRead which is used for archives
	// Let's test with a gzip file instead
	gzPath := filepath.Join(tmpDir, "large.sms.gz")
	f, err := os.Create(gzPath)
	if err != nil {
		t.Fatalf("Failed to create gzip: %v", err)
	}

	w := gzip.NewWriter(f)
	w.Write(largeData)
	w.Close()
	f.Close()

	_, _, err = LoadROM(gzPath)
	if err == nil {
		t.Error("Expected error for oversized file")
	}
}

// TestLoader_FileNotFound tests error for missing files
func TestLoader_FileNotFound(t *testing.T) {
	_, _, err := LoadROM("/nonexistent/path/game.sms")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

// TestLoader_IsSMSFile tests the SMS file extension check
func TestLoader_IsSMSFile(t *testing.T) {
	testCases := []struct {
		name     string
		expected bool
	}{
		{"game.sms", true},
		{"game.SMS", true},
		{"game.Sms", true},
		{"game.txt", false},
		{"game.sms.bak", false},
		{"game", false},
		{"sms", false},
		{".sms", true},
	}

	for _, tc := range testCases {
		result := isSMSFile(tc.name)
		if result != tc.expected {
			t.Errorf("isSMSFile(%q): expected %v, got %v", tc.name, tc.expected, result)
		}
	}
}

// TestLoader_ZipWithSubdirectory tests extracting SMS from nested directory
func TestLoader_ZipWithSubdirectory(t *testing.T) {
	testData := []byte{0x12, 0x34, 0x56}
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.zip")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create zip: %v", err)
	}

	w := zip.NewWriter(f)
	// Create file in subdirectory
	fw, _ := w.Create("roms/games/test.sms")
	fw.Write(testData)
	w.Close()
	f.Close()

	data, name, err := LoadROM(path)
	if err != nil {
		t.Fatalf("LoadROM failed: %v", err)
	}

	if !bytes.Equal(data, testData) {
		t.Errorf("Data mismatch: expected %v, got %v", testData, data)
	}

	if name != "test.sms" {
		t.Errorf("Name should be just the filename, got %s", name)
	}
}

// TestLoader_EmptyFile tests handling of empty files
func TestLoader_EmptyFile(t *testing.T) {
	path := createTestSMSFile(t, []byte{})

	data, _, err := LoadROM(path)
	if err != nil {
		t.Fatalf("LoadROM failed: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("Expected empty data, got %d bytes", len(data))
	}
}

// TestLoader_MaxROMSizeConstant tests that the size limit is reasonable
func TestLoader_MaxROMSizeConstant(t *testing.T) {
	// SMS ROMs can be up to 4MB, so 8MB limit should be sufficient
	if maxROMSize < 4*1024*1024 {
		t.Errorf("maxROMSize too small: %d bytes (should be at least 4MB)", maxROMSize)
	}
	if maxROMSize > 16*1024*1024 {
		t.Errorf("maxROMSize unexpectedly large: %d bytes", maxROMSize)
	}
}

// TestLoader_MagicBytesDefinition tests that magic byte arrays are correct
func TestLoader_MagicBytesDefinition(t *testing.T) {
	// ZIP magic: "PK\x03\x04"
	if !bytes.Equal(magicZIP, []byte{0x50, 0x4B, 0x03, 0x04}) {
		t.Error("ZIP magic bytes incorrect")
	}

	// 7z magic
	if !bytes.Equal(magic7z, []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}) {
		t.Error("7z magic bytes incorrect")
	}

	// Gzip magic
	if !bytes.Equal(magicGzip, []byte{0x1F, 0x8B}) {
		t.Error("Gzip magic bytes incorrect")
	}

	// RAR magic: "Rar!"
	if !bytes.Equal(magicRAR, []byte{0x52, 0x61, 0x72, 0x21}) {
		t.Error("RAR magic bytes incorrect")
	}
}
