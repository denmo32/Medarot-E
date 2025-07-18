package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// HitCalculator は命中・回避・防御判定に関連するロジックを担当します。
type HitCalculator struct {
	world            donburi.World
	config           *Config
	partInfoProvider *PartInfoProvider // 後で初期化
}

// NewHitCalculator は新しい HitCalculator のインスタンスを生成します。
func NewHitCalculator(world donburi.World, config *Config) *HitCalculator {
	return &HitCalculator{world: world, config: config}
}

// SetPartInfoProvider は PartInfoProvider の依存性を設定します。
func (hc *HitCalculator) SetPartInfoProvider(pip *PartInfoProvider) {
	hc.partInfoProvider = pip
}

// CalculateHit は新しいルールに基づいて命中判定を行います。
func (hc *HitCalculator) CalculateHit(attacker, target *donburi.Entry, partDef *PartDefinition) bool {
	// 攻撃側の成功度
	successRate := hc.partInfoProvider.GetSuccessRate(attacker, partDef)

	// チームバフによる成功度の上昇
	successRate *= hc.partInfoProvider.GetTeamAccuracyBuffMultiplier(attacker)

	// 防御側の回避度
	evasion := hc.partInfoProvider.GetEvasionRate(target)

	// 命中確率 = 基準値 + (成功度 - 回避度)
	chance := hc.config.Balance.Hit.BaseChance + (successRate - evasion)

	// 確率の上下限を適用
	if chance < hc.config.Balance.Hit.MinChance {
		chance = hc.config.Balance.Hit.MinChance
	}
	if chance > hc.config.Balance.Hit.MaxChance {
		chance = hc.config.Balance.Hit.MaxChance
	}

	roll := globalRand.Intn(100)
	log.Print(GlobalGameDataManager.Messages.FormatMessage("log_hit_roll", map[string]interface{}{
		"ordered_args": []interface{}{SettingsComponent.Get(attacker).Name, SettingsComponent.Get(target).Name, chance, successRate, evasion, roll},
	}))
	return float64(roll) < chance
}

// CalculateDefense は防御の成否を判定します。
func (hc *HitCalculator) CalculateDefense(attacker, target *donburi.Entry, actingPartDef *PartDefinition) bool {
	// 攻撃側の成功度
	successRate := hc.partInfoProvider.GetSuccessRate(attacker, actingPartDef)

	// 防御側の防御度
	defenseRate := hc.partInfoProvider.GetDefenseRate(target)

	// 防御成功確率 = 基準値 + (防御度 - 成功度)
	chance := hc.config.Balance.Defense.BaseChance + (defenseRate - successRate)

	// 確率の上下限を適用
	if chance < hc.config.Balance.Defense.MinChance {
		chance = hc.config.Balance.Defense.MinChance
	}
	if chance > hc.config.Balance.Defense.MaxChance {
		chance = hc.config.Balance.Defense.MaxChance
	}

	roll := globalRand.Intn(100)
	log.Print(GlobalGameDataManager.Messages.FormatMessage("log_defense_roll", map[string]interface{}{
		"ordered_args": []interface{}{SettingsComponent.Get(target).Name, defenseRate, successRate, chance, roll},
	}))
	return float64(roll) < chance
}
