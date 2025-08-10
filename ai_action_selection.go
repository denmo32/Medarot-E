package main

import (
	"log"

	"medarot-ebiten/core"
	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
)

// aiSelectAction はAI制御のメダロットの行動を決定します。
// BattleScene への依存をなくし、必要な情報を引数で受け取ります。
func aiSelectAction(
	world donburi.World,
	entry *donburi.Entry,
	battleLogic *BattleLogic,
) {
	settings := component.SettingsComponent.Get(entry)

	var slotKey core.PartSlotKey
	var selectedPartDef *core.PartDefinition
	var targetingStrategy TargetingStrategy
	var partSelectionStrategy AIPartSelectionStrategyFunc

	partInfoProvider := battleLogic.GetPartInfoProvider()
	if partInfoProvider == nil {
		log.Printf("%s: AI行動選択エラー - PartInfoProviderが初期化されていません。", settings.Name)
		return
	}
	availableParts := partInfoProvider.GetAvailableAttackParts(entry)

	if len(availableParts) == 0 {
		log.Printf("%s: AIは攻撃可能なパーツがないため待機。", settings.Name)
		return
	}

	if entry.HasComponent(component.AIComponent) {
		ai := component.AIComponent.Get(entry)
		personality, ok := PersonalityRegistry[ai.PersonalityID]
		if !ok {
			log.Printf("%s: AIエラー - PersonalityID '%s' がレジストリに見つかりません。デフォルト（ジョーカー）を使用。", settings.Name, ai.PersonalityID)
			personality = PersonalityRegistry["ジョーカー"] // フォールバック
		}
		targetingStrategy = personality.TargetingStrategy
		partSelectionStrategy = personality.PartSelectionStrategy
	} else {
		log.Printf("%s: AIエラー - AIComponentがありません。デフォルト（ジョーカー）を使用。", settings.Name)
		personality := PersonalityRegistry["ジョーカー"] // フォールバック
		targetingStrategy = personality.TargetingStrategy
		partSelectionStrategy = personality.PartSelectionStrategy
	}

	if partSelectionStrategy != nil {
		slotKey, selectedPartDef = partSelectionStrategy(entry, availableParts, world, battleLogic)
	} else {
		log.Printf("%s: AIエラー - PartSelectionStrategyがnilです。デフォルトのパーツ選択（最初のパーツ）を使用。", settings.Name)
		if len(availableParts) > 0 {
			slotKey = availableParts[0].Slot
			selectedPartDef = availableParts[0].PartDef
		}
	}

	if selectedPartDef == nil {
		log.Printf("%s: AIは戦略に基づいて選択できるパーツがありませんでした。", settings.Name)
		return
	}

	var targetEntry *donburi.Entry
	var targetPartSlot core.PartSlotKey

	if targetingStrategy != nil {
		targetEntry, targetPartSlot = targetingStrategy.SelectTarget(world, entry, battleLogic)
	} else {
		log.Printf("%s: AIエラー - TargetingStrategyがnilです。デフォルトのターゲット選択（リーダー）を使用します。", settings.Name)
		targetEntry, targetPartSlot = (&LeaderStrategy{}).SelectTarget(world, entry, battleLogic)
	}

	switch selectedPartDef.Category {
	case core.CategoryRanged:
		if targetEntry == nil {
			log.Printf("%s: AIは[射撃]の攻撃対象がいないため待機。", settings.Name)
			return
		}
		battleLogic.GetChargeInitiationSystem().StartCharge(entry, slotKey, targetEntry, targetPartSlot)
	case core.CategoryMelee:
		// 格闘の場合はターゲット選択が不要なので、nilを渡す
		battleLogic.GetChargeInitiationSystem().StartCharge(entry, slotKey, nil, "")
	case core.CategoryIntervention:
		if targetEntry == nil {
			log.Printf("%s: AIは[介入]の対象がいないため待機。", settings.Name)
			return
		}
		battleLogic.GetChargeInitiationSystem().StartCharge(entry, slotKey, targetEntry, targetPartSlot)
	default:
		log.Printf("%s: AIはパーツカテゴリ '%s' (%s) の行動を決定できませんでした。", settings.Name, selectedPartDef.PartName, selectedPartDef.Category)
	}
}

// --- AIパーツ選択戦略 ---

// SelectFirstAvailablePart は利用可能な最初のパーツを選択する単純な戦略です。
func SelectFirstAvailablePart(
	actingEntry *donburi.Entry,
	availableParts []core.AvailablePart,
	world donburi.World,
	battleLogic *BattleLogic,
) (core.PartSlotKey, *core.PartDefinition) {
	if len(availableParts) > 0 {
		return availableParts[0].Slot, availableParts[0].PartDef
	}
	return "", nil // 選択パーツなし
}

// SelectHighestPowerPart は利用可能なパーツの中で最も威力のあるパーツを選択します。
func SelectHighestPowerPart(
	actingEntry *donburi.Entry,
	availableParts []core.AvailablePart, // これは []AvailablePart{PartDef *PartDefinition, Slot PartSlotKey} です
	world donburi.World,
	battleLogic *BattleLogic,
) (core.PartSlotKey, *core.PartDefinition) {
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
	availableParts []core.AvailablePart, // これは []AvailablePart{PartDef *PartDefinition, Slot PartSlotKey} です
	world donburi.World,
	battleLogic *BattleLogic,
) (core.PartSlotKey, *core.PartDefinition) {
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
