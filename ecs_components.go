package main

import (
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ui"

	"github.com/yohamta/donburi"
)

// --- Componentの型定義 ---
// 各コンポーネントにユニークな型情報を持たせます。
var (
	SettingsComponent      = donburi.NewComponentType[component.Settings]()
	PartsComponent         = donburi.NewComponentType[component.PartsComponentData]()
	MedalComponent         = donburi.NewComponentType[component.Medal]()
	GaugeComponent         = donburi.NewComponentType[component.Gauge]()
	LogComponent           = donburi.NewComponentType[component.Log]()
	PlayerControlComponent = donburi.NewComponentType[component.PlayerControl]()

	// --- Action Components ---
	ActionIntentComponent = donburi.NewComponentType[component.ActionIntent]()
	TargetComponent       = donburi.NewComponentType[component.Target]()

	// --- State Components ---
	StateComponent = donburi.NewComponentType[component.State]()

	// --- AI Components ---
	AIComponent = donburi.NewComponentType[component.AI]()

	// --- Team Buff Component ---
	TeamBuffsComponent = donburi.NewComponentType[component.TeamBuffs]()

	// --- Status Effect Component ---
	ActiveEffectsComponent = donburi.NewComponentType[component.ActiveEffects]()

	// --- Debug Components ---
	DebugModeComponent = donburi.NewComponentType[struct{}]()

	// --- UI State Component ---
	BattleUIStateComponent = donburi.NewComponentType[ui.BattleUIState]()

	// --- Game State Component ---
	GameStateComponent = donburi.NewComponentType[component.GameStateData]()

	// --- Player Action Queue Component ---
	PlayerActionQueueComponent = donburi.NewComponentType[component.PlayerActionQueueComponentData]()

	// --- Last Action Result Component ---
	LastActionResultComponent = donburi.NewComponentType[component.ActionResult]()
)

// worldStateTag はワールド状態エンティティを識別するためのタグコンポーネントです。
var worldStateTag = donburi.NewComponentType[struct{}]()
