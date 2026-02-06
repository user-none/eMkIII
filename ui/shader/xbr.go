//go:build !libretro

package shader

import (
	_ "embed"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed shaders/xbr.kage
var xbrShaderSrc []byte

// XBRScaler handles xBR pixel art scaling with cascaded 2x passes.
// Supports 2x (1 pass), 4x (2 passes), and 8x (3 passes) scaling.
type XBRScaler struct {
	shader *ebiten.Shader // Cached compiled shader
}

// NewXBRScaler creates a new xBR scaler instance
func NewXBRScaler() *XBRScaler {
	return &XBRScaler{}
}

// Apply runs xBR scaling on the source and returns a screen-sized image.
// Automatically selects 2x, 4x, or 8x scaling based on screen size.
func (x *XBRScaler) Apply(src *ebiten.Image, screenW, screenH int) *ebiten.Image {
	if src == nil {
		return nil
	}

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	// Ensure shader is compiled
	if err := x.ensureShader(); err != nil {
		return x.scaleToScreen(src, screenW, screenH)
	}

	// Copy SubImage to regular image at (0,0) to fix coordinate issues
	// SubImages have non-zero bounds that break DrawTrianglesShader srcPos interpolation
	normalizedSrc := ebiten.NewImage(srcW, srcH)
	normalizedSrc.DrawImage(src, nil)

	// Determine optimal scale factor and number of passes
	scaleFactor := selectOptimalScale(srcW, srcH, screenW, screenH)
	passes := scaleFactorToPasses(scaleFactor)

	// Execute cascade passes
	currentInput := normalizedSrc
	var currentOutput *ebiten.Image

	for pass := 0; pass < passes; pass++ {
		inW := currentInput.Bounds().Dx()
		inH := currentInput.Bounds().Dy()
		outW := inW * 2
		outH := inH * 2

		currentOutput = ebiten.NewImage(outW, outH)
		x.runShaderPass(currentInput, currentOutput)

		// Deallocate previous input (except the original normalized source on first pass)
		if pass > 0 {
			currentInput.Deallocate()
		}
		currentInput = currentOutput
	}

	// Scale final xBR output to screen with centering
	screenBuffer := x.scaleToScreen(currentOutput, screenW, screenH)

	// Clean up
	currentOutput.Deallocate()
	normalizedSrc.Deallocate()

	return screenBuffer
}

// ensureShader compiles and caches the xBR shader
func (x *XBRScaler) ensureShader() error {
	if x.shader != nil {
		return nil
	}
	shader, err := ebiten.NewShader(xbrShaderSrc)
	if err != nil {
		return err
	}
	x.shader = shader
	return nil
}

// selectOptimalScale chooses 2, 4, or 8 based on how much scaling is needed to fit screen
func selectOptimalScale(srcW, srcH, screenW, screenH int) int {
	// Calculate aspect-ratio-preserving scale factor to fit screen
	scaleX := float64(screenW) / float64(srcW)
	scaleY := float64(screenH) / float64(srcH)
	scaleToFit := scaleX
	if scaleY < scaleX {
		scaleToFit = scaleY
	}

	// Choose smallest xBR scale that covers the target (prefer downscaling xBR output)
	if scaleToFit <= 2.0 {
		return 2
	} else if scaleToFit <= 4.0 {
		return 4
	}
	return 8
}

// scaleFactorToPasses converts scale factor to number of 2x passes
func scaleFactorToPasses(factor int) int {
	switch factor {
	case 4:
		return 2
	case 8:
		return 3
	default:
		return 1
	}
}

// runShaderPass executes one 2x xBR pass from input to output
func (x *XBRScaler) runShaderPass(input, output *ebiten.Image) {
	inW := input.Bounds().Dx()
	inH := input.Bounds().Dy()
	outW := output.Bounds().Dx()
	outH := output.Bounds().Dy()

	vertices := []ebiten.Vertex{
		{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(outW), DstY: 0, SrcX: float32(inW), SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 0, DstY: float32(outH), SrcX: 0, SrcY: float32(inH), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(outW), DstY: float32(outH), SrcX: float32(inW), SrcY: float32(inH), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	indices := []uint16{0, 1, 2, 1, 3, 2}

	op := &ebiten.DrawTrianglesShaderOptions{}
	op.Images[0] = input

	output.DrawTrianglesShader(vertices, indices, x.shader, op)
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
