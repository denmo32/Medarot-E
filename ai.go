package main

import (
	"log"
	"math/rand"

	"github.com/yohamta/donburi"
)

// SettingsComponent や PartsComponent などの変数が
// このように定義されていることを想定しています。
// var SettingsComponent = donburi.NewComponentType[Settings]()
// var PartsComponent = donburi.NewComponentType[Parts]()
// var MedalComponent = donburi.NewComponentType[Medal]()
// var StateComponent = donburi.NewComponentType[State]()

func aiSelectAction(g *Game, entry *donburi.Entry) {
	settings := SettingsComponent.Get(entry)
	medal := MedalComponent.Get(entry)
	availableParts := GetAvailableAttackParts(entry)

	if len(availableParts) == 0 {
		log.Printf("%s: AIは攻撃可能なパーツがないため待機。", settings.Name)
		return
	}

	// TODO: 性格に応じて使用パーツも変えるべきだが、今回は最初の有効なパーツを使う
	selectedPart := availableParts[0]
	slotKey := findPartSlot(entry, selectedPart)

	if selectedPart.Category == CategoryShoot {
		var targetEntry *donburi.Entry
		var targetPartSlot PartSlotKey

		switch medal.Personality {
		case "ジョーカー":
			targetEntry, targetPartSlot = selectRandomTargetPart(g, entry)
		default: // デフォルトの挙動としてリーダーを狙う
			targetEntry, targetPartSlot = selectLeaderPart(g, entry)
		}

		if targetEntry == nil {
			log.Printf("%s: AIは[SHOOT]の攻撃対象がいないため待機。", settings.Name)
			return
		}
		StartCharge(entry, slotKey, targetEntry, targetPartSlot, &g.Config.Balance)

	} else if selectedPart.Category == CategoryMelee {
		// FIGHTの場合、ターゲットは実行時に決定されるので、ここではnilでチャージ開始
		StartCharge(entry, slotKey, nil, "", &g.Config.Balance)
	}
}

// selectRandomTargetPart は「ジョーカー」性格用のターゲット選択ロジック
func selectRandomTargetPart(g *Game, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	var allEnemyEntries []*donburi.Entry
	var allEnemySlots []PartSlotKey

	candidates := getTargetCandidates(g, actingEntry)
	for _, enemyEntry := range candidates {
		partsMap := PartsComponent.Get(enemyEntry).Map
		for slot, part := range partsMap {
			if !part.IsBroken {
				allEnemyEntries = append(allEnemyEntries, enemyEntry)
				allEnemySlots = append(allEnemySlots, slot)
			}
		}
	}

	if len(allEnemyEntries) == 0 {
		return nil, ""
	}

	idx := rand.Intn(len(allEnemyEntries))
	return allEnemyEntries[idx], allEnemySlots[idx]
}

// selectLeaderPart はデフォルト用のターゲット選択ロジック
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
