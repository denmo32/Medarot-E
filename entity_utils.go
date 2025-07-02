package main

import (
	"log"

	"github.com/yohamta/donburi"
	// "github.com/yohamta/donburi/event" // Removed: No longer used
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ChangeState はエンティティの状態コンポーネントを切り替えます。
func ChangeState(entry *donburi.Entry, newStateType StateType) {
	var oldStateType StateType = -1 // Initialize with a value indicating no specific previous state or an unknown state

	// Determine old state and remove old state component
	if entry.HasComponent(IdleStateComponent) {
		oldStateType = StateTypeIdle
		entry.RemoveComponent(IdleStateComponent)
	} else if entry.HasComponent(ChargingStateComponent) {
		oldStateType = StateTypeCharging
		entry.RemoveComponent(ChargingStateComponent)
	} else if entry.HasComponent(ReadyStateComponent) {
		oldStateType = StateTypeReady
		entry.RemoveComponent(ReadyStateComponent)
	} else if entry.HasComponent(CooldownStateComponent) {
		oldStateType = StateTypeCooldown
		entry.RemoveComponent(CooldownStateComponent)
	} else if entry.HasComponent(BrokenStateComponent) {
		oldStateType = StateTypeBroken
		entry.RemoveComponent(BrokenStateComponent)
	}
	// Note: If an entity can have no state component initially, oldStateType might remain -1.

	// Log only if SettingsComponent exists, to prevent panic if called on non-medarot entities
	if entry.HasComponent(SettingsComponent) {
		log.Printf("%s のステートが変更されました: %v", SettingsComponent.Get(entry).Name, newStateType)
	}

	gauge := GaugeComponent.Get(entry)
	// action := ActionComponent.Get(entry) // Removed: No longer used directly in ChangeState

	// 新しい状態に応じた初期化処理とコンポーネントの追加
	switch newStateType {
	case StateTypeIdle:
		donburi.Add(entry, IdleStateComponent, &IdleState{})
		// Gauge and Action reset logic moved to handleStateChangeForGaugeReset event handler
	case StateTypeCharging:
		donburi.Add(entry, ChargingStateComponent, &ChargingState{})
	case StateTypeReady:
		donburi.Add(entry, ReadyStateComponent, &ReadyState{})
		if gauge != nil { // Still set gauge to 100 immediately on Ready, event might be too late for some UI
			gauge.CurrentGauge = 100
		}
	case StateTypeCooldown:
		donburi.Add(entry, CooldownStateComponent, &CooldownState{})
	case StateTypeBroken:
		donburi.Add(entry, BrokenStateComponent, &BrokenState{})
		// Gauge reset for Broken moved to handleStateChangeForGaugeReset event handler
	}

	// Publish StateChangedEvent -> Replaced with adding temporary tags
	if oldStateType != newStateType { // Only add tag if state actually changed
		// Remove any existing "JustBecame..." tags first to handle rapid state changes if necessary,
		// though typically a system would remove them after one frame.
		// For now, we assume systems will clean them up.
		switch newStateType {
		case StateTypeIdle:
			donburi.Add(entry, JustBecameIdleTagComponent, &JustBecameIdleTag{})
			if entry.HasComponent(SettingsComponent) {
				log.Printf("TagAdded: JustBecameIdleTag for %s", SettingsComponent.Get(entry).Name)
			}
		case StateTypeBroken:
			donburi.Add(entry, JustBecameBrokenTagComponent, &JustBecameBrokenTag{})
			if entry.HasComponent(SettingsComponent) {
				log.Printf("TagAdded: JustBecameBrokenTag for %s", SettingsComponent.Get(entry).Name)
			}
			// Add other JustBecame... tags here if needed for other states
		}
	}
}

// ResetAllEffects は全ての効果をリセットします。
func ResetAllEffects(world donburi.World) {
	query.NewQuery(filter.Contains(DefenseDebuffComponent)).Each(world, func(e *donburi.Entry) {
		e.RemoveComponent(DefenseDebuffComponent)
	})
	query.NewQuery(filter.Contains(EvasionDebuffComponent)).Each(world, func(e *donburi.Entry) {
		e.RemoveComponent(EvasionDebuffComponent)
	})
	log.Println("All temporary effects have been reset.")
}
