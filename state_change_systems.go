package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ProcessStateChangeSystem は、状態が変化したエンティティを処理します。
// このフレームで状態が変化したエンティティをクエリし、
// 新しい状態に応じた副作用（ゲージのリセットなど）を適用します。
func ProcessStateChangeSystem(world donburi.World) {
	query := query.NewQuery(filter.Contains(StateChangedTagComponent))

	query.Each(world, func(entry *donburi.Entry) {
		state := StateComponent.Get(entry)

		switch state.Current {
		case StateTypeIdle:
			// ゲージをリセット
			if gauge := GaugeComponent.Get(entry); gauge != nil {
				gauge.ProgressCounter = 0
				gauge.TotalDuration = 0
				gauge.CurrentGauge = 0
			}
			// 選択されていたアクションをクリア
			if entry.HasComponent(ActionIntentComponent) {
				intent := ActionIntentComponent.Get(entry)
				intent.SelectedPartKey = ""
			}
			if entry.HasComponent(TargetComponent) {
				target := TargetComponent.Get(entry)
				target.TargetEntity = nil
				target.TargetPartSlot = ""
			}
		case StateTypeBroken:
			// ゲージをリセット
			if gauge := GaugeComponent.Get(entry); gauge != nil {
				gauge.ProgressCounter = 0
				gauge.TotalDuration = 0
				gauge.CurrentGauge = 100 // 破壊されたことを示すために100%にするなど、ゲームの仕様による
			}
			// その他のクリーンアップ処理（例：すべてのアクションをキャンセル）
			// 他の状態の入力処理が必要な場合は、ここに追加します。
		}

		// 処理後にタグを削除して、次のフレームで再処理されないようにします。
		entry.RemoveComponent(StateChangedTagComponent)
	})
}
