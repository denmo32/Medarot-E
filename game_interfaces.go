package main

import (
	"github.com/yohamta/donburi"
)

// TargetingStrategy はAIのターゲット選択アルゴリズムをカプセル化するインターフェースです。
type TargetingStrategy interface {
	SelectTarget(
		world donburi.World,
		actingEntry *donburi.Entry,
		battleLogic *BattleLogic,
	) (*donburi.Entry, PartSlotKey)
}

// TraitActionHandler はカテゴリ固有のアクション処理全体をカプセル化します。
// ActionResultを返し、副作用をなくします。
type TraitActionHandler interface {
	Execute(
		actingEntry *donburi.Entry,
		world donburi.World,
		intent *ActionIntent,
		battleLogic *BattleLogic,
		gameConfig *Config,
		actingPartDef *PartDefinition,
	) ActionResult
}

// WeaponTypeEffectHandler は weapon_type 固有の追加効果を処理します。
// ActionResult を受け取り、デバフ付与などの副作用を適用します。
type WeaponTypeEffectHandler interface {
	ApplyEffect(result *ActionResult, world donburi.World, battleLogic *BattleLogic, actingPartDef *PartDefinition)
}

// PartInfoProviderInterface はパーツの状態や情報を取得・操作するロジックのインターフェースです。
type PartInfoProviderInterface interface {
	GetPartParameterValue(entry *donburi.Entry, partSlot PartSlotKey, param PartParameter) float64
	FindPartSlot(entry *donburi.Entry, partToFindInstance *PartInstanceData) PartSlotKey
	GetAvailableAttackParts(entry *donburi.Entry) []AvailablePart
	GetOverallPropulsion(entry *donburi.Entry) int
	GetOverallMobility(entry *donburi.Entry) int
	GetLegsPartDefinition(entry *donburi.Entry) (*PartDefinition, bool)
	GetSuccessRate(entry *donburi.Entry, actingPartDef *PartDefinition, selectedPartKey PartSlotKey) float64
	GetEvasionRate(entry *donburi.Entry) float64
	GetDefenseRate(entry *donburi.Entry) float64
	GetTeamAccuracyBuffMultiplier(entry *donburi.Entry) float64
	RemoveBuffsFromSource(entry *donburi.Entry, partInst *PartInstanceData)
	CalculateGaugeDuration(baseSeconds float64, entry *donburi.Entry) float64
	CalculateMedarotXPosition(entry *donburi.Entry, battlefieldWidth float32) float32
	GetGameDataManager() *GameDataManager // GameDataManagerへのアクセスを提供
}
