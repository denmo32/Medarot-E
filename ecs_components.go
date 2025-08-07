package main

import (
	"medarot-ebiten/domain"
	"medarot-ebiten/ecs"

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
	TargetComponent       = donburi.NewComponentType[ecs.Target]()

	// --- State Components ---
	StateComponent = donburi.NewComponentType[domain.State]()

	// --- AI Components ---
	AIComponent = donburi.NewComponentType[ecs.AI]()

	// --- Team Buff Component ---
	TeamBuffsComponent = donburi.NewComponentType[ecs.TeamBuffs]()

	// --- Status Effect Component ---
	ActiveEffectsComponent = donburi.NewComponentType[domain.ActiveEffects]()

	// --- Debug Components ---
	DebugModeComponent = donburi.NewComponentType[struct{}]()

	// --- UI State Component ---
	BattleUIStateComponent = donburi.NewComponentType[ecs.BattleUIState]()

	// --- Game State Component ---
	GameStateComponent = donburi.NewComponentType[domain.GameStateData]()

	// --- Player Action Queue Component ---
	PlayerActionQueueComponent = donburi.NewComponentType[ecs.PlayerActionQueueComponentData]()

	// --- Last Action Result Component ---
	LastActionResultComponent = donburi.NewComponentType[ecs.ActionResult]()
)

// worldStateTag はワールド状態エンティティを識別するためのタグコンポーネントです。
var worldStateTag = donburi.NewComponentType[struct{}]()
