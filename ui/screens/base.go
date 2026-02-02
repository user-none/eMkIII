//go:build !libretro

package screens

import (
	"github.com/ebitenui/ebitenui/widget"
)

// BaseScreen provides common scroll and focus management for screens.
// Embed this in screen structs to get scroll position preservation
// and focus restoration after rebuilds.
type BaseScreen struct {
	// Scroll container and slider for scroll position preservation
	scrollContainer *widget.ScrollContainer
	vSlider         *widget.Slider
	scrollTop       float64

	// Button references for focus restoration (maps key to button)
	focusButtons map[string]*widget.Button

	// Key of button to restore focus to after rebuild
	pendingFocus string
}

// InitBase initializes the base screen state.
// Call this in the screen's constructor.
func (b *BaseScreen) InitBase() {
	b.focusButtons = make(map[string]*widget.Button)
}

// SetScrollWidgets stores references to the scroll widgets for position preservation.
// Call this during Build() after creating the scroll container.
func (b *BaseScreen) SetScrollWidgets(scrollContainer *widget.ScrollContainer, vSlider *widget.Slider) {
	b.scrollContainer = scrollContainer
	b.vSlider = vSlider
}

// SaveScrollPosition saves the current scroll position.
// Call this before rebuilding the screen.
func (b *BaseScreen) SaveScrollPosition() {
	if b.scrollContainer != nil {
		b.scrollTop = b.scrollContainer.ScrollTop
	}
}

// RestoreScrollPosition restores the saved scroll position.
// Call this after rebuilding the screen, once the scroll container is set.
func (b *BaseScreen) RestoreScrollPosition() {
	if b.scrollContainer != nil && b.scrollTop > 0 {
		b.scrollContainer.ScrollTop = b.scrollTop
		if b.vSlider != nil {
			b.vSlider.Current = int(b.scrollTop * 1000)
		}
	}
}

// RegisterFocusButton registers a button for focus restoration.
// Call this during Build() for each focusable button.
func (b *BaseScreen) RegisterFocusButton(key string, btn *widget.Button) {
	if b.focusButtons == nil {
		b.focusButtons = make(map[string]*widget.Button)
	}
	b.focusButtons[key] = btn
}

// ClearFocusButtons clears all registered focus buttons.
// Call this at the start of Build() before registering new buttons.
func (b *BaseScreen) ClearFocusButtons() {
	b.focusButtons = make(map[string]*widget.Button)
}

// SetPendingFocus sets the key of the button to focus after rebuild.
func (b *BaseScreen) SetPendingFocus(key string) {
	b.pendingFocus = key
}

// SetDefaultFocus sets the pending focus only if no focus is currently pending.
// Use this in OnEnter() to set initial focus without overriding restored focus.
func (b *BaseScreen) SetDefaultFocus(key string) {
	if b.pendingFocus == "" {
		b.pendingFocus = key
	}
}

// GetPendingFocusButton returns the button that should receive focus after rebuild.
// Returns nil if no pending focus or button not found.
func (b *BaseScreen) GetPendingFocusButton() *widget.Button {
	if b.pendingFocus == "" {
		return nil
	}
	return b.focusButtons[b.pendingFocus]
}

// ClearPendingFocus clears the pending focus state.
func (b *BaseScreen) ClearPendingFocus() {
	b.pendingFocus = ""
}

// EnsureFocusedVisible scrolls the view to ensure the focused widget is visible.
// The isScrollableButton function should return true if the focused widget
// should trigger scrolling (e.g., game buttons but not toolbar buttons).
func (b *BaseScreen) EnsureFocusedVisible(focused widget.Focuser, isScrollableButton func(*widget.Button) bool) {
	if focused == nil || b.scrollContainer == nil {
		return
	}

	// Check if this widget should trigger scrolling
	btn, ok := focused.(*widget.Button)
	if !ok {
		return
	}
	if isScrollableButton != nil && !isScrollableButton(btn) {
		return
	}

	// Get the focused widget's rectangle
	focusWidget := focused.GetWidget()
	if focusWidget == nil {
		return
	}
	focusRect := focusWidget.Rect

	// Get the scroll container's view rect (visible area on screen)
	viewRect := b.scrollContainer.ViewRect()
	contentRect := b.scrollContainer.ContentRect()

	// If content fits in view, no scrolling needed
	if contentRect.Dy() <= viewRect.Dy() {
		return
	}

	// Current scroll offset in pixels
	maxScroll := contentRect.Dy() - viewRect.Dy()
	scrollOffset := int(b.scrollContainer.ScrollTop * float64(maxScroll))

	// Widget's position relative to view top
	widgetTopInView := focusRect.Min.Y - viewRect.Min.Y
	widgetBottomInView := focusRect.Max.Y - viewRect.Min.Y
	viewHeight := viewRect.Dy()

	// Check if widget top is above the visible area
	if widgetTopInView < 0 {
		// Scroll up: align widget top with view top
		newScrollOffset := scrollOffset + widgetTopInView
		if newScrollOffset < 0 {
			newScrollOffset = 0
		}
		b.scrollContainer.ScrollTop = float64(newScrollOffset) / float64(maxScroll)
		if b.vSlider != nil {
			b.vSlider.Current = int(b.scrollContainer.ScrollTop * 1000)
		}
	} else if widgetBottomInView > viewHeight {
		// Scroll down: align widget bottom with view bottom (minimal scroll)
		newScrollOffset := scrollOffset + (widgetBottomInView - viewHeight)
		if newScrollOffset > maxScroll {
			newScrollOffset = maxScroll
		}
		b.scrollContainer.ScrollTop = float64(newScrollOffset) / float64(maxScroll)
		if b.vSlider != nil {
			b.vSlider.Current = int(b.scrollContainer.ScrollTop * 1000)
		}
	}
}
