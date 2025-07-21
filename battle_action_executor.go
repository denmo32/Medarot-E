package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// ActionExecutor はアクションの実行に関するロジックをカプセル化します。
type ActionExecutor struct {
	world                  donburi.World
	battleLogic            *BattleLogic
	gameConfig             *Config
	statusEffectSystem     *StatusEffectSystem
	postActionEffectSystem *PostActionEffectSystem // 新しく追加したシステム
	handlers               map[Trait]TraitActionHandler
	weaponHandlers         map[WeaponType]WeaponTypeEffectHandler // WeaponTypeごとのハンドラを追加

}

// NewActionExecutor は新しいActionExecutorのインスタンスを生成します。
func NewActionExecutor(world donburi.World, battleLogic *BattleLogic, gameConfig *Config) *ActionExecutor {
	statusEffectSystem := NewStatusEffectSystem(world)                             // Create once
	postActionEffectSystem := NewPostActionEffectSystem(world, statusEffectSystem) // Use the created instance

	return &ActionExecutor{
		world:                  world,
		battleLogic:            battleLogic,
		gameConfig:             gameConfig,
		statusEffectSystem:     statusEffectSystem,     // Assign the created instance
		postActionEffectSystem: postActionEffectSystem, // Assign the new system

		handlers: map[Trait]TraitActionHandler{
			TraitShoot:    &BaseAttackHandler{},
			TraitAim:      &BaseAttackHandler{},
			TraitStrike:   &BaseAttackHandler{},
			TraitBerserk:  &BaseAttackHandler{},
			TraitSupport:  &SupportTraitExecutor{},
			TraitObstruct: &ObstructTraitExecutor{},
		},
		weaponHandlers: map[WeaponType]WeaponTypeEffectHandler{
			// 将来の拡張に備え、ここにハンドラを登録していく
			// 例: WeaponTypeThunder: &ThunderEffectHandler{},
			// 例: WeaponTypeMelt:    &MeltEffectHandler{},
		},
	}
}

// ExecuteAction は単一のアクションを実行し、その結果を返します。
func (e *ActionExecutor) ExecuteAction(actingEntry *donburi.Entry) ActionResult {
	intent := ActionIntentComponent.Get(actingEntry)
	partsComp := PartsComponent.Get(actingEntry)
	actingPartInst := partsComp.Map[intent.SelectedPartKey]
	actingPartDef, _ := e.battleLogic.GetPartInfoProvider().gameDataManager.GetPartDefinition(actingPartInst.DefinitionID)

	handler, ok := e.handlers[actingPartDef.Trait]
	if !ok {
		log.Printf("未対応のTraitです: %s", actingPartDef.Trait)
		return ActionResult{
			ActingEntry:  actingEntry,
			ActionDidHit: false,
		}
	}

	actionResult := handler.Execute(actingEntry, e.world, intent, e.battleLogic, e.gameConfig, actingPartDef)

	// チャージ時に生成された保留中の効果をActionResultにコピー
	if len(intent.PendingEffects) > 0 {
		actionResult.AppliedEffects = append(actionResult.AppliedEffects, intent.PendingEffects...)
		// 保留中の効果をクリア
		intent.PendingEffects = nil
	}

	// WeaponType に基づく追加効果を適用 (Traitの処理から独立)
	if weaponHandler, ok := e.weaponHandlers[actingPartDef.WeaponType]; ok {
		weaponHandler.ApplyEffect(&actionResult, e.world, e.battleLogic, actingPartDef)
	}

	// アクション後の共通処理を実行
	e.postActionEffectSystem.Process(&actionResult)

	return actionResult
}