package system

import (
	"log"
	"math"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
)

// DamageCalculator はダメージ計算に関連するロジックを担当します。
type DamageCalculator struct {
	world            donburi.World
	config           *data.Config
	partInfoProvider PartInfoProviderInterface
	gameDataManager  *data.GameDataManager
	rand             *rand.Rand
	logger           BattleLogger // core.BattleLogger を system.BattleLogger に変更
}

// NewDamageCalculator は新しい DamageCalculator のインスタンスを生成します。
func NewDamageCalculator(world donburi.World, config *data.Config, pip PartInfoProviderInterface, gdm *data.GameDataManager, r *rand.Rand, logger BattleLogger) *DamageCalculator { // core.BattleLogger を system.BattleLogger に変更
	return &DamageCalculator{world: world, config: config, partInfoProvider: pip, gameDataManager: gdm, rand: r, logger: logger}
}

// CalculateDamage はActionFormulaと防御の成否に基づいてダメージを計算します。
func (dc *DamageCalculator) CalculateDamage(attacker, target *donburi.Entry, actingPartDef *core.PartDefinition, selectedPartKey core.PartSlotKey, isDefended bool) (int, bool) {
	// 1. 計算式の取得
	formula, ok := dc.gameDataManager.Formulas[actingPartDef.Trait]
	if !ok || formula.ID == "" { // IDがゼロ値の場合は見つからなかったと判断
		log.Printf("警告: 特性 '%s' に対応する計算式が見つかりません。デフォルトを使用します。", actingPartDef.Trait)
		formula = dc.gameDataManager.Formulas[core.TraitShoot]
	}

	// 2. 基本パラメータの取得
	successRate := dc.partInfoProvider.GetSuccessRate(attacker, actingPartDef, selectedPartKey)
	power := float64(actingPartDef.Power)
	evasion := dc.partInfoProvider.GetEvasionRate(target)
	defenseRate := 0.0
	if isDefended {
		defenseRate = dc.partInfoProvider.GetDefenseRate(target)
	}

	// 特性による威力ボーナスを加算
	for _, bonus := range formula.PowerBonuses {
		power += dc.partInfoProvider.GetPartParameterValue(attacker, selectedPartKey, bonus.SourceParam) * bonus.Multiplier
	}

	// 3. クリティカル判定
	isCritical := false
	criticalChance := dc.config.Damage.Critical.BaseChance + (successRate * dc.config.Damage.Critical.SuccessRateFactor) + formula.CriticalRateBonus
	criticalChance = math.Max(criticalChance, dc.config.Damage.Critical.MinChance)
	criticalChance = math.Min(criticalChance, dc.config.Damage.Critical.MaxChance)

	if dc.rand.Intn(100) < int(criticalChance) {
		isCritical = true
		dc.logger.LogCriticalHit(component.SettingsComponent.Get(attacker).Name, criticalChance)
		// クリティカル時は回避度を0にする
		evasion = 0
		// TODO: クリティカルヒット時に防御度も無効化（0に）するかどうかは、将来の検討事項。
		// 現在は防御度が有効なまま。
	}

	// 4. 最終ダメージ計算
	damage := (successRate - evasion - defenseRate) / dc.config.Damage.DamageAdjustmentFactor + power
	// 乱数(±10%)
	randomFactor := 1.0 + (dc.rand.Float64()*0.2 - 0.1)
	damage *= randomFactor

	if damage < 1 {
		damage = 1
	}

	log.Printf("ダメージ計算 (%s): (%.1f - %.1f - %.1f) / %.1f + %.1f * %.2f = %d (Crit: %t, Defended: %t)",
		formula.ID, successRate, evasion, defenseRate, dc.config.Damage.DamageAdjustmentFactor, power, randomFactor, int(damage), isCritical, isDefended)

	return int(damage), isCritical
}