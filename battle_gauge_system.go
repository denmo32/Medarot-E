package main

import (
	"log"

	"medarot-ebiten/core"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ecs/entity"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdateGaugeSystem はチャージとクールダウンのゲージ進行を更新します。
func UpdateGaugeSystem(world donburi.World) {
	query.NewQuery(filter.Contains(component.StateComponent)).Each(world, func(entry *donburi.Entry) {
		state := component.StateComponent.Get(entry)

		// チャージ中またはクールダウン中のエンティティのみを処理
		if state.CurrentState != core.StateCharging && state.CurrentState != core.StateCooldown {
			return
		}

		gauge := component.GaugeComponent.Get(entry)
		gauge.ProgressCounter++
		if gauge.TotalDuration > 0 {
			gauge.CurrentGauge = (gauge.ProgressCounter / gauge.TotalDuration) * 100
		} else {
			gauge.CurrentGauge = 100 // TotalDurationが0なら即完了します。
		}

		if gauge.ProgressCounter >= gauge.TotalDuration {
			switch state.CurrentState {
			case core.StateCharging:
				state.CurrentState = core.StateReady
				actionQueueComp := entity.GetActionQueueComponent(world)
				actionQueueComp.Queue = append(actionQueueComp.Queue, entry)
				log.Printf("%s のチャージが完了。実行キューに追加。", component.SettingsComponent.Get(entry).Name)
			case core.StateCooldown:
				state.CurrentState = core.StateIdle
			}
		}
	})
}
