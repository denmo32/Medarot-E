package main

import (
	"log"
	"math/rand"
	"sort"

	"github.com/yohamta/donburi"
)

func aiSelectAction(g *Game, entry *donburi.Entry) {
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

		// ★★★ 修正点1: 新しい性格の分岐を追加 ★★★
		switch medal.Personality {
		case "クラッシャー":
			targetEntry, targetPartSlot = selectCrusherTarget(g, entry)
		case "ハンター":
			targetEntry, targetPartSlot = selectHunterTarget(g, entry)
		case "ジョーカー":
			targetEntry, targetPartSlot = selectRandomTargetPart(g, entry)
		default: // デフォルトの挙動はリーダー狙い
			targetEntry, targetPartSlot = selectLeaderPart(g, entry)
		}

		if targetEntry == nil {
			log.Printf("%s: AIは[SHOOT]の攻撃対象がいないため待機。", settings.Name)
			return
		}
		StartCharge(entry, slotKey, targetEntry, targetPartSlot, &g.Config.Balance)

	} else if selectedPart.Category == CategoryMelee {
		StartCharge(entry, slotKey, nil, "", &g.Config.Balance)
	}
}

// targetablePart はソートや選択に使うための一時的な構造体
type targetablePart struct {
	entity *donburi.Entry
	part   *Part
	slot   PartSlotKey
}

// getAllTargetableParts は相手チームの攻撃可能な全パーツのリストを返す
func getAllTargetableParts(g *Game, actingEntry *donburi.Entry, includeHead bool) []targetablePart {
	var allParts []targetablePart
	candidates := getTargetCandidates(g, actingEntry)

	for _, enemyEntry := range candidates {
		partsMap := PartsComponent.Get(enemyEntry).Map
		for slotKey, part := range partsMap {
			// 脚部パーツは常に除外
			if part.IsBroken || slotKey == PartSlotLegs {
				continue
			}
			// 頭部を含めるかどうかの判定
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

// ★★★ 修正点2: 新しい性格「クラッシャー」のロジックを追加 ★★★
func selectCrusherTarget(g *Game, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	// まずは頭部以外のパーツを取得
	targetParts := getAllTargetableParts(g, actingEntry, false)

	// 頭部以外のパーツがなければ、頭部をターゲット候補に含めて再取得
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(g, actingEntry, true)
	}

	if len(targetParts) == 0 {
		return nil, ""
	}

	// 装甲値が最も高い順（降順）にソート
	sort.Slice(targetParts, func(i, j int) bool {
		return targetParts[i].part.Armor > targetParts[j].part.Armor
	})

	selected := targetParts[0]
	return selected.entity, selected.slot
}

// ★★★ 修正点3: 新しい性格「ハンター」のロジックを追加 ★★★
func selectHunterTarget(g *Game, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	// まずは頭部以外のパーツを取得
	targetParts := getAllTargetableParts(g, actingEntry, false)

	// 頭部以外のパーツがなければ、頭部をターゲット候補に含めて再取得
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(g, actingEntry, true)
	}

	if len(targetParts) == 0 {
		return nil, ""
	}

	// 装甲値が最も低い順（昇順）にソート
	sort.Slice(targetParts, func(i, j int) bool {
		return targetParts[i].part.Armor < targetParts[j].part.Armor
	})

	selected := targetParts[0]
	return selected.entity, selected.slot
}

func selectRandomTargetPart(g *Game, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	// 頭部を含めた全ての攻撃可能パーツを取得
	allEnemyParts := getAllTargetableParts(g, actingEntry, true)
	if len(allEnemyParts) == 0 {
		return nil, ""
	}

	idx := rand.Intn(len(allEnemyParts))
	return allEnemyParts[idx].entity, allEnemyParts[idx].slot
}

func selectLeaderPart(g *Game, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	actingTeam := SettingsComponent.Get(actingEntry).Team
	opponentTeamID := Team2
	if actingTeam == Team2 {
		opponentTeamID = Team1
	}

	leader := FindLeader(g.World, opponentTeamID)
	if leader != nil && StateComponent.Get(leader).State != StateBroken {
		part := SelectRandomPartToDamage(leader)
		if part != nil {
			return leader, findPartSlot(leader, part)
		}
	}

	// リーダーがいない or 破壊済み → ジョーカーと同じ動き
	return selectRandomTargetPart(g, actingEntry)
}
