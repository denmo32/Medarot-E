package main

import (
	"medarot-ebiten/domain"
	"medarot-ebiten/ecs"

	"github.com/yohamta/donburi"
)

// UIEvent は、UIから発行されるすべてのイベントを示すマーカーインターフェースです。
type UIEvent interface {
	isUIEvent()
}

// PartSelectedUIEvent は、プレイヤーがパーツを選択したときに発行されます。
type PartSelectedUIEvent struct {
	ActingEntityID  donburi.Entity
	SelectedSlotKey domain.PartSlotKey
	TargetEntityID  donburi.Entity // 追加
}

func (e PartSelectedUIEvent) isUIEvent() {}

// TargetSelectedUIEvent は、プレイヤーがターゲットを選択したときに発行されます。
type TargetSelectedUIEvent struct {
	ActingEntityID  donburi.Entity
	SelectedSlotKey domain.PartSlotKey
	TargetEntityID  donburi.Entity
	TargetPartSlot  domain.PartSlotKey
}

func (e TargetSelectedUIEvent) isUIEvent() {}

// ActionConfirmedUIEvent は、プレイヤーがアクションを確定したときに発行されます。
type ActionConfirmedUIEvent struct {
	ActingEntityID    donburi.Entity
	SelectedPartDefID string
	SelectedSlotKey   domain.PartSlotKey
	TargetEntityID    donburi.Entity
	TargetPartSlot    domain.PartSlotKey
}

func (e ActionConfirmedUIEvent) isUIEvent() {}

// ActionCanceledUIEvent は、プレイヤーが行動選択をキャンセルしたときに発行されます。
type ActionCanceledUIEvent struct {
	ActingEntityID donburi.Entity
}

func (e ActionCanceledUIEvent) isUIEvent() {}

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
	AnimationData ecs.ActionAnimationData
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

// AnimationFinishedUIEvent は、アニメーションが終了したことをUIから通知するイベントです。
type AnimationFinishedUIEvent struct {
	Result ecs.ActionResult
}

func (e AnimationFinishedUIEvent) isUIEvent() {}
