//go:build !libretro

package settings

import (
	"github.com/ebitenui/ebitenui/widget"
	"github.com/user-none/emkiii/ui/shader"
	"github.com/user-none/emkiii/ui/storage"
	"github.com/user-none/emkiii/ui/style"
	"github.com/user-none/emkiii/ui/types"
)

// VideoSection manages video settings including shaders
type VideoSection struct {
	callback types.ScreenCallback
	config   *storage.Config
}

// NewVideoSection creates a new video section
func NewVideoSection(callback types.ScreenCallback, config *storage.Config) *VideoSection {
	return &VideoSection{
		callback: callback,
		config:   config,
	}
}

// SetConfig updates the config reference
func (v *VideoSection) SetConfig(config *storage.Config) {
	v.config = config
}

// Build creates the video section UI
func (v *VideoSection) Build(focus types.FocusManager) *widget.Container {
	// Use GridLayout so content can stretch
	section := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			// Row stretch: cropBorder=no, shaderLabel=no, shaderList=YES
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, false, true}),
			widget.GridLayoutOpts.Spacing(0, style.DefaultSpacing),
		)),
	)

	// Crop Border toggle
	section.AddChild(v.buildCropBorderRow(focus))

	// Shaders label
	shadersLabel := widget.NewText(
		widget.TextOpts.Text("Shader Effects", style.FontFace(), style.Text),
	)
	section.AddChild(shadersLabel)

	// Shaders list in scrollable container
	section.AddChild(v.buildShadersList(focus))

	return section
}

// buildCropBorderRow creates the crop border toggle row
func (v *VideoSection) buildCropBorderRow(focus types.FocusManager) *widget.Container {
	row := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(style.DefaultSpacing),
		)),
	)

	label := widget.NewText(
		widget.TextOpts.Text("Crop Border", style.FontFace(), style.Text),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)
	row.AddChild(label)

	toggleBtn := widget.NewButton(
		widget.ButtonOpts.Image(style.ActiveButtonImage(v.config.Video.CropBorder)),
		widget.ButtonOpts.Text(boolToOnOff(v.config.Video.CropBorder), style.FontFace(), style.ButtonTextColor()),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingSmall)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			v.config.Video.CropBorder = !v.config.Video.CropBorder
			storage.SaveConfig(v.config)
			focus.SetPendingFocus("crop-border")
			v.callback.RequestRebuild()
		}),
	)
	focus.RegisterFocusButton("crop-border", toggleBtn)
	row.AddChild(toggleBtn)

	return row
}

// buildShadersList creates the scrollable shaders list
func (v *VideoSection) buildShadersList(focus types.FocusManager) widget.PreferredSizeLocateableWidget {
	listContent := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(style.SmallSpacing),
		)),
	)

	for _, shaderInfo := range shader.AvailableShaders {
		listContent.AddChild(v.buildShaderRow(shaderInfo, focus))
	}

	// Wrap in scrollable container
	scrollContainer, vSlider, scrollWrapper := style.ScrollableContainer(style.ScrollableOpts{
		Content:     listContent,
		BgColor:     style.Background,
		BorderColor: style.Border,
		Spacing:     0,
		Padding:     style.SmallSpacing,
	})
	focus.SetScrollWidgets(scrollContainer, vSlider)
	focus.RestoreScrollPosition()

	return scrollWrapper
}

// buildShaderRow creates a row for a single shader with UI and Game toggle buttons
func (v *VideoSection) buildShaderRow(info shader.ShaderInfo, focus types.FocusManager) *widget.Container {
	uiEnabled := v.isShaderEnabledForUI(info.ID)
	gameEnabled := v.isShaderEnabledForGame(info.ID)

	// Use grid layout: [Info (stretch)] [UI toggle] [Game toggle]
	row := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{true, false, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(style.DefaultSpacing, 0),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
	)

	// Info column
	infoContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(style.TinySpacing),
		)),
	)

	nameLabel := widget.NewText(
		widget.TextOpts.Text(info.Name, style.FontFace(), style.Text),
	)
	infoContainer.AddChild(nameLabel)

	descLabel := widget.NewText(
		widget.TextOpts.Text(info.Description, style.FontFace(), style.TextSecondary),
	)
	infoContainer.AddChild(descLabel)

	row.AddChild(infoContainer)

	// UI toggle button
	uiBtn := widget.NewButton(
		widget.ButtonOpts.Image(style.ActiveButtonImage(uiEnabled)),
		widget.ButtonOpts.Text("UI", style.FontFace(), style.ButtonTextColor()),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingSmall)),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				VerticalPosition: widget.GridLayoutPositionCenter,
			}),
		),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			v.toggleShaderUI(info.ID)
			storage.SaveConfig(v.config)
			focus.SetPendingFocus("shader-ui-" + info.ID)
			v.callback.RequestRebuild()
		}),
	)
	focus.RegisterFocusButton("shader-ui-"+info.ID, uiBtn)
	row.AddChild(uiBtn)

	// Game toggle button
	gameBtn := widget.NewButton(
		widget.ButtonOpts.Image(style.ActiveButtonImage(gameEnabled)),
		widget.ButtonOpts.Text("Game", style.FontFace(), style.ButtonTextColor()),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(style.ButtonPaddingSmall)),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				VerticalPosition: widget.GridLayoutPositionCenter,
			}),
		),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			v.toggleShaderGame(info.ID)
			storage.SaveConfig(v.config)
			focus.SetPendingFocus("shader-game-" + info.ID)
			v.callback.RequestRebuild()
		}),
	)
	focus.RegisterFocusButton("shader-game-"+info.ID, gameBtn)
	row.AddChild(gameBtn)

	return row
}

// isShaderEnabledForUI checks if a shader is enabled for UI context
func (v *VideoSection) isShaderEnabledForUI(id string) bool {
	for _, s := range v.config.Shaders.UIShaders {
		if s == id {
			return true
		}
	}
	return false
}

// isShaderEnabledForGame checks if a shader is enabled for Game context
func (v *VideoSection) isShaderEnabledForGame(id string) bool {
	for _, s := range v.config.Shaders.GameShaders {
		if s == id {
			return true
		}
	}
	return false
}

// toggleShaderUI adds or removes a shader from the UI list
func (v *VideoSection) toggleShaderUI(id string) {
	if v.isShaderEnabledForUI(id) {
		v.config.Shaders.UIShaders = removeFromSlice(v.config.Shaders.UIShaders, id)
	} else {
		v.config.Shaders.UIShaders = append(v.config.Shaders.UIShaders, id)
	}
}

// toggleShaderGame adds or removes a shader from the Game list
func (v *VideoSection) toggleShaderGame(id string) {
	if v.isShaderEnabledForGame(id) {
		v.config.Shaders.GameShaders = removeFromSlice(v.config.Shaders.GameShaders, id)
	} else {
		v.config.Shaders.GameShaders = append(v.config.Shaders.GameShaders, id)
	}
}

// removeFromSlice removes all occurrences of value from slice
func removeFromSlice(slice []string, value string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != value {
			result = append(result, s)
		}
	}
	return result
}

// boolToOnOff converts a boolean to "On" or "Off" string
func boolToOnOff(b bool) string {
	if b {
		return "On"
	}
	return "Off"
}
