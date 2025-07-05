package main

import (
	"log"

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

		StateComponent.SetValue(entry, State{Current: StateTypeIdle, StateEnterAt: 0})
		GaugeComponent.SetValue(entry, Gauge{})
		LogComponent.SetValue(entry, Log{})
		ActionIntentComponent.SetValue(entry, ActionIntent{})
		TargetComponent.SetValue(entry, Target{})

		if loadout.Team != playerTeam { // AIのみ
			var strategy TargetingStrategy
			switch medalDef.Personality {
			case "アシスト":
				strategy = &AssistStrategy{}
			case "クラッシャー":
				strategy = &CrusherStrategy{}
			case "カウンター":
				strategy = &CounterStrategy{}
			case "チェイス":
				strategy = &ChaseStrategy{}
			case "デュエル":
				strategy = &DuelStrategy{}
			case "フォーカス":
				strategy = &FocusStrategy{}
			case "ガード":
				strategy = &GuardStrategy{}
			case "ハンター":
				strategy = &HunterStrategy{}
			case "インターセプト":
				strategy = &InterceptStrategy{}
			case "ジョーカー":
				strategy = &JokerStrategy{}
			default:
				strategy = &LeaderStrategy{} // デフォルトはリーダー狙い
			}

			partSelectionStrategy := SelectFirstAvailablePart
			switch medalDef.Personality {
			case "ハンター":
				partSelectionStrategy = SelectHighestPowerPart
			case "ジョーカー":
				partSelectionStrategy = SelectFastestChargePart
			// 他の性格に応じた戦略を追加
			}

			donburi.Add(entry, AIComponent, &AI{
				TargetingStrategy:     strategy,
				PartSelectionStrategy: partSelectionStrategy,
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
