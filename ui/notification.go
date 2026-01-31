//go:build !libretro

package ui

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"
)

// Notification displays temporary messages on screen
type Notification struct {
	message   string
	startTime time.Time
	duration  time.Duration
	fontFace  text.Face
}

// NewNotification creates a new notification system
func NewNotification() *Notification {
	return &Notification{
		fontFace: text.NewGoXFace(basicfont.Face7x13),
	}
}

// Show displays a notification message
func (n *Notification) Show(message string, duration time.Duration) {
	n.message = message
	n.startTime = time.Now()
	n.duration = duration
}

// ShowDefault displays a notification with default 3 second duration
func (n *Notification) ShowDefault(message string) {
	n.Show(message, 3*time.Second)
}

// ShowShort displays a notification with 1 second duration (for gameplay)
func (n *Notification) ShowShort(message string) {
	n.Show(message, 1*time.Second)
}

// IsVisible returns whether the notification is currently visible
func (n *Notification) IsVisible() bool {
	if n.message == "" {
		return false
	}
	return time.Since(n.startTime) < n.duration
}

// Clear removes the current notification
func (n *Notification) Clear() {
	n.message = ""
}

// Draw renders the notification
func (n *Notification) Draw(screen *ebiten.Image) {
	if !n.IsVisible() {
		return
	}

	bounds := screen.Bounds()
	screenWidth := bounds.Dx()
	screenHeight := bounds.Dy()

	// Calculate text size
	textWidth, textHeight := text.Measure(n.message, n.fontFace, 0)

	// Padding
	padding := 12
	bgWidth := int(textWidth) + padding*2
	bgHeight := int(textHeight) + padding*2

	// Position: bottom-right, 8px margin
	margin := 8
	bgX := screenWidth - bgWidth - margin
	bgY := screenHeight - bgHeight - margin

	// Draw background (black at 60% opacity)
	bg := ebiten.NewImage(bgWidth, bgHeight)
	bg.Fill(color.RGBA{0, 0, 0, 153}) // 60% opacity = 0.6 * 255 = 153

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(bgX), float64(bgY))
	screen.DrawImage(bg, opts)

	// Draw text centered in background
	textOpts := &text.DrawOptions{}
	textOpts.GeoM.Translate(float64(bgX+padding), float64(bgY+padding+int(textHeight)))
	textOpts.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, n.message, n.fontFace, textOpts)
}
