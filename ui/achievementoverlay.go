//go:build !libretro

package ui

import (
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/user-none/emkiii/ui/achievements"
	"github.com/user-none/emkiii/ui/style"
	"github.com/user-none/go-rcheevos"
)

// AchievementOverlay shows achievements during gameplay
type AchievementOverlay struct {
	visible bool
	manager *achievements.Manager

	// Badge loading state (actual cache is in manager)
	mu            sync.Mutex
	badgesPending map[uint32]bool
	// Local grayscale badge cache (cleared on close)
	grayscaleBadges map[uint32]*ebiten.Image

	// Scroll state
	scrollOffset float64
	scrollMax    float64
	itemHeight   int
	visibleItems int

	// Cached images
	cache struct {
		screenW, screenH   int
		panelW, panelH     int
		dimOverlay         *ebiten.Image
		panelBg            *ebiten.Image
		rowBg              *ebiten.Image
		rowBgWidth         int
		placeholderBadge   *ebiten.Image
		placeholderBadgeSz int
	}

	// Pre-allocated draw options
	drawOpts ebiten.DrawImageOptions
	textOpts text.DrawOptions
}

// NewAchievementOverlay creates a new achievement overlay
func NewAchievementOverlay(manager *achievements.Manager) *AchievementOverlay {
	o := &AchievementOverlay{
		manager:         manager,
		itemHeight:      style.AchievementRowHeight,
		badgesPending:   make(map[uint32]bool),
		grayscaleBadges: make(map[uint32]*ebiten.Image),
	}
	// Register callback to clear grayscale cache when achievements unlock
	if manager != nil {
		manager.SetOnUnlockCallback(o.handleUnlock)
	}
	return o
}

// Show displays the achievement overlay
func (o *AchievementOverlay) Show() {
	o.visible = true
	o.scrollOffset = 0
	o.updateScrollMax()
}

// InitForGame prepares the overlay for a new game session.
// The manager already caches achievements on LoadGame, so this just resets overlay state.
func (o *AchievementOverlay) InitForGame() {
	o.mu.Lock()
	o.grayscaleBadges = make(map[uint32]*ebiten.Image)
	o.badgesPending = make(map[uint32]bool)
	o.mu.Unlock()
	o.scrollOffset = 0
}

// Hide hides the achievement overlay
func (o *AchievementOverlay) Hide() {
	o.visible = false
	// Clear grayscale cache
	o.mu.Lock()
	o.grayscaleBadges = make(map[uint32]*ebiten.Image)
	o.mu.Unlock()
}

// handleUnlock is called when an achievement is unlocked during gameplay.
// The manager updates its cached list; we just need to clear our grayscale cache.
func (o *AchievementOverlay) handleUnlock(achievementID uint32) {
	o.mu.Lock()
	delete(o.grayscaleBadges, achievementID)
	o.mu.Unlock()
}

// IsVisible returns whether the overlay is visible
func (o *AchievementOverlay) IsVisible() bool {
	return o.visible
}

// Reset clears session state when the game ends
func (o *AchievementOverlay) Reset() {
	o.mu.Lock()
	o.grayscaleBadges = make(map[uint32]*ebiten.Image)
	o.badgesPending = make(map[uint32]bool)
	o.mu.Unlock()
	o.scrollOffset = 0
}

// getAchievements returns the cached achievements from the manager
func (o *AchievementOverlay) getAchievements() []*rcheevos.Achievement {
	if o.manager == nil {
		return nil
	}
	return o.manager.GetCachedAchievements()
}

// getGameTitle returns the cached game title from the manager
func (o *AchievementOverlay) getGameTitle() string {
	if o.manager == nil {
		return ""
	}
	return o.manager.GetCachedGameTitle()
}

// computeSummary calculates summary stats from the cached achievements
func (o *AchievementOverlay) computeSummary(achievements []*rcheevos.Achievement) (numTotal, numUnlocked, pointsTotal, pointsUnlocked uint32) {
	for _, ach := range achievements {
		numTotal++
		pointsTotal += ach.Points
		if ach.Unlocked != rcheevos.AchievementUnlockedNone {
			numUnlocked++
			pointsUnlocked += ach.Points
		}
	}
	return
}

// updateScrollMax calculates the maximum scroll offset
func (o *AchievementOverlay) updateScrollMax() {
	achievements := o.getAchievements()
	count := len(achievements)

	if o.visibleItems > 0 && count > o.visibleItems {
		o.scrollMax = float64(count - o.visibleItems)
	} else {
		o.scrollMax = 0
	}
}

// Update handles input for the overlay
func (o *AchievementOverlay) Update() {
	if !o.visible {
		return
	}

	// ESC closes overlay
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		o.Hide()
		return
	}

	// Keyboard navigation
	scrollAmount := 0.0
	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		scrollAmount = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		scrollAmount = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		scrollAmount = -float64(o.visibleItems)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		scrollAmount = float64(o.visibleItems)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		o.scrollOffset = 0
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		o.scrollOffset = o.scrollMax
		return
	}

	// Mouse wheel
	_, wheelY := ebiten.Wheel()
	if wheelY != 0 {
		scrollAmount = -wheelY * 2
	}

	// Gamepad support
	gamepadIDs := ebiten.AppendGamepadIDs(nil)
	for _, id := range gamepadIDs {
		// D-pad navigation
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			scrollAmount = -1
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			scrollAmount = 1
		}
		// B button closes
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightRight) {
			o.Hide()
			return
		}
		// Start button closes
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight) {
			o.Hide()
			return
		}
	}

	// Apply scroll
	if scrollAmount != 0 {
		o.scrollOffset += scrollAmount
		if o.scrollOffset < 0 {
			o.scrollOffset = 0
		}
		if o.scrollOffset > o.scrollMax {
			o.scrollOffset = o.scrollMax
		}
	}
}

// rebuildCache recreates cached images when screen dimensions change
func (o *AchievementOverlay) rebuildCache(screenW, screenH int) {
	// Deallocate old images
	if o.cache.dimOverlay != nil {
		o.cache.dimOverlay.Deallocate()
	}
	if o.cache.panelBg != nil {
		o.cache.panelBg.Deallocate()
	}

	o.cache.screenW = screenW
	o.cache.screenH = screenH

	// Create dim overlay
	o.cache.dimOverlay = ebiten.NewImage(screenW, screenH)
	dimColor := style.DimOverlay
	dimColor.A = 160
	o.cache.dimOverlay.Fill(dimColor)

	// Calculate panel dimensions (centered, fixed width)
	panelWidth := style.AchievementOverlayWidth
	if panelWidth > screenW-40 {
		panelWidth = screenW - 40
	}
	panelHeight := screenH * 70 / 100
	if panelHeight < 200 {
		panelHeight = 200
	}

	o.cache.panelW = panelWidth
	o.cache.panelH = panelHeight

	// Calculate visible items
	headerHeight := 80 // Space for title and summary
	contentHeight := panelHeight - headerHeight - style.AchievementOverlayPadding*2
	o.visibleItems = contentHeight / o.itemHeight
	if o.visibleItems < 1 {
		o.visibleItems = 1
	}
	o.updateScrollMax()

	// Create panel background
	o.cache.panelBg = ebiten.NewImage(panelWidth, panelHeight)
	o.cache.panelBg.Fill(style.Surface)

	// Draw panel border
	for x := 0; x < panelWidth; x++ {
		o.cache.panelBg.Set(x, 0, style.Border)
		o.cache.panelBg.Set(x, panelHeight-1, style.Border)
	}
	for y := 0; y < panelHeight; y++ {
		o.cache.panelBg.Set(0, y, style.Border)
		o.cache.panelBg.Set(panelWidth-1, y, style.Border)
	}
}

// Draw renders the overlay
func (o *AchievementOverlay) Draw(screen *ebiten.Image) {
	if !o.visible {
		return
	}

	bounds := screen.Bounds()
	screenW := bounds.Dx()
	screenH := bounds.Dy()

	// Rebuild cache if screen dimensions changed
	if o.cache.screenW != screenW || o.cache.screenH != screenH {
		o.rebuildCache(screenW, screenH)
	}

	// Draw dim overlay
	screen.DrawImage(o.cache.dimOverlay, nil)

	// Calculate panel position (centered)
	panelX := (screenW - o.cache.panelW) / 2
	panelY := (screenH - o.cache.panelH) / 2

	// Draw panel background
	o.drawOpts.GeoM.Reset()
	o.drawOpts.GeoM.Translate(float64(panelX), float64(panelY))
	screen.DrawImage(o.cache.panelBg, &o.drawOpts)

	padding := style.AchievementOverlayPadding
	contentX := panelX + padding
	contentY := panelY + padding
	contentW := o.cache.panelW - padding*2

	// Get data from manager's cache
	achievements := o.getAchievements()
	gameTitle := o.getGameTitle()

	// Draw title
	title := "Achievements"
	if gameTitle != "" {
		title = gameTitle
	}

	o.textOpts = text.DrawOptions{}
	o.textOpts.GeoM.Translate(float64(contentX+contentW/2), float64(contentY+10))
	o.textOpts.PrimaryAlign = text.AlignCenter
	o.textOpts.ColorScale.ScaleWithColor(style.Text)
	text.Draw(screen, title, *style.FontFace(), &o.textOpts)

	// Draw spectator mode indicator if enabled
	spectatorMode := o.manager.IsSpectatorMode()
	if spectatorMode {
		o.textOpts = text.DrawOptions{}
		o.textOpts.GeoM.Translate(float64(contentX+contentW/2), float64(contentY+28))
		o.textOpts.PrimaryAlign = text.AlignCenter
		o.textOpts.ColorScale.ScaleWithColor(style.Accent)
		text.Draw(screen, "[SPECTATOR MODE]", *style.FontFace(), &o.textOpts)
	}

	// Calculate vertical offset for content below title (accounts for spectator mode banner)
	headerOffset := 0
	if spectatorMode {
		headerOffset = 18
	}

	// Draw summary
	if len(achievements) > 0 {
		numTotal, numUnlocked, pointsTotal, pointsUnlocked := o.computeSummary(achievements)
		pct := 0
		if numTotal > 0 {
			pct = int(numUnlocked * 100 / numTotal)
		}
		summaryText := fmt.Sprintf("Progress: %d/%d (%d%%)    Points: %d/%d",
			numUnlocked, numTotal, pct, pointsUnlocked, pointsTotal)

		o.textOpts = text.DrawOptions{}
		o.textOpts.GeoM.Translate(float64(contentX+contentW/2), float64(contentY+35+headerOffset))
		o.textOpts.PrimaryAlign = text.AlignCenter
		o.textOpts.ColorScale.ScaleWithColor(style.TextSecondary)
		text.Draw(screen, summaryText, *style.FontFace(), &o.textOpts)
	}

	// Draw separator line
	separatorY := contentY + 55 + headerOffset
	for x := contentX; x < contentX+contentW; x++ {
		screen.Set(x, separatorY, style.Border)
	}

	// Draw achievement list or "not available" message
	listY := separatorY + 10

	if len(achievements) == 0 {
		// Show message when no achievements are available
		centerY := (panelY + o.cache.panelH) / 2
		o.textOpts = text.DrawOptions{}
		o.textOpts.GeoM.Translate(float64(contentX+contentW/2), float64(centerY))
		o.textOpts.PrimaryAlign = text.AlignCenter
		o.textOpts.ColorScale.ScaleWithColor(style.TextSecondary)
		text.Draw(screen, "Achievements not available", *style.FontFace(), &o.textOpts)
	} else {
		startIdx := int(o.scrollOffset)
		endIdx := startIdx + o.visibleItems
		if endIdx > len(achievements) {
			endIdx = len(achievements)
		}

		for i := startIdx; i < endIdx; i++ {
			ach := achievements[i]
			rowY := listY + (i-startIdx)*o.itemHeight

			// Skip if row would be below panel
			if rowY+o.itemHeight > panelY+o.cache.panelH-padding {
				break
			}

			o.drawAchievementRow(screen, ach, contentX, rowY, contentW)
		}
	}

	// Draw close hint at bottom
	hintY := panelY + o.cache.panelH - padding - 5
	o.textOpts = text.DrawOptions{}
	o.textOpts.GeoM.Translate(float64(contentX+contentW/2), float64(hintY))
	o.textOpts.PrimaryAlign = text.AlignCenter
	o.textOpts.ColorScale.ScaleWithColor(style.TextSecondary)
	text.Draw(screen, "[ESC] to close", *style.FontFace(), &o.textOpts)
}

// drawAchievementRow draws a single achievement row
func (o *AchievementOverlay) drawAchievementRow(screen *ebiten.Image, ach *rcheevos.Achievement, x, y, width int) {
	if y < 0 || y > o.cache.screenH {
		return
	}

	isUnlocked := ach.Unlocked != rcheevos.AchievementUnlockedNone
	badgeSize := 48
	textX := x + badgeSize + 10 // Badge + padding
	textWidth := width - badgeSize - 20

	// Draw row background (same for locked and unlocked) - use cached image
	if o.cache.rowBg == nil || o.cache.rowBgWidth != width {
		if o.cache.rowBg != nil {
			o.cache.rowBg.Deallocate()
		}
		o.cache.rowBg = ebiten.NewImage(width, o.itemHeight-4)
		o.cache.rowBg.Fill(style.Background)
		o.cache.rowBgWidth = width
	}

	o.drawOpts.GeoM.Reset()
	o.drawOpts.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(o.cache.rowBg, &o.drawOpts)

	// Draw badge
	o.drawBadge(screen, ach, x+5, y+2, badgeSize)

	// Title - unlocked uses primary text, locked uses secondary
	titleColor := style.Text
	if !isUnlocked {
		titleColor = style.TextSecondary
	}
	o.textOpts = text.DrawOptions{}
	o.textOpts.GeoM.Translate(float64(textX), float64(y+5))
	o.textOpts.ColorScale.ScaleWithColor(titleColor)
	text.Draw(screen, ach.Title, *style.FontFace(), &o.textOpts)

	// Description (truncate if needed)
	desc := ach.Description
	maxDescLen := textWidth / 8
	if len(desc) > maxDescLen && maxDescLen > 3 {
		desc = desc[:maxDescLen-3] + "..."
	}
	o.textOpts = text.DrawOptions{}
	o.textOpts.GeoM.Translate(float64(textX), float64(y+25))
	o.textOpts.ColorScale.ScaleWithColor(style.TextSecondary)
	text.Draw(screen, desc, *style.FontFace(), &o.textOpts)

	// Points (right-aligned) - use primary color for unlocked, secondary for locked
	pointsText := fmt.Sprintf("%d pts", ach.Points)
	pointsColor := style.TextSecondary
	if isUnlocked {
		pointsColor = style.Primary
	}
	o.textOpts = text.DrawOptions{}
	o.textOpts.GeoM.Translate(float64(x+width-10), float64(y+15))
	o.textOpts.PrimaryAlign = text.AlignEnd
	o.textOpts.ColorScale.ScaleWithColor(pointsColor)
	text.Draw(screen, pointsText, *style.FontFace(), &o.textOpts)
}

// drawBadge draws an achievement badge, fetching it async if not cached in manager
// Applies grayscale effect for locked achievements
func (o *AchievementOverlay) drawBadge(screen *ebiten.Image, ach *rcheevos.Achievement, x, y, size int) {
	isUnlocked := ach.Unlocked != rcheevos.AchievementUnlockedNone

	// Check manager's badge cache (always stores colored version)
	if o.manager != nil {
		badge := o.manager.GetBadgeImage(ach.ID)
		if badge != nil {
			// For locked achievements, use cached grayscale version
			if !isUnlocked {
				o.mu.Lock()
				grayBadge, exists := o.grayscaleBadges[ach.ID]
				o.mu.Unlock()

				if !exists {
					// Create and cache grayscale version
					grayBadge = style.ApplyGrayscale(badge)
					o.mu.Lock()
					o.grayscaleBadges[ach.ID] = grayBadge
					o.mu.Unlock()
				}
				badge = grayBadge
			}

			// Scale from 64x64 (badge size) to target size using GeoM
			bounds := badge.Bounds()
			scaleX := float64(size) / float64(bounds.Dx())
			scaleY := float64(size) / float64(bounds.Dy())

			o.drawOpts.GeoM.Reset()
			o.drawOpts.GeoM.Scale(scaleX, scaleY)
			o.drawOpts.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(badge, &o.drawOpts)
			return
		}
	}

	// Draw placeholder while loading - use cached image
	if o.cache.placeholderBadge == nil || o.cache.placeholderBadgeSz != size {
		if o.cache.placeholderBadge != nil {
			o.cache.placeholderBadge.Deallocate()
		}
		o.cache.placeholderBadge = ebiten.NewImage(size, size)
		o.cache.placeholderBadge.Fill(style.Border)
		o.cache.placeholderBadgeSz = size
	}
	o.drawOpts.GeoM.Reset()
	o.drawOpts.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(o.cache.placeholderBadge, &o.drawOpts)

	// Start async fetch if not already pending - stores in manager's cache
	o.mu.Lock()
	pending := o.badgesPending[ach.ID]
	o.mu.Unlock()

	if o.manager != nil && !pending {
		o.mu.Lock()
		o.badgesPending[ach.ID] = true
		o.mu.Unlock()

		achID := ach.ID
		o.manager.GetBadgeImageAsync(achID, func(img *ebiten.Image) {
			// Badge is now in manager's cache, clear pending flag
			o.mu.Lock()
			delete(o.badgesPending, achID)
			o.mu.Unlock()
		})
	}
}
