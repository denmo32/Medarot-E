package domain

import (
	"github.com/yohamta/donburi"
)

// NOTE: This file is allowed to depend on "github.com/yohamta/donburi".

// AvailablePart now holds PartDefinition for AI/UI to see base stats.
type AvailablePart struct {
	PartDef *PartDefinition
	Slot    PartSlotKey
}

// TargetablePart はAIがターゲット可能なパーツの情報を保持します。
type TargetablePart struct {
	Entity   *donburi.Entry
	PartInst *PartInstanceData
	PartDef  *PartDefinition
	Slot     PartSlotKey
}

// ActiveStatusEffectData は、エンティティに現在適用されている効果のデータとその残り期間を追跡します。
type ActiveStatusEffectData struct {
	EffectData   interface{}
	RemainingDur int
}

// ActionTarget はUIが選択したアクションのターゲット情報を保持します。
type ActionTarget struct {
	TargetEntityID donburi.Entity
	Slot           PartSlotKey
}

// --- Component Data Structs (donburi-dependent) ---

type PlayerActionQueueComponentData struct {
	Queue []*donburi.Entry
}

type ActiveEffects struct {
	Effects []*ActiveStatusEffectData
}

type Target struct {
	Policy         TargetingPolicyType
	TargetEntity   donburi.Entity
	TargetPartSlot PartSlotKey
}

type AI struct {
	PersonalityID     string
	TargetHistory     TargetHistoryData
	LastActionHistory LastActionHistoryData
}

type TargetHistoryData struct {
	LastAttacker *donburi.Entry
}

type LastActionHistoryData struct {
	LastHitTarget   *donburi.Entry
	LastHitPartSlot PartSlotKey
}

type TeamBuffs struct {
	Buffs map[TeamID]map[BuffType][]*BuffSource
}

type BuffSource struct {
	SourceEntry *donburi.Entry
	SourcePart  PartSlotKey
	Value       float64
}
