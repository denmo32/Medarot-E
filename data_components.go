package main

import (
	"github.com/looplab/fsm"
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

	// --- Debuff Components ---
	DefenseDebuffComponent = donburi.NewComponentType[DefenseDebuff]()
	EvasionDebuffComponent = donburi.NewComponentType[EvasionDebuff]()
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

// State はエンティティの現在の状態と関連データを保持します。
type State struct {
	FSM *fsm.FSM
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
}

// Target は、行動の対象となるエンティティとパーツを表します。
// TargetingSystemによってActionIntentが解決された後に設定されます。
type Target struct {
	TargetEntity   *donburi.Entry
	TargetPartSlot PartSlotKey
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

// --- AI関連コンポーネント ---

// AI はAI制御エンティティのすべてのデータを集約します。
type AI struct {
	TargetingStrategy     TargetingStrategy
	PartSelectionStrategy AIPartSelectionStrategyFunc
	TargetHistory         TargetHistoryData
	LastActionHistory     LastActionHistoryData
}

// --- 特性効果タグコンポーネント --- (リファクタリングにより不要に)

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

// --- 履歴データコンポーネント ---

// TargetHistoryData は、このエンティティを最後に攻撃したエンティティを記録します。
type TargetHistoryData struct {
	LastAttacker *donburi.Entry
}

// LastActionHistoryData は、このエンティティが最後に攻撃を成功させたターゲットとパーツを記録します。
type LastActionHistoryData struct {
	LastHitTarget   *donburi.Entry
	LastHitPartSlot PartSlotKey
}
