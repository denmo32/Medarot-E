package main

import (
	"log"

	"medarot-ebiten/core"
	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// StatusEffectSystem はステータス効果の適用、更新、解除を管理します。
type StatusEffectSystem struct {
	world                  donburi.World
	battleDamageCalculator *DamageCalculator // 追加
}

// NewStatusEffectSystem は新しいStatusEffectSystemのインスタンスを生成します。
func NewStatusEffectSystem(world donburi.World, damageCalculator *DamageCalculator) *StatusEffectSystem {
	return &StatusEffectSystem{
		world:                  world,
		battleDamageCalculator: damageCalculator,
	}
}

// Apply はエンティティにステータス効果を適用します。
func (s *StatusEffectSystem) Apply(entry *donburi.Entry, effectData interface{}, duration int) {
	// log.Printf("Applying effect to %s", SettingsComponent.Get(entry).Name) // Description()がなくなったため汎用的なログに

	// 効果の持続時間を管理するコンポーネントを追加
	if !entry.HasComponent(component.ActiveEffectsComponent) {
		donburi.Add(entry, component.ActiveEffectsComponent, &core.ActiveEffects{
			Effects: make([]*core.ActiveStatusEffectData, 0),
		})
	}
	activeEffects := component.ActiveEffectsComponent.Get(entry)
	activeEffects.Effects = append(activeEffects.Effects, &core.ActiveStatusEffectData{
		EffectData:   effectData,
		RemainingDur: duration,
	})
}

// Remove はエンティティからステータス効果を解除します。
func (s *StatusEffectSystem) Remove(entry *donburi.Entry, effectData interface{}) {
	// log.Printf("Removing effect from %s", SettingsComponent.Get(entry).Name) // Description()がなくなったため汎用的なログに

	if entry.HasComponent(component.ActiveEffectsComponent) {
		activeEffects := component.ActiveEffectsComponent.Get(entry)
		newEffects := make([]*core.ActiveStatusEffectData, 0)
		for _, activeEffect := range activeEffects.Effects {
			if activeEffect.EffectData != effectData {
				newEffects = append(newEffects, activeEffect)
			}
		}
		activeEffects.Effects = newEffects
	}
}

// Update は毎フレーム呼び出され、効果の持続時間を更新し、期限切れの効果を削除します。
func (s *StatusEffectSystem) Update() {
	query.NewQuery(filter.Contains(component.ActiveEffectsComponent)).Each(s.world, func(entry *donburi.Entry) {
		activeEffects := component.ActiveEffectsComponent.Get(entry)
		effectsToRemove := make([]*core.ActiveStatusEffectData, 0)

		for _, effectData := range activeEffects.Effects {
			if effectData.RemainingDur > 0 {
				effectData.RemainingDur--
			}

			// 効果のタイプに応じて処理を分岐
			switch effect := effectData.EffectData.(type) {
			case *core.DamageOverTimeEffectData:
				// 継続ダメージの処理
				if DurationDamageOverTimeEffect(effect) > 0 { // Duration()が0より大きい場合のみダメージを与える
					// ダメージ計算ロジックを呼び出す
					// ApplyDamageはBattleDamageCalculatorのメソッドではないため、直接呼び出す
					// ここでは簡略化のため、直接ダメージを適用するロジックを記述
					// 実際のゲームでは、BattleDamageCalculatorの適切なメソッドを呼び出すか、
					// ダメージ適用ロジックをStatusEffectSystemに持たせるべきです。
					targetParts := component.PartsComponent.Get(entry)
					if targetParts != nil && len(targetParts.Map) > 0 {
						// 適当なパーツにダメージを適用する例
						for _, partInst := range targetParts.Map {
							partInst.CurrentArmor -= effect.DamagePerTurn
							if partInst.CurrentArmor < 0 {
								partInst.CurrentArmor = 0
							}
							log.Printf("%s のパーツに継続ダメージ %d を与えた。残りアーマー: %d", component.SettingsComponent.Get(entry).Name, effect.DamagePerTurn, partInst.CurrentArmor)
							break // 最初のパーツにダメージを与えたら終了
						}
					}
					log.Printf("%s は継続ダメージ %d を受けた。", component.SettingsComponent.Get(entry).Name, effect.DamagePerTurn)
				}
			case *core.ChargeStopEffectData:
				// チャージ停止効果はChargeInitiationSystemで処理されるため、ここでは何もしない
			case *core.TargetRandomEffectData:
				// ターゲットランダム化効果はBattleTargetSelectorで処理されるため、ここでは何もしない
			case *core.EvasionDebuffEffectData:
				// 回避率デバフはPartInfoProviderInterfaceで処理されるため、ここでは何もしない
			case *core.DefenseDebuffEffectData:
				// 防御力デバフはPartInfoProviderInterfaceで処理されるため、ここでは何もしない
			default:
				log.Printf("未対応のステータス効果データ型です: %T", effectData.EffectData)
			}

			// 持続時間が0になった効果を削除対象としてマーク
			if effectData.RemainingDur == 0 {
				effectsToRemove = append(effectsToRemove, effectData)
			}
		}

		// 削除対象の効果をActiveEffectsComponentから除去
		for _, effectToRemove := range effectsToRemove {
			// 効果の解除ロジックを呼び出す
			switch effect := effectToRemove.EffectData.(type) {
			case *core.ChargeStopEffectData:
				RemoveChargeStopEffect(s.world, entry, effect)
			case *core.DamageOverTimeEffectData:
				RemoveDamageOverTimeEffect(s.world, entry, effect)
			case *core.TargetRandomEffectData:
				RemoveTargetRandomEffect(s.world, entry, effect)
			case *core.EvasionDebuffEffectData:
				RemoveEvasionDebuffEffect(s.world, entry, effect)
			case *core.DefenseDebuffEffectData:
				RemoveDefenseDebuffEffect(s.world, entry, effect)
			default:
				log.Printf("未対応のステータス効果データ型です（解除時）: %T", effectToRemove.EffectData)
			}
			activeEffects.Effects = removeEffect(activeEffects.Effects, effectToRemove)
		}
	})
}

// removeEffect はスライスから指定された効果を削除するヘルパー関数です。
func removeEffect(slice []*core.ActiveStatusEffectData, element *core.ActiveStatusEffectData) []*core.ActiveStatusEffectData {
	for i, v := range slice {
		if v == element {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
