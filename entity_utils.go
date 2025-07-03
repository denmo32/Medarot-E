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

	// 新しい状態に応じた初期化処理
	// このロジックは後で専用のシステムに移動します
	gauge := GaugeComponent.Get(entry)
	switch newStateType {
	case StateTypeIdle:
		// ゲージとアクションのリセットロジックは後で EnterIdleStateSystem に移動
	case StateTypeCharging:
		// チャージ開始ロジックは StartCharge にあります
	case StateTypeReady:
		if gauge != nil {
			gauge.CurrentGauge = 100
		}
	case StateTypeCooldown:
		// クールダウン開始ロジックは StartCooldownSystem にあります
	case StateTypeBroken:
		// 破壊時ロジックは後で EnterBrokenStateSystem に移動
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
