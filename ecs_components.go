package main

import (
	"medarot-ebiten/core"
	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
)

// --- Componentの型定義 ---
// 各コンポーネントにユニークな型情報を持たせます。
var (
	SettingsComponent      = donburi.NewComponentType[core.Settings]()
	PartsComponent         = donburi.NewComponentType[core.PartsComponentData]()
	MedalComponent         = donburi.NewComponentType[core.Medal]()
	GaugeComponent         = donburi.NewComponentType[core.Gauge]()
	LogComponent           = donburi.NewComponentType[core.Log]()
	PlayerControlComponent = donburi.NewComponentType[core.PlayerControl]()

	// --- Action Components ---
	ActionIntentComponent = donburi.NewComponentType[core.ActionIntent]()
	TargetComponent       = donburi.NewComponentType[component.Target]()

	// --- State Components ---
	StateComponent = donburi.NewComponentType[core.State]()

	// --- AI Components ---
	AIComponent = donburi.NewComponentType[component.AI]()

	// --- Team Buff Component ---
	TeamBuffsComponent = donburi.NewComponentType[component.TeamBuffs]()

	// --- Status Effect Component ---
	ActiveEffectsComponent = donburi.NewComponentType[core.ActiveEffects]()

	// --- Debug Components ---
	DebugModeComponent = donburi.NewComponentType[struct{}]()

	// --- Game State Component ---
	GameStateComponent = donburi.NewComponentType[core.GameStateData]()

	// --- Player Action Queue Component ---
	PlayerActionQueueComponent = donburi.NewComponentType[component.PlayerActionQueueComponentData]()

	// --- Last Action Result Component ---
	LastActionResultComponent = donburi.NewComponentType[component.ActionResult]()
)

// worldStateTag はワールド状態エンティティを識別するためのタグコンポーネントです。
var worldStateTag = donburi.NewComponentType[struct{}]()
