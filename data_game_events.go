package main

import (
	"github.com/yohamta/donburi"
)

// GameEvent は、ゲームロジックから発行されるすべてのイベントを示すマーカーインターフェースです。
type GameEvent interface {
	isGameEvent()
}

// PlayerActionRequiredGameEvent は、プレイヤーの行動選択が必要になったことを示すイベントです。
type PlayerActionRequiredGameEvent struct{}

func (e PlayerActionRequiredGameEvent) isGameEvent() {}

// ActionAnimationStartedGameEvent は、アクションアニメーションが開始されたことを示すイベントです。
type ActionAnimationStartedGameEvent struct {
	AnimationData ActionAnimationData
}

func (e ActionAnimationStartedGameEvent) isGameEvent() {}

// ActionAnimationFinishedGameEvent は、アクションアニメーションが終了したことを示すイベントです。
type ActionAnimationFinishedGameEvent struct {
	Result      ActionResult
	ActingEntry *donburi.Entry // クールダウン開始のために追加
}

func (e ActionAnimationFinishedGameEvent) isGameEvent() {}

// MessageDisplayRequestGameEvent は、メッセージ表示が必要になったことを示すイベントです。
type MessageDisplayRequestGameEvent struct {
	Messages []string
	Callback func()
}

func (e MessageDisplayRequestGameEvent) isGameEvent() {}

// MessageDisplayFinishedGameEvent は、メッセージ表示が終了したことを示すイベントです。
type MessageDisplayFinishedGameEvent struct{}

func (e MessageDisplayFinishedGameEvent) isGameEvent() {}

// GameOverGameEvent は、ゲームオーバーになったことを示すイベントです。
type GameOverGameEvent struct {
	Winner TeamID
}

func (e GameOverGameEvent) isGameEvent() {}

// HideActionModalGameEvent は、アクションモーダルを隠す必要があることを示すイベントです。
type HideActionModalGameEvent struct{}

func (e HideActionModalGameEvent) isGameEvent() {}

// ShowActionModalGameEvent は、アクションモーダルを表示する必要があることを示すイベントです。
type ShowActionModalGameEvent struct {
	ViewModel ActionModalViewModel
}

func (e ShowActionModalGameEvent) isGameEvent() {}

// ClearAnimationGameEvent は、アニメーションをクリアする必要があることを示すイベントです。
type ClearAnimationGameEvent struct{}

func (e ClearAnimationGameEvent) isGameEvent() {}

// ClearCurrentTargetGameEvent は、現在のターゲットをクリアする必要があることを示すイベントです。
type ClearCurrentTargetGameEvent struct{}

func (e ClearCurrentTargetGameEvent) isGameEvent() {}

// ActionConfirmedGameEvent は、プレイヤーがアクションを確定したことを示すイベントです。
type ActionConfirmedGameEvent struct {
	ActingEntry     *donburi.Entry
	SelectedPartDef *PartDefinition
	SelectedSlotKey PartSlotKey
	TargetEntry     *donburi.Entry
	TargetPartSlot  PartSlotKey
}

func (e ActionConfirmedGameEvent) isGameEvent() {}

// ChargeRequestedGameEvent は、チャージ開始が要求されたことを示すイベントです。
type ChargeRequestedGameEvent struct {
	ActingEntry     *donburi.Entry
	SelectedSlotKey PartSlotKey
	TargetEntry     *donburi.Entry
	TargetPartSlot  PartSlotKey
}

func (e ChargeRequestedGameEvent) isGameEvent() {}

// ActionCanceledGameEvent は、プレイヤーが行動選択をキャンセルしたことを示すイベントです。
type ActionCanceledGameEvent struct {
	ActingEntry *donburi.Entry
}

func (e ActionCanceledGameEvent) isGameEvent() {}

type GoToTitleSceneGameEvent struct{}

func (e GoToTitleSceneGameEvent) isGameEvent() {}

// UIEvent は、UIから発行されるすべてのイベントを示すマーカーインターフェースです。
type UIEvent interface {
	isUIEvent()
}

// 新しい抽象化されたUIイベント

// PartSelectedUIEvent は、プレイヤーがパーツを選択したときに発行されます。
type PartSelectedUIEvent struct {
	ActingEntry     *donburi.Entry
	SelectedPartDef *PartDefinition
	SelectedSlotKey PartSlotKey
	TargetEntry     *donburi.Entry // 追加
}

func (e PartSelectedUIEvent) isUIEvent() {}

// TargetSelectedUIEvent は、プレイヤーがターゲットを選択したときに発行されます。
type TargetSelectedUIEvent struct {
	ActingEntry     *donburi.Entry
	SelectedPartDef *PartDefinition
	SelectedSlotKey PartSlotKey
	TargetEntry     *donburi.Entry
	TargetPartSlot  PartSlotKey
}

func (e TargetSelectedUIEvent) isUIEvent() {}

// ActionConfirmedUIEvent は、プレイヤーがアクションを確定したときに発行されます。
type ActionConfirmedUIEvent struct {
	ActingEntry     *donburi.Entry
	SelectedPartDef *PartDefinition
	SelectedSlotKey PartSlotKey
	TargetEntry     *donburi.Entry
	TargetPartSlot  PartSlotKey
}

func (e ActionConfirmedUIEvent) isUIEvent() {}

// ActionCanceledUIEvent は、プレイヤーが行動選択をキャンセルしたときに発行されます。
type ActionCanceledUIEvent struct {
	ActingEntry *donburi.Entry
}

func (e ActionCanceledUIEvent) isUIEvent()   {}
func (e ActionCanceledUIEvent) isGameEvent() {}

// 既存のUIイベント (変更なし)

// ShowActionModalUIEvent は、アクションモーダルを表示するUIイベントです。
type ShowActionModalUIEvent struct {
	ViewModel ActionModalViewModel
}

func (e ShowActionModalUIEvent) isUIEvent() {}

// HideActionModalUIEvent は、アクションモーダルを隠すUIイベントです。
type HideActionModalUIEvent struct{}

func (e HideActionModalUIEvent) isUIEvent() {}

// SetAnimationUIEvent は、アニメーションを設定するUIイベントです。
type SetAnimationUIEvent struct {
	AnimationData ActionAnimationData
}

func (e SetAnimationUIEvent) isUIEvent() {}

// ClearAnimationUIEvent は、アニメーションをクリアするUIイベントです。
type ClearAnimationUIEvent struct{}

func (e ClearAnimationUIEvent) isUIEvent() {}

// ClearCurrentTargetUIEvent は、現在のターゲットをクリアするUIイベントです。
type ClearCurrentTargetUIEvent struct{}

func (e ClearCurrentTargetUIEvent) isUIEvent() {}

// MessageDisplayRequestUIEvent は、メッセージ表示を要求するUIイベントです。
type MessageDisplayRequestUIEvent struct {
	Messages []string
	Callback func()
}

func (e MessageDisplayRequestUIEvent) isUIEvent() {}

// ActionResult はアクション実行の詳細な結果を保持します。
type ActionResult struct {
	ActingEntry       *donburi.Entry
	TargetEntry       *donburi.Entry
	TargetPartSlot    PartSlotKey // ターゲットのパーツスロット
	ActionDidHit      bool        // 命中したかどうか
	IsCritical        bool        // クリティカルだったか
	OriginalDamage    int         // 元のダメージ量
	DamageDealt       int         // 実際に与えたダメージ
	TargetPartBroken  bool        // ターゲットパーツが破壊されたか
	ActionIsDefended  bool        // 攻撃が防御されたか
	ActualHitPartSlot PartSlotKey // 実際にヒットしたパーツのスロット

	// 新しいメッセージ形式のための追加フィールド
	AttackerName      string
	DefenderName      string
	ActionName        string // e.g., "パーツ名"
	ActionTrait       Trait  // e.g., "撃つ", "狙い撃ち" (Trait)
	WeaponType        WeaponType
	ActionCategory    PartCategory
	TargetPartType    string // e.g., "頭部", "脚部"
	DefendingPartType string // e.g., "頭部", "脚部"

	AppliedEffects []interface{} // アクションによって適用されるステータス効果のデータ
}

// ActionAnimationData はアニメーションの再生に必要なデータを保持します。
type ActionAnimationData struct {
	Result    ActionResult
	StartTime int
}
