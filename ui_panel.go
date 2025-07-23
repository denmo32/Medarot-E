package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// NewPanel は、指定されたオプションに基づいて汎用的なパネルウィジェットを作成します。
func NewPanel(opts *PanelOptions, imageGenerator *UIImageGenerator, font text.Face, children ...widget.PreferredSizeLocateableWidget) *widget.Container {
	var bg *image.NineSlice
	if opts.BackgroundImage != nil {
		bg = opts.BackgroundImage
	} else {
		// デフォルトの背景色
		if opts.BackgroundColor == nil {
			opts.BackgroundColor = color.NRGBA{50, 50, 70, 200}
		}
		bg = image.NewNineSliceColor(opts.BackgroundColor)
	}

	// 枠線が指定されている場合、サイバーパンク風のパネル画像を生成
	if opts.BorderThickness > 0 {
		bg = imageGenerator.createCyberpunkPanelNineSlice(opts.BorderThickness)
	}

	widgetOpts := []widget.WidgetOpt{
		widget.WidgetOpts.MinSize(opts.PanelWidth, opts.PanelHeight),
	}

	panelContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(bg),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(opts.Padding),
			widget.RowLayoutOpts.Spacing(opts.Spacing),
		)),
		widget.ContainerOpts.WidgetOpts(widgetOpts...),
	)

	if opts.Title != "" {
		var titleColor color.Color = color.White
		if opts.TitleColor != nil {
			titleColor = opts.TitleColor
		}

		titleFont := font // uiFactory.Font の代わりに直接 font を使用
		if opts.TitleFont != nil {
			titleFont = opts.TitleFont
		}

		title := widget.NewText(
			widget.TextOpts.Text(opts.Title, titleFont, titleColor),
		)
		panelContainer.AddChild(title)
	}

	contentContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(opts.Spacing),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Stretch: true,
		})),
	)
	panelContainer.AddChild(contentContainer)

	for _, child := range children {
		contentContainer.AddChild(child)
	}

	return panelContainer
}
