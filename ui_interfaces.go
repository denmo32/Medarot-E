package main

import (
	"image"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

// UIInterface はUIの描画とイベント処理に必要なメソッドを定義します。
type UIInterface interface {
	Update()
	Draw(screen *ebiten.Image, tick int, gameDataManager *GameDataManager)
	DrawBackground(screen *ebiten.Image)
	GetRootContainer() *widget.Container
	SetAnimation(anim *ActionAnimationData)
	IsAnimationFinished(tick int) bool
	ClearAnimation()
	GetCurrentAnimationResult() ActionResult
	ShowActionModal(vm ActionModalViewModel)
	HideActionModal()
	SetBattleUIState(battleUIState *BattleUIState, config *Config, battlefieldRect image.Rectangle, uiFactory *UIFactory)
	PostEvent(event UIEvent)
	IsActionModalVisible() bool
	GetActionTargetMap() map[PartSlotKey]ActionTarget
	SetCurrentTarget(entry *donburi.Entry)
	ClearCurrentTarget()
	GetBattlefieldWidgetRect() image.Rectangle
}
