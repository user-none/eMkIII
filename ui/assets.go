//go:build !libretro

package ui

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/placeholder.png
var placeholderImageData []byte

var placeholderImage *ebiten.Image

// GetPlaceholderImage returns the placeholder image for missing artwork
func GetPlaceholderImage() *ebiten.Image {
	if placeholderImage != nil {
		return placeholderImage
	}

	img, _, err := image.Decode(bytes.NewReader(placeholderImageData))
	if err != nil {
		log.Printf("Failed to decode placeholder image: %v", err)
		// Return a solid color fallback
		fallback := ebiten.NewImage(120, 90)
		fallback.Fill(Theme.Surface)
		return fallback
	}

	placeholderImage = ebiten.NewImageFromImage(img)
	return placeholderImage
}
