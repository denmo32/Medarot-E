package ui

import (
	"image/color"

	"medarot-ebiten/data"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// PlaceholderScene は「未実装」などを表示するための汎用シーンです
type PlaceholderScene struct {
	resources *data.SharedResources
	manager   *SceneManager // bamennのシーンマネージャ
	ui        *ebitenui.UI
}

// NewPlaceholderScene は新しいプレースホルダーシーンを作成します
func NewPlaceholderScene(res *data.SharedResources, manager *SceneManager, message string) *PlaceholderScene {
	p := &PlaceholderScene{
		resources: res,
		manager:   manager,
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
		widget.TextOpts.Text("クリックしてタイトルに戻る", res.Font, res.Config.UI.Colors.Gray),
	)
	panel.AddChild(subText)

	p.ui = &ebitenui.UI{Container: rootContainer}
	return p
}

func (p *PlaceholderScene) Update() error {
	p.ui.Update()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		p.manager.GoToTitleScene() // マネージャ経由で遷移
	}
	return nil
}

func (p *PlaceholderScene) Draw(screen *ebiten.Image) {
	screen.Fill(p.resources.Config.UI.Colors.Background)
	p.ui.Draw(screen)
}

func (p *PlaceholderScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return p.resources.Config.UI.Screen.Width, p.resources.Config.UI.Screen.Height
}
