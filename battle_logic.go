package main

import (
	"math/rand"
	"time"

	"github.com/yohamta/donburi"
)

// BattleLogic は戦闘関連のすべての計算ロジックをカプセル化します。
type BattleLogic struct {
	world            donburi.World // worldフィールドを追加
	config           *Config       // configフィールドを追加
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
	bl := &BattleLogic{
		world:  world,  // worldフィールドを初期化
		config: config, // configフィールドを初期化
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())), // 乱数生成器を初期化
	}

	logger := NewBattleLogger(gameDataManager) // BattleLoggerを初期化

	// ヘルパーを初期化
	bl.partInfoProvider = NewPartInfoProvider(world, config, gameDataManager)
	bl.damageCalculator = NewDamageCalculator(world, config, bl.partInfoProvider, gameDataManager, bl.rand, logger)
	bl.hitCalculator = NewHitCalculator(world, config, bl.partInfoProvider, bl.rand, logger)
	bl.targetSelector = NewTargetSelector(world, config, bl.partInfoProvider)

	return bl
}
