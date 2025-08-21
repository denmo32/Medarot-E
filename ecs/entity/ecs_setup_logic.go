package entity

import (
	"log"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"

	"medarot-ebiten/donburi"
)

// InitializeBattleWorld は戦闘ワールドのECSエンティティを初期化します。
func InitializeBattleWorld(world donburi.World, res *data.SharedResources, playerTeam core.TeamID) {
	EnsureActionQueueEntity(world) // entity. を削除

	// Ensure GameStateComponent entity exists
	gameStateEntry := world.Entry(world.Create(component.GameStateComponent, component.WorldStateTag))
	component.GameStateComponent.SetValue(gameStateEntry, core.GameStateData{CurrentState: core.StateGaugeProgress})

	// Ensure PlayerActionQueueComponent entity exists
	playerActionQueueEntry := world.Entry(world.Create(component.PlayerActionQueueComponent, component.WorldStateTag))
	component.PlayerActionQueueComponent.SetValue(playerActionQueueEntry, component.PlayerActionQueueComponentData{Queue: make([]*donburi.Entry, 0)})

	// Ensure LastActionResultComponent entity exists
	lastActionResultEntry := world.Entry(world.Create(component.LastActionResultComponent, component.WorldStateTag))
	component.LastActionResultComponent.SetValue(lastActionResultEntry, component.ActionResult{})

	teamBuffsEntry := world.Entry(world.Create(component.TeamBuffsComponent))
	component.TeamBuffsComponent.SetValue(teamBuffsEntry, component.TeamBuffs{
		Buffs: make(map[core.TeamID]map[core.BuffType][]*component.BuffSource),
	})

	CreateMedarotEntities(world, res.GameData, playerTeam, res.GameDataManager)
}

// CreateMedarotEntities はゲームデータからECSのエンティティを生成します。
func CreateMedarotEntities(world donburi.World, gameData *core.GameData, playerTeam core.TeamID, gameDataManager *data.GameDataManager) {
	for _, loadout := range gameData.Medarots {
		entry := world.Entry(world.Create(
			component.SettingsComponent,
			component.PartsComponent,
			component.MedalComponent,
			component.StateComponent,
			component.GaugeComponent,
			component.LogComponent,
			component.ActionIntentComponent,
			component.TargetComponent,
		))
		component.SettingsComponent.SetValue(entry, core.Settings{
			ID:        loadout.ID,
			Name:      loadout.Name,
			Team:      loadout.Team,
			IsLeader:  loadout.IsLeader,
			DrawIndex: loadout.DrawIndex,
		})

		partsInstanceMap := make(map[core.PartSlotKey]*core.PartInstanceData)
		partIDMap := map[core.PartSlotKey]string{
			core.PartSlotHead:     loadout.HeadID,
			core.PartSlotRightArm: loadout.RightArmID,
			core.PartSlotLeftArm:  loadout.LeftArmID,
			core.PartSlotLegs:     loadout.LegsID,
		}

		for slot, partID := range partIDMap {
			partDef, defFound := gameDataManager.GetPartDefinition(partID)
			if defFound {
				partsInstanceMap[slot] = &core.PartInstanceData{
					DefinitionID: partDef.ID,
					CurrentArmor: partDef.MaxArmor,
					IsBroken:     false,
				}
			} else {
				log.Fatalf("エラー: ID %s のパーツ定義が見つかりません。", partID)
			}
		}
		component.PartsComponent.SetValue(entry, core.PartsComponentData{Map: partsInstanceMap})

		medalDef, medalFound := gameDataManager.GetMedalDefinition(loadout.MedalID)
		if medalFound {
			component.MedalComponent.SetValue(entry, *medalDef)
		} else {
			log.Fatalf("エラー: ID %s のメダル定義が見つかりません。", loadout.MedalID)
		}

		component.StateComponent.SetValue(entry, core.State{CurrentState: core.StateIdle})
		component.GaugeComponent.SetValue(entry, core.Gauge{})
		component.LogComponent.SetValue(entry, core.Log{})
		component.ActionIntentComponent.SetValue(entry, core.ActionIntent{})
		component.TargetComponent.SetValue(entry, component.Target{})

		if loadout.Team != playerTeam { // AIのみ

			donburi.Add(entry, component.AIComponent, &component.AI{
				PersonalityID:     medalDef.Personality,
				TargetHistory:     component.TargetHistoryData{},
				LastActionHistory: component.LastActionHistoryData{},
			})
		}

		if loadout.Team == playerTeam {
			donburi.Add(entry, component.PlayerControlComponent, &core.PlayerControl{})
		}
	}
	log.Printf("%d体のメダロットエンティティを生成しました。", len(gameData.Medarots))
}

func FindLeader(world donburi.World, teamID core.TeamID) *donburi.Entry {
	var leaderEntry *donburi.Entry
	donburi.NewQuery(donburi.Contains(component.SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		settings := component.SettingsComponent.Get(entry)
		if settings.Team == teamID && settings.IsLeader {
			leaderEntry = entry
		}
	})
	return leaderEntry
}
