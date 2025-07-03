package main

import "github.com/yohamta/donburi"

// ActionContext は、アクション実行の各フェーズで情報を引き継ぐための構造体です。
type ActionContext struct {
	World donburi.World
	// General
	ActingEntry  *donburi.Entry
	ActionResult *ActionResult

	// Resolution Phase
	ActingPartInstance         *PartInstanceData
	ActingPartDef              *PartDefinition
	TargetEntry                *donburi.Entry
	TargetPartSlot             PartSlotKey
	IntendedTargetPartInstance *PartInstanceData
	IntendedTargetPartDef      *PartDefinition
	ActionHandler              ActionHandler

	// Hit Determination Phase
	ActionDidHit bool

	// Damage Application Phase
	IsCritical            bool
	OriginalDamage        int
	FinalDamageDealt      int
	ActualHitPartSlot     PartSlotKey
	ActualHitPartInstance *PartInstanceData
	ActualHitPartDef      *PartDefinition
	ActionIsDefended      bool

	// Dependencies
	DamageCalculator *DamageCalculator
	HitCalculator    *HitCalculator
	TargetSelector   *TargetSelector
	PartInfoProvider *PartInfoProvider
	GameConfig       *Config
}
