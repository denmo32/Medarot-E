package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ChangeState はエンティティの状態を更新します。
func ChangeState(entry *donburi.Entry, newStateType StateType) {
	state := StateComponent.Get(entry)
	oldStateType := state.Current

	if oldStateType == newStateType {
		return // 状態が同じ場合は何もしない
	}

	state.Current = newStateType

	// SettingsComponent が存在する場合にのみログを出力
	if entry.HasComponent(SettingsComponent) {
		log.Printf("%s のステートが変更されました: %v -> %v", SettingsComponent.Get(entry).Name, oldStateType, newStateType)
	}

	

	if oldStateType != newStateType {
		// 状態が実際に変更された場合にのみタグを追加します。
		// これにより、状態遷移システムはこのフレームで処理すべきエンティティを特定できます。
		donburi.Add(entry, StateChangedTagComponent, &StateChangedTag{})
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
	log.Println("すべての一時的な効果がリセットされました。")
}
