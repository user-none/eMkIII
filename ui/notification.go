//go:build !libretro

package ui

import (
	"image"
	"log"
	"sync"
	"time"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/user-none/emkiii/ui/style"
)

// NotificationType determines the visual style of the notification
type NotificationType int

const (
	NotificationTypeDefault     NotificationType = iota // Small, bottom-right
	NotificationTypeAchievement                         // Large, top-center, prominent
)

// Notification displays temporary messages on screen
type Notification struct {
	mu         sync.Mutex
	message    string
	subtitle   string // Secondary line (e.g., achievement description)
	startTime  time.Time
	duration   time.Duration
	notifyType NotificationType
	largeFace  *text.GoTextFace // Cached large font for achievements

	// Badge image (pre-cached by achievement manager)
	badgeImage *ebiten.Image

	// Pre-allocated images for rendering (avoid per-frame allocations)
	defaultBg     *ebiten.Image
	achievementBg *ebiten.Image
	lastBgWidth   int
	lastBgHeight  int

	// Audio stream for notification sounds (separate from game audio)
	audioStream *sdl.AudioStream
}

// NewNotification creates a new notification system
func NewNotification() *Notification {
	return &Notification{
		largeFace: style.LargeFontFace(),
	}
}

// ensureAudioStream lazily initializes the audio stream when first needed
func (n *Notification) ensureAudioStream() bool {
	if n.audioStream != nil {
		return true
	}

	// Initialize SDL audio (lazy, safe to call multiple times)
	if !ensureSDLAudio() {
		return false
	}

	spec := sdl.AudioSpec{
		Freq:     48000,
		Format:   sdl.AUDIO_S16LE,
		Channels: 2,
	}

	stream := sdl.AUDIO_DEVICE_DEFAULT_PLAYBACK.OpenAudioDeviceStream(&spec, 0)
	if stream == nil {
		return false
	}

	if err := stream.ResumeDevice(); err != nil {
		stream.Destroy()
		log.Printf("Warning: Failed to resume notification audio: %v", err)
		return false
	}

	n.audioStream = stream
	return true
}

// PlaySound plays sound data through the notification audio stream
// Sound data should be 48kHz stereo S16LE format
func (n *Notification) PlaySound(soundData []byte) {
	if len(soundData) == 0 {
		return
	}
	// Lazily init audio stream (SDL must be initialized first by game audio)
	if !n.ensureAudioStream() {
		return
	}
	if err := n.audioStream.PutData(soundData); err != nil {
		log.Printf("Failed to play notification sound: %v", err)
	}
}

// Close cleans up audio resources
func (n *Notification) Close() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.audioStream != nil {
		n.audioStream.Destroy()
		n.audioStream = nil
	}
}

// Show displays a notification message
func (n *Notification) Show(message string, duration time.Duration) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.message = message
	n.subtitle = ""
	n.startTime = time.Now()
	n.duration = duration
	n.notifyType = NotificationTypeDefault
	n.badgeImage = nil
}

// ShowDefault displays a notification with default 3 second duration
func (n *Notification) ShowDefault(message string) {
	n.Show(message, 3*time.Second)
}

// ShowShort displays a notification with 1 second duration (for gameplay)
func (n *Notification) ShowShort(message string) {
	n.Show(message, 1*time.Second)
}

// ShowAchievementWithBadge displays a prominent achievement notification with a badge image
func (n *Notification) ShowAchievementWithBadge(title, description string, badge *ebiten.Image) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.message = title
	n.subtitle = description
	n.startTime = time.Now()
	n.duration = 5 * time.Second
	n.notifyType = NotificationTypeAchievement
	n.badgeImage = badge
}

// SetBadge updates the badge image for the current notification (thread-safe)
func (n *Notification) SetBadge(badge *ebiten.Image) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.badgeImage = badge
}

// IsVisible returns whether the notification is currently visible
func (n *Notification) IsVisible() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.message == "" {
		return false
	}
	return time.Since(n.startTime) < n.duration
}

// Clear removes the current notification
func (n *Notification) Clear() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.message = ""
}

// Draw renders the notification
func (n *Notification) Draw(screen *ebiten.Image) {
	n.mu.Lock()
	if n.message == "" || time.Since(n.startTime) >= n.duration {
		n.mu.Unlock()
		return
	}

	// Copy fields under lock
	message := n.message
	subtitle := n.subtitle
	notifyType := n.notifyType
	badge := n.badgeImage
	n.mu.Unlock()

	if notifyType == NotificationTypeAchievement {
		n.drawAchievementWithData(screen, message, subtitle, badge)
	} else {
		n.drawDefaultWithData(screen, message)
	}
}

// drawDefaultWithData renders a small notification in the bottom-right corner
func (n *Notification) drawDefaultWithData(screen *ebiten.Image, message string) {
	bounds := screen.Bounds()
	screenWidth := bounds.Dx()
	screenHeight := bounds.Dy()

	// Calculate text size
	textWidth, textHeight := text.Measure(message, *style.FontFace(), 0)

	// Padding
	padding := 12
	bgWidth := int(textWidth) + padding*2
	bgHeight := int(textHeight) + padding*2

	// Position: bottom-right, 8px margin
	margin := 8
	bgX := screenWidth - bgWidth - margin
	bgY := screenHeight - bgHeight - margin

	// Reuse or create background image
	if n.defaultBg == nil || n.defaultBg.Bounds().Dx() < bgWidth || n.defaultBg.Bounds().Dy() < bgHeight {
		n.defaultBg = ebiten.NewImage(bgWidth, bgHeight)
	}
	n.defaultBg.Clear()
	overlayBg := style.OverlayBackground
	overlayBg.A = 153 // 60% opacity
	n.defaultBg.Fill(overlayBg)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(bgX), float64(bgY))
	screen.DrawImage(n.defaultBg.SubImage(image.Rect(0, 0, bgWidth, bgHeight)).(*ebiten.Image), opts)

	// Draw text centered in background
	textOpts := &text.DrawOptions{}
	textOpts.GeoM.Translate(float64(bgX+padding), float64(bgY+padding+int(textHeight)))
	textOpts.ColorScale.ScaleWithColor(style.Text)
	text.Draw(screen, message, *style.FontFace(), textOpts)
}

// drawAchievementWithData renders a prominent achievement notification at top-center
func (n *Notification) drawAchievementWithData(screen *ebiten.Image, titleText, descText string, badge *ebiten.Image) {
	bounds := screen.Bounds()
	screenWidth := bounds.Dx()

	// Use large font for title, regular for description
	largeFace := n.largeFace
	if largeFace == nil {
		largeFace = style.LargeFontFace()
	}
	if largeFace == nil {
		// Fallback to default notification if large font unavailable
		n.drawDefaultWithData(screen, titleText)
		return
	}

	// Badge dimensions (RetroAchievements badges are 64x64)
	badgeSize := 64
	badgeSpacing := 12

	// Header text
	headerText := "Achievement Unlocked"

	// Measure text
	headerWidth, headerHeight := text.Measure(headerText, *style.FontFace(), 0)
	titleWidth, titleHeight := text.Measure(titleText, largeFace, 0)
	var descWidth, descHeight float64
	if descText != "" {
		descWidth, descHeight = text.Measure(descText, *style.FontFace(), 0)
	}

	// Calculate content width (text area)
	maxTextWidth := headerWidth
	if titleWidth > maxTextWidth {
		maxTextWidth = titleWidth
	}
	if descWidth > maxTextWidth {
		maxTextWidth = descWidth
	}

	// Calculate box size
	paddingH := 20
	paddingV := 16
	spacing := 6

	// Content width includes badge + spacing + text
	contentWidth := int(maxTextWidth)
	if badge != nil {
		contentWidth += badgeSize + badgeSpacing
	}

	bgWidth := contentWidth + paddingH*2
	bgHeight := paddingV*2 + int(headerHeight) + spacing + int(titleHeight)
	if descText != "" {
		bgHeight += spacing + int(descHeight)
	}
	// Ensure minimum height for badge
	if badge != nil && bgHeight < badgeSize+paddingV*2 {
		bgHeight = badgeSize + paddingV*2
	}

	// Position: top-center, 20px from top
	bgX := (screenWidth - bgWidth) / 2
	bgY := 20

	// Reuse or create background image (only recreate if size changed)
	if n.achievementBg == nil || n.lastBgWidth != bgWidth || n.lastBgHeight != bgHeight {
		n.achievementBg = ebiten.NewImage(bgWidth, bgHeight)
		n.lastBgWidth = bgWidth
		n.lastBgHeight = bgHeight

		// Fill with dark background
		achieveBg := style.OverlayBackground
		achieveBg.A = 240 // 94% opacity
		n.achievementBg.Fill(achieveBg)

		// Draw gold border (2px)
		gold := style.Accent
		borderSize := 2
		for x := 0; x < bgWidth; x++ {
			for y := 0; y < borderSize; y++ {
				n.achievementBg.Set(x, y, gold)            // Top
				n.achievementBg.Set(x, bgHeight-1-y, gold) // Bottom
			}
		}
		for y := 0; y < bgHeight; y++ {
			for x := 0; x < borderSize; x++ {
				n.achievementBg.Set(x, y, gold)           // Left
				n.achievementBg.Set(bgWidth-1-x, y, gold) // Right
			}
		}
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(bgX), float64(bgY))
	screen.DrawImage(n.achievementBg, opts)

	// Calculate text start position (after badge if present)
	textStartX := float64(bgX + paddingH)
	if badge != nil {
		textStartX += float64(badgeSize + badgeSpacing)
	}

	// Draw badge on the left
	if badge != nil {
		badgeOpts := &ebiten.DrawImageOptions{}
		badgeBounds := badge.Bounds()
		// Scale badge to 64x64 if needed
		scaleX := float64(badgeSize) / float64(badgeBounds.Dx())
		scaleY := float64(badgeSize) / float64(badgeBounds.Dy())
		badgeOpts.GeoM.Scale(scaleX, scaleY)
		// Center badge vertically
		badgeY := float64(bgY) + float64(bgHeight-badgeSize)/2
		badgeOpts.GeoM.Translate(float64(bgX+paddingH), badgeY)
		screen.DrawImage(badge, badgeOpts)
	}

	// Draw header (small, gold)
	headerY := float64(bgY + paddingV)
	headerOpts := &text.DrawOptions{}
	headerOpts.GeoM.Translate(textStartX, headerY)
	headerOpts.ColorScale.ScaleWithColor(style.Accent)
	text.Draw(screen, headerText, *style.FontFace(), headerOpts)

	// Draw title (large, white)
	titleY := headerY + headerHeight + float64(spacing)
	titleOpts := &text.DrawOptions{}
	titleOpts.GeoM.Translate(textStartX, titleY)
	titleOpts.ColorScale.ScaleWithColor(style.Text)
	text.Draw(screen, titleText, largeFace, titleOpts)

	// Draw description (small, secondary)
	if descText != "" {
		descY := titleY + titleHeight + float64(spacing)
		descOpts := &text.DrawOptions{}
		descOpts.GeoM.Translate(textStartX, descY)
		descOpts.ColorScale.ScaleWithColor(style.TextSecondary)
		text.Draw(screen, descText, *style.FontFace(), descOpts)
	}
}
