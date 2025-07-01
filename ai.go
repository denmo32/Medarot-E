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

	if bs.partInfoProvider == nil {
		log.Printf("%s: AI行動選択エラー - PartInfoProviderが初期化されていません。", settings.Name)
		return
	}
	availableParts := bs.partInfoProvider.GetAvailableAttackParts(entry)

	if len(availableParts) == 0 {
		log.Printf("%s: AIは攻撃可能なパーツがないため待機。", settings.Name)
		return
	}

	// TODO: AIのパーツ選択ロジックをより高度化する（現在は常に最初のパーツを選択）
	selectedAvailablePart := availableParts[0]
	selectedPart := selectedAvailablePart.Part
	slotKey := selectedAvailablePart.Slot

	if selectedPart.Category == CategoryShoot {
		var targetEntry *donburi.Entry
		var targetPartSlot PartSlotKey

		switch medal.Personality {
		case "クラッシャー":
			targetEntry, targetPartSlot = selectCrusherTarget(bs, entry)
		case "ハンター":
			targetEntry, targetPartSlot = selectHunterTarget(bs, entry)
		case "ジョーカー":
			targetEntry, targetPartSlot = selectRandomTargetPartAI(bs, entry) // 名前変更: selectRandomTargetPart -> selectRandomTargetPartAI
		default:
			targetEntry, targetPartSlot = selectLeaderPart(bs, entry)
		}

		if targetEntry == nil {
			log.Printf("%s: AIは[SHOOT]の攻撃対象がいないため待機。", settings.Name)
			return
		}
		// StartCharge の第5引数を bs に変更
		StartCharge(entry, slotKey, targetEntry, targetPartSlot, bs)

	} else if selectedPart.Category == CategoryMelee {
		// StartCharge の第5引数を bs に変更
		StartCharge(entry, slotKey, nil, "", bs)
	} else {
		log.Printf("%s: AIはパーツカテゴリ '%s' の行動を決定できませんでした。", settings.Name, selectedPart.Category)
	}
}

type targetablePart struct {
	entity *donburi.Entry
	part   *Part
	slot   PartSlotKey
}

// getAllTargetableParts はAIがターゲット可能な全パーツのリストを返します。
// bs.targetSelector.GetTargetableEnemies を使用するように変更。
func getAllTargetableParts(bs *BattleScene, actingEntry *donburi.Entry, includeHead bool) []targetablePart {
	var allParts []targetablePart
	if bs.targetSelector == nil {
		log.Println("Error: getAllTargetableParts - bs.targetSelector is nil")
		return allParts
	}
	candidates := bs.targetSelector.GetTargetableEnemies(actingEntry)

	for _, enemyEntry := range candidates {
		partsMap := PartsComponent.Get(enemyEntry).Map
		for slotKey, part := range partsMap {
			// 破壊されているパーツはターゲットにしない
			if part.IsBroken {
				continue
			}
			// includeHeadがfalseの場合、頭部パーツも除外
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
	targetParts := getAllTargetableParts(bs, actingEntry, false) // 脚部以外、頭部以外
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(bs, actingEntry, true) // 脚部以外 (頭部含む)
	}
	if len(targetParts) == 0 {
		return nil, ""
	}

	// 装甲が最も高いパーツを優先
	sort.Slice(targetParts, func(i, j int) bool {
		return targetParts[i].part.Armor > targetParts[j].part.Armor
	})

	selected := targetParts[0]
	return selected.entity, selected.slot
}

func selectHunterTarget(bs *BattleScene, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(bs, actingEntry, false) // 脚部以外、頭部以外
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(bs, actingEntry, true) // 脚部以外 (頭部含む)
	}
	if len(targetParts) == 0 {
		return nil, ""
	}

	// 装甲が最も低いパーツを優先
	sort.Slice(targetParts, func(i, j int) bool {
		return targetParts[i].part.Armor < targetParts[j].part.Armor
	})

	selected := targetParts[0]
	return selected.entity, selected.slot
}

// selectRandomTargetPartAI はAI用にランダムなターゲットパーツを選択します。
// battle_logic.go の SelectRandomPartToDamage とは別物（あっちは単一ターゲットからパーツを選ぶ）。
func selectRandomTargetPartAI(bs *BattleScene, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	allEnemyParts := getAllTargetableParts(bs, actingEntry, true) // 脚部以外 (頭部含む)
	if len(allEnemyParts) == 0 {
		return nil, ""
	}

	idx := rand.Intn(len(allEnemyParts))
	return allEnemyParts[idx].entity, allEnemyParts[idx].slot
}

func selectLeaderPart(bs *BattleScene, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	if bs.targetSelector == nil || bs.partInfoProvider == nil {
		log.Println("Error: selectLeaderPart - bs.targetSelector or bs.partInfoProvider is nil")
		return selectRandomTargetPartAI(bs, actingEntry) // フォールバック
	}

	opponentTeamID := bs.targetSelector.GetOpponentTeam(actingEntry)
	leader := FindLeader(bs.world, opponentTeamID) // FindLeader は ecs_setup.go のグローバル関数

	if leader != nil && !leader.HasComponent(BrokenStateComponent) {
		// リーダーのランダムなパーツを狙う
		// TargetSelector の SelectRandomPartToDamage はターゲットエンティティ内のパーツを選ぶ
		targetPart := bs.targetSelector.SelectRandomPartToDamage(leader)
		if targetPart != nil {
			// そのパーツのスロットキーを取得
			slotKey := bs.partInfoProvider.FindPartSlot(leader, targetPart)
			if slotKey != "" {
				return leader, slotKey
			}
		}
	}
	// リーダーを狙えない場合はランダムなターゲット
	return selectRandomTargetPartAI(bs, actingEntry)
}
