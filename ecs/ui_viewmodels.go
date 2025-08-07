package ecs

import (
	"image/color"
	"medarot-ebiten/domain"

	"github.com/yohamta/donburi"
)

// BattleUIState is a singleton component that stores UI-specific data (ViewModels).
type BattleUIState struct {
	InfoPanels           map[string]InfoPanelViewModel // Map from Medarot ID to its ViewModel
	BattlefieldViewModel BattlefieldViewModel          // Add BattlefieldViewModel here
}

// ActionModalButtonViewModel は、アクション選択モーダルのボタン一つ分のデータを保持します。
type ActionModalButtonViewModel struct {
	PartName          string
	PartCategory      domain.PartCategory
	SlotKey           domain.PartSlotKey
	TargetEntityID    donburi.Entity // 射撃などのターゲットが必要な場合
	TargetPartSlot    domain.PartSlotKey
	SelectedPartDefID string
}

// ActionModalViewModel は、アクション選択モーダル全体の表示に必要なデータを保持します。
type ActionModalViewModel struct {
	ActingMedarotName string
	ActingEntityID    donburi.Entity // イベント発行時に必要
	Buttons           []ActionModalButtonViewModel
}

// InfoPanelViewModel は、単一の情報パネルUIが必要とするすべてのデータを保持します。
type InfoPanelViewModel struct {
	ID        string         // 名前表示用としてstringに戻す
	EntityID  donburi.Entity // アイコンとの対応付け用
	Name      string
	Team      domain.TeamID
	DrawIndex int
	StateStr  string
	IsLeader  bool
	Parts     map[domain.PartSlotKey]PartViewModel
}

// PartViewModel は、単一のパーツUIが必要とするデータを保持します。
type PartViewModel struct {
	PartName     string
	PartType     domain.PartType
	CurrentArmor int
	MaxArmor     int
	IsBroken     bool
}

// BattlefieldViewModel は、バトルフィールド全体の描画に必要なデータを保持します。
type BattlefieldViewModel struct {
	Icons     []*IconViewModel
	DebugMode bool
}

// IconViewModel は、個々のメダロットアイコンの描画に必要なデータを保持します。
type IconViewModel struct {
	EntryID       donburi.Entity // 元のdonburi.Entryを特定するためのID (uint32 から donburi.Entity に変更)
	X, Y          float32
	Color         color.Color
	IsLeader      bool
	State         domain.StateType
	GaugeProgress float64 // 0.0 to 1.0
	DebugText     string
}
