package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// InitializeBattleWorld は戦闘ワールドのECSエンティティを初期化します。
	func InitializeBattleWorld(world donburi.World, res *SharedResources, playerTeam TeamID) {
	EnsureActionQueueEntity(world)

	// Ensure GameStateComponent entity exists
	gameStateEntry := world.Entry(world.Create(GameStateComponent, worldStateTag))
	GameStateComponent.SetValue(gameStateEntry, GameStateData{CurrentState: StatePlaying})

	// Ensure PlayerActionQueueComponent entity exists
	playerActionQueueEntry := world.Entry(world.Create(PlayerActionQueueComponent, worldStateTag))
	PlayerActionQueueComponent.SetValue(playerActionQueueEntry, PlayerActionQueueComponentData{Queue: make([]*donburi.Entry, 0)})

	// Ensure LastActionResultComponent entity exists
	lastActionResultEntry := world.Entry(world.Create(LastActionResultComponent, worldStateTag))
	LastActionResultComponent.SetValue(lastActionResultEntry, ActionResult{})

	// Ensure BattlePhaseComponent entity exists
	battlePhaseEntry := world.Entry(world.Create(BattlePhaseComponent, worldStateTag))
	BattlePhaseComponent.SetValue(battlePhaseEntry, BattlePhaseData{CurrentPhase: PhaseGaugeProgress})

	

	teamBuffsEntry := world.Entry(world.Create(TeamBuffsComponent))
	TeamBuffsComponent.SetValue(teamBuffsEntry, TeamBuffs{
		Buffs: make(map[TeamID]map[BuffType][]*BuffSource),
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
func CreateMedarotEntities(world donburi.World, gameData *GameData, playerTeam TeamID, gameDataManager *GameDataManager) {
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
			partDef, defFound := gameDataManager.GetPartDefinition(partID)
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

		medalDef, medalFound := gameDataManager.GetMedalDefinition(loadout.MedalID)
		if medalFound {
			MedalComponent.SetValue(entry, *medalDef)
		} else {
			log.Fatalf("エラー: ID %s のメダル定義が見つかりません。", loadout.MedalID)
		}

		StateComponent.SetValue(entry, State{CurrentState: StateIdle})
		GaugeComponent.SetValue(entry, Gauge{})
		LogComponent.SetValue(entry, Log{})
		ActionIntentComponent.SetValue(entry, ActionIntent{})
		TargetComponent.SetValue(entry, Target{})

		if loadout.Team != playerTeam { // AIのみ

			donburi.Add(entry, AIComponent, &AI{
				PersonalityID:     medalDef.Personality,
				TargetHistory:     TargetHistoryData{},
				LastActionHistory: LastActionHistoryData{},
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
