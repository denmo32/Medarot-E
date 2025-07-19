package main

import (
	

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// UIFactory はUIコンポーネントの生成とスタイリングを一元的に管理します。
type UIFactory struct {
	Config          *Config
	Font            text.Face
	MessageManager  *MessageManager
	GameDataManager *GameDataManager // 追加
	imageGenerator  *UIImageGenerator // 追加
}

// NewUIFactory は新しいUIFactoryのインスタンスを作成します。
func NewUIFactory(config *Config, font text.Face, messageManager *MessageManager, gameDataManager *GameDataManager) *UIFactory {
	return &UIFactory{
		Config:          config,
		Font:            font,
		MessageManager:  messageManager,
		GameDataManager: gameDataManager, // 追加
		imageGenerator:  NewUIImageGenerator(config),
	}
}

// NewCyberpunkButton はサイバーパンクスタイルのボタンを生成します。
func (f *UIFactory) NewCyberpunkButton(
	text string,
	buttonTextColor *widget.ButtonTextColor,
	clickedHandler func(args *widget.ButtonClickedEventArgs),
	cursorEnteredHandler func(args *widget.ButtonHoverEventArgs),
	cursorExitedHandler func(args *widget.ButtonHoverEventArgs),
) *widget.Button {
	buttonImage := f.imageGenerator.createCyberpunkButtonImageSet(5) // thicknessは固定値で良いか、Configから取得するか検討

	if buttonTextColor == nil {
		buttonTextColor = &widget.ButtonTextColor{Idle: f.Config.UI.Colors.White}
	}

	return widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text(text, f.Font, buttonTextColor),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
		widget.ButtonOpts.ClickedHandler(clickedHandler),
		widget.ButtonOpts.CursorEnteredHandler(cursorEnteredHandler),
		widget.ButtonOpts.CursorExitedHandler(cursorExitedHandler),
	)
}

// createCyberpunkPanelNineSlice は、サイバーパンク風のパネル用NineSlice画像を生成します。
// グラデーション背景と立体的な枠線が特徴です。
func (f *UIFactory) createCyberpunkPanelNineSlice(thickness float32) *image.NineSlice {
	return f.imageGenerator.createCyberpunkPanelNineSlice(thickness)
}