//go:build !libretro

package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/user-none/emkiii/ui/style"
)

// SearchOverlay displays a search filter at the bottom-left of the screen
type SearchOverlay struct {
	text      string
	active    bool              // Currently capturing keyboard input
	onChanged func(text string) // Callback when text changes
}

// NewSearchOverlay creates a new search overlay with the given change callback
func NewSearchOverlay(onChanged func(text string)) *SearchOverlay {
	return &SearchOverlay{
		onChanged: onChanged,
	}
}

// IsVisible returns true if the search has text (overlay should be shown)
func (s *SearchOverlay) IsVisible() bool {
	return s.text != ""
}

// IsActive returns true if the overlay is capturing keyboard input
func (s *SearchOverlay) IsActive() bool {
	return s.active
}

// Activate starts capturing keyboard input
func (s *SearchOverlay) Activate() {
	s.active = true
}

// Clear removes all search text and deactivates
func (s *SearchOverlay) Clear() {
	s.text = ""
	s.active = false
	if s.onChanged != nil {
		s.onChanged(s.text)
	}
}

// HandleInput processes keyboard input when active.
// Returns true if input was handled (should not propagate to navigation).
func (s *SearchOverlay) HandleInput() bool {
	if !s.active {
		return false
	}

	// Arrow keys deactivate but keep filter - let navigation proceed
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) ||
		ebiten.IsKeyPressed(ebiten.KeyArrowDown) ||
		ebiten.IsKeyPressed(ebiten.KeyArrowLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		s.active = false
		return false // Let navigation proceed
	}

	// Backspace removes last character
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(s.text) > 0 {
		s.text = s.text[:len(s.text)-1]
		if s.onChanged != nil {
			s.onChanged(s.text)
		}
		return true
	}

	// Character input
	chars := ebiten.AppendInputChars(nil)
	if len(chars) > 0 {
		for _, c := range chars {
			// Don't add the '/' that activated search
			if c != '/' || s.text != "" {
				s.text += string(c)
			}
		}
		if s.onChanged != nil {
			s.onChanged(s.text)
		}
		return true
	}

	return true // Active, consume input even if nothing typed
}

// Draw renders the search overlay at bottom-left
func (s *SearchOverlay) Draw(screen *ebiten.Image) {
	if !s.IsVisible() && !s.active {
		return
	}

	bounds := screen.Bounds()
	screenHeight := bounds.Dy()

	// Build display text
	displayText := "Filter: " + s.text
	if s.active {
		displayText += "_" // Cursor when active
	}

	// Calculate text size
	textWidth, textHeight := text.Measure(displayText, *style.FontFace(), 0)

	// Padding
	padding := 12
	bgWidth := int(textWidth) + padding*2
	bgHeight := int(textHeight) + padding*2

	// Position: bottom-left, 8px margin (mirrors Notification at bottom-right)
	margin := 8
	bgX := margin
	bgY := screenHeight - bgHeight - margin

	// Draw background (black at 60% opacity)
	bg := ebiten.NewImage(bgWidth, bgHeight)
	overlayBg := style.OverlayBackground
	overlayBg.A = 153 // 60% opacity
	bg.Fill(overlayBg)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(bgX), float64(bgY))
	screen.DrawImage(bg, opts)

	// Draw text
	textOpts := &text.DrawOptions{}
	textOpts.GeoM.Translate(float64(bgX+padding), float64(bgY+padding))
	textOpts.ColorScale.ScaleWithColor(style.Text)
	text.Draw(screen, displayText, *style.FontFace(), textOpts)
}
