package main

import (
	"math/rand"
	"time"

	"github.com/yohamta/donburi"
)

// BattleLogic は戦闘関連のすべての計算ロジックをカプセル化します。
type BattleLogic struct {
	damageCalculator *DamageCalculator
	hitCalculator    *HitCalculator
	targetSelector   *TargetSelector
	partInfoProvider PartInfoProviderInterface
	rand             *rand.Rand // 追加: 乱数生成器
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

// NewBattleLogic は BattleLogic とそのすべての依存ヘルパーを初期化します。
func NewBattleLogic(world donburi.World, config *Config, gameDataManager *GameDataManager) *BattleLogic {
	logger := NewBattleLogger(gameDataManager) // BattleLoggerを初期化
	partInfoProvider := NewPartInfoProvider(world, config, gameDataManager)
	damageCalculator := NewDamageCalculator(world, config, partInfoProvider, gameDataManager, rand.New(rand.NewSource(time.Now().UnixNano())), logger)
	hitCalculator := NewHitCalculator(world, config, partInfoProvider, rand.New(rand.NewSource(time.Now().UnixNano())), logger)
	targetSelector := NewTargetSelector(world, config, partInfoProvider)

	return &BattleLogic{
		damageCalculator: damageCalculator,
		hitCalculator:    hitCalculator,
		targetSelector:   targetSelector,
		partInfoProvider: partInfoProvider,
		rand:             rand.New(rand.NewSource(time.Now().UnixNano())), // 乱数生成器を初期化
	}
}
