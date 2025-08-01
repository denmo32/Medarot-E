package main

import (
	"github.com/ebitenui/ebitenui/widget"
)

func createMessageWindow(message string, uiFactory *UIFactory) widget.PreferredSizeLocateableWidget {
	c := uiFactory.Config.UI

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
		widget.TextOpts.Text(message, uiFactory.Font, c.Colors.White),
	)
	contentContainer.AddChild(messageTextWidget)

	continueTextStr := "クリックして続行..."
	if uiFactory.MessageManager != nil {
		continueTextStr = uiFactory.MessageManager.FormatMessage("ui_click_to_continue", nil)
	}
	continueTextWidget := widget.NewText(
		widget.TextOpts.Text(continueTextStr, uiFactory.Font, c.Colors.Gray),
		widget.TextOpts.Position(widget.TextPositionEnd, widget.TextPositionEnd),
	)
	contentContainer.AddChild(continueTextWidget)

	return contentContainer
}
