package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ChangeState はエンティティの状態コンポーネントを切り替えます。
func ChangeState(entry *donburi.Entry, newStateType StateType) {
	var oldStateType StateType = -1 // 特定の前の状態がないか、不明な状態を示す値で初期化

	// 古い状態を決定し、古い状態コンポーネントを削除
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
	// 注意: エンティティが最初に状態コンポーネントを持たない場合、oldStateType は -1 のままになる可能性があります。

	// SettingsComponent が存在する場合にのみログを出力し、メダロット以外のエンティティで呼び出された場合のパニックを防ぎます
	if entry.HasComponent(SettingsComponent) {
		log.Printf("%s のステートが変更されました: %v -> %v", SettingsComponent.Get(entry).Name, oldStateType, newStateType)
	}

	gauge := GaugeComponent.Get(entry)

	// 新しい状態に応じた初期化処理とコンポーネントの追加
	switch newStateType {
	case StateTypeIdle:
		donburi.Add(entry, IdleStateComponent, &IdleState{})
		// ゲージとアクションのリセットロジックは ProcessStateEffectsSystem の JustBecameIdleTag の処理に移動
	case StateTypeCharging:
		donburi.Add(entry, ChargingStateComponent, &ChargingState{})
	case StateTypeReady:
		donburi.Add(entry, ReadyStateComponent, &ReadyState{})
		if gauge != nil { // UIのために、準備完了時に即座にゲージを100に設定します（イベント処理では遅すぎる場合があるため）
			gauge.CurrentGauge = 100
		}
	case StateTypeCooldown:
		donburi.Add(entry, CooldownStateComponent, &CooldownState{})
	case StateTypeBroken:
		donburi.Add(entry, BrokenStateComponent, &BrokenState{})
		// 破壊状態のゲージリセットは ProcessStateEffectsSystem の JustBecameBrokenTag の処理に移動
	}

	// StateChangedEvent の発行 -> 一時的なタグの追加に置き換え
	if oldStateType != newStateType { // 実際に状態が変更された場合にのみタグを追加
		// 必要に応じて迅速な状態変化に対応するため、既存の "JustBecame..." タグを最初に削除します。
		// ただし、通常はシステムが1フレーム後にそれらを削除します。
		// 現状では、システムがそれらをクリーンアップすると仮定します。
		switch newStateType {
		case StateTypeIdle:
			donburi.Add(entry, JustBecameIdleTagComponent, &JustBecameIdleTag{})
			if entry.HasComponent(SettingsComponent) {
				log.Printf("タグ追加: %s に JustBecameIdleTag", SettingsComponent.Get(entry).Name)
			}
		case StateTypeBroken:
			donburi.Add(entry, JustBecameBrokenTagComponent, &JustBecameBrokenTag{})
			if entry.HasComponent(SettingsComponent) {
				log.Printf("タグ追加: %s に JustBecameBrokenTag", SettingsComponent.Get(entry).Name)
			}
			// 他の状態に対して必要であれば、他の JustBecame... タグをここに追加
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
	log.Println("すべての一時的な効果がリセットされました。")
}
