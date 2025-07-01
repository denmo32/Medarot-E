package main

import (
	"log"
	"math/rand"
	"sort"

	"github.com/yohamta/donburi"
)

func aiSelectAction(bs *BattleScene, entry *donburi.Entry) {
	settings := SettingsComponent.Get(entry)
	medal := MedalComponent.Get(entry)
	availableParts := GetAvailableAttackParts(entry)

	if len(availableParts) == 0 {
		log.Printf("%s: AIは攻撃可能なパーツがないため待機。", settings.Name)
		return
	}

	selectedPart := availableParts[0]
	slotKey := findPartSlot(entry, selectedPart)

	if selectedPart.Category == CategoryShoot {
		var targetEntry *donburi.Entry
		var targetPartSlot PartSlotKey

		switch medal.Personality {
		case "クラッシャー":
			targetEntry, targetPartSlot = selectCrusherTarget(bs, entry)
		case "ハンター":
			targetEntry, targetPartSlot = selectHunterTarget(bs, entry)
		case "ジョーカー":
			targetEntry, targetPartSlot = selectRandomTargetPart(bs, entry)
		default:
			targetEntry, targetPartSlot = selectLeaderPart(bs, entry)
		}

		if targetEntry == nil {
			log.Printf("%s: AIは[SHOOT]の攻撃対象がいないため待機。", settings.Name)
			return
		}
		StartCharge(entry, slotKey, targetEntry, targetPartSlot, &bs.resources.Config.Balance)

	} else if selectedPart.Category == CategoryMelee {
		StartCharge(entry, slotKey, nil, "", &bs.resources.Config.Balance)
	}
}

type targetablePart struct {
	entity *donburi.Entry
	part   *Part
	slot   PartSlotKey
}

func getAllTargetableParts(bs *BattleScene, actingEntry *donburi.Entry, includeHead bool) []targetablePart {
	var allParts []targetablePart
	candidates := getTargetCandidates(bs, actingEntry)

	for _, enemyEntry := range candidates {
		partsMap := PartsComponent.Get(enemyEntry).Map
		for slotKey, part := range partsMap {
			if part.IsBroken || slotKey == PartSlotLegs {
				continue
			}
			if !includeHead && slotKey == PartSlotHead {
				continue
			}
			allParts = append(allParts, targetablePart{
				entity: enemyEntry,
				part:   part,
				slot:   slotKey,
			})
		}
	}
	return allParts
}

func selectCrusherTarget(bs *BattleScene, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(bs, actingEntry, false)

	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(bs, actingEntry, true)
	}

	if len(targetParts) == 0 {
		return nil, ""
	}

	sort.Slice(targetParts, func(i, j int) bool {
		return targetParts[i].part.Armor > targetParts[j].part.Armor
	})

	selected := targetParts[0]
	return selected.entity, selected.slot
}

func selectHunterTarget(bs *BattleScene, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(bs, actingEntry, false)

	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(bs, actingEntry, true)
	}

	if len(targetParts) == 0 {
		return nil, ""
	}

	sort.Slice(targetParts, func(i, j int) bool {
		return targetParts[i].part.Armor < targetParts[j].part.Armor
	})

	selected := targetParts[0]
	return selected.entity, selected.slot
}

func selectRandomTargetPart(bs *BattleScene, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	allEnemyParts := getAllTargetableParts(bs, actingEntry, true)
	if len(allEnemyParts) == 0 {
		return nil, ""
	}

	idx := rand.Intn(len(allEnemyParts))
	return allEnemyParts[idx].entity, allEnemyParts[idx].slot
}

func selectLeaderPart(bs *BattleScene, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	actingTeam := SettingsComponent.Get(actingEntry).Team
	opponentTeamID := Team2
	if actingTeam == Team2 {
		opponentTeamID = Team1
	}

	leader := FindLeader(bs.world, opponentTeamID)
	if leader != nil && StateComponent.Get(leader).State != StateBroken {
		part := SelectRandomPartToDamage(leader)
		if part != nil {
			return leader, findPartSlot(leader, part)
		}
	}

	return selectRandomTargetPart(bs, actingEntry)
}
