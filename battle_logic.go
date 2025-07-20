package main

import (
	"github.com/yohamta/donburi"
)

// BattleLogic は戦闘関連のすべての計算ロジックをカプセル化します。
type BattleLogic struct {
	damageCalculator *DamageCalculator
	hitCalculator    *HitCalculator
	targetSelector   *TargetSelector
	partInfoProvider *PartInfoProvider
}

// GetDamageCalculator は DamageCalculator のインスタンスを返します。
func (bl *BattleLogic) GetDamageCalculator() *DamageCalculator {
	return bl.damageCalculator
}

// GetHitCalculator は HitCalculator のインスタンスを返します。
func (bl *BattleLogic) GetHitCalculator() *HitCalculator {
	return bl.hitCalculator
}

// GetTargetSelector は TargetSelector のインスタンスを返します。
func (bl *BattleLogic) GetTargetSelector() *TargetSelector {
	return bl.targetSelector
}

// GetPartInfoProvider は PartInfoProvider のインスタンスを返します。
func (bl *BattleLogic) GetPartInfoProvider() *PartInfoProvider {
	return bl.partInfoProvider
}

// NewBattleLogic は BattleLogic とそのすべての依存ヘルパーを初期化します。
func NewBattleLogic(world donburi.World, config *Config, gameDataManager *GameDataManager) *BattleLogic {
	bl := &BattleLogic{}

	// ヘルパーを初期化
	bl.partInfoProvider = NewPartInfoProvider(world, config, gameDataManager)
	bl.damageCalculator = NewDamageCalculator(world, config)
	bl.hitCalculator = NewHitCalculator(world, config)
	bl.targetSelector = NewTargetSelector(world, config)

	// ヘルパー間の依存性注入は不要になったため削除
	// bl.damageCalculator.SetPartInfoProvider(bl.partInfoProvider)
	// bl.hitCalculator.SetPartInfoProvider(bl.partInfoProvider)
	// bl.targetSelector.SetPartInfoProvider(bl.partInfoProvider)

	return bl
}
