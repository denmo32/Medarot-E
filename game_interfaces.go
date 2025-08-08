package main

import (
	"math/rand"

	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
)

// TargetingStrategy はAIのターゲット選択アルゴリズムをカプセル化するインターフェースです。
type TargetingStrategy interface {
	SelectTarget(
		world donburi.World,
		actingEntry *donburi.Entry,
		battleLogic *BattleLogic,
	) (*donburi.Entry, component.PartSlotKey)
}

type BattleLogger interface {
	LogHitCheck(attackerName, targetName string, chance, successRate, evasion float64, roll int)
	LogDefenseCheck(targetName string, defenseRate, successRate, chance float64, roll int)
	LogCriticalHit(attackerName string, chance float64)
	LogPartBroken(medarotName, partName, partID string)
	LogActionInitiated(attackerName string, actionTrait component.Trait, weaponType component.WeaponType, category component.PartCategory)
	LogAttackMiss(attackerName, skillName, targetName string)
	LogDamageDealt(defenderName, targetPartType string, damage int)
	LogDefenseSuccess(targetName, defensePartName string, originalDamage, actualDamage int, isCritical bool)
}

// TraitActionHandler はカテゴリ固有のアクション処理全体をカプセル化します。
// ActionResultを返し、副作用をなくします。
type TraitActionHandler interface {
	Execute(
		actingEntry *donburi.Entry,
		world donburi.World,
		intent *component.ActionIntent,
		damageCalculator *DamageCalculator,
		hitCalculator *HitCalculator,
		targetSelector *TargetSelector,
		partInfoProvider PartInfoProviderInterface,
		gameConfig *Config,
		actingPartDef *component.PartDefinition,
		rand *rand.Rand,
	) component.ActionResult
}

// WeaponTypeEffectHandler は weapon_type 固有の追加効果を処理します。
// ActionResult を受け取り、デバフ付与などの副作用を適用します。
type WeaponTypeEffectHandler interface {
	ApplyEffect(result *component.ActionResult, world donburi.World, damageCalculator *DamageCalculator, hitCalculator *HitCalculator, targetSelector *TargetSelector, partInfoProvider PartInfoProviderInterface, actingPartDef *component.PartDefinition, rand *rand.Rand)
}

// PartInfoProviderInterface はパーツの状態や情報を取得・操作するロジックのインターフェースです。
type PartInfoProviderInterface interface {
	// パーツのパラメータ値を取得するメソッド
	GetPartParameterValue(entry *donburi.Entry, partSlot component.PartSlotKey, param component.PartParameter) float64

	// パーツスロットを検索するメソッド
	FindPartSlot(entry *donburi.Entry, partToFindInstance *component.PartInstanceData) component.PartSlotKey

	// 利用可能な攻撃パーツを取得するメソッド
	GetAvailableAttackParts(entry *donburi.Entry) []component.AvailablePart

	// 全体的な推進力と機動力を取得するメソッド
	GetOverallPropulsion(entry *donburi.Entry) int
	GetOverallMobility(entry *donburi.Entry) int

	// 脚部パーツの定義を取得するメソッド
	GetLegsPartDefinition(entry *donburi.Entry) (*component.PartDefinition, bool)

	// 成功度、回避度、防御度を取得するメソッド
	GetSuccessRate(entry *donburi.Entry, actingPartDef *component.PartDefinition, selectedPartKey component.PartSlotKey) float64
	GetEvasionRate(entry *donburi.Entry) float64
	GetDefenseRate(entry *donburi.Entry) float64

	// チームの命中率バフ乗数を取得するメソッド
	GetTeamAccuracyBuffMultiplier(entry *donburi.Entry) float64

	// バフを削除するメソッド
	RemoveBuffsFromSource(entry *donburi.Entry, partInst *component.PartInstanceData)

	// ゲージの持続時間を計算するメソッド
	CalculateGaugeDuration(baseSeconds float64, entry *donburi.Entry) float64

	// メダロットの正規化された行動進行度を取得するメソッド
	GetNormalizedActionProgress(entry *donburi.Entry) float32

	// GameDataManagerへのアクセスを提供するメソッド
	GetGameDataManager() *GameDataManager
}
