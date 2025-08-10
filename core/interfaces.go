package core

import (
	"image"

	"github.com/yohamta/donburi"
)

// UIMediator defines the interface for communication from game logic (ECS) to the UI.
// This helps to break the circular dependency between ecs/system and ui.
type UIMediator interface {
	// ViewModelFactory methods needed by ecs/system
	BuildInfoPanelViewModel(entry *donburi.Entry) (InfoPanelViewModel, error)
	BuildBattlefieldViewModel(world donburi.World, battlefieldRect image.Rectangle) (BattlefieldViewModel, error)
	BuildActionModalViewModel(actingEntry *donburi.Entry, actionTargetMap map[PartSlotKey]ActionTarget) (ActionModalViewModel, error)

	// UIMessageDisplayManager methods needed by ecs/system
	EnqueueMessage(msg string, callback func())
	EnqueueMessageQueue(messages []string, callback func())
	IsMessageFinished() bool

	// Other UIInterface methods needed by ecs/system
	ShowActionModal(vm ActionModalViewModel)
	HideActionModal()
	PostUIEvent(event any) // A generic event posting method
	ClearAnimation()
	ClearCurrentTarget()
	IsActionModalVisible() bool // 追加
}

// ViewModel and other shared type definitions are now located in core/types.go.
