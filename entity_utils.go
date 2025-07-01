package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ChangeState はエンティティの状態コンポーネントを切り替えます。
func ChangeState(entry *donburi.Entry, newStateType StateType) {
	// 既存の状態コンポーネントをすべて削除
	if entry.HasComponent(IdleStateComponent) {
		entry.RemoveComponent(IdleStateComponent)
	}
	if entry.HasComponent(ChargingStateComponent) {
		entry.RemoveComponent(ChargingStateComponent)
	}
	if entry.HasComponent(ReadyStateComponent) {
		entry.RemoveComponent(ReadyStateComponent)
	}
	if entry.HasComponent(CooldownStateComponent) {
		entry.RemoveComponent(CooldownStateComponent)
	}
	if entry.HasComponent(BrokenStateComponent) {
		entry.RemoveComponent(BrokenStateComponent)
	}

	// Log only if SettingsComponent exists, to prevent panic if called on non-medarot entities
	if entry.HasComponent(SettingsComponent) {
		log.Printf("%s のステートが変更されました: %v", SettingsComponent.Get(entry).Name, newStateType)
	}


	gauge := GaugeComponent.Get(entry)
	action := ActionComponent.Get(entry)

	// 新しい状態に応じた初期化処理とコンポーネントの追加
	switch newStateType {
	case StateTypeIdle:
		donburi.Add(entry, IdleStateComponent, &IdleState{})
		if gauge != nil {
			gauge.CurrentGauge = 0
			gauge.ProgressCounter = 0
			gauge.TotalDuration = 0
		}
		if action != nil {
			action.SelectedPartKey = ""
			action.TargetPartSlot = ""
			action.TargetEntity = nil
		}
	case StateTypeCharging:
		donburi.Add(entry, ChargingStateComponent, &ChargingState{})
	case StateTypeReady:
		donburi.Add(entry, ReadyStateComponent, &ReadyState{})
		if gauge != nil {
			gauge.CurrentGauge = 100
		}
	case StateTypeCooldown:
		donburi.Add(entry, CooldownStateComponent, &CooldownState{})
	case StateTypeBroken:
		donburi.Add(entry, BrokenStateComponent, &BrokenState{})
		if gauge != nil {
			gauge.CurrentGauge = 0
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
