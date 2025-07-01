package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// PlaceholderScene は「未実装」などを表示するための汎用シーンです
type PlaceholderScene struct {
	resources *SharedResources
	ui        *ebitenui.UI
	nextScene SceneType
}

// NewPlaceholderScene は新しいプレースホルダーシーンを作成します
func NewPlaceholderScene(res *SharedResources, message string) *PlaceholderScene {
	p := &PlaceholderScene{
		resources: res,
		nextScene: SceneTypeCustomize, // 自分自身のシーンタイプを初期値に
	}

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	panel := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(20),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)
	rootContainer.AddChild(panel)

	messageText := widget.NewText(
		widget.TextOpts.Text(message, res.Font, color.White),
	)
	panel.AddChild(messageText)

	subText := widget.NewText(
		// ★★★ 修正点: color.Gray を Config の色に修正 ★★★
		widget.TextOpts.Text("Click to return to Title", res.Font, res.Config.UI.Colors.Gray),
	)
	panel.AddChild(subText)

	p.ui = &ebitenui.UI{Container: rootContainer}
	return p
}

func (p *PlaceholderScene) Update() (SceneType, error) {
	p.ui.Update()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		p.nextScene = SceneTypeTitle
	}
	return p.nextScene, nil
}

func (p *PlaceholderScene) Draw(screen *ebiten.Image) {
	screen.Fill(p.resources.Config.UI.Colors.Background)
	p.ui.Draw(screen)
}
