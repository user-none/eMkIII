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

// shaderSources maps shader IDs to their Kage source code
var shaderSources = map[string][]byte{
	"crt":        crtShaderSrc,
	"scanlines":  scanlinesShaderSrc,
	"bloom":      bloomShaderSrc,
	"lcd":        lcdShaderSrc,
	"colorbleed": colorbleedShaderSrc,
	"dotmatrix":  dotmatrixShaderSrc,
	"ntsc":       ntscShaderSrc,
	"gamma":      gammaShaderSrc,
}

// Manager handles shader compilation, caching, and application
type Manager struct {
	// Compiled shader cache
	shaders map[string]*ebiten.Shader

	// Intermediate buffers for shader chaining (ping-pong)
	bufferA *ebiten.Image
	bufferB *ebiten.Image
}

// NewManager creates a new shader manager
func NewManager() *Manager {
	return &Manager{
		shaders: make(map[string]*ebiten.Shader),
	}
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
		if err := m.LoadShader(id); err != nil {
			log.Printf("Warning: failed to load shader %s: %v", id, err)
		}
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

// ApplyShaders draws src to dst with the specified shader chain applied.
// If shaderIDs is empty, src is drawn directly to dst.
// Returns true if shaders were applied, false if direct draw was used.
func (m *Manager) ApplyShaders(dst, src *ebiten.Image, shaderIDs []string) bool {
	if len(shaderIDs) == 0 {
		// No shaders, direct draw
		op := &ebiten.DrawImageOptions{}
		dst.DrawImage(src, op)
		return false
	}

	// Load any missing shaders
	for _, id := range shaderIDs {
		if _, ok := m.shaders[id]; !ok {
			if err := m.LoadShader(id); err != nil {
				log.Printf("Warning: shader %s not available: %v", id, err)
			}
		}
	}

	// Filter to only valid shaders
	validShaders := make([]*ebiten.Shader, 0, len(shaderIDs))
	for _, id := range shaderIDs {
		if s, ok := m.shaders[id]; ok {
			validShaders = append(validShaders, s)
		}
	}

	if len(validShaders) == 0 {
		// No valid shaders, direct draw
		op := &ebiten.DrawImageOptions{}
		dst.DrawImage(src, op)
		return false
	}

	srcW, srcH := src.Bounds().Dx(), src.Bounds().Dy()

	// Single shader - apply directly to dst
	if len(validShaders) == 1 {
		op := &ebiten.DrawRectShaderOptions{}
		op.Images[0] = src
		dst.DrawRectShader(srcW, srcH, validShaders[0], op)
		return true
	}

	// Multiple shaders - chain through ping-pong buffers
	m.ensureBuffers(srcW, srcH)

	// Track current input for each pass
	currentInput := src
	buffers := [2]*ebiten.Image{m.bufferA, m.bufferB}

	for i, shader := range validShaders {
		op := &ebiten.DrawRectShaderOptions{}
		op.Images[0] = currentInput

		if i == len(validShaders)-1 {
			// Last shader writes to destination
			dst.DrawRectShader(srcW, srcH, shader, op)
		} else {
			// Intermediate shaders write to ping-pong buffer
			outputBuffer := buffers[i%2]
			outputBuffer.Clear()
			outputBuffer.DrawRectShader(srcW, srcH, shader, op)
			currentInput = outputBuffer
		}
	}

	return true
}

// HasShader returns true if the shader ID is available
func (m *Manager) HasShader(id string) bool {
	_, ok := shaderSources[id]
	return ok
}

// IsLoaded returns true if the shader is compiled and ready
func (m *Manager) IsLoaded(id string) bool {
	_, ok := m.shaders[id]
	return ok
}
