package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// UIAnimationDrawer はUIアニメーションの描画に特化した構造体です。
type UIAnimationDrawer struct {
	config           *Config
	animationManager *BattleAnimationManager
	font             text.Face // 追加
}

// NewUIAnimationDrawer は新しいUIAnimationDrawerインスタンスを作成します。
func NewUIAnimationDrawer(config *Config, animationManager *BattleAnimationManager, font text.Face) *UIAnimationDrawer {
	return &UIAnimationDrawer{
		config:           config,
		animationManager: animationManager,
		font:             font, // 追加
	}
}

// Draw は現在のアニメーションを画面に描画します。
func (d *UIAnimationDrawer) Draw(screen *ebiten.Image, tick int, battlefieldVM BattlefieldViewModel, battlefieldWidget *BattlefieldWidget) {
	anim := d.animationManager.currentAnimation
	if anim == nil {
		return
	}

	progress := float64(tick - anim.StartTime)

	var attackerVM, targetVM *IconViewModel
	for _, icon := range battlefieldVM.Icons {
		if icon.EntryID == uint32(anim.Result.ActingEntry.Id()) {
			attackerVM = icon
		}
		if anim.Result.TargetEntry != nil && icon.EntryID == uint32(anim.Result.TargetEntry.Id()) {
			targetVM = icon
		}
	}

	// 攻撃者とターゲットが両方見つかった場合のみアニメーションを実行
	if attackerVM != nil && targetVM != nil {
		// アニメーションのタイミング設定
		const firstPingDuration = 30.0
		const secondPingDuration = 30.0
		const delayBetweenPings = 0.0 // 連続して再生

		// 1回目のピング（攻撃者） - 拡大
		if progress >= 0 && progress < firstPingDuration {
			pingProgress := progress / firstPingDuration
			battlefieldWidget.drawPingAnimation(screen, attackerVM.X, attackerVM.Y, pingProgress, true)
		}

		// 2回目のピング（ターゲット） - 縮小
		secondPingStart := firstPingDuration + delayBetweenPings
		if progress >= secondPingStart && progress < secondPingStart+secondPingDuration {
			pingProgress := (progress - secondPingStart) / secondPingDuration
			battlefieldWidget.drawPingAnimation(screen, targetVM.X, targetVM.Y, pingProgress, false)
		}

		// ダメージポップアップのアニメーション
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
				// Animation is in progress
				popupProgress := popupStartProgress / popupDuration
				if popupProgress < peakTimeRatio {
					// Phase 1: Rising
					phaseProgress := popupProgress / peakTimeRatio
					yOffset = float32(phaseProgress * peakHeight)
				} else {
					// Phase 2: Settling down
					phaseProgress := (popupProgress - peakTimeRatio) / (1.0 - peakTimeRatio)
					yOffset = float32(peakHeight - (phaseProgress * (peakHeight - settleHeight)))
				}
			} else {
				// Animation is finished, hold the settled position
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
			text.Draw(screen, fmt.Sprintf("-%d", anim.Result.OriginalDamage), d.font, drawOpts) // d.fontを使用
		}
	}
}
