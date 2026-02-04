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
		ID:          "crt",
		Name:        "CRT",
		Description: "Curved screen with RGB separation and vignette",
	},
	{
		ID:          "scanlines",
		Name:        "Scanlines",
		Description: "Horizontal scanline effect",
	},
}

// GetShaderByID returns the shader info for the given ID, or nil if not found
func GetShaderByID(id string) *ShaderInfo {
	for i := range AvailableShaders {
		if AvailableShaders[i].ID == id {
			return &AvailableShaders[i]
		}
	}
	return nil
}
