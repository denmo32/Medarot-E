package main

import (
	"github.com/yohamta/donburi"
)

// --- Componentの型定義 ---
// 各Componentにユニークな型情報を持たせる
var (
	SettingsComponent      = donburi.NewComponentType[Settings]()
	PartsComponent         = donburi.NewComponentType[PartsComponentData]() // Changed from Parts
	MedalComponent         = donburi.NewComponentType[Medal]()
	GaugeComponent         = donburi.NewComponentType[Gauge]()
	ActionComponent        = donburi.NewComponentType[Action]()
	LogComponent           = donburi.NewComponentType[Log]()
	PlayerControlComponent = donburi.NewComponentType[PlayerControl]()
	// EffectsComponent       = donburi.NewComponentType[Effects]()

	// ★★★ 以下を新しく追加 ★★★
	DefenseDebuffComponent = donburi.NewComponentType[DefenseDebuff]()
	EvasionDebuffComponent = donburi.NewComponentType[EvasionDebuff]()

	IdleStateComponent     = donburi.NewComponentType[IdleState]()
	ChargingStateComponent = donburi.NewComponentType[ChargingState]()
	ReadyStateComponent    = donburi.NewComponentType[ReadyState]()
	CooldownStateComponent = donburi.NewComponentType[CooldownState]()
	BrokenStateComponent   = donburi.NewComponentType[BrokenState]()

	// For AI Targeting Strategy
	TargetingStrategyComponent = donburi.NewComponentType[TargetingStrategyComponentData]()
)

// --- Componentの構造体定義 ---
// Settings はメダロットの不変的な設定を保持する
type Settings struct {
	ID        string
	Name      string
	Team      TeamID
	IsLeader  bool
	DrawIndex int // 描画順やY座標の決定に使用
}

// Parts (now PartsComponentData) はメダロットのパーツ一式を保持する
type PartsComponentData struct {
	Map map[PartSlotKey]*PartInstanceData // Changed from *Part to *PartInstanceData
}

// 新しい状態タグコンポーネント
type IdleState struct{}
type ChargingState struct{}
type ReadyState struct{}
type CooldownState struct{}
type BrokenState struct{}

// Gauge はチャージやクールダウンの進行状況を保持する
type Gauge struct {
	ProgressCounter float64
	TotalDuration   float64
	CurrentGauge    float64 // 0-100
}

// Action は選択された行動とターゲットを保持する
type Action struct {
	SelectedPartKey PartSlotKey
	TargetPartSlot  PartSlotKey
	TargetEntity    *donburi.Entry
}

// Log は最後に行われた行動の結果を保持する
type Log struct {
	LastActionLog string
}

// PlayerControl はプレイヤーが操作するエンティティであることを示すタグ
type PlayerControl struct{}

// Effects はメダロットにかかっている一時的な効果（バフ・デバフ）を管理します
// type Effects struct {
//	EvasionRateMultiplier float64 // 回避率の倍率 (例: 0.5で半減)
//	DefenseRateMultiplier float64 // 防御率の倍率 (例: 0.5で半減)
//}

// ★★★ 以下を新しく追加 ★★★
// 防御率デバフ効果
type DefenseDebuff struct {
	Multiplier float64 // 防御率に乗算される値 (例: 0.5)
}

// 回避率デバフ効果
type EvasionDebuff struct {
	Multiplier float64 // 回避率に乗算される値 (例: 0.5)
}

// --- AI Targeting Strategy Component ---

// TargetingStrategyFunc defines the function signature for a targeting strategy.
// It needs access to the world (for querying entities), the acting AI entity,
// the targetSelector helper, and potentially partInfoProvider.
type TargetingStrategyFunc func(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector, // from battle_logic.go
	partInfoProvider *PartInfoProvider, // from battle_logic.go
) (*donburi.Entry, PartSlotKey) // Returns target entity and target part slot

// TargetingStrategyComponentData holds the targeting strategy for an AI entity.
type TargetingStrategyComponentData struct {
	Strategy TargetingStrategyFunc
}

// --- Trait Effect Tag Components ---
// These are added to an entity when a part with a specific trait is being used for an action.

// ActingWithBerserkTraitTag indicates the entity is currently performing an action with a BERSERK trait part.
type ActingWithBerserkTraitTag struct{}

var ActingWithBerserkTraitTagComponent = donburi.NewComponentType[ActingWithBerserkTraitTag]()

// ActingWithAimTraitTag indicates the entity is currently performing an action with an AIM trait part.
type ActingWithAimTraitTag struct{}

var ActingWithAimTraitTagComponent = donburi.NewComponentType[ActingWithAimTraitTag]()

// Potentially others: ActingWithStrikeTraitTag, etc.

// --- Temporary State Change Tags ---
// These tags are added when an entity transitions to a specific state,
// and are typically removed by a system after processing the side effects of that state change.

// JustBecameIdleTag indicates the entity has just transitioned to the Idle state.
type JustBecameIdleTag struct{}

var JustBecameIdleTagComponent = donburi.NewComponentType[JustBecameIdleTag]()

// JustBecameBrokenTag indicates the entity has just transitioned to the Broken state.
type JustBecameBrokenTag struct{}

var JustBecameBrokenTagComponent = donburi.NewComponentType[JustBecameBrokenTag]()

// --- Action Modifier Component ---
// Temporarily added to an entity before action calculation (hit/damage)
// to aggregate all modifiers from traits, skills, buffs, debuffs, etc.
type ActionModifierComponentData struct {
	// Critical Hit Modifiers
	CriticalRateBonus  int     // e.g., +10 for +10%
	CriticalMultiplier float64 // If a trait changes the base crit multiplier, default 0 to use system base

	// Power/Damage Modifiers
	PowerAdditiveBonus    int     // e.g., +20 Power
	PowerMultiplierBonus  float64 // e.g., 1.5 for +50% Power (applied after additive)
	DamageAdditiveBonus   int     // Flat damage bonus applied at the very end
	DamageMultiplierBonus float64 // Overall damage multiplier (e.g., from buffs/debuffs)

	// Accuracy/Evasion Modifiers
	AccuracyAdditiveBonus int // e.g., +10 Accuracy
	// EvasionAdditiveBonus  int     // For target's evasion, usually handled by debuffs on target
	// AccuracyMultiplier    float64 // e.g., 1.1 for +10%
}

var ActionModifierComponent = donburi.NewComponentType[ActionModifierComponentData]()

// --- AI Part Selection Strategy Component ---

// AIPartSelectionStrategyFunc defines the function signature for an AI part selection strategy.
// It takes the acting AI entity and a list of available parts, and returns the chosen part and its slot.
type AIPartSelectionStrategyFunc func(
	actingEntry *donburi.Entry,
	availableParts []AvailablePart, // AvailablePart is defined in battle_logic.go
	world donburi.World, // For more complex strategies needing world access
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
) (PartSlotKey, *PartDefinition) // Returns selected part's slot key and its definition

// AIPartSelectionStrategyComponentData holds the part selection strategy for an AI entity.
type AIPartSelectionStrategyComponentData struct {
	Strategy AIPartSelectionStrategyFunc
}

var AIPartSelectionStrategyComponent = donburi.NewComponentType[AIPartSelectionStrategyComponentData]()
