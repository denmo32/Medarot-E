package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
)

func createMessageWindow(message string, uiFactory *UIFactory) widget.PreferredSizeLocateableWidget {
	c := uiFactory.Config.UI

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(20)),
		)),
	)

	// テキストウィジェットを作成
	messageTextWidget := widget.NewText(
		widget.TextOpts.Text(message, uiFactory.Font, c.Colors.White),
	)

	continueTextStr := "クリックして続行..."
	if uiFactory.MessageManager != nil {
		continueTextStr = uiFactory.MessageManager.FormatMessage("ui_click_to_continue", nil)
	}
	continueTextWidget := widget.NewText(
		widget.TextOpts.Text(continueTextStr, uiFactory.Font, c.Colors.Gray),
		widget.TextOpts.Position(widget.TextPositionEnd, widget.TextPositionEnd),
	)

	// NewPanel を使用してメッセージウィンドウを作成
	panel := NewPanel(&PanelOptions{
		Padding:         widget.NewInsetsSimple(15),
		Spacing:         10,
		BackgroundColor: color.Transparent, // 背景色を透過
		BorderColor:     color.White,       // 白い枠線
		BorderThickness: 0,                 // 枠線の太さ
	}, uiFactory.imageGenerator, uiFactory.Font, messageTextWidget, continueTextWidget) // uiFactory.imageGeneratorとuiFactory.Fontを渡す

	// パネルを親コンテナ全体に広げる
	panel.RootContainer.GetWidget().LayoutData = widget.AnchorLayoutData{ // RootContainer を使用
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		StretchHorizontal:  true,
		StretchVertical:    true,
	}
	root.AddChild(panel.RootContainer) // RootContainer を追加

	return root
}
