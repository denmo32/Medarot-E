package main

import (
	"medarot-ebiten/domain"

	"github.com/yohamta/donburi"
)

// --- Componentの型定義 ---
// 各コンポーネントにユニークな型情報を持たせます。
var (
	SettingsComponent      = donburi.NewComponentType[domain.Settings]()
	PartsComponent         = donburi.NewComponentType[domain.PartsComponentData]()
	MedalComponent         = donburi.NewComponentType[domain.Medal]()
	GaugeComponent         = donburi.NewComponentType[domain.Gauge]()
	LogComponent           = donburi.NewComponentType[domain.Log]()
	PlayerControlComponent = donburi.NewComponentType[domain.PlayerControl]()

	// --- Action Components ---
	ActionIntentComponent = donburi.NewComponentType[domain.ActionIntent]()
	TargetComponent       = donburi.NewComponentType[domain.Target]()

	// --- State Components ---
	StateComponent = donburi.NewComponentType[domain.State]()

	// --- AI Components ---
	AIComponent = donburi.NewComponentType[domain.AI]()

	// --- Team Buff Component ---
	TeamBuffsComponent = donburi.NewComponentType[domain.TeamBuffs]()

	// --- Status Effect Component ---
	ActiveEffectsComponent = donburi.NewComponentType[domain.ActiveEffects]()

	// --- Debug Components ---
	DebugModeComponent = donburi.NewComponentType[struct{}]()

	// --- UI State Component ---
	BattleUIStateComponent = donburi.NewComponentType[BattleUIState]()

	// --- Game State Component ---
	GameStateComponent = donburi.NewComponentType[domain.GameStateData]()

	// --- Player Action Queue Component ---
	PlayerActionQueueComponent = donburi.NewComponentType[domain.PlayerActionQueueComponentData]()

	// --- Last Action Result Component ---
	LastActionResultComponent = donburi.NewComponentType[ActionResult]()
)

// worldStateTag はワールド状態エンティティを識別するためのタグコンポーネントです。
var worldStateTag = donburi.NewComponentType[struct{}]()

// BattleUIState is a singleton component that stores UI-specific data (ViewModels).
type BattleUIState struct {
	InfoPanels           map[string]InfoPanelViewModel // Map from Medarot ID to its ViewModel
	BattlefieldViewModel BattlefieldViewModel          // Add BattlefieldViewModel here
}