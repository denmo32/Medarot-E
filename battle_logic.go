package main

import (
	"github.com/yohamta/donburi"
)

// BattleLogic は戦闘関連のすべての計算ロジックをカプセル化します。
type BattleLogic struct {
	DamageCalculator *DamageCalculator
	HitCalculator    *HitCalculator
	TargetSelector   *TargetSelector
	PartInfoProvider *PartInfoProvider
}

// GetPartInfoProvider は PartInfoProvider のインスタンスを返します。
func (bl *BattleLogic) GetPartInfoProvider() *PartInfoProvider {
	return bl.PartInfoProvider
}

// NewBattleLogic は BattleLogic とそのすべての依存ヘルパーを初期化します。
func NewBattleLogic(world donburi.World, config *Config, gameDataManager *GameDataManager) *BattleLogic {
	bl := &BattleLogic{}

	// ヘルパーを初期化
	bl.PartInfoProvider = NewPartInfoProvider(world, config, gameDataManager)
	bl.DamageCalculator = NewDamageCalculator(world, config)
	bl.HitCalculator = NewHitCalculator(world, config)
	bl.TargetSelector = NewTargetSelector(world, config)

	// ヘルパー間の依存性注入は不要になったため削除
	// bl.DamageCalculator.SetPartInfoProvider(bl.PartInfoProvider)
	// bl.HitCalculator.SetPartInfoProvider(bl.PartInfoProvider)
	// bl.TargetSelector.SetPartInfoProvider(bl.PartInfoProvider)

	return bl
}
