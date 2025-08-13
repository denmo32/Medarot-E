package ui

import (
	"medarot-ebiten/core"

	"github.com/yohamta/donburi"
)

// BattleUIState is a singleton component that stores UI-specific data (ViewModels).
// It is defined in the ui package to avoid circular dependencies with ecs/component.
type BattleUIState struct {
	InfoPanels           map[string]core.InfoPanelViewModel // Map from Medarot ID to its ViewModel
	BattlefieldViewModel core.BattlefieldViewModel
	ActionModalVisible   bool
	ActionModalViewModel *core.ActionModalViewModel
}

// BattleUIStateComponent is the component type for the BattleUIState.
var BattleUIStateComponent = donburi.NewComponentType[BattleUIState]()
