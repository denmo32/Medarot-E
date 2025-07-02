package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// PlayerControlComponent の型定義がこうなっていることを想定
// type PlayerControl struct{}
// var PlayerControlComponent = donburi.NewComponentType[PlayerControl]()

// CreateMedarotEntities はゲームデータからECSのエンティティを生成する
func CreateMedarotEntities(world donburi.World, gameData *GameData, playerTeam TeamID) {
	for _, loadout := range gameData.Medarots {
		entry := world.Entry(world.Create(
			SettingsComponent,
			PartsComponent,
			MedalComponent,
			IdleStateComponent, // ★★★ 追加 ★★★
			GaugeComponent,
			ActionComponent,
			LogComponent,
			TargetingStrategyComponent,
			AIPartSelectionStrategyComponent, // Added AIPartSelectionStrategyComponent
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
				log.Printf("Warning: Part definition not found for ID %s. Using placeholder.", partID)
				partsInstanceMap[slot] = &PartInstanceData{
					DefinitionID: "placeholder_" + string(slot),
					CurrentArmor: 0,
					IsBroken:     true,
				}
			}
		}
		PartsComponent.SetValue(entry, PartsComponentData{Map: partsInstanceMap})

		medalDef, medalFound := GlobalGameDataManager.GetMedalDefinition(loadout.MedalID)
		if medalFound {
			MedalComponent.SetValue(entry, *medalDef)
		} else {
			log.Printf("Warning: Medal definition not found for ID %s. Using fallback.", loadout.MedalID)
			fallbackMedal := Medal{ID: "fallback", Name: "フォールバック", Personality: "ジョーカー", SkillLevel: 1}
			MedalComponent.SetValue(entry, fallbackMedal)
			medalDef = &fallbackMedal // Ensure medalDef is not nil for the switch below
		}

		GaugeComponent.SetValue(entry, Gauge{})
		ActionComponent.SetValue(entry, Action{TargetPartSlot: ""})
		LogComponent.SetValue(entry, Log{})

		var strategy TargetingStrategyFunc
		// medalDef is guaranteed to be non-nil here due to the fallback logic above
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

		if loadout.Team != playerTeam { // Only for AI
			partSelectionStrategy := SelectFirstAvailablePart
			// Example: if medalDef.Personality == "Aggressive" { partSelectionStrategy = SelectHighestPowerPart }
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
