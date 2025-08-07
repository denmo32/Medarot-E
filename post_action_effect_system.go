package main

import (
	"log"
	"medarot-ebiten/domain"

	"github.com/yohamta/donburi"
)

// PostActionEffectSystem は、アクション実行後の効果を処理するECSシステムです。
type PostActionEffectSystem struct {
	world              donburi.World
	statusEffectSystem *StatusEffectSystem
	gameDataManager    *GameDataManager          // 追加
	partInfoProvider   PartInfoProviderInterface // 追加
}

// NewPostActionEffectSystem は新しいPostActionEffectSystemのインスタンスを生成します。
func NewPostActionEffectSystem(world donburi.World, statusEffectSystem *StatusEffectSystem, gameDataManager *GameDataManager, partInfoProvider PartInfoProviderInterface) *PostActionEffectSystem {
	return &PostActionEffectSystem{
		world:              world,
		statusEffectSystem: statusEffectSystem,
		gameDataManager:    gameDataManager,
		partInfoProvider:   partInfoProvider,
	}
}

// Process は、ActionResultに基づいてアクション後の効果を処理します。
func (s *PostActionEffectSystem) Process(result *ActionResult) {
	if result == nil {
		return
	}

	// 1. 適用されるべきステータス効果を処理
	if len(result.AppliedEffects) > 0 {
		// 効果の適用対象を決定する（通常は行動者自身）
		// 将来的には効果ごとに対象を指定できるように拡張可能
		targetEntry := result.ActingEntry
		if targetEntry != nil {
			for _, effectData := range result.AppliedEffects {
				// effectDataの型に応じてApplyを呼び出す
				switch effect := effectData.(type) {
				case *ChargeStopEffectData:
					s.statusEffectSystem.Apply(targetEntry, effect, effect.DurationTurns)
				case *DamageOverTimeEffectData:
					s.statusEffectSystem.Apply(targetEntry, effect, effect.DurationTurns)
				case *TargetRandomEffectData:
					s.statusEffectSystem.Apply(targetEntry, effect, effect.DurationTurns)
				case *EvasionDebuffEffectData:
					s.statusEffectSystem.Apply(targetEntry, effect, 0) // Duration 0
				case *DefenseDebuffEffectData:
					s.statusEffectSystem.Apply(targetEntry, effect, 0) // Duration 0
				default:
					log.Printf("警告: 未知の適用効果タイプです: %T", effectData)
				}
			}
		}
	}

	// 2. ダメージ適用とパーツ破壊の状態遷移
	if result.TargetPartInstance != nil && result.DamageToApply > 0 {
		// ダメージを適用
		result.TargetPartInstance.CurrentArmor -= result.DamageToApply
		if result.TargetPartInstance.CurrentArmor <= 0 {
			result.TargetPartInstance.CurrentArmor = 0
			result.TargetPartInstance.IsBroken = true
		}
		result.IsTargetPartBroken = result.TargetPartInstance.IsBroken // 結果に反映

		// パーツ破壊時のログメッセージ
		if result.TargetPartInstance.IsBroken {
			settings := SettingsComponent.Get(result.TargetEntry)
			partDef, defFound := s.gameDataManager.GetPartDefinition(result.TargetPartInstance.DefinitionID)
			partNameForLog := "(不明パーツ)"
			if defFound {
				partNameForLog = partDef.PartName
			}
			log.Print(s.gameDataManager.Messages.FormatMessage("log_part_broken_notification", map[string]interface{}{
				"ordered_args": []interface{}{settings.Name, partNameForLog, result.TargetPartInstance.DefinitionID},
			}))

			// パーツ破壊時にバフを解除する
			s.partInfoProvider.RemoveBuffsFromSource(result.TargetEntry, result.TargetPartInstance)
		}
	}

	// 3. 頭部パーツ破壊による機能停止
	if result.TargetEntry != nil && result.IsTargetPartBroken && result.ActualHitPartSlot == domain.PartSlotHead {
		state := StateComponent.Get(result.TargetEntry)
		state.CurrentState = domain.StateBroken
	}

	// 4. 行動後のクリーンアップ
	if result.ActingEntry != nil && result.ActingEntry.HasComponent(ActiveEffectsComponent) {
		activeEffects := ActiveEffectsComponent.Get(result.ActingEntry)
		effectsToRemove := []interface{}{} // interface{}のスライスに変更
		for _, activeEffect := range activeEffects.Effects {
			if activeEffect.RemainingDur == 0 {
				effectsToRemove = append(effectsToRemove, activeEffect.EffectData) // EffectDataを直接追加
			}
		}
		for _, effectData := range effectsToRemove { // effectDataに名前変更
			s.statusEffectSystem.Remove(result.ActingEntry, effectData) // effectDataを渡す
		}
	}
}
