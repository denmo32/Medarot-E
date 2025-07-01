package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// TitleScene はタイトル画面のシーンです
type TitleScene struct {
	resources *SharedResources
	ui        *ebitenui.UI
	nextScene SceneType
}

// NewTitleScene は新しいタイトルシーンを作成します
func NewTitleScene(res *SharedResources) *TitleScene {
	t := &TitleScene{
		resources: res,
		nextScene: SceneTypeTitle,
	}

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	panel := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(20),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(50)),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)
	rootContainer.AddChild(panel)

	titleText := widget.NewText(
		widget.TextOpts.Text("Medarot E", res.Font, color.White),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
	)
	panel.AddChild(titleText)

	buttonImage := &widget.ButtonImage{
		Idle:    image.NewNineSliceColor(res.Config.UI.Colors.Gray),
		Hover:   image.NewNineSliceColor(color.RGBA{180, 180, 180, 255}),
		Pressed: image.NewNineSliceColor(color.RGBA{100, 100, 100, 255}),
	}
	buttonTextColor := &widget.ButtonTextColor{Idle: color.White}

	battleButton := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Battle", res.Font, buttonTextColor),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(10)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			t.nextScene = SceneTypeBattle
		}),
	)
	panel.AddChild(battleButton)

	customizeButton := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Customize", res.Font, buttonTextColor),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(10)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			// ★★★ この行を修正 ★★★
			t.nextScene = SceneTypeCustomize
		}),
	)
	panel.AddChild(customizeButton)

	t.ui = &ebitenui.UI{Container: rootContainer}
	return t
}

func (t *TitleScene) Update() (SceneType, error) {
	t.ui.Update()
	return t.nextScene, nil
}

func (t *TitleScene) Draw(screen *ebiten.Image) {
	screen.Fill(t.resources.Config.UI.Colors.Background)
	t.ui.Draw(screen)
}
