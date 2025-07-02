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
	gameConfig *Config,
) {
	settings := SettingsComponent.Get(entry)

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
	var selectedPartDef *PartDefinition

	foundPartSelectionStrategy := false
	if entry.HasComponent(AIPartSelectionStrategyComponent) {
		partStrategyComp := AIPartSelectionStrategyComponent.Get(entry)
		if partStrategyComp.Strategy != nil {
			slotKey, selectedPartDef = partStrategyComp.Strategy(entry, availableParts, world, partInfoProvider, targetSelector)
			foundPartSelectionStrategy = true
		}
	}

	if !foundPartSelectionStrategy {
		log.Printf("%s: AIエラー - AIPartSelectionStrategyComponentが見つからないか有効なStrategyがないため、デフォルトのパーツ選択（最初のパーツ）を使用。", settings.Name)
		if len(availableParts) > 0 {
			slotKey = availableParts[0].Slot
			selectedPartDef = availableParts[0].PartDef
		}
	}

	if selectedPartDef == nil {
		log.Printf("%s: AIは戦略に基づいて選択できるパーツがありませんでした。", settings.Name)
		return
	}

	switch selectedPartDef.Category {
	case CategoryShoot:
		var targetEntry *donburi.Entry
		var targetPartSlot PartSlotKey

		// コンポーネントからターゲティング戦略を取得
		var strategyComp *TargetingStrategyComponentData
		var found bool
		if entry.HasComponent(TargetingStrategyComponent) {
			sComp := TargetingStrategyComponent.Get(entry)
			if sComp.Strategy != nil {
				strategyComp = sComp
				found = true
			}
		}

		if !found {
			log.Printf("%s: AIエラー - TargetingStrategyComponentが見つからないか、Strategyがnilです。デフォルトのターゲット選択を使用します。", settings.Name)
			targetEntry, targetPartSlot = selectLeaderPart(world, entry, targetSelector, partInfoProvider) // フォールバック
		} else {
			targetEntry, targetPartSlot = strategyComp.Strategy(world, entry, targetSelector, partInfoProvider)
		}

		if targetEntry == nil {
			log.Printf("%s: AIは[射撃]の攻撃対象がいないため待機。", settings.Name)
			return
		}
		StartCharge(entry, slotKey, targetEntry, targetPartSlot, world, gameConfig, partInfoProvider)

	case CategoryMelee:
		StartCharge(entry, slotKey, nil, "", world, gameConfig, partInfoProvider)
	default:
		// 選択されたパーツが射撃でも格闘でもない場合、またはStartChargeにつながらないカテゴリの場合のログ
		// (例: サポート、防御カテゴリで戦略があった場合など)
		log.Printf("%s: AIはパーツカテゴリ '%s' (%s) の行動を決定できませんでした。", settings.Name, selectedPartDef.PartName, selectedPartDef.Category)
	}
}

type targetablePart struct {
	entity   *donburi.Entry
	partInst *PartInstanceData
	partDef  *PartDefinition // 静的なステータスに簡単にアクセスできるように定義も格納
	slot     PartSlotKey
}

// getAllTargetableParts はAIがターゲット可能な全パーツのインスタンスと定義のリストを返します。
func getAllTargetableParts(actingEntry *donburi.Entry, targetSelector *TargetSelector, includeHead bool) []targetablePart {
	var allParts []targetablePart
	if targetSelector == nil {
		log.Println("エラー: getAllTargetableParts - targetSelectorがnilです。")
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
			if partInst.IsBroken { // インスタンスの破損状態を確認
				continue
			}
			// includeHeadがfalseの場合、頭部パーツも除外 (これはパーツ種別なのでDefinitionから)
			partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID)
			if !defFound {
				log.Printf("警告: getAllTargetableParts - PartDefinition %s が見つかりません。", partInst.DefinitionID)
				continue
			}
			if !includeHead && partDef.Type == PartTypeHead { // 定義のPartTypeHeadと比較
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
	targetParts := getAllTargetableParts(actingEntry, targetSelector, false) // 脚部以外、頭部以外
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(actingEntry, targetSelector, true) // 脚部以外 (頭部含む)
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
	targetParts := getAllTargetableParts(actingEntry, targetSelector, false) // 脚部以外、頭部以外
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(actingEntry, targetSelector, true) // 脚部以外 (頭部含む)
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
	partInfoProvider *PartInfoProvider, // TargetingStrategyFuncに合わせるため追加 (ここでは直接使用しない)
) (*donburi.Entry, PartSlotKey) {
	allEnemyParts := getAllTargetableParts(actingEntry, targetSelector, true) // 脚部以外 (頭部含む)
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
		log.Println("エラー: selectLeaderPart - targetSelector または partInfoProvider がnilです。")
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

// --- AIパーツ選択戦略 ---

// SelectFirstAvailablePart は利用可能な最初のパーツを選択する単純な戦略です。
func SelectFirstAvailablePart(
	actingEntry *donburi.Entry,
	availableParts []AvailablePart,
	world donburi.World,
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
) (PartSlotKey, *PartDefinition) {
	if len(availableParts) > 0 {
		return availableParts[0].Slot, availableParts[0].PartDef
	}
	return "", nil // 選択パーツなし
}

// SelectHighestPowerPart は利用可能なパーツの中で最も威力のあるパーツを選択します。
func SelectHighestPowerPart(
	actingEntry *donburi.Entry,
	availableParts []AvailablePart, // これは []AvailablePart{PartDef *PartDefinition, Slot PartSlotKey} です
	world donburi.World,
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
) (PartSlotKey, *PartDefinition) {
	if len(availableParts) == 0 {
		return "", nil
	}
	currentBestPartDef := availableParts[0].PartDef
	currentBestSlot := availableParts[0].Slot
	for _, ap := range availableParts[1:] { // ap は AvailablePart です
		if ap.PartDef.Power > currentBestPartDef.Power {
			currentBestPartDef = ap.PartDef
			currentBestSlot = ap.Slot
		}
	}
	return currentBestSlot, currentBestPartDef
}

// SelectFastestChargePart はチャージ時間が最も短いパーツを選択します。
func SelectFastestChargePart(
	actingEntry *donburi.Entry,
	availableParts []AvailablePart, // これは []AvailablePart{PartDef *PartDefinition, Slot PartSlotKey} です
	world donburi.World,
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
) (PartSlotKey, *PartDefinition) {
	if len(availableParts) == 0 {
		return "", nil
	}
	currentBestPartDef := availableParts[0].PartDef
	currentBestSlot := availableParts[0].Slot
	for _, ap := range availableParts[1:] { // ap は AvailablePart です
		if ap.PartDef.Charge < currentBestPartDef.Charge { // チャージ時間が短いほど速い
			currentBestPartDef = ap.PartDef
			currentBestSlot = ap.Slot
		}
	}
	return currentBestSlot, currentBestPartDef
}
