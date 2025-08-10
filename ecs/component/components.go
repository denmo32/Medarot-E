package component

// ECSのCに相当するコンポーネント定義を集約します。
// coreパッケージ及びdonburiライブラリへの依存を想定しています。

import (
	"medarot-ebiten/core"

	"github.com/yohamta/donburi"
)



// TargetablePart はAIがターゲット可能なパーツの情報を保持します。
type TargetablePart struct {
	Entity   *donburi.Entry
	PartInst *core.PartInstanceData
	PartDef  *core.PartDefinition
	Slot     core.PartSlotKey
}

// ActionTarget はUIが選択したアクションのターゲット情報を保持します。
type ActionTarget struct {
	TargetEntityID donburi.Entity
	Slot           core.PartSlotKey
}

// --- Component Data Structs (donburi-dependent) ---

type PlayerActionQueueComponentData struct {
	Queue []*donburi.Entry
}

type ActionQueueComponentData struct {
	Queue []*donburi.Entry
}

type Target struct {
	Policy         core.TargetingPolicyType
	TargetEntity   donburi.Entity
	TargetPartSlot core.PartSlotKey
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
	LastHitPartSlot core.PartSlotKey
}

type TeamBuffs struct {
	Buffs map[core.TeamID]map[core.BuffType][]*BuffSource
}

type BuffSource struct {
	SourceEntry *donburi.Entry
	SourcePart  core.PartSlotKey
	Value       float64
}

// ActionResult はアクション実行の詳細な結果を保持します。
type ActionResult struct {
	// アクションの実行者とターゲットに関する情報
	ActingEntry    *donburi.Entry
	TargetEntry    *donburi.Entry
	TargetPartSlot core.PartSlotKey // ターゲットのパーツスロット

	// アクションの結果に関する情報
	ActionDidHit      bool             // 命中したかどうか
	IsCritical        bool             // クリティカルだったか
	OriginalDamage    int              // 元のダメージ量
	DamageDealt       int              // 実際に与えたダメージ
	ActionIsDefended  bool             // 攻撃が防御されたか
	ActualHitPartSlot core.PartSlotKey // 実際にヒットしたパーツのスロット

	// メッセージ表示のための情報
	AttackerName      string
	DefenderName      string
	ActionName        string     // e.g., "パーツ名"
	ActionTrait       core.Trait // e.g., "撃つ", "狙い撃ち" (Trait)
	WeaponType        core.WeaponType
	ActionCategory    core.PartCategory
	TargetPartType    string // e.g., "頭部", "脚部"
	DefendingPartType string // e.g., "頭部", "脚部"

	// PostActionEffectSystem で処理される情報
	AppliedEffects     []interface{}          // アクションによって適用されるステータス効果のデータ
	DamageToApply      int                    // 実際に適用するダメージ量
	TargetPartInstance *core.PartInstanceData // ダメージを受けるパーツインスタンスへのポインタ
	IsTargetPartBroken bool                   // ダメージ適用後にパーツが破壊されたか
}

// ActionAnimationData はアニメーションの再生に必要なデータを保持します。
type ActionAnimationData struct {
	Result    ActionResult
	StartTime int
}
