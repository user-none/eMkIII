//go:build !libretro

package style

import (
	"fmt"
	goimage "image"
	"image/draw"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	xdraw "golang.org/x/image/draw"
)

// ScaleImage scales an image to fit within maxWidth x maxHeight while preserving aspect ratio.
// Returns an ebiten.Image suitable for display.
// Scaling is done on CPU to avoid creating large temporary GPU textures.
func ScaleImage(src goimage.Image, maxWidth, maxHeight int) *ebiten.Image {
	bounds := src.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Calculate scale to fit within max dimensions
	scaleX := float64(maxWidth) / float64(srcWidth)
	scaleY := float64(maxHeight) / float64(srcHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Calculate new dimensions
	newWidth := int(float64(srcWidth) * scale)
	newHeight := int(float64(srcHeight) * scale)

	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	// Scale on CPU using approximate bilinear interpolation (fast with good quality)
	dstRect := goimage.Rect(0, 0, newWidth, newHeight)
	scaled := goimage.NewRGBA(dstRect)
	xdraw.ApproxBiLinear.Scale(scaled, dstRect, src, bounds, draw.Over, nil)

	// Create Ebiten image from the small scaled image only
	return ebiten.NewImageFromImage(scaled)
}

// TruncateStart truncates a string from the start, keeping the end portion.
// Returns the truncated string and whether truncation occurred.
// Useful for file paths where the end (filename) is most relevant.
func TruncateStart(s string, maxLen int) (string, bool) {
	if len(s) <= maxLen {
		return s, false
	}
	if maxLen <= 3 {
		return s[len(s)-maxLen:], true
	}
	return "..." + s[len(s)-maxLen+3:], true
}

// TruncateEnd truncates a string from the end, keeping the start portion.
// Returns the truncated string and whether truncation occurred.
// Useful for titles where the beginning is most relevant.
func TruncateEnd(s string, maxLen int) (string, bool) {
	if len(s) <= maxLen {
		return s, false
	}
	if maxLen <= 3 {
		return s[:maxLen], true
	}
	return s[:maxLen-3] + "...", true
}

// FormatPlayTime formats a duration in seconds into a human-readable string.
// Returns "—" for 0 seconds, "< 1m" for under a minute,
// or a formatted string like "2h 30m" or "45m".
func FormatPlayTime(seconds int64) string {
	if seconds == 0 {
		return "-"
	}
	if seconds < 60 {
		return "< 1m"
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// FormatLastPlayed formats a Unix timestamp into a relative or absolute date string.
// Returns "Never" for 0, "Today"/"Yesterday" for recent dates,
// "Jan 2" for this year, or "Jan 2, 2006" for previous years.
func FormatLastPlayed(timestamp int64) string {
	if timestamp == 0 {
		return "Never"
	}

	t := time.Unix(timestamp, 0)
	now := time.Now()

	// Check if same day
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return "Today"
	}

	// Check if yesterday
	yesterday := now.AddDate(0, 0, -1)
	if t.Year() == yesterday.Year() && t.YearDay() == yesterday.YearDay() {
		return "Yesterday"
	}

	// This year - show month and day
	if t.Year() == now.Year() {
		return t.Format("Jan 2")
	}

	// Previous years - show full date
	return t.Format("Jan 2, 2006")
}

// FormatDate formats a Unix timestamp as a date string.
// Returns "Unknown" for 0, otherwise "Jan 2, 2006".
func FormatDate(timestamp int64) string {
	if timestamp == 0 {
		return "Unknown"
	}
	return time.Unix(timestamp, 0).Format("Jan 2, 2006")
}
