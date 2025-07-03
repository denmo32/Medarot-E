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
			gauge.CurrentGauge = 100 // TotalDurationが0なら即完了します。
		}

		if gauge.ProgressCounter >= gauge.TotalDuration {
			if entry.HasComponent(ChargingStateComponent) {
				// ChangeState は entity_utils.go に移動することを想定しています。
				ChangeState(entry, StateTypeReady) // world を渡す必要があれば ChangeState のシグネチャ変更も検討します。
				actionQueueComp := GetActionQueueComponent(world)
				actionQueueComp.Queue = append(actionQueueComp.Queue, entry)
				log.Printf("%s のチャージが完了。実行キューに追加。", SettingsComponent.Get(entry).Name)
			} else if entry.HasComponent(CooldownStateComponent) {
				// ChangeState は entity_utils.go に移動することを想定しています。
				ChangeState(entry, StateTypeIdle) // world を渡す必要があれば ChangeState のシグネチャ変更も検討します。
				// バーサーク特性の場合、クールダウン終了時に効果をリセットします。
				actionComp := ActionComponent.Get(entry)
				if actionComp.SelectedPartKey != "" {
					partsComp := PartsComponent.Get(entry)
					if partsComp != nil && partsComp.Map != nil {
						if partInst, ok := partsComp.Map[actionComp.SelectedPartKey]; ok && partInst != nil {
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
