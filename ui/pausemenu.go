//go:build !libretro

package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/user-none/emkiii/ui/style"
)

// PauseMenuOption represents a menu option
type PauseMenuOption int

const (
	PauseMenuResume PauseMenuOption = iota
	PauseMenuLibrary
	PauseMenuExit
	PauseMenuOptionCount
)

// PauseMenu handles the in-game pause menu
type PauseMenu struct {
	visible       bool
	selectedIndex int
	onResume      func()
	onLibrary     func()
	onExit        func()

	// Cached layout info for mouse hit testing
	buttonRects []image.Rectangle
}

// NewPauseMenu creates a new pause menu
func NewPauseMenu(onResume, onLibrary, onExit func()) *PauseMenu {
	return &PauseMenu{
		visible:       false,
		selectedIndex: 0,
		onResume:      onResume,
		onLibrary:     onLibrary,
		onExit:        onExit,
		buttonRects:   make([]image.Rectangle, PauseMenuOptionCount),
	}
}

// Show displays the pause menu
func (m *PauseMenu) Show() {
	m.visible = true
	m.selectedIndex = 0
}

// Hide hides the pause menu
func (m *PauseMenu) Hide() {
	m.visible = false
}

// IsVisible returns whether the menu is visible
func (m *PauseMenu) IsVisible() bool {
	return m.visible
}

// Update handles input for the pause menu
func (m *PauseMenu) Update() {
	if !m.visible {
		return
	}

	// ESC closes menu (same as Resume)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		m.handleSelect()
		return
	}

	// Keyboard navigation
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		m.selectedIndex--
		if m.selectedIndex < 0 {
			m.selectedIndex = int(PauseMenuOptionCount) - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		m.selectedIndex++
		if m.selectedIndex >= int(PauseMenuOptionCount) {
			m.selectedIndex = 0
		}
	}

	// Keyboard selection
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		m.handleSelect()
		return
	}

	// Mouse click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		for i, rect := range m.buttonRects {
			if image.Pt(mx, my).In(rect) {
				m.selectedIndex = i
				m.handleSelect()
				return
			}
		}
	}

	// Mouse hover for selection highlight
	mx, my := ebiten.CursorPosition()
	for i, rect := range m.buttonRects {
		if image.Pt(mx, my).In(rect) {
			m.selectedIndex = i
			break
		}
	}

	// Gamepad support
	gamepadIDs := ebiten.AppendGamepadIDs(nil)
	for _, id := range gamepadIDs {
		// D-pad navigation
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			m.selectedIndex--
			if m.selectedIndex < 0 {
				m.selectedIndex = int(PauseMenuOptionCount) - 1
			}
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			m.selectedIndex++
			if m.selectedIndex >= int(PauseMenuOptionCount) {
				m.selectedIndex = 0
			}
		}

		// A/Cross button selects
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom) {
			m.handleSelect()
			return
		}

		// B/Circle button acts as Resume
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightRight) {
			m.selectedIndex = int(PauseMenuResume)
			m.handleSelect()
			return
		}

		// Start button acts as Resume
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight) {
			m.selectedIndex = int(PauseMenuResume)
			m.handleSelect()
			return
		}
	}
}

// handleSelect processes the current selection
func (m *PauseMenu) handleSelect() {
	switch PauseMenuOption(m.selectedIndex) {
	case PauseMenuResume:
		m.Hide()
		if m.onResume != nil {
			m.onResume()
		}
	case PauseMenuLibrary:
		m.Hide()
		if m.onLibrary != nil {
			m.onLibrary()
		}
	case PauseMenuExit:
		m.Hide()
		if m.onExit != nil {
			m.onExit()
		}
	}
}

// Draw renders the pause menu
func (m *PauseMenu) Draw(screen *ebiten.Image) {
	if !m.visible {
		return
	}

	// Dim overlay (50% black)
	bounds := screen.Bounds()
	screenW := bounds.Dx()
	screenH := bounds.Dy()
	dimOverlay := ebiten.NewImage(screenW, screenH)
	dimOverlay.Fill(color.RGBA{0, 0, 0, 128})
	screen.DrawImage(dimOverlay, nil)

	// Menu panel dimensions - proportional to screen size
	panelWidth := screenW * 40 / 100
	if panelWidth < 150 {
		panelWidth = 150
	}
	if panelWidth > 300 {
		panelWidth = 300
	}

	// Button dimensions - proportional to panel
	buttonWidth := panelWidth * 80 / 100
	buttonHeight := screenH * 8 / 100
	if buttonHeight < 30 {
		buttonHeight = 30
	}
	if buttonHeight > 50 {
		buttonHeight = 50
	}

	buttonSpacing := buttonHeight / 4
	padding := buttonHeight / 2

	// Calculate panel height based on content
	options := []string{"Resume", "Library", "Exit"}
	panelHeight := padding*2 + len(options)*buttonHeight + (len(options)-1)*buttonSpacing

	panelX := (screenW - panelWidth) / 2
	panelY := (screenH - panelHeight) / 2

	// Draw panel background
	panelBg := ebiten.NewImage(panelWidth, panelHeight)
	panelBg.Fill(style.Surface)

	// Draw panel border
	for x := 0; x < panelWidth; x++ {
		panelBg.Set(x, 0, style.Border)
		panelBg.Set(x, panelHeight-1, style.Border)
	}
	for y := 0; y < panelHeight; y++ {
		panelBg.Set(0, y, style.Border)
		panelBg.Set(panelWidth-1, y, style.Border)
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(panelX), float64(panelY))
	screen.DrawImage(panelBg, opts)

	// Draw menu options and cache button rects for hit testing
	startY := panelY + padding

	for i, optionText := range options {
		buttonX := panelX + (panelWidth-buttonWidth)/2
		buttonY := startY + i*(buttonHeight+buttonSpacing)

		// Cache button rect for mouse hit testing
		m.buttonRects[i] = image.Rect(buttonX, buttonY, buttonX+buttonWidth, buttonY+buttonHeight)

		// Button background
		buttonBg := ebiten.NewImage(buttonWidth, buttonHeight)
		if i == m.selectedIndex {
			buttonBg.Fill(style.Primary)
		} else {
			buttonBg.Fill(style.Surface)
			// Draw border
			for x := 0; x < buttonWidth; x++ {
				buttonBg.Set(x, 0, style.Border)
				buttonBg.Set(x, buttonHeight-1, style.Border)
			}
			for y := 0; y < buttonHeight; y++ {
				buttonBg.Set(0, y, style.Border)
				buttonBg.Set(buttonWidth-1, y, style.Border)
			}
		}

		btnOpts := &ebiten.DrawImageOptions{}
		btnOpts.GeoM.Translate(float64(buttonX), float64(buttonY))
		screen.DrawImage(buttonBg, btnOpts)

		// Draw text centered
		textOpts := &text.DrawOptions{}
		textOpts.GeoM.Translate(float64(buttonX+buttonWidth/2), float64(buttonY+buttonHeight/2))
		textOpts.PrimaryAlign = text.AlignCenter
		textOpts.SecondaryAlign = text.AlignCenter
		textOpts.ColorScale.ScaleWithColor(style.Text)
		text.Draw(screen, optionText, *style.FontFace(), textOpts)
	}
}
