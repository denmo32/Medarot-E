package main

import (
	"log"
	"math"

	"github.com/yohamta/donburi"
)

// DamageCalculator はダメージ計算に関連するロジックを担当します。
type DamageCalculator struct {
	world  donburi.World
	config *Config
	// partInfoProvider *PartInfoProvider // 削除
}

// NewDamageCalculator は新しい DamageCalculator のインスタンスを生成します。
func NewDamageCalculator(world donburi.World, config *Config) *DamageCalculator {
	return &DamageCalculator{world: world, config: config}
}

// SetPartInfoProvider は PartInfoProvider の依存性を設定します。 // 削除
// func (dc *DamageCalculator) SetPartInfoProvider(pip *PartInfoProvider) {
// 	dc.partInfoProvider = pip
// }

// ApplyDamage はパーツインスタンスにダメージを適用し、メダロットの状態を更新します。
func (dc *DamageCalculator) ApplyDamage(entry *donburi.Entry, partInst *PartInstanceData, damage int, battleLogic *BattleLogic) {
	if damage < 0 {
		damage = 0
	}
	partInst.CurrentArmor -= damage
	if partInst.CurrentArmor <= 0 {
		partInst.CurrentArmor = 0
		partInst.IsBroken = true
		settings := SettingsComponent.Get(entry)
		// Get PartDefinition for logging PartName
		partDef, defFound := battleLogic.GetPartInfoProvider().gameDataManager.GetPartDefinition(partInst.DefinitionID)
		partNameForLog := "(不明パーツ)"
		if defFound {
			partNameForLog = partDef.PartName
		}
		log.Print(battleLogic.GetPartInfoProvider().gameDataManager.Messages.FormatMessage("log_part_broken_notification", map[string]interface{}{
			"ordered_args": []interface{}{settings.Name, partNameForLog, partInst.DefinitionID},
		}))

		// パーツ破壊時にバフを解除する
		battleLogic.GetPartInfoProvider().RemoveBuffsFromSource(entry, partInst)
	}
}

// CalculateDamage はActionFormulaに基づいてダメージを計算します。
func (dc *DamageCalculator) CalculateDamage(attacker, target *donburi.Entry, actingPartDef *PartDefinition, selectedPartKey PartSlotKey, battleLogic *BattleLogic) (int, bool) {
	// 1. 計算式の取得
	formula, ok := FormulaManager[actingPartDef.Trait]
	if !ok {
		log.Printf("警告: 特性 '%s' に対応する計算式が見つかりません。デフォルトを使用します。", actingPartDef.Trait)
		formula = FormulaManager[TraitShoot]
	}

	// 2. 基本パラメータの取得
	successRate := battleLogic.GetPartInfoProvider().GetSuccessRate(attacker, actingPartDef, selectedPartKey)
	power := float64(actingPartDef.Power)

	// 特性による威力ボーナスを加算
	if formula != nil {
		for _, bonus := range formula.PowerBonuses {
			power += battleLogic.GetPartInfoProvider().GetPartParameterValue(attacker, selectedPartKey, bonus.SourceParam) * bonus.Multiplier
		}
	}
	evasion := battleLogic.GetPartInfoProvider().GetEvasionRate(target)

	// クリティカル判定
	isCritical := false
	criticalChance := dc.config.Balance.Damage.Critical.BaseChance + (successRate * dc.config.Balance.Damage.Critical.SuccessRateFactor) + formula.CriticalRateBonus

	// クリティカル率の上下限を適用
	criticalChance = math.Max(criticalChance, dc.config.Balance.Damage.Critical.MinChance)
	criticalChance = math.Min(criticalChance, dc.config.Balance.Damage.Critical.MaxChance)

	if globalRand.Intn(100) < int(criticalChance) {
		isCritical = true
		log.Printf("%s の攻撃がクリティカル！ (確率: %.1f%%)", SettingsComponent.Get(attacker).Name, criticalChance)
		// クリティカル時は回避度を0にする
		evasion = 0
	}

	// 5. 最終ダメージ計算
	damage := (successRate-evasion)/dc.config.Balance.Damage.DamageAdjustmentFactor + power
	// 乱数(±10%)
	randomFactor := 1.0 + (globalRand.Float64()*0.2 - 0.1)
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
func (dc *DamageCalculator) GenerateActionLog(attacker, target *donburi.Entry, actingPartDef *PartDefinition, targetPartDef *PartDefinition, damage int, isCritical bool, didHit bool, battleLogic *BattleLogic) string {
	attackerSettings := SettingsComponent.Get(attacker)
	targetSettings := SettingsComponent.Get(target)
	skillName := "(不明なスキル)"
	if actingPartDef != nil {
		skillName = actingPartDef.PartName
	}

	if !didHit {
		return battleLogic.GetPartInfoProvider().gameDataManager.Messages.FormatMessage("attack_miss", map[string]interface{}{
			"attacker_name": attackerSettings.Name,
			"skill_name":    skillName,
			"target_name":   targetSettings.Name,
		})
	}

	targetPartNameStr := "(不明部位)"
	if targetPartDef != nil {
		targetPartNameStr = targetPartDef.PartName
	}

	params := map[string]interface{}{
		"attacker_name":    attackerSettings.Name,
		"skill_name":       skillName,
		"target_name":      targetSettings.Name,
		"target_part_name": targetPartNameStr,
		"damage":           damage,
	}

	if isCritical {
		return battleLogic.GetPartInfoProvider().gameDataManager.Messages.FormatMessage("critical_hit", params)
	}
	return battleLogic.GetPartInfoProvider().gameDataManager.Messages.FormatMessage("attack_hit", params)
}

// CalculateReducedDamage は防御成功時のダメージを計算します。
func (dc *DamageCalculator) CalculateReducedDamage(originalDamage int, targetEntry *donburi.Entry, battleLogic *BattleLogic) int {
	// ダメージ軽減ロジック: ダメージ = 元ダメージ - 脚部パーツの防御力
	defenseValue := battleLogic.GetPartInfoProvider().GetPartParameterValue(targetEntry, PartSlotLegs, Defense)
	reducedDamage := originalDamage - int(defenseValue)
	if reducedDamage < 1 {
		reducedDamage = 1 // 最低でも1ダメージは保証
	}
	log.Printf("防御成功！ ダメージ軽減: %d -> %d (脚部パーツ防御力: %d)", originalDamage, reducedDamage, int(defenseValue))
	return reducedDamage
}

// GenerateActionLogDefense は防御時のアクションログを生成します。
// defensePartDef は防御に使用されたパーツの定義
func (dc *DamageCalculator) GenerateActionLogDefense(target *donburi.Entry, defensePartDef *PartDefinition, damageDealt int, originalDamage int, isCritical bool, battleLogic *BattleLogic) string {
	targetSettings := SettingsComponent.Get(target)
	defensePartNameStr := "(不明なパーツ)"
	if defensePartDef != nil {
		defensePartNameStr = defensePartDef.PartName
	}

	params := map[string]interface{}{
		"target_name":       targetSettings.Name,
		"defense_part_name": defensePartNameStr,
		"original_damage":   originalDamage,
		"actual_damage":     damageDealt,
	}

	if isCritical {
		return battleLogic.GetPartInfoProvider().gameDataManager.Messages.FormatMessage("defense_success_critical", params)
	}
	return battleLogic.GetPartInfoProvider().gameDataManager.Messages.FormatMessage("defense_success", params)
}
