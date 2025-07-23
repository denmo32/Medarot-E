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
	LogComponent           = donburi.NewComponentType[Log]()
	PlayerControlComponent = donburi.NewComponentType[PlayerControl]()

	// --- Action Components ---
	ActionIntentComponent = donburi.NewComponentType[ActionIntent]()
	TargetComponent       = donburi.NewComponentType[Target]()

	// --- State Components ---
	StateComponent = donburi.NewComponentType[State]()

	// --- AI Components ---
	AIComponent = donburi.NewComponentType[AI]()

	// --- Team Buff Component ---
	TeamBuffsComponent = donburi.NewComponentType[TeamBuffs]()

	// --- Status Effect Component ---
	ActiveEffectsComponent = donburi.NewComponentType[ActiveEffects]()

	// --- Debug Components ---
	DebugModeComponent = donburi.NewComponentType[struct{}]()
)

// worldStateTag はワールド状態エンティティを識別するためのタグコンポーネントです。
var worldStateTag = donburi.NewComponentType[struct{}]()

// ActiveEffects はエンティティに現在適用されているすべてのステータス効果を保持します。
type ActiveEffects struct {
	Effects []*ActiveStatusEffect
}

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

// State はエンティティの現在の状態と関連データを保持します。
type State struct {
	CurrentState StateType
}

// Gauge はチャージやクールダウンの進行状況を保持します。
type Gauge struct {
	ProgressCounter float64
	TotalDuration   float64
	CurrentGauge    float64 // 0-100
}

// ActionIntent は、AIまたはプレイヤーによって決定された行動の「意図」を表します。
// これは、ターゲットがまだ解決されていない段階です。
type ActionIntent struct {
	SelectedPartKey PartSlotKey
	PendingEffects  []StatusEffect // チャージ開始時などに適用が予定される効果
}

// Target は、行動の対象となるエンティティとパーツ、およびその決定方針を表します。
// TargetingSystemによってActionIntentが解決された後に設定されます。
type Target struct {
	Policy         TargetingPolicyType
	TargetEntity   donburi.Entity // *donburi.Entry から donburi.Entity に変更
	TargetPartSlot PartSlotKey
}

// Log は最後に行われた行動の結果を保持します。
type Log struct {
	LastActionLog string
}

// PlayerControl はプレイヤーが操作するエンティティであることを示すタグコンポーネントです。
type PlayerControl struct{}

// AI はAI制御エンティティのすべてのデータを集約します。
type AI struct {
	PersonalityID     string
	TargetHistory     TargetHistoryData
	LastActionHistory LastActionHistoryData
}

// TargetHistoryData は、このエンティティを最後に攻撃したエンティティを記録します。
type TargetHistoryData struct {
	LastAttacker *donburi.Entry
}

// LastActionHistoryData は、このエンティティが最後に攻撃を成功させたターゲットとパーツを記録します。
type LastActionHistoryData struct {
	LastHitTarget   *donburi.Entry
	LastHitPartSlot PartSlotKey
}

// TeamBuffs はチーム全体にかかるバフ効果を管理します。
// このコンポーネントを持つエンティティはワールドに1つだけ存在することを想定しています。
type TeamBuffs struct {
	// Buffs[TeamID][BuffType]
	Buffs map[TeamID]map[BuffType][]*BuffSource
}

// BuffSource は、どのエンティティのどのパーツからバフが発生しているかを記録します。
type BuffSource struct {
	SourceEntry *donburi.Entry
	SourcePart  PartSlotKey
	Value       float64 // 効果量 (例: 命中率1.2倍なら1.2)
}
