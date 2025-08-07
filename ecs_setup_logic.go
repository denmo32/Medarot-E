package main

import (
	"log"

	"medarot-ebiten/domain"
	"medarot-ebiten/ecs"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// InitializeBattleWorld は戦闘ワールドのECSエンティティを初期化します。
func InitializeBattleWorld(world donburi.World, res *SharedResources, playerTeam domain.TeamID) {
	EnsureActionQueueEntity(world)

	// Ensure GameStateComponent entity exists
	gameStateEntry := world.Entry(world.Create(GameStateComponent, worldStateTag))
	GameStateComponent.SetValue(gameStateEntry, domain.GameStateData{CurrentState: domain.StateGaugeProgress})

	// Ensure PlayerActionQueueComponent entity exists
	playerActionQueueEntry := world.Entry(world.Create(PlayerActionQueueComponent, worldStateTag))
	PlayerActionQueueComponent.SetValue(playerActionQueueEntry, ecs.PlayerActionQueueComponentData{Queue: make([]*donburi.Entry, 0)})

	// Ensure LastActionResultComponent entity exists
	lastActionResultEntry := world.Entry(world.Create(LastActionResultComponent, worldStateTag))
	LastActionResultComponent.SetValue(lastActionResultEntry, ecs.ActionResult{})

	teamBuffsEntry := world.Entry(world.Create(TeamBuffsComponent))
	TeamBuffsComponent.SetValue(teamBuffsEntry, ecs.TeamBuffs{
		Buffs: make(map[domain.TeamID]map[domain.BuffType][]*ecs.BuffSource),
	})

	// Initialize BattleUIStateComponent
	battleUIStateEntry := world.Entry(world.Create(BattleUIStateComponent))
	if battleUIStateEntry.Valid() {
		BattleUIStateComponent.SetValue(battleUIStateEntry, BattleUIState{
			InfoPanels: make(map[string]InfoPanelViewModel),
		})
		log.Println("BattleUIStateComponent successfully created and initialized.")
	} else {
		log.Println("ERROR: Failed to create BattleUIStateComponent entry.")
	}

	CreateMedarotEntities(world, res.GameData, playerTeam, res.GameDataManager)
}

// CreateMedarotEntities はゲームデータからECSのエンティティを生成します。
func CreateMedarotEntities(world donburi.World, gameData *domain.GameData, playerTeam domain.TeamID, gameDataManager *GameDataManager) {
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
		SettingsComponent.SetValue(entry, domain.Settings{
			ID:        loadout.ID,
			Name:      loadout.Name,
			Team:      loadout.Team,
			IsLeader:  loadout.IsLeader,
			DrawIndex: loadout.DrawIndex,
		})

		partsInstanceMap := make(map[domain.PartSlotKey]*domain.PartInstanceData)
		partIDMap := map[domain.PartSlotKey]string{
			domain.PartSlotHead:     loadout.HeadID,
			domain.PartSlotRightArm: loadout.RightArmID,
			domain.PartSlotLeftArm:  loadout.LeftArmID,
			domain.PartSlotLegs:     loadout.LegsID,
		}

		for slot, partID := range partIDMap {
			partDef, defFound := gameDataManager.GetPartDefinition(partID)
			if defFound {
				partsInstanceMap[slot] = &domain.PartInstanceData{
					DefinitionID: partDef.ID,
					CurrentArmor: partDef.MaxArmor,
					IsBroken:     false,
				}
			} else {
				log.Fatalf("エラー: ID %s のパーツ定義が見つかりません。", partID)
			}
		}
		PartsComponent.SetValue(entry, domain.PartsComponentData{Map: partsInstanceMap})

		medalDef, medalFound := gameDataManager.GetMedalDefinition(loadout.MedalID)
		if medalFound {
			MedalComponent.SetValue(entry, *medalDef)
		} else {
			log.Fatalf("エラー: ID %s のメダル定義が見つかりません。", loadout.MedalID)
		}

		StateComponent.SetValue(entry, domain.State{CurrentState: domain.StateIdle})
		GaugeComponent.SetValue(entry, domain.Gauge{})
		LogComponent.SetValue(entry, domain.Log{})
		ActionIntentComponent.SetValue(entry, domain.ActionIntent{})
		TargetComponent.SetValue(entry, ecs.Target{})

		if loadout.Team != playerTeam { // AIのみ

			donburi.Add(entry, AIComponent, &ecs.AI{
				PersonalityID:     medalDef.Personality,
				TargetHistory:     ecs.TargetHistoryData{},
				LastActionHistory: ecs.LastActionHistoryData{},
			})
		}

		if loadout.Team == playerTeam {
			donburi.Add(entry, PlayerControlComponent, &domain.PlayerControl{})
		}
	}
	log.Printf("%d体のメダロットエンティティを生成しました。", len(gameData.Medarots))
}

func FindLeader(world donburi.World, teamID domain.TeamID) *donburi.Entry {
	var leaderEntry *donburi.Entry
	query.NewQuery(filter.Contains(SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		settings := SettingsComponent.Get(entry)
		if settings.Team == teamID && settings.IsLeader {
			leaderEntry = entry
		}
	})
	return leaderEntry
}
