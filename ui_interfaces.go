package main

import (
	"image"

	"medarot-ebiten/domain"
	"medarot-ebiten/ecs"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

// UIInterface はUIの描画とイベント処理に必要なメソッドを定義します。
type UIInterface interface {
	Update(tick int)
	Draw(screen *ebiten.Image, tick int, gameDataManager *GameDataManager)
	DrawBackground(screen *ebiten.Image)
	GetRootContainer() *widget.Container
	SetAnimation(anim *ecs.ActionAnimationData)
	ClearAnimation()
	ShowActionModal(vm ecs.ActionModalViewModel)
	HideActionModal()
	SetBattleUIState(battleUIState *ecs.BattleUIState, config *Config, battlefieldRect image.Rectangle, uiFactory *UIFactory)
	PostEvent(event UIEvent)
	IsActionModalVisible() bool
	IsAnimationFinished(tick int) bool
	GetCurrentAnimationResult() ecs.ActionResult
	GetActionTargetMap() map[domain.PartSlotKey]ecs.ActionTarget
	SetCurrentTarget(entityID donburi.Entity)
	ClearCurrentTarget()
	GetBattlefieldWidgetRect() image.Rectangle
	GetMessageDisplayManager() *UIMessageDisplayManager
	GetEventChannel() chan UIEvent
}
