package scene

import (
	"image/color"
	"medarot-ebiten/internal/game"
	"medarot-ebiten/internal/ui"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type SharedResources struct {
	GameData        *game.GameData
	Config          game.Config
	UIConfig        ui.UIConfig
	Font            text.Face
	GameDataManager *game.GameDataManager // 追加
	ButtonImage     *widget.ButtonImage
}

type SceneManagerChanger interface {
	GoToTitleScene()
	GoToBattleScene()
	GoToCustomizeScene()
	GoToTestUIScene()
}

// TitleScene はタイトル画面のシーンです
type TitleScene struct {
	resources *SharedResources
	manager   SceneManagerChanger // シーンマネージャへの参照
	ui        *ebitenui.UI
}

// NewTitleScene は新しいタイトルシーンを作成します
func NewTitleScene(res *SharedResources, manager SceneManagerChanger) *TitleScene {
	t := &TitleScene{
		resources: res,
		manager:   manager, // マネージャを保持
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
		Idle:    image.NewNineSliceColor(res.UIConfig.Colors.Gray),
		Hover:   image.NewNineSliceColor(color.RGBA{180, 180, 180, 255}),
		Pressed: image.NewNineSliceColor(color.RGBA{100, 100, 100, 255}),
	}
	buttonTextColor := &widget.ButtonTextColor{Idle: color.White}

	battleButton := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Battle", res.Font, buttonTextColor),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(10)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			// マネージャ経由でシーン遷移を依頼
			t.manager.GoToBattleScene()
		}),
	)
	panel.AddChild(battleButton)

	customizeButton := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Customize", res.Font, buttonTextColor),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(10)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			// マネージャ経由でシーン遷移を依頼
			t.manager.GoToCustomizeScene()
		}),
	)
	panel.AddChild(customizeButton)

	testUIButton := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Test UI", res.Font, buttonTextColor),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(10)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			t.manager.GoToTestUIScene()
		}),
	)
	panel.AddChild(testUIButton)

	t.ui = &ebitenui.UI{Container: rootContainer}
	return t
}

// Update はUIの状態を更新します。bamennに準拠し、errorのみを返します。
func (t *TitleScene) Update() error {
	t.ui.Update()
	return nil
}

// Draw はUIを描画します
func (t *TitleScene) Draw(screen *ebiten.Image) {
	screen.Fill(t.resources.UIConfig.Colors.Background)
	t.ui.Draw(screen)
}

// Layout はEbitenのレイアウト計算を行います。bamennのシーンとして必須です。
func (t *TitleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return t.resources.UIConfig.Screen.Width, t.resources.UIConfig.Screen.Height
}
