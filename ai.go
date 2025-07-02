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
	var selectedPartDef *PartDefinition // Changed to selectedPartDef

	foundPartSelectionStrategy := false
	if entry.HasComponent(AIPartSelectionStrategyComponent) {
		partStrategyComp := AIPartSelectionStrategyComponent.Get(entry)
		if partStrategyComp.Strategy != nil {
			slotKey, selectedPartDef = partStrategyComp.Strategy(entry, availableParts, world, partInfoProvider, targetSelector)
			foundPartSelectionStrategy = true
		}
	}

	if !foundPartSelectionStrategy {
		log.Printf("%s: AIエラー - AIPartSelectionStrategyComponentが見つからないか有効なStrategyがないため、デフォルトのパーツ選択(最初のパーツ)を使用。", settings.Name)
		if len(availableParts) > 0 {
			slotKey = availableParts[0].Slot
			selectedPartDef = availableParts[0].PartDef // Use PartDef from AvailablePart
		}
	}

	if selectedPartDef == nil { // Check selectedPartDef
		log.Printf("%s: AIは戦略に基づいて選択できるパーツがありませんでした。", settings.Name)
		return
	}

	if selectedPartDef.Category == CategoryShoot { // Use selectedPartDef.Category
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

	} else if selectedPartDef.Category == CategoryMelee { // Use selectedPartDef.Category
		// StartCharge のシグネチャ変更に対応
		StartCharge(entry, slotKey, nil, "", world, gameConfig, partInfoProvider)
	} else {
		// Log with selectedPartDef.Category if it's not SHOOT or MELEE but still somehow selected
		// Or if it's a category that doesn't lead to StartCharge (e.g. SUPPORT, DEFENSE if they had strategies)
		log.Printf("%s: AIはパーツカテゴリ '%s' (%s) の行動を決定できませんでした。", settings.Name, selectedPartDef.PartName, selectedPartDef.Category)
	}
}

type targetablePart struct {
	entity   *donburi.Entry
	partInst *PartInstanceData // Changed from part *Part to partInst *PartInstanceData
	partDef  *PartDefinition   // Also store definition for easy access to static stats
	slot     PartSlotKey
}

// getAllTargetableParts はAIがターゲット可能な全パーツのインスタンスと定義のリストを返します。
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
		partsComp := PartsComponent.Get(enemyEntry)
		if partsComp == nil {
			continue
		}
		for slotKey, partInst := range partsComp.Map {
			if partInst.IsBroken { // Check instance for broken state
				continue
			}
			// includeHeadがfalseの場合、頭部パーツも除外 (これはパーツ種別なのでDefinitionから)
			partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID)
			if !defFound {
				log.Printf("Warning: getAllTargetableParts - PartDefinition %s not found.", partInst.DefinitionID)
				continue
			}
			if !includeHead && partDef.Type == PartTypeHead { // Compare with PartTypeHead from definition
				continue
			}
			allParts = append(allParts, targetablePart{
				entity:   enemyEntry,
				partInst: partInst,
				partDef:  partDef,
				slot:     slotKey,
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

	// 装甲が最も高いパーツを優先 (現在の耐久力で比較)
	sort.Slice(targetParts, func(i, j int) bool {
		return targetParts[i].partInst.CurrentArmor > targetParts[j].partInst.CurrentArmor
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

	// 装甲が最も低いパーツを優先 (現在の耐久力で比較)
	sort.Slice(targetParts, func(i, j int) bool {
		return targetParts[i].partInst.CurrentArmor < targetParts[j].partInst.CurrentArmor
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
) (PartSlotKey, *PartDefinition) { // Return PartDefinition
	if len(availableParts) > 0 {
		return availableParts[0].Slot, availableParts[0].PartDef
	}
	return "", nil // No part selected
}

// SelectHighestPowerPart selects the part with the highest power among available parts.
func SelectHighestPowerPart(
	actingEntry *donburi.Entry,
	availableParts []AvailablePart, // This is []AvailablePart{PartDef *PartDefinition, Slot PartSlotKey}
	world donburi.World,
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
) (PartSlotKey, *PartDefinition) { // Return PartDefinition
	if len(availableParts) == 0 {
		return "", nil
	}
	currentBestPartDef := availableParts[0].PartDef
	currentBestSlot := availableParts[0].Slot
	for _, ap := range availableParts[1:] { // ap is AvailablePart
		if ap.PartDef.Power > currentBestPartDef.Power {
			currentBestPartDef = ap.PartDef
			currentBestSlot = ap.Slot
		}
	}
	return currentBestSlot, currentBestPartDef
}

// SelectFastestChargePart selects the part with the lowest charge time.
func SelectFastestChargePart(
	actingEntry *donburi.Entry,
	availableParts []AvailablePart, // This is []AvailablePart{PartDef *PartDefinition, Slot PartSlotKey}
	world donburi.World,
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
) (PartSlotKey, *PartDefinition) { // Return PartDefinition
	if len(availableParts) == 0 {
		return "", nil
	}
	currentBestPartDef := availableParts[0].PartDef
	currentBestSlot := availableParts[0].Slot
	for _, ap := range availableParts[1:] { // ap is AvailablePart
		if ap.PartDef.Charge < currentBestPartDef.Charge { // Lower charge time is faster
			currentBestPartDef = ap.PartDef
			currentBestSlot = ap.Slot
		}
	}
	return currentBestSlot, currentBestPartDef
}
