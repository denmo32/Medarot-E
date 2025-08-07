package domain

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
	ViewModel any // ActionModalViewModel への依存をなくす
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

// PlayerActionProcessedGameEvent は、プレイヤーの行動が処理されたことを示すイベントです。
type PlayerActionProcessedGameEvent struct {
	ActingEntry *donburi.Entry
}

func (e PlayerActionProcessedGameEvent) isGameEvent() {}

// PlayerActionSelectFinishedGameEvent は、プレイヤーの行動選択フェーズが完了したことを示すイベントです。
type PlayerActionSelectFinishedGameEvent struct{}

func (e PlayerActionSelectFinishedGameEvent) isGameEvent() {}

// GoToTitleSceneGameEvent は、タイトルシーンへの遷移を要求するイベントです。
type GoToTitleSceneGameEvent struct{}

func (e GoToTitleSceneGameEvent) isGameEvent() {}

// StateChangeRequestedGameEvent は、ゲームの状態変更が要求されたことを示すイベントです。
type StateChangeRequestedGameEvent struct {
	NextState GameState
}

func (e StateChangeRequestedGameEvent) isGameEvent() {}
