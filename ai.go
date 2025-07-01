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

	selected := availableParts[0]
	// ★★★ findPartSlotの呼び出しを削除し、直接スロットキーを取得 ★★★
	selectedPart := selected.Part
	slotKey := selected.Slot
	// slotKey := findPartSlot(entry, selectedPart) // ← 削除

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
	// ★★★ 古い関数呼び出しを新しい共通関数に置き換え ★★★
	candidates := GetTargetableEnemies(bs.world, actingEntry)

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
	// actingTeam := SettingsComponent.Get(actingEntry).Team
	// ★★★ 古いロジックを新しい共通関数に置き換え ★★★
	opponentTeamID := GetOpponentTeam(actingEntry)

	leader := FindLeader(bs.world, opponentTeamID)
	if leader != nil && !leader.HasComponent(BrokenStateComponent) {
		part := SelectRandomPartToDamage(leader)
		if part != nil {
			return leader, findPartSlot(leader, part)
		}
	}

	return selectRandomTargetPart(bs, actingEntry)
}
