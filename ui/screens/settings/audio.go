//go:build !libretro

package settings

import (
	"github.com/ebitenui/ebitenui/widget"
	"github.com/user-none/emkiii/ui/storage"
	"github.com/user-none/emkiii/ui/style"
	"github.com/user-none/emkiii/ui/types"
)

// AudioSection manages audio settings
type AudioSection struct {
	callback types.ScreenCallback
	config   *storage.Config
}

// NewAudioSection creates a new audio section
func NewAudioSection(callback types.ScreenCallback, config *storage.Config) *AudioSection {
	return &AudioSection{
		callback: callback,
		config:   config,
	}
}

// SetConfig updates the config reference
func (a *AudioSection) SetConfig(config *storage.Config) {
	a.config = config
}

// Build creates the audio section UI
func (a *AudioSection) Build(focus types.FocusManager) *widget.Container {
	section := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(style.DefaultSpacing),
		)),
	)

	// Mute Game Audio toggle
	section.AddChild(a.buildMuteRow(focus))

	a.setupNavigation(focus)

	return section
}

// setupNavigation registers navigation zones for the audio section
func (a *AudioSection) setupNavigation(focus types.FocusManager) {
	focus.RegisterNavZone("audio-mute", types.NavZoneHorizontal, []string{"audio-mute"}, 0)
}

// buildMuteRow creates the mute toggle row
func (a *AudioSection) buildMuteRow(focus types.FocusManager) *widget.Container {
	row := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(style.DefaultSpacing),
		)),
	)

	label := widget.NewText(
		widget.TextOpts.Text("Mute Game Audio", style.FontFace(), style.Text),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)
	row.AddChild(label)

	toggleBtn := widget.NewButton(
		widget.ButtonOpts.Image(style.ActiveButtonImage(a.config.Audio.Muted)),
		widget.ButtonOpts.Text(boolToOnOff(a.config.Audio.Muted), style.FontFace(), style.ButtonTextColor()),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingSmall)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			a.config.Audio.Muted = !a.config.Audio.Muted
			storage.SaveConfig(a.config)
			focus.SetPendingFocus("audio-mute")
			a.callback.RequestRebuild()
		}),
	)
	focus.RegisterFocusButton("audio-mute", toggleBtn)
	row.AddChild(toggleBtn)

	return row
}
