package main

import (
	"log"
	"math"
	"math/rand"

	"github.com/yohamta/donburi"
)

// DamageCalculator はダメージ計算に関連するロジックを担当します。
type DamageCalculator struct {
	world            donburi.World
	config           *Config
	partInfoProvider PartInfoProviderInterface
	gameDataManager  *GameDataManager
	rand             *rand.Rand
	logger           BattleLogger // 追加
}

// NewDamageCalculator は新しい DamageCalculator のインスタンスを生成します。
func NewDamageCalculator(world donburi.World, config *Config, pip PartInfoProviderInterface, gdm *GameDataManager, r *rand.Rand, logger BattleLogger) *DamageCalculator {
	return &DamageCalculator{world: world, config: config, partInfoProvider: pip, gameDataManager: gdm, rand: r, logger: logger}
}

// SetPartInfoProvider は PartInfoProvider の依存性を設定します。 // 削除
// func (dc *DamageCalculator) SetPartInfoProvider(pip *PartInfoProvider) {
// 	dc.partInfoProvider = pip
// }



// CalculateDamage はActionFormulaに基づいてダメージを計算します。
func (dc *DamageCalculator) CalculateDamage(attacker, target *donburi.Entry, actingPartDef *PartDefinition, selectedPartKey PartSlotKey, battleLogic *BattleLogic) (int, bool) {
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
func (dc *DamageCalculator) GenerateActionLog(attacker, target *donburi.Entry, actingPartDef *PartDefinition, targetPartDef *PartDefinition, damage int, isCritical bool, didHit bool) string {
	panic("GenerateActionLog should not be called directly. Use BattleLogger.")
}

// CalculateReducedDamage は防御成功時のダメージを計算します。
func (dc *DamageCalculator) CalculateReducedDamage(originalDamage int, targetEntry *donburi.Entry) int {
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
func (dc *DamageCalculator) GenerateActionLogDefense(target *donburi.Entry, defensePartDef *PartDefinition, damageDealt int, originalDamage int, isCritical bool) string {
	panic("GenerateActionLogDefense should not be called directly. Use BattleLogger.")
}
