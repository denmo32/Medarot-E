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
	// medal := MedalComponent.Get(entry) // Removed: No longer used directly here for personality switch

	if partInfoProvider == nil {
		log.Printf("%s: AI行動選択エラー - PartInfoProviderが初期化されていません。", settings.Name)
		return
	}
	availableParts := partInfoProvider.GetAvailableAttackParts(entry)

	if len(availableParts) == 0 {
		log.Printf("%s: AIは攻撃可能なパーツがないため待機。", settings.Name)
		return
	}

	// TODO: AIのパーツ選択ロジックをより高度化する（現在は常に最初のパーツを選択） -> Strategyパターンで対応
	var slotKey PartSlotKey
	var selectedPart *Part

	foundPartSelectionStrategy := false
	if entry.HasComponent(AIPartSelectionStrategyComponent) {
		partStrategyComp := AIPartSelectionStrategyComponent.Get(entry)
		if partStrategyComp.Strategy != nil {
			slotKey, selectedPart = partStrategyComp.Strategy(entry, availableParts, world, partInfoProvider, targetSelector)
			foundPartSelectionStrategy = true
		}
	}

	if !foundPartSelectionStrategy {
		log.Printf("%s: AIエラー - AIPartSelectionStrategyComponentが見つからないか有効なStrategyがないため、デフォルトのパーツ選択(最初のパーツ)を使用。", settings.Name)
		if len(availableParts) > 0 {
			slotKey = availableParts[0].Slot
			selectedPart = availableParts[0].Part
		}
	}

	if selectedPart == nil {
		log.Printf("%s: AIは戦略に基づいて選択できるパーツがありませんでした。", settings.Name)
		return
	}

	if selectedPart.Category == CategoryShoot {
		var targetEntry *donburi.Entry
		var targetPartSlot PartSlotKey

		// Get targeting strategy from component
		var strategyComp *TargetingStrategyComponentData
		var found bool
		if entry.HasComponent(TargetingStrategyComponent) { // Check if component exists
			sComp := TargetingStrategyComponent.Get(entry) // Get the component
			if sComp.Strategy != nil {
				strategyComp = sComp
				found = true
			}
		}

		if !found {
			log.Printf("%s: AIエラー - TargetingStrategyComponentが見つからないか、Strategyがnilです。デフォルトのターゲット選択を使用します。", settings.Name)
			targetEntry, targetPartSlot = selectLeaderPart(world, entry, targetSelector, partInfoProvider) // Fallback
		} else {
			targetEntry, targetPartSlot = strategyComp.Strategy(world, entry, targetSelector, partInfoProvider)
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
	partInfoProvider *PartInfoProvider, // Added to match TargetingStrategyFunc, though not directly used here
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
		return selectRandomTargetPartAI(world, actingEntry, targetSelector, partInfoProvider) // フォールバック
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
	return selectRandomTargetPartAI(world, actingEntry, targetSelector, partInfoProvider)
}

// --- AI Part Selection Strategies ---

// SelectFirstAvailablePart is a simple strategy that selects the first available part.
func SelectFirstAvailablePart(
	actingEntry *donburi.Entry,
	availableParts []AvailablePart,
	world donburi.World,
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
) (PartSlotKey, *Part) {
	if len(availableParts) > 0 {
		return availableParts[0].Slot, availableParts[0].Part
	}
	return "", nil // No part selected
}

// SelectHighestPowerPart selects the part with the highest power among available parts.
// (Skeleton for future implementation)
func SelectHighestPowerPart(
	actingEntry *donburi.Entry,
	availableParts []AvailablePart,
	world donburi.World,
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
) (PartSlotKey, *Part) {
	if len(availableParts) == 0 {
		return "", nil
	}
	bestPart := availableParts[0].Part
	bestSlot := availableParts[0].Slot
	for _, ap := range availableParts[1:] {
		if ap.Part.Power > bestPart.Power {
			bestPart = ap.Part
			bestSlot = ap.Slot
		}
	}
	return bestSlot, bestPart
}

// SelectFastestChargePart selects the part with the lowest charge time.
// (Skeleton for future implementation)
func SelectFastestChargePart(
	actingEntry *donburi.Entry,
	availableParts []AvailablePart,
	world donburi.World,
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
) (PartSlotKey, *Part) {
	if len(availableParts) == 0 {
		return "", nil
	}
	bestPart := availableParts[0].Part
	bestSlot := availableParts[0].Slot
	// Assuming lower charge value is better
	for _, ap := range availableParts[1:] {
		if ap.Part.Charge < bestPart.Charge { // Lower charge time is faster
			bestPart = ap.Part
			bestSlot = ap.Slot
		}
	}
	return bestSlot, bestPart
}
