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

// UIFactory はUIコンポーネントの生成とスタイリングを一元的に管理します。
type UIFactory struct {
	Config          *Config
	Font            text.Face
	MessageManager  *MessageManager
	GameDataManager *GameDataManager // 追加
}

// NewUIFactory は新しいUIFactoryのインスタンスを作成します。
func NewUIFactory(config *Config, font text.Face, messageManager *MessageManager, gameDataManager *GameDataManager) *UIFactory {
	return &UIFactory{
		Config:          config,
		Font:            font,
		MessageManager:  messageManager,
		GameDataManager: gameDataManager, // 追加
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
	buttonImage := f.createCyberpunkButtonImageSet(5) // thicknessは固定値で良いか、Configから取得するか検討

	if buttonTextColor == nil {
		buttonTextColor = &widget.ButtonTextColor{Idle: f.Config.UI.Colors.White}
	}

	return widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text(text, f.Font, buttonTextColor),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
		widget.ButtonOpts.ClickedHandler(clickedHandler),
		widget.ButtonOpts.CursorEnteredHandler(cursorEnteredHandler),
		widget.ButtonOpts.CursorExitedHandler(cursorExitedHandler),
	)
}

// createCyberpunkButtonImageSet は、サイバーパンク風のボタン画像セットを生成します。
func (f *UIFactory) createCyberpunkButtonImageSet(thickness float32) *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:    f.createCyberpunkButtonNineSlice(color.RGBA{0, 20, 40, 255}, color.RGBA{20, 40, 80, 255}, color.RGBA{0, 191, 255, 255}, thickness),
		Hover:   f.createCyberpunkButtonNineSlice(color.RGBA{10, 30, 50, 255}, color.RGBA{30, 50, 90, 255}, color.RGBA{0, 221, 255, 255}, thickness),
		Pressed: f.createCyberpunkButtonNineSlice(color.RGBA{20, 40, 60, 255}, color.RGBA{40, 60, 100, 255}, color.RGBA{0, 255, 255, 255}, thickness),
	}
}

// createCyberpunkButtonNineSlice は、ボタン用のNineSlice画像を生成します。
func (f *UIFactory) createCyberpunkButtonNineSlice(startColor, endColor, borderColor color.Color, thickness float32) *image.NineSlice {
	tileSize := 64
	borderInset := int(thickness)

	img := ebiten.NewImage(tileSize, tileSize)
	f.drawGradient(img, startColor, endColor)

	highlightColor, shadowColor := f.createHighlightAndShadowColors(borderColor)

	vector.StrokeLine(img, 0, 0, float32(tileSize), 0, thickness, highlightColor, false)
	vector.StrokeLine(img, 0, 0, 0, float32(tileSize), thickness, highlightColor, false)
	vector.StrokeLine(img, 0, float32(tileSize), float32(tileSize), float32(tileSize), thickness, shadowColor, false)
	vector.StrokeLine(img, float32(tileSize), 0, float32(tileSize), float32(tileSize), thickness, shadowColor, false)

	return image.NewNineSlice(img,
		[3]int{borderInset, tileSize - 2*borderInset, borderInset},
		[3]int{borderInset, tileSize - 2*borderInset, borderInset})
}

// drawGradient は、指定された画像に線形グラデーションを描画します。
func (f *UIFactory) drawGradient(img *ebiten.Image, startColor, endColor color.Color) {
	size := img.Bounds().Size()
	sr, sg, sb, sa := startColor.RGBA()
	er, eg, eb, ea := endColor.RGBA()

	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			ratio := float64(y) / float64(size.Y-1)
			r := f.lerp(float64(sr), float64(er), ratio)
			g := f.lerp(float64(sg), float64(eg), ratio)
			b := f.lerp(float64(sb), float64(eb), ratio)
			a := f.lerp(float64(sa), float64(ea), ratio)
			img.Set(x, y, color.RGBA64{uint16(r), uint16(g), uint16(b), uint16(a)})
		}
	}
}

// lerp は線形補間を行います。
func (f *UIFactory) lerp(start, end, ratio float64) float64 {
	return start*(1-ratio) + end*ratio
}

// createHighlightAndShadowColors は、ベースカラーから明るい色と暗い色を生成します。
func (f *UIFactory) createHighlightAndShadowColors(baseColor color.Color) (highlight color.Color, shadow color.Color) {
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

// createCyberpunkPanelNineSlice は、サイバーパンク風のパネル用NineSlice画像を生成します。
// グラデーション背景と立体的な枠線が特徴です。
func (f *UIFactory) createCyberpunkPanelNineSlice(thickness float32) *image.NineSlice {
	tileSize := 64
	borderInset := int(thickness)

	img := ebiten.NewImage(tileSize, tileSize)

	// サイバーパンク風のグラデーション背景を描画
	startColor := color.RGBA{R: 0, G: 20, B: 40, A: 255}
	endColor := color.RGBA{R: 20, G: 40, B: 80, A: 255}
	f.drawGradient(img, startColor, endColor)

	// 枠線の色を定義
	borderColor := color.RGBA{R: 0, G: 191, B: 255, A: 255} // ディープスカイブルー
	highlightColor, shadowColor := f.createHighlightAndShadowColors(borderColor)

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
