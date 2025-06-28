package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

func createMessageWindow(game *Game) widget.PreferredSizeLocateableWidget {
	c := game.Config.UI

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(20)),
		)),
	)

	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{20, 20, 30, 220})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(15)),
			widget.RowLayoutOpts.Spacing(10),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionEnd,
			StretchVertical:    false,
		})),
	)
	root.AddChild(panel)

	panel.AddChild(widget.NewText(
		widget.TextOpts.Text(game.message, game.MplusFont, c.Colors.White),
	))
	panel.AddChild(widget.NewText(
		widget.TextOpts.Text("クリックして続行...", game.MplusFont, c.Colors.Gray),
		widget.TextOpts.Position(widget.TextPositionEnd, widget.TextPositionEnd),
	))

	return root
}

// [REMOVED] showUIMessage と hideUIMessage を削除
