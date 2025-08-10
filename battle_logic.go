package main

import (
	"math/rand"
	"medarot-ebiten/data"

	"github.com/yohamta/donburi"
)

// BattleLogic は戦闘関連のすべての計算ロジックをカプセル化します。
type BattleLogic struct {
	damageCalculator       *DamageCalculator
	hitCalculator          *HitCalculator
	targetSelector         *TargetSelector
	partInfoProvider       PartInfoProviderInterface
	chargeInitiationSystem *ChargeInitiationSystem // 追加
	rand                   *rand.Rand
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
func (bl *BattleLogic) GetPartInfoProvider() PartInfoProviderInterface {
	return bl.partInfoProvider
}

// GetChargeInitiationSystem は ChargeInitiationSystem のインスタンスを返します。
func (bl *BattleLogic) GetChargeInitiationSystem() *ChargeInitiationSystem {
	return bl.chargeInitiationSystem
}

// NewBattleLogic は BattleLogic とそのすべての依存ヘルパーを初期化します。
func NewBattleLogic(world donburi.World, config *data.Config, gameDataManager *data.GameDataManager, rng *rand.Rand) *BattleLogic {
	logger := data.NewBattleLogger(gameDataManager) // BattleLoggerを初期化
	partInfoProvider := NewPartInfoProvider(world, config, gameDataManager)
	damageCalculator := NewDamageCalculator(world, config, partInfoProvider, gameDataManager, rng, logger)
	hitCalculator := NewHitCalculator(world, config, partInfoProvider, rng, logger)
	targetSelector := NewTargetSelector(world, config, partInfoProvider)
	chargeInitiationSystem := NewChargeInitiationSystem(world, partInfoProvider)

	return &BattleLogic{
		damageCalculator:       damageCalculator,
		hitCalculator:          hitCalculator,
		targetSelector:         targetSelector,
		partInfoProvider:       partInfoProvider,
		chargeInitiationSystem: chargeInitiationSystem,
		rand:                   rng,
	}
}
