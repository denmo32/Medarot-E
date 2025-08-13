package ui

import (
	"fmt"
	"image/color"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/event"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// UIAnimationDrawer はUIアニメーションの描画に特化した構造体です。
type UIAnimationDrawer struct {
	config           *data.Config
	currentAnimation *component.ActionAnimationData
	font             text.Face
	eventChannel     chan event.GameEvent
}

// NewUIAnimationDrawer は新しいUIAnimationDrawerインスタンスを作成します。
func NewUIAnimationDrawer(config *data.Config, font text.Face, eventChannel chan event.GameEvent) *UIAnimationDrawer {
	return &UIAnimationDrawer{
		config:       config,
		font:         font,
		eventChannel: eventChannel,
	}
}

// Update はアニメーションの進行状況を更新し、終了した場合はイベントを発行します。
func (d *UIAnimationDrawer) Update(tick float64) {
	if d.currentAnimation == nil {
		return
	}

	if d.IsAnimationFinished(tick) {
		d.eventChannel <- event.ActionAnimationFinishedGameEvent{Result: d.currentAnimation.Result}
		d.ClearAnimation()
	}
}

// SetAnimation は現在再生するアニメーションを設定します。
func (d *UIAnimationDrawer) SetAnimation(anim *component.ActionAnimationData) {
	d.currentAnimation = anim
}

// IsAnimationFinished は現在のアニメーションが完了したかどうかを返します。
func (d *UIAnimationDrawer) IsAnimationFinished(tick float64) bool {
	if d.currentAnimation == nil {
		return true
	}
	const totalAnimationDuration = 120
	return float64(tick-float64(d.currentAnimation.StartTime)) >= totalAnimationDuration
}

// ClearAnimation は現在のアニメーションをクリアします。
func (d *UIAnimationDrawer) ClearAnimation() {
	d.currentAnimation = nil
}

// GetCurrentAnimationResult は現在のアニメーションの結果を返します。
func (d *UIAnimationDrawer) GetCurrentAnimationResult() component.ActionResult {
	return d.currentAnimation.Result
}

// Draw は現在のアニメーションを画面に描画します。
func (d *UIAnimationDrawer) Draw(screen *ebiten.Image, tick float64, battlefieldVM core.BattlefieldViewModel) {
	anim := d.currentAnimation
	if anim == nil {
		return
	}

	progress := tick - float64(anim.StartTime)

	var attackerVM, targetVM *core.IconViewModel
	for _, icon := range battlefieldVM.Icons {
		if icon.EntryID == anim.Result.ActingEntry.Entity() {
			attackerVM = icon
		}
		if anim.Result.TargetEntry != nil && icon.EntryID == anim.Result.TargetEntry.Entity() {
			targetVM = icon
		}
	}

	if attackerVM != nil && targetVM != nil {
		const firstPingDuration = 30.0
		const secondPingDuration = 30.0
		const delayBetweenPings = 0.0

		if progress >= 0 && progress < firstPingDuration {
			pingProgress := progress / firstPingDuration
			d.drawPingAnimation(screen, attackerVM.X, attackerVM.Y, pingProgress, true)
		}

		secondPingStart := firstPingDuration + delayBetweenPings
		if progress >= secondPingStart && progress < secondPingStart+secondPingDuration {
			pingProgress := (progress - secondPingStart) / secondPingDuration
			d.drawPingAnimation(screen, targetVM.X, targetVM.Y, pingProgress, false)
		}

		const popupDelay = 60
		const popupDuration = 60
		const peakTimeRatio = 0.6
		const peakHeight = 40.0
		const settleHeight = 30.0

		popupStartProgress := progress - popupDelay
		var yOffset float32
		alpha := float32(1.0)

		if popupStartProgress >= 0 {
			if popupStartProgress < popupDuration {
				popupProgress := popupStartProgress / popupDuration
				if popupProgress < peakTimeRatio {
					phaseProgress := popupProgress / peakTimeRatio
					yOffset = float32(phaseProgress * peakHeight)
				} else {
					phaseProgress := (popupProgress - peakTimeRatio) / (1.0 - peakTimeRatio)
					yOffset = float32(peakHeight - (phaseProgress * (peakHeight - settleHeight)))
				}
			} else {
				yOffset = settleHeight
			}

			x := targetVM.X
			y := targetVM.Y - 20 - yOffset

			drawOpts := &text.DrawOptions{}
			drawOpts.GeoM.Scale(1.5, 1.5)
			drawOpts.GeoM.Translate(float64(x), float64(y))
			drawOpts.LayoutOptions = text.LayoutOptions{
				PrimaryAlign:   text.AlignCenter,
				SecondaryAlign: text.AlignCenter,
			}
			r, g, b, a := d.config.UI.Colors.Red.RGBA()
			cr := float32(r) / 0xffff
			cg := float32(g) / 0xffff
			cb := float32(b) / 0xffff
			ca := float32(a) / 0xffff
			drawOpts.DrawImageOptions.ColorScale.Scale(cr, cg, cb, ca)
			drawOpts.DrawImageOptions.ColorScale.ScaleAlpha(alpha)
			text.Draw(screen, fmt.Sprintf("-%d", anim.Result.OriginalDamage), d.font, drawOpts)
		}
	}
}

// drawPingAnimation は、指定された中心にレーダーのようなピングアニメーションを描画します。
func (d *UIAnimationDrawer) drawPingAnimation(screen *ebiten.Image, centerX, centerY float32, progress float64, expand bool) {
	if progress < 0 || progress > 1 {
		return
	}

	maxRadius := float32(40.0)
	pingColor := color.RGBA{R: 0, G: 255, B: 255, A: 255}

	var radius float32
	if expand {
		radius = maxRadius * float32(progress)
	} else {
		radius = maxRadius * (1.0 - float32(progress))
	}
	alpha := 1.0 - progress

	r, g, b, _ := pingColor.RGBA()
	finalColor := color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(255 * alpha),
	}

	vector.StrokeCircle(screen, centerX, centerY, radius, 2, finalColor, true)
	vector.StrokeCircle(screen, centerX, centerY, radius*0.4, 1.5, finalColor, true)
}