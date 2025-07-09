package main

import (
	"context"
	"log"

	"github.com/looplab/fsm"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// CreateMedarotEntities はゲームデータからECSのエンティティを生成します。
func CreateMedarotEntities(world donburi.World, gameData *GameData, playerTeam TeamID) {
	for _, loadout := range gameData.Medarots {
		entry := world.Entry(world.Create(
			SettingsComponent,
			PartsComponent,
			MedalComponent,
			StateComponent,
			GaugeComponent,
			LogComponent,
			ActionIntentComponent,
			TargetComponent,
		))
		SettingsComponent.SetValue(entry, Settings{
			ID:        loadout.ID,
			Name:      loadout.Name,
			Team:      loadout.Team,
			IsLeader:  loadout.IsLeader,
			DrawIndex: loadout.DrawIndex,
		})

		partsInstanceMap := make(map[PartSlotKey]*PartInstanceData)
		partIDMap := map[PartSlotKey]string{
			PartSlotHead:     loadout.HeadID,
			PartSlotRightArm: loadout.RightArmID,
			PartSlotLeftArm:  loadout.LeftArmID,
			PartSlotLegs:     loadout.LegsID,
		}

		for slot, partID := range partIDMap {
			partDef, defFound := GlobalGameDataManager.GetPartDefinition(partID)
			if defFound {
				partsInstanceMap[slot] = &PartInstanceData{
					DefinitionID: partDef.ID,
					CurrentArmor: partDef.MaxArmor,
					IsBroken:     false,
				}
			} else {
				log.Fatalf("エラー: ID %s のパーツ定義が見つかりません。", partID)
			}
		}
		PartsComponent.SetValue(entry, PartsComponentData{Map: partsInstanceMap})

		medalDef, medalFound := GlobalGameDataManager.GetMedalDefinition(loadout.MedalID)
		if medalFound {
			MedalComponent.SetValue(entry, *medalDef)
		} else {
			log.Fatalf("エラー: ID %s のメダル定義が見つかりません。", loadout.MedalID)
		}

		// FSMのセットアップ
		fsmInstance := fsm.NewFSM(
			string(StateIdle),
			fsm.Events{
				// イベント名, 遷移元, 遷移先
				{Name: "charge", Src: []string{string(StateIdle)}, Dst: string(StateCharging)},
				{Name: "action_ready", Src: []string{string(StateCharging)}, Dst: string(StateReady)},
				{Name: "cooldown", Src: []string{string(StateReady)}, Dst: string(StateCooldown)},
				{Name: "cooldown_finish", Src: []string{string(StateCooldown)}, Dst: string(StateIdle)},
				{Name: "break", Src: []string{string(StateIdle), string(StateCharging), string(StateReady), string(StateCooldown)}, Dst: string(StateBroken)},
			},
			fsm.Callbacks{
				
				string(StateIdle): func(ctx context.Context, e *fsm.Event) {
					// e.Args[0] には donburi.Entry が渡される想定
					if len(e.Args) > 0 {
						if entry, ok := e.Args[0].(*donburi.Entry); ok {
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
						}
					}
				},
				string(StateBroken): func(ctx context.Context, e *fsm.Event) {
					// e.Args[0] には donburi.Entry が渡される想定
					if len(e.Args) > 0 {
						if entry, ok := e.Args[0].(*donburi.Entry); ok {
							// ゲージをリセット
							if gauge := GaugeComponent.Get(entry); gauge != nil {
								gauge.ProgressCounter = 0
								gauge.TotalDuration = 0
								gauge.CurrentGauge = 100 // 破壊されたことを示す
							}
						}
					}
				},
			},
		)

		StateComponent.SetValue(entry, State{FSM: fsmInstance})
		GaugeComponent.SetValue(entry, Gauge{})
		LogComponent.SetValue(entry, Log{})
		ActionIntentComponent.SetValue(entry, ActionIntent{})
		TargetComponent.SetValue(entry, Target{})

		if loadout.Team != playerTeam { // AIのみ
			personality, ok := PersonalityRegistry[medalDef.Personality]
			if !ok {
				log.Printf("警告: 性格 '%s' がレジストリに見つかりません。デフォルトを使用します。", medalDef.Personality)
				personality = PersonalityRegistry["リーダー"] // デフォルトの性格
			}

			donburi.Add(entry, AIComponent, &AI{
				TargetingStrategy:     personality.TargetingStrategy,
				PartSelectionStrategy: personality.PartSelectionStrategy,
				TargetHistory:         TargetHistoryData{},
				LastActionHistory:     LastActionHistoryData{},
			})
		}

		if loadout.Team == playerTeam {
			donburi.Add(entry, PlayerControlComponent, &PlayerControl{})
		}
	}
	log.Printf("%d体のメダロットエンティティを生成しました。", len(gameData.Medarots))
}

func FindLeader(world donburi.World, teamID TeamID) *donburi.Entry {
	var leaderEntry *donburi.Entry
	query.NewQuery(filter.Contains(SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		settings := SettingsComponent.Get(entry)
		if settings.Team == teamID && settings.IsLeader {
			leaderEntry = entry
		}
	})
	return leaderEntry
}
