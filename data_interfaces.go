package main

import (
	"image"
	
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/yohamta/donburi"
)

// TargetingStrategy はAIのターゲット選択アルゴリズムをカプセル化するインターフェースです。
type TargetingStrategy interface {
	SelectTarget(
		world donburi.World,
		actingEntry *donburi.Entry,
		battleLogic *BattleLogic,
	) (*donburi.Entry, PartSlotKey)
}

// TraitActionHandler はカテゴリ固有のアクション処理全体をカプセル化します。
// ActionResultを返し、副作用をなくします。
type TraitActionHandler interface {
	Execute(
		actingEntry *donburi.Entry,
		world donburi.World,
		intent *ActionIntent,
		battleLogic *BattleLogic,
		gameConfig *Config,
		actingPartDef *PartDefinition,
	) ActionResult
}

// WeaponTypeEffectHandler は weapon_type 固有の追加効果を処理します。
// ActionResult を受け取り、デバフ付与などの副作用を適用します。
type WeaponTypeEffectHandler interface {
	ApplyEffect(result *ActionResult, world donburi.World, battleLogic *BattleLogic, actingPartDef *PartDefinition)
}

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
