package main

import (
	"medarot-ebiten/domain"

	"github.com/yohamta/donburi"
)

// AIPartSelectionStrategyFunc はAIパーツ選択戦略の関数シグネチャを定義します。
// 行動するAIエンティティと利用可能なパーツのリストを受け取り、選択されたパーツとそのスロットを返します。
type AIPartSelectionStrategyFunc func(
	actingEntry *donburi.Entry,
	availableParts []domain.AvailablePart,
	world donburi.World,
	battleLogic *BattleLogic,
) (domain.PartSlotKey, *domain.PartDefinition)

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
