//go:build !libretro

package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"
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
	fontFace      text.Face
	onResume      func()
	onLibrary     func()
	onExit        func()
}

// NewPauseMenu creates a new pause menu
func NewPauseMenu(onResume, onLibrary, onExit func()) *PauseMenu {
	return &PauseMenu{
		visible:       false,
		selectedIndex: 0,
		fontFace:      text.NewGoXFace(basicfont.Face7x13),
		onResume:      onResume,
		onLibrary:     onLibrary,
		onExit:        onExit,
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

	// ESC or Start closes menu (same as Resume)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		m.handleSelect()
		return
	}

	// Navigation
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

	// Selection
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		m.handleSelect()
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
		}

		// B/Circle button acts as Resume
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightRight) {
			m.selectedIndex = int(PauseMenuResume)
			m.handleSelect()
		}

		// Start button acts as Resume
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight) {
			m.selectedIndex = int(PauseMenuResume)
			m.handleSelect()
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
	// Panel is ~40% of screen width, with min/max limits
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
	panelBg.Fill(color.RGBA{0x25, 0x25, 0x3a, 0xff}) // Theme.Surface

	// Draw panel border
	borderColor := color.RGBA{0x3a, 0x3a, 0x5a, 0xff} // Theme.Border
	for x := 0; x < panelWidth; x++ {
		panelBg.Set(x, 0, borderColor)
		panelBg.Set(x, panelHeight-1, borderColor)
	}
	for y := 0; y < panelHeight; y++ {
		panelBg.Set(0, y, borderColor)
		panelBg.Set(panelWidth-1, y, borderColor)
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(panelX), float64(panelY))
	screen.DrawImage(panelBg, opts)

	// Draw menu options
	startY := panelY + padding

	for i, optionText := range options {
		buttonX := panelX + (panelWidth-buttonWidth)/2
		buttonY := startY + i*(buttonHeight+buttonSpacing)

		// Button background
		buttonBg := ebiten.NewImage(buttonWidth, buttonHeight)
		if i == m.selectedIndex {
			buttonBg.Fill(color.RGBA{0x4a, 0x4a, 0x8a, 0xff}) // Theme.Primary
		} else {
			buttonBg.Fill(color.RGBA{0x25, 0x25, 0x3a, 0xff}) // Theme.Surface
			// Draw border
			for x := 0; x < buttonWidth; x++ {
				buttonBg.Set(x, 0, borderColor)
				buttonBg.Set(x, buttonHeight-1, borderColor)
			}
			for y := 0; y < buttonHeight; y++ {
				buttonBg.Set(0, y, borderColor)
				buttonBg.Set(buttonWidth-1, y, borderColor)
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
		textOpts.ColorScale.ScaleWithColor(color.White)
		text.Draw(screen, optionText, m.fontFace, textOpts)
	}
}

// GetBounds returns the menu panel bounds for click detection
func (m *PauseMenu) GetBounds(screenWidth, screenHeight int) image.Rectangle {
	// Same proportional calculations as Draw
	panelWidth := screenWidth * 40 / 100
	if panelWidth < 150 {
		panelWidth = 150
	}
	if panelWidth > 300 {
		panelWidth = 300
	}

	buttonHeight := screenHeight * 8 / 100
	if buttonHeight < 30 {
		buttonHeight = 30
	}
	if buttonHeight > 50 {
		buttonHeight = 50
	}

	buttonSpacing := buttonHeight / 4
	padding := buttonHeight / 2
	numOptions := 3
	panelHeight := padding*2 + numOptions*buttonHeight + (numOptions-1)*buttonSpacing

	panelX := (screenWidth - panelWidth) / 2
	panelY := (screenHeight - panelHeight) / 2
	return image.Rect(panelX, panelY, panelX+panelWidth, panelY+panelHeight)
}
