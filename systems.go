package main

import (
	"log"
	"sort"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// Functions below were moved or are no longer needed in this file.
// - StartCooldown (moved to action_queue_system.go as StartCooldownSystem)
// - ResetAllEffects (moved to entity_utils.go)
// - ExecuteAction (logic moved to action_queue_system.go as executeActionLogic)
// - SystemProcessReadyQueue (replaced by UpdateActionQueueSystem in action_queue_system.go)
// - ChangeState (moved to entity_utils.go)
