package ui

import (
	"github.com/yohamta/donburi"
)

// BattleUIState is a singleton component that can store UI-specific data if needed.
// It is defined in the ui package to avoid circular dependencies with ecs/component.
type BattleUIState struct {
	IsActionModalVisible bool
}

// BattleUIStateComponent is the component type for the BattleUIState.
var BattleUIStateComponent = donburi.NewComponentType[BattleUIState]()
