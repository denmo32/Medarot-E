package system

import (
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
)

// HitCalculator は命中・回避・防御判定に関連するロジックを担当します。
type HitCalculator struct {
	world            donburi.World
	config           *data.Config
	partInfoProvider PartInfoProviderInterface
	rand             *rand.Rand
	logger           BattleLogger // 追加
}

// NewHitCalculator は新しい HitCalculator のインスタンスを生成します。
func NewHitCalculator(world donburi.World, config *data.Config, pip PartInfoProviderInterface, r *rand.Rand, logger BattleLogger) *HitCalculator {
	return &HitCalculator{world: world, config: config, partInfoProvider: pip, rand: r, logger: logger}
}

// CalculateHit は新しいルールに基づいて命中判定を行います。
func (hc *HitCalculator) CalculateHit(attacker, target *donburi.Entry, partDef *core.PartDefinition, selectedPartKey core.PartSlotKey) bool {
	// 攻撃側の成功度
	successRate := hc.partInfoProvider.GetSuccessRate(attacker, partDef, selectedPartKey)

	// チームバフによる成功度の上昇
	successRate *= hc.partInfoProvider.GetTeamAccuracyBuffMultiplier(attacker)

	// 防御側の回避度
	evasion := hc.partInfoProvider.GetEvasionRate(target)

	// 命中確率 = 基準値 + (成功度 - 回避度)
	// config.Balance.Hit を config.Hit に変更
	chance := hc.config.Hit.BaseChance + (successRate - evasion)

	// 確率の上下限を適用
	// config.Balance.Hit を config.Hit に変更
	if chance < hc.config.Hit.MinChance {
		chance = hc.config.Hit.MinChance
	}
	if chance > hc.config.Hit.MaxChance {
		chance = hc.config.Hit.MaxChance
	}

	roll := hc.rand.Intn(100)
	hc.logger.LogHitCheck(component.SettingsComponent.Get(attacker).Name, component.SettingsComponent.Get(target).Name, chance, successRate, evasion, roll)
	return float64(roll) < chance
}

// CalculateDefense は防御の成否を判定します。
func (hc *HitCalculator) CalculateDefense(attacker, target *donburi.Entry, actingPartDef *core.PartDefinition, selectedPartKey core.PartSlotKey) bool {
	// 攻撃側の成功度
	successRate := hc.partInfoProvider.GetSuccessRate(attacker, actingPartDef, selectedPartKey)

	// 防御側の防御度
	defenseRate := hc.partInfoProvider.GetDefenseRate(target)

	// 防御成功確率 = 基準値 + (防御度 - 成功度)
	// config.Balance.Defense を config.Defense に変更
	chance := hc.config.Defense.BaseChance + (defenseRate - successRate)

	// 確率の上下限を適用
	// config.Balance.Defense を config.Defense に変更
	if chance < hc.config.Defense.MinChance {
		chance = hc.config.Defense.MinChance
	}
	if chance > hc.config.Defense.MaxChance {
		chance = hc.config.Defense.MaxChance
	}

	roll := hc.rand.Intn(100)
	hc.logger.LogDefenseCheck(component.SettingsComponent.Get(target).Name, defenseRate, successRate, chance, roll)
	return float64(roll) < chance
}