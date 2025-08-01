package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// UIPanel は、汎用的なUIパネルとそのコンテンツコンテナを保持します。
type UIPanel struct {
	RootContainer    *widget.Container
	ContentContainer *widget.Container
	centerContent    bool // コンテンツを中央に配置するかどうか
}

// SetContent は、パネルのコンテンツコンテナの子要素をクリアし、新しいウィジェットを追加します。
// nil を渡すとコンテンツがクリアされます。
func (p *UIPanel) SetContent(w widget.PreferredSizeLocateableWidget) {
	p.ContentContainer.RemoveChildren()
	if w != nil {
		if p.centerContent {
			// 中央配置が有効な場合、AnchorLayoutDataを適用
			w.GetWidget().LayoutData = widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}
		}
		p.ContentContainer.AddChild(w)
	}
}

// PanelOptions は、汎用パネルを作成するための設定を保持します。
type PanelOptions struct {
	PanelWidth      int
	PanelHeight     int
	Title           string
	Padding         widget.Insets
	Spacing         int
	BackgroundColor color.Color
	BackgroundImage *image.NineSlice
	TitleColor      color.Color
	TitleFont       text.Face
	BorderColor     color.Color
	BorderThickness float32
	CenterContent   bool // コンテンツを中央に配置するかどうか
}

// NewPanel は、指定されたオプションに基づいて汎用的なパネルウィジェットを作成します。
func NewPanel(opts *PanelOptions, imageGenerator *UIImageGenerator, font text.Face, children ...widget.PreferredSizeLocateableWidget) *UIPanel {
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

	rootContainer := widget.NewContainer(
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

		titleFont := font
		if opts.TitleFont != nil {
			titleFont = opts.TitleFont
		}

		title := widget.NewText(
			widget.TextOpts.Text(opts.Title, titleFont, titleColor),
		)
		rootContainer.AddChild(title)
	}

	if opts.CenterContent {
		// コンテンツを中央に配置する場合、StackedLayoutを使用
		contentContainer := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewStackedLayout()),
			widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			})),
		)
		rootContainer.AddChild(contentContainer)

		for _, child := range children {
			// 中央配置が有効な場合、AnchorLayoutDataを適用
			child.GetWidget().LayoutData = widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}
			contentContainer.AddChild(child)
		}

		return &UIPanel{
			RootContainer:    rootContainer,
			ContentContainer: contentContainer,
			centerContent:    opts.CenterContent, // 設定を保存
		}
	} else {
		// それ以外の場合はRowLayoutを使用
		contentContainer := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(opts.Spacing),
			)),
			widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			})),
		)
		rootContainer.AddChild(contentContainer)

		for _, child := range children {
			contentContainer.AddChild(child)
		}

		return &UIPanel{
			RootContainer:    rootContainer,
			ContentContainer: contentContainer,
			centerContent:    opts.CenterContent, // 設定を保存
		}
	}
}
