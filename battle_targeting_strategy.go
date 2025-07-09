package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// AIPersonality はAIの性格に関連する戦略をカプセル化します。
type AIPersonality struct {
	TargetingStrategy     TargetingStrategy
	PartSelectionStrategy AIPartSelectionStrategyFunc
}

// PersonalityRegistry は、性格名をキーとしてAIPersonalityを保持するグローバルなマップです。
var PersonalityRegistry = map[string]AIPersonality{
	"ハンター": {
		TargetingStrategy:     &HunterStrategy{},
		PartSelectionStrategy: SelectHighestPowerPart,
	},
	"クラッシャー": {
		TargetingStrategy:     &CrusherStrategy{},
		PartSelectionStrategy: SelectHighestPowerPart,
	},
	"ジョーカー": {
		TargetingStrategy:     &JokerStrategy{},
		PartSelectionStrategy: SelectFastestChargePart,
	},
	"リーダー": { // デフォルト/フォールバック用
		TargetingStrategy:     &LeaderStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"アシスト": {
		TargetingStrategy:     &AssistStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"カウンター": {
		TargetingStrategy:     &CounterStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"チェイス": {
		TargetingStrategy:     &ChaseStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"デュエル": {
		TargetingStrategy:     &DuelStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"フォーカス": {
		TargetingStrategy:     &FocusStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"ガード": {
		TargetingStrategy:     &GuardStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"インターセプト": {
		TargetingStrategy:     &InterceptStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
}

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

	var slotKey PartSlotKey
	var selectedPartDef *PartDefinition

	if entry.HasComponent(AIComponent) {
		ai := AIComponent.Get(entry)
		if ai.PartSelectionStrategy != nil {
			slotKey, selectedPartDef = ai.PartSelectionStrategy(entry, availableParts, world, partInfoProvider, targetSelector)
		} else {
			// 戦略が設定されていない場合のフォールバック
			log.Printf("%s: AIエラー - PartSelectionStrategyが設定されていません。デフォルトのパーツ選択（最初のパーツ）を使用。", settings.Name)
			if len(availableParts) > 0 {
				slotKey = availableParts[0].Slot
				selectedPartDef = availableParts[0].PartDef
			}
		}
	} else {
		// AIComponentがない場合のフォールバック
		log.Printf("%s: AIエラー - AIComponentがありません。デフォルトのパーツ選択（最初のパーツ）を使用。", settings.Name)
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
	var targetPartSlot PartSlotKey

	if entry.HasComponent(AIComponent) {
		ai := AIComponent.Get(entry)
		if ai.TargetingStrategy != nil {
			targetEntry, targetPartSlot = ai.TargetingStrategy.SelectTarget(world, entry, targetSelector, partInfoProvider)
		} else {
			// 戦略が設定されていない場合のフォールバック
			log.Printf("%s: AIエラー - TargetingStrategyが設定されていません。デフォルトのターゲット選択（リーダー）を使用します。", settings.Name)
			targetEntry, targetPartSlot = (&LeaderStrategy{}).SelectTarget(world, entry, targetSelector, partInfoProvider)
		}
	} else {
		// AIComponentがない場合のフォールバック
		log.Printf("%s: AIエラー - AIComponentがありません。デフォルトのターゲット選択（リーダー）を使用します。", settings.Name)
		targetEntry, targetPartSlot = (&LeaderStrategy{}).SelectTarget(world, entry, targetSelector, partInfoProvider)
	}

	if selectedPartDef.Category == CategoryShoot {
		if targetEntry == nil {
			log.Printf("%s: AIは[射撃]の攻撃対象がいないため待機。", settings.Name)
			return
		}
		StartCharge(entry, slotKey, targetEntry, targetPartSlot, world, gameConfig, partInfoProvider)
	} else if selectedPartDef.Category == CategoryMelee {
		// 格闘の場合はターゲット選択が不要なので、nilを渡す
		StartCharge(entry, slotKey, nil, "", world, gameConfig, partInfoProvider)
	} else {
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
