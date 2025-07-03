package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

func createMessageWindow(bs *BattleScene) widget.PreferredSizeLocateableWidget {
	c := bs.resources.Config.UI

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(20)),
		)),
	)

	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{20, 20, 30, 220})), // 半透明の背景
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(15)), // パネル内のパディング
			widget.RowLayoutOpts.Spacing(10),                         // メッセージと「クリックして続行」の間のスペース
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter, // 水平方向中央
			VerticalPosition:   widget.AnchorLayoutPositionEnd,    // 垂直方向下部
			StretchVertical:    false,                             // 垂直方向には引き伸ばさない
		})),
	)
	root.AddChild(panel)

	panel.AddChild(widget.NewText(
		widget.TextOpts.Text(bs.message, bs.resources.Font, c.Colors.White), // メッセージ本文
	))
	panel.AddChild(widget.NewText(
		widget.TextOpts.Text("クリックして続行...", bs.resources.Font, c.Colors.Gray),    // 続行を促すテキスト
		widget.TextOpts.Position(widget.TextPositionEnd, widget.TextPositionEnd), // テキストを右下に配置
	))

	return root
}
