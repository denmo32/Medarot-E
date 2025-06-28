package main

import (
	"fmt"
	"image"
	"image/color"
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

// 各ComponentTypeが定義されていることを想定しています
// var SettingsComponent = donburi.NewComponentType[Settings]()
// var StateComponent = donburi.NewComponentType[State]()
// var GaugeComponent = donburi.NewComponentType[Gauge]()

type BattlefieldWidget struct {
	*widget.Container
	game         *Game
	medarotIcons []*CustomIconWidget
}

type CustomIconWidget struct {
	entry *donburi.Entry
	game  *Game
	xPos  float32
	yPos  float32
	rect  image.Rectangle
}

func NewBattlefieldWidget(game *Game) *BattlefieldWidget {
	bf := &BattlefieldWidget{
		game:         game,
		medarotIcons: make([]*CustomIconWidget, 0),
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
	// 修正: filter.With を filter.Contains に変更
	query.NewQuery(filter.Contains(SettingsComponent)).Each(bf.game.World, func(entry *donburi.Entry) {
		icon := NewCustomIconWidget(entry, bf.game)
		bf.medarotIcons = append(bf.medarotIcons, icon)
	})
}

func NewCustomIconWidget(entry *donburi.Entry, game *Game) *CustomIconWidget {
	return &CustomIconWidget{
		entry: entry,
		game:  game,
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
	radius := w.game.Config.UI.Battlefield.IconRadius
	vector.DrawFilledCircle(screen, centerX, centerY, radius, iconColor, true)

	// 修正: ComponentType.Get(entry) を使用
	settings := SettingsComponent.Get(w.entry)
	if settings.IsLeader {
		vector.StrokeCircle(screen, centerX, centerY, radius+3, 2,
			w.game.Config.UI.Colors.Leader, true)
	}
	w.drawStateIndicator(screen, centerX, centerY)
}

func (w *CustomIconWidget) drawDebugInfo(screen *ebiten.Image) {
	if !w.game.DebugMode {
		return
	}

	// 修正: 未使用の settings 変数を削除し、各ComponentType.Get(entry) を使用
	state := StateComponent.Get(w.entry)
	gauge := GaugeComponent.Get(w.entry)

	debugText := fmt.Sprintf(
		"State: %s\nGauge: %.1f\nProg: %.1f / %.1f",
		state.State,
		gauge.CurrentGauge,
		gauge.ProgressCounter,
		gauge.TotalDuration,
	)

	x := int(w.xPos + 20)
	y := int(w.yPos - 20)
	ebitenutil.DebugPrintAt(screen, debugText, x, y)
}

func (w *CustomIconWidget) drawStateIndicator(screen *ebiten.Image, centerX, centerY float32) {
	// 修正: ComponentType.Get(entry) を使用
	state := StateComponent.Get(w.entry)
	switch state.State {
	case StateBroken:
		lineWidth := float32(2)
		size := float32(6)
		vector.StrokeLine(screen, centerX-size, centerY-size,
			centerX+size, centerY+size, lineWidth,
			w.game.Config.UI.Colors.White, true)
		vector.StrokeLine(screen, centerX-size, centerY+size,
			centerX+size, centerY-size, lineWidth,
			w.game.Config.UI.Colors.White, true)
	case StateReady:
		if (w.game.TickCount/30)%2 == 0 {
			vector.StrokeCircle(screen, centerX, centerY,
				w.game.Config.UI.Battlefield.IconRadius+5, 2,
				w.game.Config.UI.Colors.Yellow, true)
		}
	case StateCooldown, StateCharging:
		w.drawCooldownGauge(screen, centerX, centerY)
	}
}

func (w *CustomIconWidget) drawCooldownGauge(screen *ebiten.Image, centerX, centerY float32) {
	// 修正: ComponentType.Get(entry) を使用
	gauge := GaugeComponent.Get(w.entry)
	radius := w.game.Config.UI.Battlefield.IconRadius + 8
	progress := gauge.CurrentGauge / 100.0
	vector.StrokeCircle(screen, centerX, centerY, radius, 2,
		w.game.Config.UI.Colors.Gray, true)
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
				w.game.Config.UI.Colors.Yellow, true)
		}
	}
}

func (w *CustomIconWidget) getIconColor() color.Color {
	// 修正: ComponentType.Get(entry) を使用
	settings := SettingsComponent.Get(w.entry)
	// 修正: ComponentType.Get(entry) を使用
	state := StateComponent.Get(w.entry)

	if state.State == StateBroken {
		return w.game.Config.UI.Colors.Broken
	}
	if settings.Team == Team1 {
		return w.game.Config.UI.Colors.Team1
	}
	return w.game.Config.UI.Colors.Team2
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
		x, y := bf.calculateIconPosition(icon.entry, width, height)
		icon.xPos = offsetX + x
		icon.yPos = offsetY + y
		icon.rect = image.Rect(
			int(icon.xPos-10), int(icon.yPos-10),
			int(icon.xPos+10), int(icon.yPos+10),
		)
	}
}

func (bf *BattlefieldWidget) calculateIconPosition(entry *donburi.Entry, width, height float32) (float32, float32) {
	// 修正: ComponentType.Get(entry) を使用
	settings := SettingsComponent.Get(entry)
	// 修正: ComponentType.Get(entry) を使用
	state := StateComponent.Get(entry)
	// 修正: ComponentType.Get(entry) を使用
	gauge := GaugeComponent.Get(entry)

	progress := float32(gauge.CurrentGauge / 100.0)
	yPos := (height / float32(PlayersPerTeam+1)) * (float32(settings.DrawIndex) + 1)
	homeX, execX := width*0.1, width*0.4
	if settings.Team == Team2 {
		homeX, execX = width*0.9, width*0.6
	}

	var xPos float32
	switch state.State {
	case StateCharging:
		xPos = homeX + (execX-homeX)*progress
	case StateReady:
		xPos = execX
	case StateCooldown:
		xPos = execX - (execX-homeX)*progress
	case StateIdle, StateBroken:
		xPos = homeX
	default:
		xPos = homeX
	}
	return xPos, yPos
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
		bf.game.Config.UI.Battlefield.LineWidth,
		bf.game.Config.UI.Colors.Gray, false)
	team1HomeX := offsetX + width*0.1
	team2HomeX := offsetX + width*0.9
	team1ExecX := offsetX + width*0.4
	team2ExecX := offsetX + width*0.6
	for i := 0; i < PlayersPerTeam; i++ {
		yPos := offsetY + (height/float32(PlayersPerTeam+1))*(float32(i)+1)
		vector.StrokeCircle(screen, team1HomeX, yPos,
			bf.game.Config.UI.Battlefield.HomeMarkerRadius,
			bf.game.Config.UI.Battlefield.LineWidth,
			bf.game.Config.UI.Colors.Gray, true)
		vector.StrokeCircle(screen, team2HomeX, yPos,
			bf.game.Config.UI.Battlefield.HomeMarkerRadius,
			// 修正: bf.game.Code を bf.game.Config に変更
			bf.game.Config.UI.Battlefield.LineWidth,
			bf.game.Config.UI.Colors.Gray, true)
	}
	vector.StrokeLine(screen, team1ExecX, offsetY, team1ExecX, offsetY+height,
		bf.game.Config.UI.Battlefield.LineWidth,
		bf.game.Config.UI.Colors.White, true)
	vector.StrokeLine(screen, team2ExecX, offsetY, team2ExecX, offsetY+height,
		bf.game.Config.UI.Battlefield.LineWidth,
		bf.game.Config.UI.Colors.White, true)
}
