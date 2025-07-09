package main

import (
	"context"
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdateGaugeSystem はチャージとクールダウンのゲージ進行を更新します。
func UpdateGaugeSystem(world donburi.World) {
	query.NewQuery(filter.Contains(StateComponent)).Each(world, func(entry *donburi.Entry) {
		state := StateComponent.Get(entry)

		// チャージ中またはクールダウン中のエンティティのみを処理
		if !state.FSM.Is(string(StateCharging)) && !state.FSM.Is(string(StateCooldown)) {
			return
		}

		gauge := GaugeComponent.Get(entry)
		gauge.ProgressCounter++
		if gauge.TotalDuration > 0 {
			gauge.CurrentGauge = (gauge.ProgressCounter / gauge.TotalDuration) * 100
		} else {
			gauge.CurrentGauge = 100 // TotalDurationが0なら即完了します。
		}

		if gauge.ProgressCounter >= gauge.TotalDuration {
			ctx := context.Background()
			if state.FSM.Is(string(StateCharging)) {
				err := state.FSM.Event(ctx, "action_ready", entry)
				if err != nil {
					log.Printf("Error transitioning to ready state for %s: %v", SettingsComponent.Get(entry).Name, err)
				}
				actionQueueComp := GetActionQueueComponent(world)
				actionQueueComp.Queue = append(actionQueueComp.Queue, entry)
				log.Printf("%s のチャージが完了。実行キューに追加。", SettingsComponent.Get(entry).Name)
			} else if state.FSM.Is(string(StateCooldown)) {
				err := state.FSM.Event(ctx, "cooldown_finish", entry)
				if err != nil {
					log.Printf("Error transitioning to idle state for %s: %v", SettingsComponent.Get(entry).Name, err)
				}
			}
		}
	})
}
