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
			IdleStateComponent,
			GaugeComponent,
			ActionComponent,
			LogComponent,
			TargetingStrategyComponent,
			AIPartSelectionStrategyComponent,
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

		GaugeComponent.SetValue(entry, Gauge{})
		ActionComponent.SetValue(entry, Action{TargetPartSlot: ""})
		LogComponent.SetValue(entry, Log{})

		var strategy TargetingStrategyFunc
		// 上記のフォールバックロジックにより、medalDefはここでnilでないことが保証されます
		switch medalDef.Personality {
		case "クラッシャー":
			strategy = selectCrusherTarget
		case "ハンター":
			strategy = selectHunterTarget
		case "ジョーカー":
			strategy = selectRandomTargetPartAI
		default:
			strategy = selectLeaderPart
		}
		TargetingStrategyComponent.SetValue(entry, TargetingStrategyComponentData{Strategy: strategy})

		if loadout.Team != playerTeam { // AIのみ
			partSelectionStrategy := SelectFirstAvailablePart
			// 例: if medalDef.Personality == "Aggressive" { partSelectionStrategy = SelectHighestPowerPart }
			AIPartSelectionStrategyComponent.SetValue(entry, AIPartSelectionStrategyComponentData{Strategy: partSelectionStrategy})
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
