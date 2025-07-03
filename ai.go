package main

import (
	"log"

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
		var found bool
		if entry.HasComponent(TargetingStrategyComponent) {
			sComp := TargetingStrategyComponent.Get(entry)
			if sComp.Strategy != nil { // Strategyはインターフェース型
				targetEntry, targetPartSlot = sComp.Strategy.SelectTarget(world, entry, targetSelector, partInfoProvider)
				found = true
			}
		}

		if !found {
			log.Printf("%s: AIエラー - TargetingStrategyComponentが見つからないか、Strategyがnilです。デフォルトのターゲット選択を使用します。", settings.Name)
			// フォールバックとしてLeaderStrategyを直接インスタンス化して使用
			targetEntry, targetPartSlot = (&LeaderStrategy{}).SelectTarget(world, entry, targetSelector, partInfoProvider)
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
