package main

import (
	"image"
	"image/color"
	"math"

	"medarot-ebiten/domain"

	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// BattlefieldWidget はバトルフィールドの描画に必要なデータを保持します。
type BattlefieldWidget struct {
	*widget.Container
	config       *Config
	whitePixel   *ebiten.Image
	viewModel    *BattlefieldViewModel
	bgImage      *ebiten.Image   // 背景画像を直接保持
	customWidget *widget.Graphic // カスタム描画ウィジェット
}

func NewBattlefieldWidget(config *Config) *BattlefieldWidget {
	whiteImg := ebiten.NewImage(1, 1)
	whiteImg.Fill(color.White)

	bf := &BattlefieldWidget{
		config:     config,
		whitePixel: whiteImg,
	}

	// 背景画像を読み込み
	bf.bgImage = r.LoadImage(ImageBattleBackground).Data

	// カスタム描画用のGraphicウィジェットを作成
	bf.customWidget = widget.NewGraphic(
		widget.GraphicOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				StretchHorizontal:  true,
				StretchVertical:    true,
			}),
		),
		widget.GraphicOpts.ImageNineSlice(eimage.NewNineSliceColor(color.Transparent)), // 透明な背景
	)

	// コンテナを作成
	bf.Container = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// カスタム描画ウィジェットをコンテナに追加
	bf.Container.AddChild(bf.customWidget)

	return bf
}

// drawBackgroundImage は背景画像をクリッピングして描画します。
func (bf *BattlefieldWidget) drawBackgroundImage(screen *ebiten.Image, rect image.Rectangle) {
	if bf.bgImage == nil {
		// 背景画像がない場合は単色で塗りつぶし
		screen.Fill(color.RGBA{R: 20, G: 30, B: 50, A: 255})
		return
	}

	imgW, imgH := bf.bgImage.Size()
	viewW, viewH := rect.Dx(), rect.Dy()

	// 画像のアスペクト比と表示領域のアスペクト比を比較
	imgAspect := float64(imgW) / float64(imgH)
	viewAspect := float64(viewW) / float64(viewH)

	var scale float64
	var srcRect image.Rectangle

	if imgAspect > viewAspect {
		// 画像が横長の場合、高さを表示領域に合わせ、横をクリッピング
		scale = float64(viewH) / float64(imgH)
		scaledImgW := float64(imgW) * scale
		clipX := (scaledImgW - float64(viewW)) / 2
		srcRect = image.Rect(int(clipX/scale), 0, int((clipX+float64(viewW))/scale), imgH)
	} else {
		// 画像が縦長の場合、幅を表示領域に合わせ、縦をクリッピング
		scale = float64(viewW) / float64(imgW)
		scaledImgH := float64(imgH) * scale
		clipY := (scaledImgH - float64(viewH)) / 2
		srcRect = image.Rect(0, int(clipY/scale), imgW, int((clipY+float64(viewH))/scale))
	}

	op := &ebiten.DrawImageOptions{}
	m := ebiten.GeoM{}
	m.Scale(scale, scale)
	m.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	op.GeoM = m
	op.Filter = ebiten.FilterLinear // スケーリング時の品質を向上

	screen.DrawImage(bf.bgImage.SubImage(srcRect).(*ebiten.Image), op)
}

// SetViewModel はViewModelを設定し、描画更新をトリガーします
func (bf *BattlefieldWidget) SetViewModel(vm BattlefieldViewModel) {
	bf.viewModel = &vm
	// カスタム描画ウィジェットの再描画をトリガー
	if bf.customWidget != nil {
		bf.customWidget.GetWidget().Disabled = false // 強制的に再描画をトリガー
	}
}

// Draw はバトルフィールドのすべての要素を描画します。
func (bf *BattlefieldWidget) Draw(screen *ebiten.Image, targetIconVM *IconViewModel, tick int) {
	// コンテナの描画領域を取得
	rect := bf.Container.GetWidget().Rect
	if rect.Dx() == 0 || rect.Dy() == 0 {
		return
	}

	// 1. 背景画像の描画（クリッピング）
	bf.drawBackgroundImage(screen, rect)

	// 2. 枠線、ホームマーカー、実行ラインの描画（背景レイヤー）
	bf.drawBattlefieldLines(screen, rect)

	// 2. アイコンの描画（中間レイヤー）
	bf.drawIcons(screen, rect)

	// 3. ターゲットインジケーターの描画（前景レイヤー）
	bf.DrawTargetIndicator(screen, targetIconVM, tick)

	// 4. デバッグ情報の描画（最前面レイヤー）
	if bf.viewModel != nil && bf.viewModel.DebugMode {
		bf.drawDebugInfo(screen, rect)
	}
}

// drawBattlefieldLines はバトルフィールドの線やマーカーを描画します
func (bf *BattlefieldWidget) drawBattlefieldLines(screen *ebiten.Image, rect image.Rectangle) {
	width := float32(rect.Dx())
	height := float32(rect.Dy())
	offsetX := float32(rect.Min.X)
	offsetY := float32(rect.Min.Y)

	// 外枠線
	vector.StrokeRect(screen, offsetX, offsetY, width, height,
		bf.config.UI.Battlefield.LineWidth,
		bf.config.UI.Colors.Gray, false)

	// チーム位置の計算
	team1HomeX := offsetX + width*bf.config.UI.Battlefield.Team1HomeX
	team2HomeX := offsetX + width*bf.config.UI.Battlefield.Team2HomeX
	team1ExecX := offsetX + width*bf.config.UI.Battlefield.Team1ExecutionLineX
	team2ExecX := offsetX + width*bf.config.UI.Battlefield.Team2ExecutionLineX

	// ホームマーカー
	// game_settings.json の UI.Battlefield.MedarotVerticalSpacingFactor を使用してY座標を計算します。
	for i := 0; i < domain.PlayersPerTeam; i++ {
		yPos := offsetY + (height/float32(domain.PlayersPerTeam+1))*(float32(i)+1)

		// チーム1のホームマーカー
		vector.StrokeCircle(screen, team1HomeX, yPos,
			bf.config.UI.Battlefield.HomeMarkerRadius,
			bf.config.UI.Battlefield.LineWidth,
			bf.config.UI.Colors.Gray, true)

		// チーム2のホームマーカー
		vector.StrokeCircle(screen, team2HomeX, yPos,
			bf.config.UI.Battlefield.HomeMarkerRadius,
			bf.config.UI.Battlefield.LineWidth,
			bf.config.UI.Colors.Gray, true)
	}

	// 実行ライン
	vector.StrokeLine(screen, team1ExecX, offsetY, team1ExecX, offsetY+height,
		bf.config.UI.Battlefield.LineWidth,
		bf.config.UI.Colors.White, true)
	vector.StrokeLine(screen, team2ExecX, offsetY, team2ExecX, offsetY+height,
		bf.config.UI.Battlefield.LineWidth,
		bf.config.UI.Colors.White, true)
}

// drawIcons はメダロットアイコンを描画します
func (bf *BattlefieldWidget) drawIcons(screen *ebiten.Image, rect image.Rectangle) {
	if bf.viewModel == nil {
		return
	}

	for _, iconVM := range bf.viewModel.Icons {
		bf.drawSingleIcon(screen, iconVM, rect)
	}
}

// drawSingleIcon は単一のアイコンを描画します
func (bf *BattlefieldWidget) drawSingleIcon(screen *ebiten.Image, iconVM *IconViewModel, _ image.Rectangle) {
	centerX := iconVM.X
	centerY := iconVM.Y
	iconColor := iconVM.Color
	radius := bf.config.UI.Battlefield.IconRadius

	// メインアイコン（塗りつぶし円）
	vector.DrawFilledCircle(screen, centerX, centerY, radius, iconColor, true)

	// リーダーマーク
	if iconVM.IsLeader {
		vector.StrokeCircle(screen, centerX, centerY, radius+3, 2,
			bf.config.UI.Colors.Leader, true)
	}

	// 状態インジケーター
	bf.drawStateIndicator(screen, iconVM, centerX, centerY)
}

// drawStateIndicator は状態インジケーターを描画します
func (bf *BattlefieldWidget) drawStateIndicator(screen *ebiten.Image, iconVM *IconViewModel, centerX, centerY float32) {
	switch iconVM.State {
	case domain.StateBroken:
		// X印を描画
		lineWidth := float32(2)
		size := float32(6)
		vector.StrokeLine(screen, centerX-size, centerY-size,
			centerX+size, centerY+size, lineWidth,
			bf.config.UI.Colors.White, true)
		vector.StrokeLine(screen, centerX-size, centerY+size,
			centerX+size, centerY-size, lineWidth,
			bf.config.UI.Colors.White, true)

	case domain.StateReady:
		// 準備完了の点滅効果（静的版）
		vector.StrokeCircle(screen, centerX, centerY,
			bf.config.UI.Battlefield.IconRadius+5, 2,
			bf.config.UI.Colors.Yellow, true)

	case domain.StateCharging, domain.StateCooldown:
		// ゲージ表示
		bf.drawCooldownGauge(screen, iconVM, centerX, centerY)
	}
}

// drawCooldownGauge はクールダウンゲージを描画します
func (bf *BattlefieldWidget) drawCooldownGauge(screen *ebiten.Image, iconVM *IconViewModel, centerX, centerY float32) {
	radius := bf.config.UI.Battlefield.IconRadius + 8
	progress := iconVM.GaugeProgress

	// 背景の円
	vector.StrokeCircle(screen, centerX, centerY, radius, 2,
		bf.config.UI.Colors.Gray, true)

	// プログレス表示
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
				bf.config.UI.Colors.Yellow, true)
		}
	}
}

// drawDebugInfo はデバッグ情報を描画します
func (bf *BattlefieldWidget) drawDebugInfo(screen *ebiten.Image, rect image.Rectangle) {
	if bf.viewModel == nil {
		return
	}

	for _, iconVM := range bf.viewModel.Icons {
		if iconVM.DebugText == "" {
			continue
		}

		x := int(iconVM.X + 20)
		y := int(iconVM.Y - 20)

		// デバッグテキストが画面外に出ないように調整
		if x > rect.Max.X-200 {
			x = int(iconVM.X - 150)
		}
		if y < rect.Min.Y+40 {
			y = int(iconVM.Y + 40)
		}

		ebitenutil.DebugPrintAt(screen, iconVM.DebugText, x, y)
	}
}

// DrawTargetIndicator はターゲットインジケーターを描画します
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
	const innerRadiusRatio = 0.4

	// 時間経過に基づいて半径を計算
	angle := float32(tick) * animationSpeed
	normalizedSin := (math.Sin(float64(angle)) + 1) / 2
	outerRadius := minOuterRadius + (maxOuterRadius-minOuterRadius)*float32(normalizedSin)
	innerRadius := outerRadius * innerRadiusRatio

	// 線の太さもアニメーション
	const minStrokeWidth = 1.5
	const maxStrokeWidth = 3.0
	strokeWidth := minStrokeWidth + (maxStrokeWidth-minStrokeWidth)*float32(normalizedSin)

	// 外側と内側の円を描画
	vector.StrokeCircle(screen, tx, ty, outerRadius, strokeWidth, indicatorColor, true)
	vector.StrokeCircle(screen, tx, ty, innerRadius, strokeWidth*0.8, indicatorColor, true)
}

// 既存の互換性メソッド（廃止予定）
func (bf *BattlefieldWidget) DrawIcons(screen *ebiten.Image) {
	rect := bf.Container.GetWidget().Rect
	bf.drawIcons(screen, rect)
}

func (bf *BattlefieldWidget) DrawDebug(screen *ebiten.Image) {
	rect := bf.Container.GetWidget().Rect
	bf.drawDebugInfo(screen, rect)
}

func (bf *BattlefieldWidget) DrawBackground(screen *ebiten.Image) {
	// 背景はContainerが自動描画するため、何もしない
}
