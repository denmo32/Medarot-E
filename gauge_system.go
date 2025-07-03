package main

import (
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
		if state.Current != StateTypeCharging && state.Current != StateTypeCooldown {
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
			if state.Current == StateTypeCharging {
				ChangeState(entry, StateTypeReady)
				actionQueueComp := GetActionQueueComponent(world)
				actionQueueComp.Queue = append(actionQueueComp.Queue, entry)
				log.Printf("%s のチャージが完了。実行キューに追加。", SettingsComponent.Get(entry).Name)
			} else if state.Current == StateTypeCooldown {
				ChangeState(entry, StateTypeIdle)
				// バーサーク特性の場合、クールダウン終了時に効果をリセットします。
				intent := ActionIntentComponent.Get(entry)
				if intent.SelectedPartKey != "" {
					partsComp := PartsComponent.Get(entry)
					if partsComp != nil && partsComp.Map != nil {
						if partInst, ok := partsComp.Map[intent.SelectedPartKey]; ok && partInst != nil {
							if partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID); defFound {
								if partDef.Trait == TraitBerserk {
									ResetAllEffects(world)
								}
							}
						}
					}
				}
			}
		}
	})
}
