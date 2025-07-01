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
			// ★★★ 削除
			// StateComponent,
			IdleStateComponent, // ★★★ 追加 ★★★
			GaugeComponent,
			ActionComponent,
			LogComponent,
			TargetingStrategyComponent,
			AIPartSelectionStrategyComponent, // Added AIPartSelectionStrategyComponent
			// EffectsComponent,
		))
		SettingsComponent.SetValue(entry, Settings{
			ID:        loadout.ID,
			Name:      loadout.Name,
			Team:      loadout.Team,
			IsLeader:  loadout.IsLeader,
			DrawIndex: loadout.DrawIndex,
		})
		partsMap := make(map[PartSlotKey]*Part)
		partIDMap := map[PartSlotKey]string{
			PartSlotHead:     loadout.HeadID,
			PartSlotRightArm: loadout.RightArmID,
			PartSlotLeftArm:  loadout.LeftArmID,
			PartSlotLegs:     loadout.LegsID,
		}
		for slot, partID := range partIDMap {
			if p, exists := gameData.AllParts[partID]; exists {
				newPart := *p
				newPart.IsBroken = false
				partsMap[slot] = &newPart
			} else {
				placeholderPart := &Part{ID: "placeholder", PartName: "なし", IsBroken: true}
				partsMap[slot] = placeholderPart
			}
		}
		PartsComponent.SetValue(entry, Parts{Map: partsMap})
		medal := findMedalByID(gameData.Medals, loadout.MedalID)
		if medal == nil {
			medal = &Medal{ID: "fallback", Name: "フォールバック", SkillLevel: 1}
		}
		MedalComponent.SetValue(entry, *medal)
		// ★★★ 削除
		// StateComponent.SetValue(entry, State{State: StateIdle})
		GaugeComponent.SetValue(entry, Gauge{})
		ActionComponent.SetValue(entry, Action{TargetPartSlot: ""}) // TargetPartSlotを初期化
		LogComponent.SetValue(entry, Log{})

		// ★★★ EffectsComponentの初期値を設定 ★★★
		// EffectsComponent.SetValue(entry, Effects{
		//	EvasionRateMultiplier: 1.0, // 通常は1.0 (100%)
		//	DefenseRateMultiplier: 1.0, // 通常は1.0 (100%)
		// })

		// Set Targeting Strategy based on medal personality
		var strategy TargetingStrategyFunc
		switch medal.Personality {
		case "クラッシャー":
			strategy = selectCrusherTarget
		case "ハンター":
			strategy = selectHunterTarget
		case "ジョーカー":
			strategy = selectRandomTargetPartAI
		default: // デフォルトや不明な性格の場合
			strategy = selectLeaderPart // Default to targeting leader or a fallback like random
		}
		TargetingStrategyComponent.SetValue(entry, TargetingStrategyComponentData{Strategy: strategy})

		// Set default AI Part Selection Strategy for non-player entities
		if loadout.Team != playerTeam { // Only for AI
			// Default strategy, can be customized later based on AI type or medal
			partSelectionStrategy := SelectFirstAvailablePart
			// Example: if medal.Personality == "Aggressive" { partSelectionStrategy = SelectHighestPowerPart }
			AIPartSelectionStrategyComponent.SetValue(entry, AIPartSelectionStrategyComponentData{Strategy: partSelectionStrategy})
		}

		// プレイヤーチームならPlayerControlタグを追加
		if loadout.Team == playerTeam {
			// 修正: 第3引数を追加
			donburi.Add(entry, PlayerControlComponent, &PlayerControl{})
		}
	}
	log.Printf("%d体のメダロットエンティティを生成しました。", len(gameData.Medarots))
}

// ( ... findMedalByID, FindLeader は変更なし ... )

func findMedalByID(allMedals []Medal, id string) *Medal {
	for _, medal := range allMedals {
		if medal.ID == id {
			newMedal := medal
			return &newMedal
		}
	}
	return nil
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
