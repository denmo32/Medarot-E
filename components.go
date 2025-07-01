package main

import (
	"github.com/yohamta/donburi"
)

// --- Componentの型定義 ---
// 各Componentにユニークな型情報を持たせる
var (
	SettingsComponent      = donburi.NewComponentType[Settings]()
	PartsComponent         = donburi.NewComponentType[Parts]()
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

// Parts はメダロットのパーツ一式を保持する
type Parts struct {
	Map map[PartSlotKey]*Part
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
