package system

import (
	"log"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/donburi"
)

// aiSelectAction はAI制御のメダロットの行動を決定します。
// BattleLogicへの依存をなくし、必要なインターフェースとシステムを直接引数で受け取ります。
func aiSelectAction(
	world donburi.World,
	entry *donburi.Entry,
	partInfoProvider PartInfoProviderInterface,
	chargeSystem *ChargeInitiationSystem,
	targetSelector *TargetSelector,
	// randの型を *core.Rand から正しい *rand.Rand に修正しました。
	rand *rand.Rand,
) {
	settings := component.SettingsComponent.Get(entry)

	// 利用可能な攻撃パーツを取得
	availableParts := partInfoProvider.GetAvailableAttackParts(entry)
	if len(availableParts) == 0 {
		log.Printf("%s: AIは攻撃可能なパーツがないため待機。", settings.Name)
		return
	}

	// AIの性格に基づいた戦略を取得
	var targetingStrategy TargetingStrategy
	var partSelectionStrategy AIPartSelectionStrategyFunc
	if entry.HasComponent(component.AIComponent) {
		ai := component.AIComponent.Get(entry)
		personality, ok := PersonalityRegistry[ai.PersonalityID]
		if !ok {
			log.Printf("%s: AIエラー - PersonalityID '%s' がレジストリに見つかりません。デフォルト（リーダー）を使用。", settings.Name, ai.PersonalityID)
			personality = PersonalityRegistry["リーダー"] // フォールバック
		}
		targetingStrategy = personality.TargetingStrategy
		partSelectionStrategy = personality.PartSelectionStrategy
	} else {
		// AIコンポーネントがない場合のフォールバック
		log.Printf("%s: AIエラー - AIComponentがありません。デフォルト（リーダー）を使用。", settings.Name)
		personality := PersonalityRegistry["リーダー"] // フォールバック
		targetingStrategy = personality.TargetingStrategy
		partSelectionStrategy = personality.PartSelectionStrategy
	}

	// 1. パーツ選択戦略の実行
	// この戦略はパーツの静的データのみに依存するため、多くの引数は不要です。
	slotKey, selectedPartDef := partSelectionStrategy(entry, availableParts)
	if selectedPartDef == nil {
		log.Printf("%s: AIは戦略に基づいて選択できるパーツがありませんでした。", settings.Name)
		return
	}

	// 2. ターゲット選択戦略の実行
	// ターゲット選択はWorldの状態に依存するため、必要なシステムを渡します。
	targetEntry, targetPartSlot := targetingStrategy.SelectTarget(world, entry, targetSelector, partInfoProvider, rand)

	// 3. 行動開始
	// カテゴリに応じてチャージを開始します。
	switch selectedPartDef.Category {
	case core.CategoryRanged, core.CategoryIntervention:
		if targetEntry == nil {
			log.Printf("%s: AIは[%s]の攻撃対象がいないため待機。", settings.Name, selectedPartDef.Category)
			return
		}
		chargeSystem.StartCharge(entry, slotKey, targetEntry, targetPartSlot)
	case core.CategoryMelee:
		// 格闘の場合は実行時にターゲットが決まるため、ここではターゲットを指定しません。
		chargeSystem.StartCharge(entry, slotKey, nil, "")
	default:
		log.Printf("%s: AIはパーツカテゴリ '%s' (%s) の行動を決定できませんでした。", settings.Name, selectedPartDef.PartName, selectedPartDef.Category)
	}
}

// --- AIパーツ選択戦略 ---

// SelectFirstAvailablePart は利用可能な最初のパーツを選択する単純な戦略です。
// この関数は引数にWorldや他のシステムを必要としないため、シグネチャが簡潔になります。
func SelectFirstAvailablePart(
	actingEntry *donburi.Entry,
	availableParts []core.AvailablePart,
) (core.PartSlotKey, *core.PartDefinition) {
	if len(availableParts) > 0 {
		return availableParts[0].Slot, availableParts[0].PartDef
	}
	return "", nil // 選択パーツなし
}

// SelectHighestPowerPart は利用可能なパーツの中で最も威力のあるパーツを選択します。
func SelectHighestPowerPart(
	actingEntry *donburi.Entry,
	availableParts []core.AvailablePart,
) (core.PartSlotKey, *core.PartDefinition) {
	if len(availableParts) == 0 {
		return "", nil
	}
	var bestPartDef *core.PartDefinition
	var bestSlot core.PartSlotKey
	maxPower := -1

	for _, ap := range availableParts {
		if ap.PartDef.Power > maxPower {
			maxPower = ap.PartDef.Power
			bestPartDef = ap.PartDef
			bestSlot = ap.Slot
		}
	}
	return bestSlot, bestPartDef
}

// SelectFastestChargePart はチャージ時間が最も短いパーツを選択します。
func SelectFastestChargePart(
	actingEntry *donburi.Entry,
	availableParts []core.AvailablePart,
) (core.PartSlotKey, *core.PartDefinition) {
	if len(availableParts) == 0 {
		return "", nil
	}
	var bestPartDef *core.PartDefinition
	var bestSlot core.PartSlotKey
	minCharge := int(^uint(0) >> 1) // intの最大値

	for _, ap := range availableParts {
		if ap.PartDef.Charge < minCharge {
			minCharge = ap.PartDef.Charge
			bestPartDef = ap.PartDef
			bestSlot = ap.Slot
		}
	}
	return bestSlot, bestPartDef
}