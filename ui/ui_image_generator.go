package ui

import (
	"image/color"
	"math"

	"medarot-ebiten/data"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// UIImageGenerator はUIコンポーネントの画像生成ロジックをカプセル化します。
type UIImageGenerator struct {
	config *data.Config
}

// NewUIImageGenerator は新しいUIImageGeneratorのインスタンスを作成します。
func NewUIImageGenerator(config *data.Config) *UIImageGenerator {
	return &UIImageGenerator{
		config: config,
	}
}

// createCyberpunkButtonImageSet は、サイバーパンク風のボタン画像セットを生成します。
func (g *UIImageGenerator) createCyberpunkButtonImageSet(thickness float32) *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:    g.createCyberpunkButtonNineSlice(color.RGBA{R: 0, G: 60, B: 120, A: 255}, color.RGBA{R: 0, G: 100, B: 200, A: 255}, color.RGBA{R: 0, G: 191, B: 255, A: 255}, thickness),
		Hover:   g.createCyberpunkButtonNineSlice(color.RGBA{R: 0, G: 128, B: 0, A: 255}, color.RGBA{R: 50, G: 205, B: 50, A: 255}, color.RGBA{R: 0, G: 255, B: 0, A: 255}, thickness),
		Pressed: g.createCyberpunkButtonNineSlice(color.RGBA{R: 0, G: 150, B: 0, A: 255}, color.RGBA{R: 100, G: 255, B: 100, A: 255}, color.RGBA{R: 50, G: 255, B: 50, A: 255}, thickness),
	}
}

// createCyberpunkButtonNineSlice は、ボタン用のNineSlice画像を生成します。
func (g *UIImageGenerator) createCyberpunkButtonNineSlice(startColor, endColor, borderColor color.Color, thickness float32) *image.NineSlice {
	tileSize := 64
	borderInset := int(thickness)

	img := ebiten.NewImage(tileSize, tileSize)
	g.drawGradient(img, startColor, endColor)

	highlightColor, shadowColor := g.createHighlightAndShadowColors(borderColor)

	vector.StrokeLine(img, 0, 0, float32(tileSize), 0, thickness, highlightColor, false)
	vector.StrokeLine(img, 0, 0, 0, float32(tileSize), thickness, highlightColor, false)
	vector.StrokeLine(img, 0, float32(tileSize), float32(tileSize), float32(tileSize), thickness, shadowColor, false)
	vector.StrokeLine(img, float32(tileSize), 0, float32(tileSize), float32(tileSize), thickness, shadowColor, false)

	return image.NewNineSlice(img,
		[3]int{borderInset, tileSize - 2*borderInset, borderInset},
		[3]int{borderInset, tileSize - 2*borderInset, borderInset})
}

// drawGradient は、指定された画像に線形グラデーションを描画します。
func (g *UIImageGenerator) drawGradient(img *ebiten.Image, startColor, endColor color.Color) {
	size := img.Bounds().Size()
	sr, sg, sb, sa := startColor.RGBA()
	er, eg, eb, ea := endColor.RGBA()

	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			ratio := float64(y) / float64(size.Y-1)
			r := g.lerp(float64(sr), float64(er), ratio)
			c := g.lerp(float64(sg), float64(eg), ratio)
			b := g.lerp(float64(sb), float64(eb), ratio)
			a := g.lerp(float64(sa), float64(ea), ratio)
			img.Set(x, y, color.RGBA64{uint16(r), uint16(c), uint16(b), uint16(a)})
		}
	}
}

// lerp は線形補間を行います。
func (g *UIImageGenerator) lerp(start, end, ratio float64) float64 {
	return start*(1-ratio) + end*ratio
}

// createHighlightAndShadowColors は、ベースカラーから明るい色と暗い色を生成します。
func (g *UIImageGenerator) createHighlightAndShadowColors(baseColor color.Color) (highlight color.Color, shadow color.Color) {
	r, gVal, b, a := baseColor.RGBA()

	// ハイライト色 (明るくする)
	hr := uint16(math.Min(0xffff, float64(r)*1.5))
	hg := uint16(math.Min(0xffff, float64(gVal)*1.5))
	hb := uint16(math.Min(0xffff, float64(b)*1.5))
	highlight = color.RGBA64{hr, hg, hb, uint16(a)}

	// シャドウ色 (暗くする)
	sr := uint16(float64(r) * 0.5)
	sg := uint16(float64(gVal) * 0.5)
	sb := uint16(float64(b) * 0.5)
	shadow = color.RGBA64{sr, sg, sb, uint16(a)}

	return highlight, shadow
}

// createCyberpunkPanelNineSlice は、サイバーパンク風のパネル用NineSlice画像を生成します。
// グラデーション背景と立体的な枠線が特徴です。
func (g *UIImageGenerator) createCyberpunkPanelNineSlice(thickness float32) *image.NineSlice {
	tileSize := 64
	borderInset := int(thickness)

	img := ebiten.NewImage(tileSize, tileSize)

	// サイバーパンク風のグラデーション背景を描画
	startColor := color.RGBA{R: 0, G: 20, B: 40, A: 255}
	endColor := color.RGBA{R: 20, G: 40, B: 80, A: 255}
	g.drawGradient(img, startColor, endColor)

	// 枠線の色を定義
	borderColor := color.RGBA{R: 0, G: 191, B: 255, A: 255} // ディープスカイブルー
	highlightColor, shadowColor := g.createHighlightAndShadowColors(borderColor)

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
