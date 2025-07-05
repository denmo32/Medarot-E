package main

import (
	"github.com/yohamta/donburi"
)

// UIEvent は、UIから発行されるすべてのイベントを示すマーカーインターフェースです。
type UIEvent interface {
	isUIEvent()
}

// PlayerActionSelectedEvent は、プレイヤーが使用するパーツを選択したときに発行されます。
type PlayerActionSelectedEvent struct {
	ActingEntry     *donburi.Entry
	SelectedPartDef *PartDefinition
	SelectedSlotKey PartSlotKey
}

func (e PlayerActionSelectedEvent) isUIEvent() {}

// PlayerActionCancelEvent は、プレイヤーが行動選択をキャンセルしたときに発行されます。
type PlayerActionCancelEvent struct {
	ActingEntry *donburi.Entry
}

func (e PlayerActionCancelEvent) isUIEvent() {}