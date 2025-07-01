package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdateGaugeSystem はチャージとクールダウンのゲージ進行を更新します。
// このシステムは BattleScene に直接依存しません。
func UpdateGaugeSystem(world donburi.World) {
	query.NewQuery(filter.Or(
		filter.Contains(ChargingStateComponent),
		filter.Contains(CooldownStateComponent),
	)).Each(world, func(entry *donburi.Entry) {
		gauge := GaugeComponent.Get(entry)
		gauge.ProgressCounter++
		if gauge.TotalDuration > 0 {
			gauge.CurrentGauge = (gauge.ProgressCounter / gauge.TotalDuration) * 100
		} else {
			gauge.CurrentGauge = 100 // TotalDurationが0なら即完了
		}

		if gauge.ProgressCounter >= gauge.TotalDuration {
			if entry.HasComponent(ChargingStateComponent) {
				// ChangeState は entity_state_utils.go に移動することを想定
				ChangeState(entry, StateTypeReady) // world を渡す必要があれば ChangeState のシグネチャ変更も検討
				actionQueueComp := GetActionQueueComponent(world)
				actionQueueComp.Queue = append(actionQueueComp.Queue, entry)
				log.Printf("%s のチャージが完了。実行キューに追加。", SettingsComponent.Get(entry).Name)
			} else if entry.HasComponent(CooldownStateComponent) {
				// ChangeState は entity_state_utils.go に移動することを想定
				ChangeState(entry, StateTypeIdle) // world を渡す必要があれば ChangeState のシグネチャ変更も検討
				// バーサーク特性の場合、クールダウン終了時に効果をリセット
				actionComp := ActionComponent.Get(entry)
				if actionComp.SelectedPartKey != "" { // 選択パーツキーが存在するか確認
					// PartsComponent と ActionComponent は entry から取得可能
					parts := PartsComponent.Get(entry)
					if parts != nil && parts.Map != nil { // nilチェックを追加
						part := parts.Map[actionComp.SelectedPartKey]
						if part != nil && part.Trait == TraitBerserk {
							ResetAllEffects(world) // ResetAllEffects は world を引数に取る
						}
					}
				}
			}
		}
	})
}
