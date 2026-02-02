//go:build !libretro

package screens

import (
	"github.com/user-none/emkiii/ui/types"
)

// Re-export interfaces from types package for backward compatibility
type (
	ScreenCallback = types.ScreenCallback
	FocusRestorer  = types.FocusRestorer
	FocusManager   = types.FocusManager
)
