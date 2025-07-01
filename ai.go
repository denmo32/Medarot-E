package main

import (
	"log"
	"math/rand"
	"sort"

	"github.com/yohamta/donburi"
)

// aiSelectAction はAI制御のメダロットの行動を決定します。
// BattleScene への依存をなくし、必要な情報を引数で受け取ります。
func aiSelectAction(
	world donburi.World,
	entry *donburi.Entry,
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
	gameConfig *Config, // StartCharge が Config を必要とするため
) {
	settings := SettingsComponent.Get(entry)
	medal := MedalComponent.Get(entry)

	if partInfoProvider == nil {
		log.Printf("%s: AI行動選択エラー - PartInfoProviderが初期化されていません。", settings.Name)
		return
	}
	availableParts := partInfoProvider.GetAvailableAttackParts(entry)

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
			targetEntry, targetPartSlot = selectCrusherTarget(world, entry, targetSelector, partInfoProvider)
		case "ハンター":
			targetEntry, targetPartSlot = selectHunterTarget(world, entry, targetSelector, partInfoProvider)
		case "ジョーカー":
			targetEntry, targetPartSlot = selectRandomTargetPartAI(world, entry, targetSelector)
		default:
			targetEntry, targetPartSlot = selectLeaderPart(world, entry, targetSelector, partInfoProvider)
		}

		if targetEntry == nil {
			log.Printf("%s: AIは[SHOOT]の攻撃対象がいないため待機。", settings.Name)
			return
		}
		// StartCharge のシグネチャ変更に対応
		StartCharge(entry, slotKey, targetEntry, targetPartSlot, world, gameConfig, partInfoProvider)

	} else if selectedPart.Category == CategoryMelee {
		// StartCharge のシグネチャ変更に対応
		StartCharge(entry, slotKey, nil, "", world, gameConfig, partInfoProvider)
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
func getAllTargetableParts(
	world donburi.World, // world を追加 (未使用だが一貫性のため)
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	includeHead bool,
) []targetablePart {
	var allParts []targetablePart
	if targetSelector == nil {
		log.Println("Error: getAllTargetableParts - targetSelector is nil")
		return allParts
	}
	// targetSelector.GetTargetableEnemies は world を引数に取るように変更される想定
	// (現状は world を内部で持っているが、将来的には引数で渡す方が良い)
	candidates := targetSelector.GetTargetableEnemies(actingEntry)

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

func selectCrusherTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider, // getAllTargetableParts が必要とする可能性を考慮 (現状は未使用)
) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(world, actingEntry, targetSelector, false) // 脚部以外、頭部以外
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(world, actingEntry, targetSelector, true) // 脚部以外 (頭部含む)
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

func selectHunterTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider, // getAllTargetableParts が必要とする可能性を考慮 (現状は未使用)
) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(world, actingEntry, targetSelector, false) // 脚部以外、頭部以外
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(world, actingEntry, targetSelector, true) // 脚部以外 (頭部含む)
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
func selectRandomTargetPartAI(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
) (*donburi.Entry, PartSlotKey) {
	allEnemyParts := getAllTargetableParts(world, actingEntry, targetSelector, true) // 脚部以外 (頭部含む)
	if len(allEnemyParts) == 0 {
		return nil, ""
	}

	idx := rand.Intn(len(allEnemyParts))
	return allEnemyParts[idx].entity, allEnemyParts[idx].slot
}

func selectLeaderPart(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) (*donburi.Entry, PartSlotKey) {
	if targetSelector == nil || partInfoProvider == nil {
		log.Println("Error: selectLeaderPart - targetSelector or partInfoProvider is nil")
		return selectRandomTargetPartAI(world, actingEntry, targetSelector) // フォールバック
	}

	opponentTeamID := targetSelector.GetOpponentTeam(actingEntry)
	leader := FindLeader(world, opponentTeamID) // FindLeader は ecs_setup.go のグローバル関数 (world を引数に取る)

	if leader != nil && !leader.HasComponent(BrokenStateComponent) {
		targetPart := targetSelector.SelectRandomPartToDamage(leader)
		if targetPart != nil {
			slotKey := partInfoProvider.FindPartSlot(leader, targetPart)
			if slotKey != "" {
				return leader, slotKey
			}
		}
	}
	return selectRandomTargetPartAI(world, actingEntry, targetSelector)
}
