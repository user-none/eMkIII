//go:build !libretro

package shader

import (
	_ "embed"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed shaders/crt.kage
var crtShaderSrc []byte

//go:embed shaders/scanlines.kage
var scanlinesShaderSrc []byte

//go:embed shaders/bloom.kage
var bloomShaderSrc []byte

//go:embed shaders/lcd.kage
var lcdShaderSrc []byte

//go:embed shaders/colorbleed.kage
var colorbleedShaderSrc []byte

//go:embed shaders/dotmatrix.kage
var dotmatrixShaderSrc []byte

//go:embed shaders/ntsc.kage
var ntscShaderSrc []byte

//go:embed shaders/gamma.kage
var gammaShaderSrc []byte

//go:embed shaders/halation.kage
var halationShaderSrc []byte

//go:embed shaders/rfnoise.kage
var rfnoiseShaderSrc []byte

//go:embed shaders/rollingband.kage
var rollingbandShaderSrc []byte

//go:embed shaders/vhs.kage
var vhsShaderSrc []byte

//go:embed shaders/interlace.kage
var interlaceShaderSrc []byte

//go:embed shaders/monochrome.kage
var monochromeShaderSrc []byte

//go:embed shaders/sepia.kage
var sepiaShaderSrc []byte

// shaderSources maps shader IDs to their Kage source code
var shaderSources = map[string][]byte{
	"crt":         crtShaderSrc,
	"scanlines":   scanlinesShaderSrc,
	"bloom":       bloomShaderSrc,
	"lcd":         lcdShaderSrc,
	"colorbleed":  colorbleedShaderSrc,
	"dotmatrix":   dotmatrixShaderSrc,
	"ntsc":        ntscShaderSrc,
	"gamma":       gammaShaderSrc,
	"halation":    halationShaderSrc,
	"rfnoise":     rfnoiseShaderSrc,
	"rollingband": rollingbandShaderSrc,
	"vhs":         vhsShaderSrc,
	"interlace":   interlaceShaderSrc,
	"monochrome":  monochromeShaderSrc,
	"sepia":       sepiaShaderSrc,
}

// specialEffects lists effect IDs that appear in the shader menu
// but are implemented directly in Go rather than as Kage shaders
var specialEffects = map[string]bool{
	"ghosting": true,
}

// isSpecialEffect returns true if the ID is a special effect, not a compiled shader
func isSpecialEffect(id string) bool {
	return specialEffects[id]
}

// Manager handles shader compilation, caching, and application
type Manager struct {
	// Compiled shader cache
	shaders map[string]*ebiten.Shader

	// Intermediate buffers for shader chaining (ping-pong)
	bufferA *ebiten.Image
	bufferB *ebiten.Image

	// Ghosting buffer for phosphor persistence (persistent across frames)
	ghostingBuffer *ebiten.Image

	// Frame counter for animated shaders
	frame int
}

// NewManager creates a new shader manager
func NewManager() *Manager {
	return &Manager{
		shaders: make(map[string]*ebiten.Shader),
	}
}

// IncrementFrame advances the frame counter for animated shaders
func (m *Manager) IncrementFrame() {
	m.frame++
}

// Frame returns the current frame count
func (m *Manager) Frame() int {
	return m.frame
}

// LoadShader compiles and caches a shader by ID
func (m *Manager) LoadShader(id string) error {
	// Already loaded?
	if _, ok := m.shaders[id]; ok {
		return nil
	}

	// Get source
	src, ok := shaderSources[id]
	if !ok {
		return fmt.Errorf("unknown shader: %s", id)
	}

	// Compile
	shader, err := ebiten.NewShader(src)
	if err != nil {
		return fmt.Errorf("failed to compile shader %s: %w", id, err)
	}

	m.shaders[id] = shader
	return nil
}

// PreloadShaders loads all shaders in the given list
func (m *Manager) PreloadShaders(ids []string) {
	for _, id := range ids {
		if isSpecialEffect(id) {
			continue
		}
		if err := m.LoadShader(id); err != nil {
			log.Printf("Warning: failed to load shader %s: %v", id, err)
		}
	}
}

// ensureGhostingBuffer creates or resizes the ghosting buffer to match dimensions
func (m *Manager) ensureGhostingBuffer(width, height int) {
	if m.ghostingBuffer != nil {
		bw, bh := m.ghostingBuffer.Bounds().Dx(), m.ghostingBuffer.Bounds().Dy()
		if bw != width || bh != height {
			m.ghostingBuffer.Deallocate()
			m.ghostingBuffer = nil
		}
	}
	if m.ghostingBuffer == nil {
		m.ghostingBuffer = ebiten.NewImage(width, height)
	}
}

// ensureBuffers creates or resizes the ping-pong buffers to match dimensions
func (m *Manager) ensureBuffers(width, height int) {
	// Check if bufferA needs (re)creation
	if m.bufferA != nil {
		bw, bh := m.bufferA.Bounds().Dx(), m.bufferA.Bounds().Dy()
		if bw != width || bh != height {
			m.bufferA.Deallocate()
			m.bufferA = nil
		}
	}
	if m.bufferA == nil {
		m.bufferA = ebiten.NewImage(width, height)
	}

	// Check if bufferB needs (re)creation
	if m.bufferB != nil {
		bw, bh := m.bufferB.Bounds().Dx(), m.bufferB.Bounds().Dy()
		if bw != width || bh != height {
			m.bufferB.Deallocate()
			m.bufferB = nil
		}
	}
	if m.bufferB == nil {
		m.bufferB = ebiten.NewImage(width, height)
	}
}

// applyGhosting applies the ghosting pre-processing step.
// It updates the ghosting buffer and returns a ghosted image.
func (m *Manager) applyGhosting(src *ebiten.Image) *ebiten.Image {
	srcW, srcH := src.Bounds().Dx(), src.Bounds().Dy()
	m.ensureGhostingBuffer(srcW, srcH)
	m.ensureBuffers(srcW, srcH)

	// Update ghosting buffer: buffer = buffer * 0.6 + src * 0.4
	// Step 1: Copy ghostingBuffer at 60% to bufferA
	m.bufferA.Clear()
	decayOp := &ebiten.DrawImageOptions{}
	decayOp.ColorScale.Scale(0.6, 0.6, 0.6, 1.0)
	m.bufferA.DrawImage(m.ghostingBuffer, decayOp)

	// Step 2: Add current at 40% to bufferA (additive blend)
	addOp := &ebiten.DrawImageOptions{}
	addOp.ColorScale.Scale(0.4, 0.4, 0.4, 1.0)
	addOp.Blend = ebiten.Blend{
		BlendFactorSourceRGB:        ebiten.BlendFactorOne,
		BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
		BlendFactorDestinationRGB:   ebiten.BlendFactorOne,
		BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
		BlendOperationRGB:           ebiten.BlendOperationAdd,
		BlendOperationAlpha:         ebiten.BlendOperationAdd,
	}
	m.bufferA.DrawImage(src, addOp)

	// Step 3: Copy bufferA to ghostingBuffer for next frame
	m.ghostingBuffer.Clear()
	m.ghostingBuffer.DrawImage(m.bufferA, nil)

	// Return the blended result (bufferA already contains it)
	return m.bufferA
}

// hasGhosting returns true if "ghosting" is in the shader list
func hasGhosting(shaderIDs []string) bool {
	for _, id := range shaderIDs {
		if id == "ghosting" {
			return true
		}
	}
	return false
}

// removeGhosting returns a new slice without "ghosting"
func removeGhosting(shaderIDs []string) []string {
	result := make([]string, 0, len(shaderIDs))
	for _, id := range shaderIDs {
		if id != "ghosting" {
			result = append(result, id)
		}
	}
	return result
}

// ApplyShaders draws src to dst with the specified shader chain applied.
// If shaderIDs is empty, src is drawn directly to dst.
// Ghosting is handled as a pre-processing step before other shaders.
// Returns true if shaders were applied, false if direct draw was used.
func (m *Manager) ApplyShaders(dst, src *ebiten.Image, shaderIDs []string) bool {
	if len(shaderIDs) == 0 {
		// No shaders, direct draw
		op := &ebiten.DrawImageOptions{}
		dst.DrawImage(src, op)
		return false
	}

	// Handle ghosting as pre-processing
	effectiveInput := src
	remainingShaders := shaderIDs

	if hasGhosting(shaderIDs) {
		effectiveInput = m.applyGhosting(src)
		remainingShaders = removeGhosting(shaderIDs)
	}

	// If no remaining shaders, draw the (possibly ghosted) input to destination
	if len(remainingShaders) == 0 {
		op := &ebiten.DrawImageOptions{}
		dst.DrawImage(effectiveInput, op)
		return true
	}

	// Load any missing shaders
	for _, id := range remainingShaders {
		if _, ok := m.shaders[id]; !ok {
			if err := m.LoadShader(id); err != nil {
				log.Printf("Warning: shader %s not available: %v", id, err)
			}
		}
	}

	// Filter to only shaders that compiled successfully
	validShaders := make([]*ebiten.Shader, 0, len(remainingShaders))
	for _, id := range remainingShaders {
		if s, ok := m.shaders[id]; ok {
			validShaders = append(validShaders, s)
		}
	}

	if len(validShaders) == 0 {
		// No valid shaders, draw the effective input
		op := &ebiten.DrawImageOptions{}
		dst.DrawImage(effectiveInput, op)
		return hasGhosting(shaderIDs) // Return true if ghosting was applied
	}

	srcW, srcH := effectiveInput.Bounds().Dx(), effectiveInput.Bounds().Dy()

	// Uniforms for animated shaders
	uniforms := map[string]interface{}{
		"Time": float32(m.frame),
	}

	// Single shader case - draw directly to destination
	if len(validShaders) == 1 {
		op := &ebiten.DrawRectShaderOptions{}
		op.Images[0] = effectiveInput
		op.Uniforms = uniforms
		dst.DrawRectShader(srcW, srcH, validShaders[0], op)
		return true
	}

	// Multiple shaders - chain through ping-pong buffers
	m.ensureBuffers(srcW, srcH)

	// Track current input for each pass
	currentInput := effectiveInput
	buffers := [2]*ebiten.Image{m.bufferA, m.bufferB}
	bufferIndex := 1

	for i, shader := range validShaders {
		op := &ebiten.DrawRectShaderOptions{}
		op.Images[0] = currentInput
		op.Uniforms = uniforms

		if i == len(validShaders)-1 {
			// Last shader writes to destination
			dst.DrawRectShader(srcW, srcH, shader, op)
		} else {
			// Intermediate shaders write to ping-pong buffer
			outputBuffer := buffers[bufferIndex%2]
			outputBuffer.Clear()
			outputBuffer.DrawRectShader(srcW, srcH, shader, op)
			currentInput = outputBuffer
			bufferIndex++
		}
	}

	return true
}
