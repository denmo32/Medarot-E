package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ProcessStateEffectsSystem は、一時的な "JustBecame..." タグを探すことによって、
// 状態が変更されたばかりのエンティティを処理します。
func ProcessStateEffectsSystem(world donburi.World) {
	// アイドル状態になったばかりのエンティティを処理
	query.NewQuery(filter.Contains(JustBecameIdleTagComponent)).Each(world, func(entry *donburi.Entry) {
		if entry.HasComponent(SettingsComponent) { // ログコンテキスト
			log.Printf("システム: %s の JustBecameIdleTag を処理中。ゲージとアクションをリセットします。", SettingsComponent.Get(entry).Name)
		}

		if gauge := GaugeComponent.Get(entry); gauge != nil {
			gauge.CurrentGauge = 0
			gauge.ProgressCounter = 0
			gauge.TotalDuration = 0
		}
		if action := ActionComponent.Get(entry); action != nil {
			action.SelectedPartKey = ""
			action.TargetPartSlot = ""
			action.TargetEntity = nil
		}

		// 処理後にタグを削除
		entry.RemoveComponent(JustBecameIdleTagComponent)
	})

	// 破壊状態になったばかりのエンティティを処理
	query.NewQuery(filter.Contains(JustBecameBrokenTagComponent)).Each(world, func(entry *donburi.Entry) {
		if entry.HasComponent(SettingsComponent) { // ログコンテキスト
			log.Printf("システム: %s の JustBecameBrokenTag を処理中。ゲージをリセットします。", SettingsComponent.Get(entry).Name)
		}

		if gauge := GaugeComponent.Get(entry); gauge != nil {
			gauge.CurrentGauge = 0
			// 破壊状態の他のリセットは、他のシステムまたはクリティカルな場合は直接 ChangeState で処理される可能性があります
		}

		// 処理後にタグを削除
		entry.RemoveComponent(JustBecameBrokenTagComponent)
	})

	// 他の "JustBecame..." タグが作成された場合は、ここで処理を追加
}
