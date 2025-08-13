package ui

import (
	"github.com/yohamta/donburi"
)

// BattleUIState is a singleton component that can store UI-specific data if needed.
// It is defined in the ui package to avoid circular dependencies with ecs/component.
// Currently, it's empty as UI state is managed within BattleUIManager.
type BattleUIState struct {
}

// BattleUIStateComponent is the component type for the BattleUIState.
var BattleUIStateComponent = donburi.NewComponentType[BattleUIState]()
