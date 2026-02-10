//go:build !libretro

package style

import (
	"testing"
)

func TestTruncateStart(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		maxLen      int
		expected    string
		shouldTrunc bool
	}{
		{"shorter than max", "hello", 10, "hello", false},
		{"exact length", "hello", 5, "hello", false},
		{"truncated with ellipsis", "/Users/john/very/long/path/to/file.sms", 20, ".../path/to/file.sms", true},
		{"maxLen 3", "abcdef", 3, "def", true},
		{"maxLen 2", "abcdef", 2, "ef", true},
		{"maxLen 1", "abcdef", 1, "f", true},
		{"empty string", "", 5, "", false},
		{"single char no trunc", "a", 5, "a", false},
		{"truncate to 4", "abcdef", 4, "...f", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, truncated := TruncateStart(tc.input, tc.maxLen)
			if got != tc.expected {
				t.Errorf("TruncateStart(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.expected)
			}
			if truncated != tc.shouldTrunc {
				t.Errorf("TruncateStart(%q, %d) truncated = %v, want %v", tc.input, tc.maxLen, truncated, tc.shouldTrunc)
			}
		})
	}
}

func TestTruncateEnd(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		maxLen      int
		expected    string
		shouldTrunc bool
	}{
		{"shorter than max", "hello", 10, "hello", false},
		{"exact length", "hello", 5, "hello", false},
		{"truncated with ellipsis", "Sonic the Hedgehog in Very Long Title", 20, "Sonic the Hedgeho...", true},
		{"maxLen 3", "abcdef", 3, "abc", true},
		{"maxLen 2", "abcdef", 2, "ab", true},
		{"maxLen 1", "abcdef", 1, "a", true},
		{"empty string", "", 5, "", false},
		{"single char no trunc", "a", 5, "a", false},
		{"truncate to 4", "abcdef", 4, "a...", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, truncated := TruncateEnd(tc.input, tc.maxLen)
			if got != tc.expected {
				t.Errorf("TruncateEnd(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.expected)
			}
			if truncated != tc.shouldTrunc {
				t.Errorf("TruncateEnd(%q, %d) truncated = %v, want %v", tc.input, tc.maxLen, truncated, tc.shouldTrunc)
			}
		})
	}
}

func TestFormatPlayTime(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int64
		expected string
	}{
		{"zero", 0, "-"},
		{"1 second", 1, "< 1m"},
		{"30 seconds", 30, "< 1m"},
		{"59 seconds", 59, "< 1m"},
		{"exactly 1 minute", 60, "1m"},
		{"5 minutes", 300, "5m"},
		{"59 minutes", 3540, "59m"},
		{"exactly 1 hour", 3600, "1h 0m"},
		{"1 hour 30 minutes", 5400, "1h 30m"},
		{"2 hours 15 minutes", 8100, "2h 15m"},
		{"24 hours", 86400, "24h 0m"},
		{"100 hours 59 minutes", 363540, "100h 59m"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatPlayTime(tc.seconds)
			if got != tc.expected {
				t.Errorf("FormatPlayTime(%d) = %q, want %q", tc.seconds, got, tc.expected)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	t.Run("zero timestamp", func(t *testing.T) {
		got := FormatDate(0)
		if got != "Unknown" {
			t.Errorf("FormatDate(0) = %q, want \"Unknown\"", got)
		}
	})

	t.Run("valid timestamp returns date", func(t *testing.T) {
		// Use a mid-day timestamp to avoid timezone boundary issues
		// 1609509600 = 2021-01-01 14:00:00 UTC
		got := FormatDate(1609509600)
		if got == "Unknown" {
			t.Error("FormatDate should not return \"Unknown\" for non-zero timestamp")
		}
		// Should contain year 2021 (or Dec 2020 depending on timezone, but not "Unknown")
		if len(got) < 5 {
			t.Errorf("FormatDate returned unexpectedly short result: %q", got)
		}
	})
}

func TestFormatLastPlayed(t *testing.T) {
	t.Run("zero timestamp", func(t *testing.T) {
		got := FormatLastPlayed(0)
		if got != "Never" {
			t.Errorf("FormatLastPlayed(0) = %q, want %q", got, "Never")
		}
	})

	t.Run("very old timestamp", func(t *testing.T) {
		// 1609459200 = Jan 1, 2021 - should show full date with year
		got := FormatLastPlayed(1609459200)
		if got == "Never" || got == "Today" || got == "Yesterday" {
			t.Errorf("FormatLastPlayed(1609459200) = %q, expected a date with year", got)
		}
	})
}
