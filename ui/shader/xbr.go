//go:build !libretro

package shader

import (
	_ "embed"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed shaders/xbr.kage
var xbrShaderSrc []byte

// XBRScaler handles xBR pixel art scaling.
type XBRScaler struct{}

// NewXBRScaler creates a new xBR scaler instance
func NewXBRScaler() *XBRScaler {
	return &XBRScaler{}
}

// Apply runs xBR scaling on the source and returns a screen-sized image.
func (x *XBRScaler) Apply(src *ebiten.Image, screenW, screenH int) *ebiten.Image {
	if src == nil {
		return nil
	}

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	// Copy SubImage to regular image at (0,0) to fix coordinate issues
	// SubImages have non-zero bounds that break DrawTrianglesShader srcPos interpolation
	normalizedSrc := ebiten.NewImage(srcW, srcH)
	normalizedSrc.DrawImage(src, nil)

	// Create shader
	shader, err := ebiten.NewShader(xbrShaderSrc)
	if err != nil {
		normalizedSrc.Deallocate()
		return x.scaleToScreen(src, screenW, screenH)
	}

	// Create 2x buffer for shader output
	outW := srcW * 2
	outH := srcH * 2
	shaderOutput := ebiten.NewImage(outW, outH)

	// Run shader with vertices mapping dst to src coordinates
	vertices := []ebiten.Vertex{
		{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(outW), DstY: 0, SrcX: float32(srcW), SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 0, DstY: float32(outH), SrcX: 0, SrcY: float32(srcH), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(outW), DstY: float32(outH), SrcX: float32(srcW), SrcY: float32(srcH), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	indices := []uint16{0, 1, 2, 1, 3, 2}

	op := &ebiten.DrawTrianglesShaderOptions{}
	op.Images[0] = normalizedSrc

	shaderOutput.DrawTrianglesShader(vertices, indices, shader, op)

	// Scale shader output to screen with centering
	screenBuffer := x.scaleToScreen(shaderOutput, screenW, screenH)

	// Clean up
	shaderOutput.Deallocate()
	normalizedSrc.Deallocate()

	return screenBuffer
}

// scaleToScreen scales src to fit screen, centered with aspect ratio preserved
func (x *XBRScaler) scaleToScreen(src *ebiten.Image, screenW, screenH int) *ebiten.Image {
	srcW := float64(src.Bounds().Dx())
	srcH := float64(src.Bounds().Dy())

	// Calculate scale to fit
	scaleX := float64(screenW) / srcW
	scaleY := float64(screenH) / srcH
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Calculate centering offset
	scaledW := srcW * scale
	scaledH := srcH * scale
	offsetX := (float64(screenW) - scaledW) / 2
	offsetY := (float64(screenH) - scaledH) / 2

	// Create screen buffer and draw centered
	screenBuffer := ebiten.NewImage(screenW, screenH)

	drawOp := &ebiten.DrawImageOptions{}
	drawOp.GeoM.Scale(scale, scale)
	drawOp.GeoM.Translate(offsetX, offsetY)
	drawOp.Filter = ebiten.FilterNearest
	screenBuffer.DrawImage(src, drawOp)

	return screenBuffer
}
