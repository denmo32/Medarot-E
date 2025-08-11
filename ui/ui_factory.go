package ui

import (
	"medarot-ebiten/data"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// UIFactory はUIコンポーネントの生成とスタイリングを一元的に管理します。
type UIFactory struct {
	Config            *data.Config
	Font              text.Face // 通常のフォント
	ModalButtonFont   text.Face // モーダルボタン用フォント
	MessageWindowFont text.Face // メッセージウィンドウ用フォント
	MessageManager    *data.MessageManager
	imageGenerator    *UIImageGenerator
}

// NewUIFactory は新しいUIFactoryのインスタンスを作成します。
func NewUIFactory(config *data.Config, font text.Face, modalButtonFont text.Face, messageWindowFont text.Face, messageManager *data.MessageManager) *UIFactory {
	return &UIFactory{
		Config:            config,
		Font:              font,
		ModalButtonFont:   modalButtonFont,
		MessageWindowFont: messageWindowFont,
		MessageManager:    messageManager,
		imageGenerator:    NewUIImageGenerator(config),
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
		widget.ButtonOpts.Text(text, f.ModalButtonFont, buttonTextColor),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
		widget.ButtonOpts.ClickedHandler(clickedHandler),
		widget.ButtonOpts.CursorEnteredHandler(cursorEnteredHandler),
		widget.ButtonOpts.CursorExitedHandler(cursorExitedHandler),
	)
}
