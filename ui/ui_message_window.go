package ui

import (
	"github.com/ebitenui/ebitenui/widget"
)

// MessageWindow はゲーム内メッセージを表示するUIコンポーネントです。
type MessageWindow struct {
	widget    widget.PreferredSizeLocateableWidget
	uiFactory *UIFactory
	isVisible bool
}

// NewMessageWindow は新しいMessageWindowのインスタンスを作成します。
func NewMessageWindow(uiFactory *UIFactory) *MessageWindow {
	// 初期状態では空のコンテナを持つ
	container := widget.NewContainer()
	return &MessageWindow{
		widget:    container,
		uiFactory: uiFactory,
		isVisible: false,
	}
}

// Widget はこのコンポーネントのルートウィジェットを返します。
func (m *MessageWindow) Widget() widget.PreferredSizeLocateableWidget {
	return m.widget
}

// IsVisible はウィンドウが表示されているかどうかを返します。
func (m *MessageWindow) IsVisible() bool {
	return m.isVisible
}

// SetMessage は表示するメッセージをセットし、ウィンドウを表示状態にします。
func (m *MessageWindow) SetMessage(message string) {
	m.widget = m.createUI(message)
	m.isVisible = true
}

// Hide はウィンドウを非表示にし、内容をクリアします。
func (m *MessageWindow) Hide() {
	m.widget = widget.NewContainer()
	m.isVisible = false
}

// createUI はメッセージから実際のUIウィジェットを構築します。
func (m *MessageWindow) createUI(message string) widget.PreferredSizeLocateableWidget {
	c := m.uiFactory.Config.UI

	// コンテンツを格納するコンテナを作成
	contentContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(15)), // パディングはここで設定
			widget.RowLayoutOpts.Spacing(10),
		)),
	)

	// テキストウィジェットを作成
	messageTextWidget := widget.NewText(
		widget.TextOpts.Text(message, m.uiFactory.MessageWindowFont, c.Colors.White),
	)
	contentContainer.AddChild(messageTextWidget)

	continueTextStr := "クリックして続行..."
	if m.uiFactory.MessageManager != nil {
		continueTextStr = m.uiFactory.MessageManager.FormatMessage("ui_click_to_continue", nil)
	}
	continueTextWidget := widget.NewText(
		widget.TextOpts.Text(continueTextStr, m.uiFactory.MessageWindowFont, c.Colors.Gray),
		widget.TextOpts.Position(widget.TextPositionEnd, widget.TextPositionEnd),
	)
	contentContainer.AddChild(continueTextWidget)

	return contentContainer
}