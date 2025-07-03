package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	uiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

type BattlefieldWidget struct {
	*widget.Container
	scene        *BattleScene
	medarotIcons []*CustomIconWidget
	whitePixel   *ebiten.Image
}

type CustomIconWidget struct {
	entry *donburi.Entry
	scene *BattleScene
	xPos  float32
	yPos  float32
	rect  image.Rectangle
}

func NewBattlefieldWidget(bs *BattleScene) *BattlefieldWidget {
	whiteImg := ebiten.NewImage(1, 1)
	whiteImg.Fill(color.White)

	bf := &BattlefieldWidget{
		scene:        bs,
		medarotIcons: make([]*CustomIconWidget, 0),
		whitePixel:   whiteImg,
	}
	bf.Container = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			uiimage.NewNineSliceColor(color.NRGBA{20, 30, 40, 255}),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	bf.createMedarotIcons()
	return bf
}

func (bf *BattlefieldWidget) createMedarotIcons() {
	query.NewQuery(filter.Contains(SettingsComponent)).Each(bf.scene.world, func(entry *donburi.Entry) {
		icon := NewCustomIconWidget(entry, bf.scene)
		bf.medarotIcons = append(bf.medarotIcons, icon)
	})
}

func NewCustomIconWidget(entry *donburi.Entry, bs *BattleScene) *CustomIconWidget {
	return &CustomIconWidget{
		entry: entry,
		scene: bs,
		rect:  image.Rect(0, 0, 20, 20),
	}
}

func (w *CustomIconWidget) Render(screen *ebiten.Image) {
	if w.rect.Dx() == 0 || w.rect.Dy() == 0 {
		return
	}
	centerX := w.xPos
	centerY := w.yPos
	iconColor := w.getIconColor()
	radius := w.scene.resources.Config.UI.Battlefield.IconRadius
	vector.DrawFilledCircle(screen, centerX, centerY, radius, iconColor, true)

	settings := SettingsComponent.Get(w.entry)
	if settings.IsLeader {
		vector.StrokeCircle(screen, centerX, centerY, radius+3, 2,
			w.scene.resources.Config.UI.Colors.Leader, true)
	}
	w.drawStateIndicator(screen, centerX, centerY)
}

func (w *CustomIconWidget) drawDebugInfo(screen *ebiten.Image) {
	if !w.scene.debugMode {
		return
	}

	gauge := GaugeComponent.Get(w.entry)

	var stateStr string
	if w.entry.HasComponent(IdleStateComponent) {
		stateStr = "待機"
	} else if w.entry.HasComponent(ChargingStateComponent) {
		stateStr = "チャージ中"
	} else if w.entry.HasComponent(ReadyStateComponent) {
		stateStr = "実行準備"
	} else if w.entry.HasComponent(CooldownStateComponent) {
		stateStr = "クールダウン"
	} else if w.entry.HasComponent(BrokenStateComponent) {
		stateStr = "機能停止"
	}

	debugText := fmt.Sprintf(
		"State: %s\nGauge: %.1f\nProg: %.1f / %.1f",
		stateStr,
		gauge.CurrentGauge,
		gauge.ProgressCounter,
		gauge.TotalDuration,
	)

	x := int(w.xPos + 20)
	y := int(w.yPos - 20)
	ebitenutil.DebugPrintAt(screen, debugText, x, y)
}

func (w *CustomIconWidget) drawStateIndicator(screen *ebiten.Image, centerX, centerY float32) {
	if w.entry.HasComponent(BrokenStateComponent) {
		lineWidth := float32(2)
		size := float32(6)
		vector.StrokeLine(screen, centerX-size, centerY-size,
			centerX+size, centerY+size, lineWidth,
			w.scene.resources.Config.UI.Colors.White, true)
		vector.StrokeLine(screen, centerX-size, centerY+size,
			centerX+size, centerY-size, lineWidth,
			w.scene.resources.Config.UI.Colors.White, true)
	} else if w.entry.HasComponent(ReadyStateComponent) {
		if (w.scene.tickCount/30)%2 == 0 {
			vector.StrokeCircle(screen, centerX, centerY,
				w.scene.resources.Config.UI.Battlefield.IconRadius+5, 2,
				w.scene.resources.Config.UI.Colors.Yellow, true)
		}
	} else if w.entry.HasComponent(CooldownStateComponent) || w.entry.HasComponent(ChargingStateComponent) {
		w.drawCooldownGauge(screen, centerX, centerY)
	}
}

func (w *CustomIconWidget) drawCooldownGauge(screen *ebiten.Image, centerX, centerY float32) {
	gauge := GaugeComponent.Get(w.entry)
	radius := w.scene.resources.Config.UI.Battlefield.IconRadius + 8
	progress := gauge.CurrentGauge / 100.0
	vector.StrokeCircle(screen, centerX, centerY, radius, 2,
		w.scene.resources.Config.UI.Colors.Gray, true)
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
				w.scene.resources.Config.UI.Colors.Yellow, true)
		}
	}
}

func (w *CustomIconWidget) getIconColor() color.Color {
	settings := SettingsComponent.Get(w.entry)

	if w.entry.HasComponent(BrokenStateComponent) {
		return w.scene.resources.Config.UI.Colors.Broken
	}
	if settings.Team == Team1 {
		return w.scene.resources.Config.UI.Colors.Team1
	}
	return w.scene.resources.Config.UI.Colors.Team2
}

func (bf *BattlefieldWidget) UpdatePositions() {
	rect := bf.Container.GetWidget().Rect
	if rect.Dx() == 0 || rect.Dy() == 0 {
		return
	}
	width := float32(rect.Dx())
	height := float32(rect.Dy())
	offsetX := float32(rect.Min.X)
	offsetY := float32(rect.Min.Y)

	for _, icon := range bf.medarotIcons {
		if bf.scene.partInfoProvider == nil {
			log.Println("Error: BattlefieldWidget.UpdatePositions - partInfoProvider is nil")
			continue
		}
		x := bf.scene.partInfoProvider.CalculateIconXPosition(icon.entry, width)
		settings := SettingsComponent.Get(icon.entry)
		y := (height / float32(PlayersPerTeam+1)) * (float32(settings.DrawIndex) + 1)
		icon.xPos = offsetX + x
		icon.yPos = offsetY + y
		icon.rect = image.Rect(
			int(icon.xPos-10), int(icon.yPos-10),
			int(icon.xPos+10), int(icon.yPos+10),
		)
	}
}

func (bf *BattlefieldWidget) DrawIcons(screen *ebiten.Image) {
	for _, icon := range bf.medarotIcons {
		icon.Render(screen)
	}
}

func (bf *BattlefieldWidget) DrawDebug(screen *ebiten.Image) {
	for _, icon := range bf.medarotIcons {
		icon.drawDebugInfo(screen)
	}
}

func (bf *BattlefieldWidget) DrawTargetIndicator(screen *ebiten.Image, targetEntry *donburi.Entry) {
	var targetIcon *CustomIconWidget
	for _, icon := range bf.medarotIcons {
		if icon.entry == targetEntry {
			targetIcon = icon
			break
		}
	}
	if targetIcon == nil {
		return
	}
	tx, ty := targetIcon.xPos, targetIcon.yPos

	indicatorColor := bf.scene.resources.Config.UI.Colors.Yellow
	iconRadius := bf.scene.resources.Config.UI.Battlefield.IconRadius
	indicatorHeight := bf.scene.resources.Config.UI.Battlefield.TargetIndicator.Height
	indicatorWidth := bf.scene.resources.Config.UI.Battlefield.TargetIndicator.Width
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
		bf.scene.resources.Config.UI.Battlefield.LineWidth,
		bf.scene.resources.Config.UI.Colors.Gray, false)
	team1HomeX := offsetX + width*0.1
	team2HomeX := offsetX + width*0.9
	team1ExecX := offsetX + width*0.4
	team2ExecX := offsetX + width*0.6
	for i := 0; i < PlayersPerTeam; i++ {
		yPos := offsetY + (height/float32(PlayersPerTeam+1))*(float32(i)+1)
		vector.StrokeCircle(screen, team1HomeX, yPos,
			bf.scene.resources.Config.UI.Battlefield.HomeMarkerRadius,
			bf.scene.resources.Config.UI.Battlefield.LineWidth,
			bf.scene.resources.Config.UI.Colors.Gray, true)
		vector.StrokeCircle(screen, team2HomeX, yPos,
			bf.scene.resources.Config.UI.Battlefield.HomeMarkerRadius,
			bf.scene.resources.Config.UI.Battlefield.LineWidth,
			bf.scene.resources.Config.UI.Colors.Gray, true)
	}
	vector.StrokeLine(screen, team1ExecX, offsetY, team1ExecX, offsetY+height,
		bf.scene.resources.Config.UI.Battlefield.LineWidth,
		bf.scene.resources.Config.UI.Colors.White, true)
	vector.StrokeLine(screen, team2ExecX, offsetY, team2ExecX, offsetY+height,
		bf.scene.resources.Config.UI.Battlefield.LineWidth,
		bf.scene.resources.Config.UI.Colors.White, true)
}
