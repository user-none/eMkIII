//go:build !libretro

package shader

// ShaderInfo describes an available shader effect
type ShaderInfo struct {
	ID          string // Unique identifier used in config
	Name        string // Display name for UI
	Description string // Brief description of the effect
}

// AvailableShaders lists all shaders that can be enabled
var AvailableShaders = []ShaderInfo{
	{
		ID:          "ghosting",
		Name:        "Phosphor Persistence",
		Description: "Ghost trails from slow CRT phosphor decay",
	},
	{
		ID:          "crt",
		Name:        "CRT",
		Description: "Curved screen with RGB separation and vignette",
	},
	{
		ID:          "scanlines",
		Name:        "Scanlines",
		Description: "Horizontal scanline effect",
	},
	{
		ID:          "bloom",
		Name:        "Phosphor Glow",
		Description: "Bright pixels glow into neighbors like CRT phosphors",
	},
	{
		ID:          "lcd",
		Name:        "LCD Grid",
		Description: "Visible pixel grid with RGB subpixels like handhelds",
	},
	{
		ID:          "colorbleed",
		Name:        "Color Bleed",
		Description: "Horizontal color bleeding from composite video",
	},
	{
		ID:          "dotmatrix",
		Name:        "Dot Matrix",
		Description: "Circular pixels like CRT phosphor dots",
	},
	{
		ID:          "ntsc",
		Name:        "NTSC Artifacts",
		Description: "Color fringing at edges from NTSC encoding",
	},
	{
		ID:          "gamma",
		Name:        "CRT Gamma",
		Description: "Non-linear brightness curve of CRT displays",
	},
}
