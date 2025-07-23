package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// TestUIScene はEbitenUIの検証を行うためのシーンです
type TestUIScene struct {
	resources *SharedResources
	manager   *SceneManager
	ui        *ebitenui.UI
}

// NewTestUIScene は新しいTestUISceneを作成します
func NewTestUIScene(res *SharedResources, manager *SceneManager) *TestUIScene {
	t := &TestUIScene{
		resources: res,
		manager:   manager,
	}

	// メインコンテナ（縦に2つの領域を配置）
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{true, true}), // 上部と下部の2行
			widget.GridLayoutOpts.Spacing(10, 10),
		)),
	)

	// 上部コンテナ（3列のパネル用）
	topContainer := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{})),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{true}), // 3カラム
			widget.GridLayoutOpts.Spacing(10, 0),
		)),
	)
	rootContainer.AddChild(topContainer)

	// 上部左のパネル
	topLeftPanel := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{})),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.RGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0x80})), // 赤
	)
	topContainer.AddChild(topLeftPanel)

	// 上部中央のパネル
	topCenterPanel := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{})),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.RGBA{R: 0x00, G: 0xFF, B: 0x00, A: 0x80})), // 緑
	)
	topContainer.AddChild(topCenterPanel)

	// 上部右のパネル
	topRightPanel := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{})),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.RGBA{R: 0x00, G: 0x00, B: 0xFF, A: 0x80})), // 青
	)
	topContainer.AddChild(topRightPanel)

	// 下部の大きなパネル
	bottomPanel := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{})),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.RGBA{R: 0xFF, G: 0xFF, B: 0x00, A: 0x80})), // 黄色
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),                                                         // ボタンの配置のためのレイアウト
	)
	rootContainer.AddChild(bottomPanel)

	// タイトルに戻るボタンをbottomPanelに追加
	button := widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
		widget.ButtonOpts.Image(res.ButtonImage),
		widget.ButtonOpts.Text("タイトルに戻る", res.Font, &widget.ButtonTextColor{
			Idle: color.White,
		}),
		widget.ButtonOpts.TextPadding(widget.Insets{
			Left:  30,
			Right: 30,
		}),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			t.manager.GoToTitleScene()
		}),
	)
	bottomPanel.AddChild(button)

	t.ui = &ebitenui.UI{Container: rootContainer}
	return t
}

func (t *TestUIScene) Update() error {
	t.ui.Update()
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		t.manager.GoToTitleScene()
	}
	return nil
}

func (t *TestUIScene) Draw(screen *ebiten.Image) {
	screen.Fill(t.resources.Config.UI.Colors.Background)
	t.ui.Draw(screen)
}

func (t *TestUIScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return t.resources.Config.UI.Screen.Width, t.resources.Config.UI.Screen.Height
}
