package ui

import (
	"image"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

// UIInterface はUIの描画とイベント処理に必要なメソッドを定義します。
type UIInterface interface {
	Update(tick int)
	Draw(screen *ebiten.Image, tick int, gameDataManager *data.GameDataManager)
	DrawBackground(screen *ebiten.Image)
	GetRootContainer() *widget.Container
	SetAnimation(anim *component.ActionAnimationData)
	ClearAnimation()
	ShowActionModal(vm core.ActionModalViewModel)
	HideActionModal()
	SetBattleUIState(battleUIState *BattleUIState, config *data.Config, battlefieldRect image.Rectangle, uiFactory *UIFactory)
	PostEvent(event UIEvent)
	IsActionModalVisible() bool
	IsAnimationFinished(tick int) bool
	GetCurrentAnimationResult() component.ActionResult
	GetActionTargetMap() map[core.PartSlotKey]core.ActionTarget
	SetCurrentTarget(entityID donburi.Entity)
	ClearCurrentTarget()
	GetBattlefieldWidgetRect() image.Rectangle
	GetMessageDisplayManager() *UIMessageDisplayManager
	GetEventChannel() chan UIEvent
	GetConfig() *data.Config
}
