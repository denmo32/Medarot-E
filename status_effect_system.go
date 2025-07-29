package main

import (
	"log"
	"math"
	"math/rand" // 追加

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// DamageCalculator はダメージ計算に関連するロジックを担当します。
type BattleDamageCalculator struct { // 名前をDamageCalculatorからBattleDamageCalculatorに変更
	world            donburi.World
	config           *Config
	partInfoProvider PartInfoProviderInterface
	gameDataManager  *GameDataManager
	rand             *rand.Rand
	logger           BattleLogger
}

// NewBattleDamageCalculator は新しい BattleDamageCalculator のインスタンスを生成します。
func NewBattleDamageCalculator(world donburi.World, config *Config, pip PartInfoProviderInterface, gdm *GameDataManager, r *rand.Rand, logger BattleLogger) *BattleDamageCalculator {
	return &BattleDamageCalculator{world: world, config: config, partInfoProvider: pip, gameDataManager: gdm, rand: r, logger: logger}
}

// CalculateDamage はActionFormulaに基づいてダメージを計算します。
func (dc *BattleDamageCalculator) CalculateDamage(attacker, target *donburi.Entry, actingPartDef *PartDefinition, selectedPartKey PartSlotKey, battleLogic *BattleLogic) (int, bool) {
	// 1. 計算式の取得
	formula, ok := dc.gameDataManager.Formulas[actingPartDef.Trait]
	if !ok || formula.ID == "" { // IDがゼロ値の場合は見つからなかったと判断
		log.Printf("警告: 特性 '%s' に対応する計算式が見つかりません。デフォルトを使用します。", actingPartDef.Trait)
		formula = dc.gameDataManager.Formulas[TraitShoot]
	}

	// 2. 基本パラメータの取得
	successRate := dc.partInfoProvider.GetSuccessRate(attacker, actingPartDef, selectedPartKey)
	power := float64(actingPartDef.Power)

	// 特性による威力ボーナスを加算
	// formula は常に有効な ActionFormula 構造体なので nil チェックは不要
	for _, bonus := range formula.PowerBonuses {
		power += dc.partInfoProvider.GetPartParameterValue(attacker, selectedPartKey, bonus.SourceParam) * bonus.Multiplier
	}
	evasion := dc.partInfoProvider.GetEvasionRate(target)

	// クリティカル判定
	isCritical := false
	criticalChance := dc.config.Balance.Damage.Critical.BaseChance + (successRate * dc.config.Balance.Damage.Critical.SuccessRateFactor) + formula.CriticalRateBonus

	// クリティカル率の上下限を適用
	criticalChance = math.Max(criticalChance, dc.config.Balance.Damage.Critical.MinChance)
	criticalChance = math.Min(criticalChance, dc.config.Balance.Damage.Critical.MaxChance)

	if dc.rand.Intn(100) < int(criticalChance) {
		isCritical = true
		dc.logger.LogCriticalHit(SettingsComponent.Get(attacker).Name, criticalChance)
		// クリティカル時は回避度を0にする
		evasion = 0
	}

	// 5. 最終ダメージ計算
	damage := (successRate-evasion)/dc.config.Balance.Damage.DamageAdjustmentFactor + power
	// 乱数(±10%)
	randomFactor := 1.0 + (dc.rand.Float64()*0.2 - 0.1)
	damage *= randomFactor

	if damage < 1 {
		damage = 1
	}

	log.Printf("ダメージ計算 (%s): (%.1f - %.1f) / %.1f + %.1f * %.2f = %d (Crit: %t)",
		formula.ID, successRate, evasion, dc.config.Balance.Damage.DamageAdjustmentFactor, power, randomFactor, int(damage), isCritical)

	return int(damage), isCritical
}

// GenerateActionLog は行動の結果ログを生成します。
// targetPartDef はダメージを受けたパーツの定義 (nilの場合あり)
// actingPartDef は攻撃に使用されたパーツの定義
func (dc *BattleDamageCalculator) GenerateActionLog(attacker, target *donburi.Entry, actingPartDef *PartDefinition, targetPartDef *PartDefinition, damage int, isCritical bool, didHit bool) string {
	panic("GenerateActionLog should not be called directly. Use BattleLogger.")
}

// CalculateReducedDamage は防御成功時のダメージを計算します。
func (dc *BattleDamageCalculator) CalculateReducedDamage(originalDamage int, targetEntry *donburi.Entry) int {
	// ダメージ軽減ロジック: ダメージ = 元ダメージ - 脚部パーツの防御力
	defenseValue := dc.partInfoProvider.GetPartParameterValue(targetEntry, PartSlotLegs, Defense)
	reducedDamage := originalDamage - int(defenseValue)
	if reducedDamage < 1 {
		reducedDamage = 1 // 最低でも1ダメージは保証
	}
	log.Printf("防御成功！ ダメージ軽減: %d -> %d (脚部パーツ防御力: %d)", originalDamage, reducedDamage, int(defenseValue))
	return reducedDamage
}

// GenerateActionLogDefense は防御時のアクションログを生成します。
// defensePartDef は防御に使用されたパーツの定義
func (dc *BattleDamageCalculator) GenerateActionLogDefense(target *donburi.Entry, defensePartDef *PartDefinition, damageDealt int, originalDamage int, isCritical bool) string {
	panic("GenerateActionLogDefense should not be called directly. Use BattleLogger.")
}

// StatusEffectSystem はステータス効果の適用、更新、解除を管理します。
type StatusEffectSystem struct {
	world                  donburi.World
	battleDamageCalculator *BattleDamageCalculator // 追加
}

// NewStatusEffectSystem は新しいStatusEffectSystemのインスタンスを生成します。
func NewStatusEffectSystem(world donburi.World, bdc *BattleDamageCalculator) *StatusEffectSystem {
	return &StatusEffectSystem{
		world:                  world,
		battleDamageCalculator: bdc,
	}
}

// Apply はエンティティにステータス効果を適用します。
func (s *StatusEffectSystem) Apply(entry *donburi.Entry, effectData interface{}, duration int) {
	// log.Printf("Applying effect to %s", SettingsComponent.Get(entry).Name) // Description()がなくなったため汎用的なログに

	// 効果の持続時間を管理するコンポーネントを追加
	if !entry.HasComponent(ActiveEffectsComponent) {
		donburi.Add(entry, ActiveEffectsComponent, &ActiveEffects{
			Effects: make([]*ActiveStatusEffectData, 0),
		})
	}
	activeEffects := ActiveEffectsComponent.Get(entry)
	activeEffects.Effects = append(activeEffects.Effects, &ActiveStatusEffectData{
		EffectData:   effectData,
		RemainingDur: duration,
	})
}

// Remove はエンティティからステータス効果を解除します。
func (s *StatusEffectSystem) Remove(entry *donburi.Entry, effectData interface{}) {
	// log.Printf("Removing effect from %s", SettingsComponent.Get(entry).Name) // Description()がなくなったため汎用的なログに

	if entry.HasComponent(ActiveEffectsComponent) {
		activeEffects := ActiveEffectsComponent.Get(entry)
		newEffects := make([]*ActiveStatusEffectData, 0)
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
	query.NewQuery(filter.Contains(ActiveEffectsComponent)).Each(s.world, func(entry *donburi.Entry) {
		activeEffects := ActiveEffectsComponent.Get(entry)
		effectsToRemove := make([]*ActiveStatusEffectData, 0)

		for _, effectData := range activeEffects.Effects {
			if effectData.RemainingDur > 0 {
				effectData.RemainingDur--
			}

			// 効果のタイプに応じて処理を分岐
			switch effect := effectData.EffectData.(type) {
			case *DamageOverTimeEffectData:
				// 継続ダメージの処理
				if DurationDamageOverTimeEffect(effect) > 0 { // Duration()が0より大きい場合のみダメージを与える
					// ダメージ計算ロジックを呼び出す
					// ApplyDamageはBattleDamageCalculatorのメソッドではないため、直接呼び出す
					// ここでは簡略化のため、直接ダメージを適用するロジックを記述
					// 実際のゲームでは、BattleDamageCalculatorの適切なメソッドを呼び出すか、
					// ダメージ適用ロジックをStatusEffectSystemに持たせるべきです。
					targetParts := PartsComponent.Get(entry)
					if targetParts != nil && len(targetParts.Map) > 0 {
						// 適当なパーツにダメージを適用する例
						for _, partInst := range targetParts.Map {
							partInst.CurrentArmor -= effect.DamagePerTurn
							if partInst.CurrentArmor < 0 {
								partInst.CurrentArmor = 0
							}
							log.Printf("%s のパーツに継続ダメージ %d を与えた。残りアーマー: %d", SettingsComponent.Get(entry).Name, effect.DamagePerTurn, partInst.CurrentArmor)
							break // 最初のパーツにダメージを与えたら終了
						}
					}
					log.Printf("%s は継続ダメージ %d を受けた。", SettingsComponent.Get(entry).Name, effect.DamagePerTurn)
				}
			case *ChargeStopEffectData:
				// チャージ停止効果はChargeInitiationSystemで処理されるため、ここでは何もしない
			case *TargetRandomEffectData:
				// ターゲットランダム化効果はBattleTargetSelectorで処理されるため、ここでは何もしない
			case *EvasionDebuffEffectData:
				// 回避率デバフはPartInfoProviderInterfaceで処理されるため、ここでは何もしない
			case *DefenseDebuffEffectData:
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
			case *ChargeStopEffectData:
				RemoveChargeStopEffect(s.world, entry, effect)
			case *DamageOverTimeEffectData:
				RemoveDamageOverTimeEffect(s.world, entry, effect)
			case *TargetRandomEffectData:
				RemoveTargetRandomEffect(s.world, entry, effect)
			case *EvasionDebuffEffectData:
				RemoveEvasionDebuffEffect(s.world, entry, effect)
			case *DefenseDebuffEffectData:
				RemoveDefenseDebuffEffect(s.world, entry, effect)
			default:
				log.Printf("未対応のステータス効果データ型です（解除時）: %T", effectToRemove.EffectData)
			}
			activeEffects.Effects = removeEffect(activeEffects.Effects, effectToRemove)
		}
	})
}

// removeEffect はスライスから指定された効果を削除するヘルパー関数です。
func removeEffect(slice []*ActiveStatusEffectData, element *ActiveStatusEffectData) []*ActiveStatusEffectData {
	for i, v := range slice {
		if v == element {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
