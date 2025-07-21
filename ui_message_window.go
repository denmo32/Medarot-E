package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	// "github.com/hajimehoshi/ebiten/v2/text/v2" // 削除
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
		BackgroundColor: c.Colors.Background, // 不透明な背景色を設定
		BorderColor:     color.White,         // 白い枠線
		BorderThickness: 2,                   // 枠線の太さ
	}, uiFactory.imageGenerator, uiFactory.Font, messageTextWidget, continueTextWidget) // uiFactory.imageGeneratorとuiFactory.Fontを渡す

	// パネルを画面下部中央に配置
	panel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionEnd,
		StretchVertical:    false,
	}
	root.AddChild(panel)

	return root
}