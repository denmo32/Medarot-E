package main

import (
	"image"
	"image/color"
	"math"

	// uiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type BattlefieldWidget struct {
	*widget.Container
	config          *Config
	whitePixel      *ebiten.Image
	viewModel       *BattlefieldViewModel
	backgroundImage *ebiten.Image
}

type CustomIconWidget struct {
	viewModel *IconViewModel
	config    *Config
	rect      image.Rectangle
}

func NewBattlefieldWidget(config *Config) *BattlefieldWidget {
	whiteImg := ebiten.NewImage(1, 1)
	whiteImg.Fill(color.White)

	bgImage := r.LoadImage(ImageBattleBackground).Data

	bf := &BattlefieldWidget{
		config:          config,
		whitePixel:      whiteImg,
		backgroundImage: bgImage,
	}
	bf.Container = widget.NewContainer(
		// 背景画像はBattleSceneで描画するため、ここでは設定しない
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	return bf
}

func NewCustomIconWidget(vm *IconViewModel, config *Config) *CustomIconWidget {
	return &CustomIconWidget{
		viewModel: vm,
		config:    config,
		rect:      image.Rect(0, 0, 20, 20),
	}
}

func (w *CustomIconWidget) Render(screen *ebiten.Image) {
	if w.rect.Dx() == 0 || w.rect.Dy() == 0 {
		return
	}
	centerX := w.viewModel.X
	centerY := w.viewModel.Y
	iconColor := w.viewModel.Color
	radius := w.config.UI.Battlefield.IconRadius
	vector.DrawFilledCircle(screen, centerX, centerY, radius, iconColor, true)

	if w.viewModel.IsLeader {
		vector.StrokeCircle(screen, centerX, centerY, radius+3, 2,
			w.config.UI.Colors.Leader, true)
	}
	w.drawStateIndicator(screen, centerX, centerY)
}

func (w *CustomIconWidget) drawDebugInfo(screen *ebiten.Image) {
	// デバッグモードはBattleSceneで管理されるため、ここではViewModelのDebugTextを使用
	if w.viewModel.DebugText == "" {
		return
	}

	x := int(w.viewModel.X + 20)
	y := int(w.viewModel.Y - 20)
	ebitenutil.DebugPrintAt(screen, w.viewModel.DebugText, x, y)
}

func (w *CustomIconWidget) drawStateIndicator(screen *ebiten.Image, centerX, centerY float32) {
	switch w.viewModel.State {
	case StateBroken:
		lineWidth := float32(2)
		size := float32(6)
		vector.StrokeLine(screen, centerX-size, centerY-size,
			centerX+size, centerY+size, lineWidth,
			w.config.UI.Colors.White, true)
		vector.StrokeLine(screen, centerX-size, centerY+size,
			centerX+size, centerY-size, lineWidth,
			w.config.UI.Colors.White, true)
	case StateReady:
		// tickCountはBattleSceneで管理されるため、ここではViewModelに含めない
		// このアニメーションはBattleSceneのUpdateで制御されるべき
		// 現状はViewModelにtickCountがないため、アニメーションは停止
		// if (w.scene.tickCount/30)%2 == 0 {
		// 	vector.StrokeCircle(screen, centerX, centerY,
		// 		w.config.UI.Battlefield.IconRadius+5, 2,
		// 		w.config.UI.Colors.Yellow, true)
		// }
	case StateCharging, StateCooldown:
		w.drawCooldownGauge(screen, centerX, centerY)
	}
}

func (w *CustomIconWidget) drawCooldownGauge(screen *ebiten.Image, centerX, centerY float32) {
	radius := w.config.UI.Battlefield.IconRadius + 8
	progress := w.viewModel.GaugeProgress
	vector.StrokeCircle(screen, centerX, centerY, radius, 2,
		w.config.UI.Colors.Gray, true)
	if progress > 0 {
		steps := int(progress * 32)
		for i := 0; i < steps; i++ {
			angle := float64(i) * 2 * math.Pi / 32
			nextAngle := float64(i+1) * 2 * math.Pi / 32
			x1 := centerX + radius*float32(math.Cos(angle-math.Pi/2))
			y1 := centerY + radius*float32(math.Sin(angle-math.Pi/2))
			x2 := centerX + radius*float32(math.Cos(nextAngle-math.Pi/2))
			y2 := centerY + radius*float32(math.Sin(nextAngle-math.Pi/2))
			vector.StrokeLine(screen, x1, y1, x2, y2, 3,
				w.config.UI.Colors.Yellow, true)
		}
	}
}

// getIconColor はViewModelから直接色を取得するため不要になります。
// func (w *CustomIconWidget) getIconColor() color.Color {
// 	return w.viewModel.Color
// }

func (bf *BattlefieldWidget) SetViewModel(vm BattlefieldViewModel) {
	bf.viewModel = &vm
}

// Draw はバトルフィールドのすべての要素を描画します。
// targetIconVM はターゲットインジケーターを描画するためのIconViewModelです。
func (bf *BattlefieldWidget) Draw(screen *ebiten.Image, targetIconVM *IconViewModel, tick int) {
	// 背景の描画はBattleSceneで行うため、ここでは行わない

	// アイコンの描画
	if bf.viewModel != nil {
		for _, iconVM := range bf.viewModel.Icons {
			iconWidget := NewCustomIconWidget(iconVM, bf.config)
			iconWidget.Render(screen)
		}

		// デバッグ情報の描画
		if bf.viewModel.DebugMode {
			for _, iconVM := range bf.viewModel.Icons {
				iconWidget := NewCustomIconWidget(iconVM, bf.config)
				iconWidget.drawDebugInfo(screen)
			}
		}
	}

	// ターゲットインジケーターの描画
	bf.DrawTargetIndicator(screen, targetIconVM, tick)
}

func (bf *BattlefieldWidget) DrawIcons(screen *ebiten.Image) {
	if bf.viewModel == nil {
		return
	}
	for _, iconVM := range bf.viewModel.Icons {
		iconWidget := NewCustomIconWidget(iconVM, bf.config)
		iconWidget.Render(screen)
	}
}

func (bf *BattlefieldWidget) DrawDebug(screen *ebiten.Image) {
	if bf.viewModel == nil {
		return
	}
	for _, iconVM := range bf.viewModel.Icons {
		iconWidget := NewCustomIconWidget(iconVM, bf.config)
		iconWidget.drawDebugInfo(screen)
	}
}

func (bf *BattlefieldWidget) DrawTargetIndicator(screen *ebiten.Image, targetIconVM *IconViewModel, tick int) {
	if targetIconVM == nil {
		return
	}
	tx, ty := targetIconVM.X, targetIconVM.Y
	indicatorColor := color.RGBA{R: 0, G: 255, B: 255, A: 255} // ネオン風の水色

	// アニメーションパラメータ
	const animationSpeed = 0.1
	const minOuterRadius = 15.0
	const maxOuterRadius = 25.0
	const innerRadiusRatio = 0.4 // 内側の円の半径を外側の円に対する割合で指定

	// 時間経過に基づいて半径を計算（sin波で拡大・縮小）
	angle := float32(tick) * animationSpeed
	// sinの結果は-1から1なので、0から1の範囲に変換
	normalizedSin := (math.Sin(float64(angle)) + 1) / 2
	// 半径をminとmaxの間で変動させる
	outerRadius := minOuterRadius + (maxOuterRadius-minOuterRadius)*float32(normalizedSin)
	innerRadius := outerRadius * innerRadiusRatio

	// 線の太さもアニメーションさせる
	const minStrokeWidth = 1.5
	const maxStrokeWidth = 3.0
	strokeWidth := minStrokeWidth + (maxStrokeWidth-minStrokeWidth)*float32(normalizedSin)

	// 外側の円を描画
	vector.StrokeCircle(screen, tx, ty, outerRadius, strokeWidth, indicatorColor, true)
	// 内側の円を描画
	vector.StrokeCircle(screen, tx, ty, innerRadius, strokeWidth*0.8, indicatorColor, true)
}

func (bf *BattlefieldWidget) DrawBackground(screen *ebiten.Image) {
	rect := bf.Container.GetWidget().Rect
	if rect.Dx() == 0 || rect.Dy() == 0 {
		return
	}

	// Draw the background image to fill the widget, cropping as needed.
	if bf.backgroundImage != nil {
		bgW, bgH := bf.backgroundImage.Size()
		widgetW, widgetH := rect.Dx(), rect.Dy()

		scale := math.Max(float64(widgetW)/float64(bgW), float64(widgetH)/float64(bgH))

		drawW, drawH := float64(bgW)*scale, float64(bgH)*scale
		dx, dy := (float64(widgetW)-drawW)/2, (float64(widgetH)-drawH)/2

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(float64(rect.Min.X)+dx, float64(rect.Min.Y)+dy)
		// 明度を下げるためにColorScaleを適用
		dimmingFactor := 0.48 // 0.0 (真っ暗) から 1.0 (元の明るさ) の間で調整
		op.ColorScale.Scale(float32(dimmingFactor), float32(dimmingFactor), float32(dimmingFactor), 1.0)
		screen.DrawImage(bf.backgroundImage, op)
	}

	width := float32(rect.Dx())
	height := float32(rect.Dy())
	offsetX := float32(rect.Min.X)
	offsetY := float32(rect.Min.Y)
	vector.StrokeRect(screen, offsetX, offsetY, width, height,
		bf.config.UI.Battlefield.LineWidth,
		bf.config.UI.Colors.Gray, false)
	team1HomeX := offsetX + width*0.1
	team2HomeX := offsetX + width*0.9
	team1ExecX := offsetX + width*0.4
	team2ExecX := offsetX + width*0.6
	for i := 0; i < PlayersPerTeam; i++ {
		yPos := offsetY + (height/float32(PlayersPerTeam+1))*(float32(i)+1)
		vector.StrokeCircle(screen, team1HomeX, yPos,
			bf.config.UI.Battlefield.HomeMarkerRadius,
			bf.config.UI.Battlefield.LineWidth,
			bf.config.UI.Colors.Gray, true)
		vector.StrokeCircle(screen, team2HomeX, yPos,
			bf.config.UI.Battlefield.HomeMarkerRadius,
			bf.config.UI.Battlefield.LineWidth,
			bf.config.UI.Colors.Gray, true)
	}
	vector.StrokeLine(screen, team1ExecX, offsetY, team1ExecX, offsetY+height,
		bf.config.UI.Battlefield.LineWidth,
		bf.config.UI.Colors.White, true)
	vector.StrokeLine(screen, team2ExecX, offsetY, team2ExecX, offsetY+height,
		bf.config.UI.Battlefield.LineWidth,
		bf.config.UI.Colors.White, true)
}
