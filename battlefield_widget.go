package main

import (
	"image"
	"image/color"
	"math"

	uiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type BattlefieldWidget struct {
	*widget.Container
	config     *Config
	whitePixel *ebiten.Image
	viewModel  *BattlefieldViewModel
}

type CustomIconWidget struct {
	viewModel *IconViewModel
	config    *Config
	rect      image.Rectangle
}

func NewBattlefieldWidget(config *Config) *BattlefieldWidget {
	whiteImg := ebiten.NewImage(1, 1)
	whiteImg.Fill(color.White)

	bf := &BattlefieldWidget{
		config:     config,
		whitePixel: whiteImg,
	}
	bf.Container = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			uiimage.NewNineSliceColor(color.NRGBA{20, 30, 40, 255}),
		),
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
	case StateTypeBroken:
		lineWidth := float32(2)
		size := float32(6)
		vector.StrokeLine(screen, centerX-size, centerY-size,
			centerX+size, centerY+size, lineWidth,
			w.config.UI.Colors.White, true)
		vector.StrokeLine(screen, centerX-size, centerY+size,
			centerX+size, centerY-size, lineWidth,
			w.config.UI.Colors.White, true)
	case StateTypeReady:
		// tickCountはBattleSceneで管理されるため、ここではViewModelに含めない
		// このアニメーションはBattleSceneのUpdateで制御されるべき
		// 現状はViewModelにtickCountがないため、アニメーションは停止
		// if (w.scene.tickCount/30)%2 == 0 {
		// 	vector.StrokeCircle(screen, centerX, centerY,
		// 		w.config.UI.Battlefield.IconRadius+5, 2,
		// 		w.config.UI.Colors.Yellow, true)
		// }
	case StateTypeCharging, StateTypeCooldown:
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

func (bf *BattlefieldWidget) DrawTargetIndicator(screen *ebiten.Image, targetIconVM *IconViewModel) {
	if targetIconVM == nil {
		return
	}
	tx, ty := targetIconVM.X, targetIconVM.Y

	indicatorColor := bf.config.UI.Colors.Yellow
	iconRadius := bf.config.UI.Battlefield.IconRadius
	indicatorHeight := bf.config.UI.Battlefield.TargetIndicator.Height
	indicatorWidth := bf.config.UI.Battlefield.TargetIndicator.Width
	margin := float32(5)

	p1x := tx - indicatorWidth/2
	p1y := ty - iconRadius - margin - indicatorHeight
	p2x := tx + indicatorWidth/2
	p2y := p1y
	p3x := tx
	p3y := ty - iconRadius - margin

	vertices := []ebiten.Vertex{
		{DstX: p1x, DstY: p1y},
		{DstX: p2x, DstY: p2y},
		{DstX: p3x, DstY: p3y},
	}
	r, g, b, a := indicatorColor.RGBA()
	cr := float32(r) / 65535
	cg := float32(g) / 65535
	cb := float32(b) / 65535
	ca := float32(a) / 65535
	for i := range vertices {
		vertices[i].ColorR = cr
		vertices[i].ColorG = cg
		vertices[i].ColorB = cb
		vertices[i].ColorA = ca
	}
	indices := []uint16{0, 1, 2}
	screen.DrawTriangles(vertices, indices, bf.whitePixel, &ebiten.DrawTrianglesOptions{})
}

func (bf *BattlefieldWidget) DrawBackground(screen *ebiten.Image) {
	rect := bf.Container.GetWidget().Rect
	if rect.Dx() == 0 || rect.Dy() == 0 {
		return
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
