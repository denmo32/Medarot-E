package main

import (
	"image/color"
	"math"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// PanelOptions は、汎用パネルを作成するための設定を保持します。
type PanelOptions struct {
	PanelWidth      int
	PanelHeight     int
	Title           string
	Padding         widget.Insets
	Spacing         int
	BackgroundColor color.Color      // 新しく追加
	BackgroundImage *image.NineSlice // これはカスタム画像用
	TitleColor      color.Color
	TitleFont       text.Face
	BorderColor     color.Color // 枠線の色
	BorderThickness float32     // 枠線の太さ
}

// NewPanel は、指定されたオプションに基づいて汎用的なパネルウィジェットを作成します。
func NewPanel(opts *PanelOptions, children ...widget.PreferredSizeLocateableWidget) *widget.Container {
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
		bg = createCyberpunkPanelNineSlice(opts.BorderThickness)
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

		titleFont := GlobalGameDataManager.Font
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

// createCyberpunkPanelNineSlice は、サイバーパンク風のパネル用NineSlice画像を生成します。
// グラデーション背景と立体的な枠線が特徴です。
func createCyberpunkPanelNineSlice(thickness float32) *image.NineSlice {
	tileSize := 64
	borderInset := int(thickness)

	img := ebiten.NewImage(tileSize, tileSize)

	// サイバーパンク風のグラデーション背景を描画
	startColor := color.RGBA{R: 0, G: 20, B: 40, A: 255}
	endColor := color.RGBA{R: 20, G: 40, B: 80, A: 255}
	drawGradient(img, startColor, endColor)

	// 枠線の色を定義
	borderColor := color.RGBA{R: 0, G: 191, B: 255, A: 255} // ディープスカイブルー
	highlightColor, shadowColor := createHighlightAndShadowColors(borderColor)

	// 上辺と左辺にハイライト
	vector.StrokeLine(img, 0, 0, float32(tileSize), 0, thickness, highlightColor, false) // Top
	vector.StrokeLine(img, 0, 0, 0, float32(tileSize), thickness, highlightColor, false) // Left

	// 下辺と右辺にシャドウ
	vector.StrokeLine(img, 0, float32(tileSize), float32(tileSize), float32(tileSize), thickness, shadowColor, false) // Bottom
	vector.StrokeLine(img, float32(tileSize), 0, float32(tileSize), float32(tileSize), thickness, shadowColor, false) // Right

	return image.NewNineSlice(img,
		[3]int{borderInset, tileSize - 2*borderInset, borderInset},
		[3]int{borderInset, tileSize - 2*borderInset, borderInset})
}

// drawGradient は、指定された画像に線形グラデーションを描画します。
func drawGradient(img *ebiten.Image, startColor, endColor color.Color) {
	size := img.Bounds().Size()
	sr, sg, sb, sa := startColor.RGBA()
	er, eg, eb, ea := endColor.RGBA()

	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			ratio := float64(y) / float64(size.Y-1)
			r := lerp(float64(sr), float64(er), ratio)
			g := lerp(float64(sg), float64(eg), ratio)
			b := lerp(float64(sb), float64(eb), ratio)
			a := lerp(float64(sa), float64(ea), ratio)
			img.Set(x, y, color.RGBA64{uint16(r), uint16(g), uint16(b), uint16(a)})
		}
	}
}

// lerp は線形補間を行います。
func lerp(start, end, ratio float64) float64 {
	return start*(1-ratio) + end*ratio
}

// createHighlightAndShadowColors は、ベースカラーから明るい色と暗い色を生成します。
func createHighlightAndShadowColors(baseColor color.Color) (highlight color.Color, shadow color.Color) {
	r, g, b, a := baseColor.RGBA()

	// ハイライト色 (明るくする)
	hr := uint16(math.Min(0xffff, float64(r)*1.5))
	hg := uint16(math.Min(0xffff, float64(g)*1.5))
	hb := uint16(math.Min(0xffff, float64(b)*1.5))
	highlight = color.RGBA64{hr, hg, hb, uint16(a)}

	// シャドウ色 (暗くする)
	sr := uint16(float64(r) * 0.5)
	sg := uint16(float64(g) * 0.5)
	sb := uint16(float64(b) * 0.5)
	shadow = color.RGBA64{sr, sg, sb, uint16(a)}

	return highlight, shadow
}

// createCyberpunkButtonImageSet は、サイバーパンク風のボタン画像セットを生成します。
func createCyberpunkButtonImageSet(thickness float32) *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:    createCyberpunkButtonNineSlice(color.RGBA{0, 20, 40, 255}, color.RGBA{20, 40, 80, 255}, color.RGBA{0, 191, 255, 255}, thickness),
		Hover:   createCyberpunkButtonNineSlice(color.RGBA{10, 30, 50, 255}, color.RGBA{30, 50, 90, 255}, color.RGBA{0, 221, 255, 255}, thickness),
		Pressed: createCyberpunkButtonNineSlice(color.RGBA{20, 40, 60, 255}, color.RGBA{40, 60, 100, 255}, color.RGBA{0, 255, 255, 255}, thickness),
	}
}

// createCyberpunkButtonNineSlice は、ボタン用のNineSlice画像を生成します。
func createCyberpunkButtonNineSlice(startColor, endColor, borderColor color.Color, thickness float32) *image.NineSlice {
	tileSize := 64
	borderInset := int(thickness)

	img := ebiten.NewImage(tileSize, tileSize)
	drawGradient(img, startColor, endColor)

	highlightColor, shadowColor := createHighlightAndShadowColors(borderColor)

	vector.StrokeLine(img, 0, 0, float32(tileSize), 0, thickness, highlightColor, false)
	vector.StrokeLine(img, 0, 0, 0, float32(tileSize), thickness, highlightColor, false)
	vector.StrokeLine(img, 0, float32(tileSize), float32(tileSize), float32(tileSize), thickness, shadowColor, false)
	vector.StrokeLine(img, float32(tileSize), 0, float32(tileSize), float32(tileSize), thickness, shadowColor, false)

	return image.NewNineSlice(img,
		[3]int{borderInset, tileSize - 2*borderInset, borderInset},
		[3]int{borderInset, tileSize - 2*borderInset, borderInset})
}
