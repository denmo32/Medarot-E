package main

import (
	"image"

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
	SetAnimation(anim *ActionAnimationData)
	ClearAnimation()
	ShowActionModal(vm ActionModalViewModel)
	HideActionModal()
	SetBattleUIState(battleUIState *BattleUIState, config *Config, battlefieldRect image.Rectangle, uiFactory *UIFactory)
	PostEvent(event UIEvent)
	IsActionModalVisible() bool
	IsAnimationFinished(tick int) bool
	GetCurrentAnimationResult() ActionResult
	GetActionTargetMap() map[PartSlotKey]ActionTarget
	SetCurrentTarget(entityID donburi.Entity)
	ClearCurrentTarget()
	GetBattlefieldWidgetRect() image.Rectangle
	GetMessageDisplayManager() *UIMessageDisplayManager
	GetEventChannel() chan UIEvent
}
