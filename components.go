package main

import (
	"github.com/yohamta/donburi"
)

// --- Componentの型定義 ---
// 各コンポーネントにユニークな型情報を持たせます。
var (
	SettingsComponent      = donburi.NewComponentType[Settings]()
	PartsComponent         = donburi.NewComponentType[PartsComponentData]()
	MedalComponent         = donburi.NewComponentType[Medal]()
	GaugeComponent         = donburi.NewComponentType[Gauge]()
	ActionComponent        = donburi.NewComponentType[Action]()
	LogComponent           = donburi.NewComponentType[Log]()
	PlayerControlComponent = donburi.NewComponentType[PlayerControl]()
	// EffectsComponent       = donburi.NewComponentType[Effects]() // 現在未使用

	DefenseDebuffComponent = donburi.NewComponentType[DefenseDebuff]()
	EvasionDebuffComponent = donburi.NewComponentType[EvasionDebuff]()

	IdleStateComponent     = donburi.NewComponentType[IdleState]()
	ChargingStateComponent = donburi.NewComponentType[ChargingState]()
	ReadyStateComponent    = donburi.NewComponentType[ReadyState]()
	CooldownStateComponent = donburi.NewComponentType[CooldownState]()
	BrokenStateComponent   = donburi.NewComponentType[BrokenState]()

	TargetingStrategyComponent = donburi.NewComponentType[TargetingStrategyComponentData]()
)

// --- コンポーネントの構造体定義 ---

// Settings はメダロットの不変的な設定を保持します。
type Settings struct {
	ID        string
	Name      string
	Team      TeamID
	IsLeader  bool
	DrawIndex int // 描画順やY座標の決定に使用されます。
}

// PartsComponentData はメダロットのパーツ一式を保持します。
type PartsComponentData struct {
	Map map[PartSlotKey]*PartInstanceData
}

// 状態タグコンポーネント
type IdleState struct{}
type ChargingState struct{}
type ReadyState struct{}
type CooldownState struct{}
type BrokenState struct{}

// Gauge はチャージやクールダウンの進行状況を保持します。
type Gauge struct {
	ProgressCounter float64
	TotalDuration   float64
	CurrentGauge    float64 // 0-100
}

// Action は選択された行動とターゲットを保持します。
type Action struct {
	SelectedPartKey PartSlotKey
	TargetPartSlot  PartSlotKey
	TargetEntity    *donburi.Entry
}

// Log は最後に行われた行動の結果を保持します。
type Log struct {
	LastActionLog string
}

// PlayerControl はプレイヤーが操作するエンティティであることを示すタグコンポーネントです。
type PlayerControl struct{}

// Effects はメダロットにかかっている一時的な効果（バフ・デバフ）を管理します。 (現在未使用)
// type Effects struct {
//	EvasionRateMultiplier float64 // 回避率の倍率 (例: 0.5で半減)
//	DefenseRateMultiplier float64 // 防御率の倍率 (例: 0.5で半減)
//}

// DefenseDebuff は防御力デバフ効果を表します。
type DefenseDebuff struct {
	Multiplier float64 // 防御率に乗算される値 (例: 0.5)
}

// EvasionDebuff は回避力デバフ効果を表します。
type EvasionDebuff struct {
	Multiplier float64 // 回避率に乗算される値 (例: 0.5)
}

// --- AIターゲティング戦略コンポーネント ---

// TargetingStrategyComponentData はAIエンティティのターゲティング戦略を保持します。
// StrategyフィールドはTargetingStrategyインターフェースを実装した構造体のポインタを保持します。
type TargetingStrategyComponentData struct {
	Strategy TargetingStrategy
}

// --- 特性効果タグコンポーネント ---
// これらは、特定の特性を持つパーツがアクションに使用されているときにエンティティに追加されます。

// ActingWithBerserkTraitTag は、エンティティが現在BERSERK特性パーツでアクションを実行していることを示します。
type ActingWithBerserkTraitTag struct{}

var ActingWithBerserkTraitTagComponent = donburi.NewComponentType[ActingWithBerserkTraitTag]()

// ActingWithAimTraitTag は、エンティティが現在AIM特性パーツでアクションを実行していることを示します。
type ActingWithAimTraitTag struct{}

var ActingWithAimTraitTagComponent = donburi.NewComponentType[ActingWithAimTraitTag]()

// --- 一時的な状態変化タグ ---
// これらのタグは、エンティティが特定の状態に遷移したときに追加され、
// 通常、その状態変化の副作用を処理した後にシステムによって削除されます。

// JustBecameIdleTag は、エンティティがアイドル状態に遷移したばかりであることを示します。
type JustBecameIdleTag struct{}

var JustBecameIdleTagComponent = donburi.NewComponentType[JustBecameIdleTag]()

// JustBecameBrokenTag は、エンティティが破壊状態に遷移したばかりであることを示します。
type JustBecameBrokenTag struct{}

var JustBecameBrokenTagComponent = donburi.NewComponentType[JustBecameBrokenTag]()

// --- アクション修飾コンポーネント ---
// アクション計算（ヒット/ダメージ）の前にエンティティに一時的に追加され、
// 特性、スキル、バフ、デバフなどからのすべての修飾子を集約します。
type ActionModifierComponentData struct {
	// クリティカルヒット修飾子
	CriticalRateBonus  int     // 例: +10 で +10%
	CriticalMultiplier float64 // 特性が基本クリティカル乗数を変更する場合。デフォルト0でシステムの基本値を使用
	// 威力/ダメージ修飾子
	PowerAdditiveBonus    int     // 例: +20 威力
	PowerMultiplierBonus  float64 // 例: 1.5 で +50% 威力 (加算後に適用)
	DamageAdditiveBonus   int     // 最後に適用される固定ダメージボーナス
	DamageMultiplierBonus float64 // 全体的なダメージ乗数 (例: バフ/デバフから)
	// 命中/回避修飾子
	AccuracyAdditiveBonus int // 例: +10 命中
	// EvasionAdditiveBonus  int     // ターゲットの回避用、通常はターゲットのデバフで処理
	// AccuracyMultiplier    float64 // 例: 1.1 で +10%
}

var ActionModifierComponent = donburi.NewComponentType[ActionModifierComponentData]()

// --- AIパーツ選択戦略コンポーネント ---

// AIPartSelectionStrategyFunc はAIパーツ選択戦略の関数シグネチャを定義します。
// 行動するAIエンティティと利用可能なパーツのリストを受け取り、選択されたパーツとそのスロットを返します。
type AIPartSelectionStrategyFunc func(
	actingEntry *donburi.Entry,
	availableParts []AvailablePart,
	world donburi.World, // より複雑な戦略でワールドアクセスが必要な場合
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
) (PartSlotKey, *PartDefinition) // 選択されたパーツのスロットキーとその定義を返します。

// AIPartSelectionStrategyComponentData はAIエンティティのパーツ選択戦略を保持します。
type AIPartSelectionStrategyComponentData struct {
	Strategy AIPartSelectionStrategyFunc
}

var AIPartSelectionStrategyComponent = donburi.NewComponentType[AIPartSelectionStrategyComponentData]()
