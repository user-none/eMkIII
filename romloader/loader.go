// Package romloader handles loading ROM files from various sources,
// including compressed archives (ZIP, 7z, gzip, tar.gz, RAR).
package romloader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Magic bytes for format detection
var (
	magicZIP    = []byte{0x50, 0x4B, 0x03, 0x04}
	magicZIPEnd = []byte{0x50, 0x4B, 0x05, 0x06} // empty zip
	magic7z     = []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}
	magicGzip   = []byte{0x1F, 0x8B}
	magicRAR    = []byte{0x52, 0x61, 0x72, 0x21} // "Rar!"
)

// Maximum ROM size (8MB safety limit)
const maxROMSize = 8 * 1024 * 1024

// ErrNoSMSFile is returned when no .sms file is found in an archive
var ErrNoSMSFile = errors.New("no .sms file found in archive")

// ErrUnsupportedFormat is returned for unrecognized file formats
var ErrUnsupportedFormat = errors.New("unsupported file format")

// ErrFileTooLarge is returned when extracted content exceeds size limit
var ErrFileTooLarge = errors.New("file exceeds maximum size limit")

// formatType represents the detected file format
type formatType int

const (
	formatUnknown formatType = iota
	formatRawSMS
	formatZIP
	format7z
	formatGzip
	formatRAR
)

// LoadROM loads a ROM from a file path. It automatically detects and extracts
// from archives. Returns the ROM data, the filename of the ROM (useful for display),
// and any error encountered.
func LoadROM(path string) ([]byte, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Read header for magic byte detection
	header := make([]byte, 16)
	n, err := f.Read(header)
	if err != nil && err != io.EOF {
		return nil, "", fmt.Errorf("failed to read file header: %w", err)
	}
	header = header[:n]

	// Detect format
	format := detectFormat(header, path)

	// Reset file position
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, "", fmt.Errorf("failed to seek file: %w", err)
	}

	switch format {
	case formatRawSMS:
		data, err := limitedRead(f)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read ROM: %w", err)
		}
		return data, filepath.Base(path), nil

	case formatZIP:
		return extractFromZIP(path)

	case format7z:
		return extractFrom7z(path)

	case formatGzip:
		return extractFromGzip(path)

	case formatRAR:
		return extractFromRAR(path)

	default:
		return nil, "", fmt.Errorf("%w: %s", ErrUnsupportedFormat, path)
	}
}

// detectFormat determines the file format based on magic bytes and extension
func detectFormat(header []byte, path string) formatType {
	ext := strings.ToLower(filepath.Ext(path))

	// Check magic bytes first (more reliable)
	if len(header) >= 4 {
		if bytes.HasPrefix(header, magicZIP) || bytes.HasPrefix(header, magicZIPEnd) {
			return formatZIP
		}
		if bytes.HasPrefix(header, magicRAR) {
			return formatRAR
		}
	}
	if len(header) >= 6 && bytes.HasPrefix(header, magic7z) {
		return format7z
	}
	if len(header) >= 2 && bytes.HasPrefix(header, magicGzip) {
		return formatGzip
	}

	// Fall back to extension
	switch ext {
	case ".sms":
		return formatRawSMS
	case ".zip":
		return formatZIP
	case ".7z":
		return format7z
	case ".gz", ".tgz":
		return formatGzip
	case ".rar":
		return formatRAR
	}

	// Check for .tar.gz
	if strings.HasSuffix(strings.ToLower(path), ".tar.gz") {
		return formatGzip
	}

	// If we have some data but don't recognize it, assume raw SMS for .sms-like files
	// Otherwise return unknown
	return formatUnknown
}

// isSMSFile checks if a filename has a .sms extension (case-insensitive)
func isSMSFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".sms")
}

// limitedRead reads from r up to maxROMSize bytes, returning an error if exceeded
func limitedRead(r io.Reader) ([]byte, error) {
	lr := io.LimitReader(r, maxROMSize+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if len(data) > maxROMSize {
		return nil, ErrFileTooLarge
	}
	return data, nil
}
