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

// shaderSources maps shader IDs to their Kage source code
var shaderSources = map[string][]byte{
	"crt":       crtShaderSrc,
	"scanlines": scanlinesShaderSrc,
}

// Manager handles shader compilation, caching, and application
type Manager struct {
	// Compiled shader cache
	shaders map[string]*ebiten.Shader

	// Intermediate buffer for shader chaining
	buffer *ebiten.Image
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

// getOrCreateBuffer returns a buffer matching the given dimensions
func (m *Manager) getOrCreateBuffer(width, height int) *ebiten.Image {
	if m.buffer != nil {
		bw, bh := m.buffer.Bounds().Dx(), m.buffer.Bounds().Dy()
		if bw == width && bh == height {
			return m.buffer
		}
		// Size changed, dispose old buffer
		m.buffer.Deallocate()
	}
	m.buffer = ebiten.NewImage(width, height)
	return m.buffer
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

	// Multiple shaders - chain through intermediate buffer
	buffer := m.getOrCreateBuffer(srcW, srcH)

	// First shader: src -> buffer
	buffer.Clear()
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	buffer.DrawRectShader(srcW, srcH, validShaders[0], op)

	// Middle shaders: ping-pong between src-copy and buffer
	// For simplicity with 2 shaders (our current case), we go straight to dst
	// For more shaders, we'd need a second buffer for ping-pong

	// Last shader: buffer -> dst
	op = &ebiten.DrawRectShaderOptions{}
	op.Images[0] = buffer
	dst.DrawRectShader(srcW, srcH, validShaders[len(validShaders)-1], op)

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
