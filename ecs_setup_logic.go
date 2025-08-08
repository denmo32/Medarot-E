package main

import (
	"log"

	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ui"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// InitializeBattleWorld は戦闘ワールドのECSエンティティを初期化します。
func InitializeBattleWorld(world donburi.World, res *SharedResources, playerTeam component.TeamID) {
	EnsureActionQueueEntity(world)

	// Ensure GameStateComponent entity exists
	gameStateEntry := world.Entry(world.Create(GameStateComponent, worldStateTag))
	GameStateComponent.SetValue(gameStateEntry, component.GameStateData{CurrentState: component.StateGaugeProgress})

	// Ensure PlayerActionQueueComponent entity exists
	playerActionQueueEntry := world.Entry(world.Create(PlayerActionQueueComponent, worldStateTag))
	PlayerActionQueueComponent.SetValue(playerActionQueueEntry, component.PlayerActionQueueComponentData{Queue: make([]*donburi.Entry, 0)})

	// Ensure LastActionResultComponent entity exists
	lastActionResultEntry := world.Entry(world.Create(LastActionResultComponent, worldStateTag))
	LastActionResultComponent.SetValue(lastActionResultEntry, component.ActionResult{})

	teamBuffsEntry := world.Entry(world.Create(TeamBuffsComponent))
	TeamBuffsComponent.SetValue(teamBuffsEntry, component.TeamBuffs{
		Buffs: make(map[component.TeamID]map[component.BuffType][]*component.BuffSource),
	})

	// Initialize BattleUIStateComponent
	battleUIStateEntry := world.Entry(world.Create(ui.BattleUIStateComponent))
	if battleUIStateEntry.Valid() {
		ui.BattleUIStateComponent.SetValue(battleUIStateEntry, ui.BattleUIState{
			InfoPanels: make(map[string]ui.InfoPanelViewModel),
		})
		log.Println("BattleUIStateComponent successfully created and initialized.")
	} else {
		log.Println("ERROR: Failed to create BattleUIStateComponent entry.")
	}

	CreateMedarotEntities(world, res.GameData, playerTeam, res.GameDataManager)
}

// CreateMedarotEntities はゲームデータからECSのエンティティを生成します。
func CreateMedarotEntities(world donburi.World, gameData *component.GameData, playerTeam component.TeamID, gameDataManager *GameDataManager) {
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
		SettingsComponent.SetValue(entry, component.Settings{
			ID:        loadout.ID,
			Name:      loadout.Name,
			Team:      loadout.Team,
			IsLeader:  loadout.IsLeader,
			DrawIndex: loadout.DrawIndex,
		})

		partsInstanceMap := make(map[component.PartSlotKey]*component.PartInstanceData)
		partIDMap := map[component.PartSlotKey]string{
			component.PartSlotHead:     loadout.HeadID,
			component.PartSlotRightArm: loadout.RightArmID,
			component.PartSlotLeftArm:  loadout.LeftArmID,
			component.PartSlotLegs:     loadout.LegsID,
		}

		for slot, partID := range partIDMap {
			partDef, defFound := gameDataManager.GetPartDefinition(partID)
			if defFound {
				partsInstanceMap[slot] = &component.PartInstanceData{
					DefinitionID: partDef.ID,
					CurrentArmor: partDef.MaxArmor,
					IsBroken:     false,
				}
			} else {
				log.Fatalf("エラー: ID %s のパーツ定義が見つかりません。", partID)
			}
		}
		PartsComponent.SetValue(entry, component.PartsComponentData{Map: partsInstanceMap})

		medalDef, medalFound := gameDataManager.GetMedalDefinition(loadout.MedalID)
		if medalFound {
			MedalComponent.SetValue(entry, *medalDef)
		} else {
			log.Fatalf("エラー: ID %s のメダル定義が見つかりません。", loadout.MedalID)
		}

		StateComponent.SetValue(entry, component.State{CurrentState: component.StateIdle})
		GaugeComponent.SetValue(entry, component.Gauge{})
		LogComponent.SetValue(entry, component.Log{})
		ActionIntentComponent.SetValue(entry, component.ActionIntent{})
		TargetComponent.SetValue(entry, component.Target{})

		if loadout.Team != playerTeam { // AIのみ

			donburi.Add(entry, AIComponent, &component.AI{
				PersonalityID:     medalDef.Personality,
				TargetHistory:     component.TargetHistoryData{},
				LastActionHistory: component.LastActionHistoryData{},
			})
		}

		if loadout.Team == playerTeam {
			donburi.Add(entry, PlayerControlComponent, &component.PlayerControl{})
		}
	}
	log.Printf("%d体のメダロットエンティティを生成しました。", len(gameData.Medarots))
}

func FindLeader(world donburi.World, teamID component.TeamID) *donburi.Entry {
	var leaderEntry *donburi.Entry
	query.NewQuery(filter.Contains(SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		settings := SettingsComponent.Get(entry)
		if settings.Team == teamID && settings.IsLeader {
			leaderEntry = entry
		}
	})
	return leaderEntry
}
