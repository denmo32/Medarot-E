package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/query"
)

// ActionQueueComponentData stores the queue of entities ready to act.
type ActionQueueComponentData struct {
	Queue []*donburi.Entry
}

// ActionQueueComponentType is the component type for ActionQueueComponentData.
var ActionQueueComponentType = donburi.NewComponentType[ActionQueueComponentData]()

// worldStateTag is a tag component to identify the world state entity.
var worldStateTag = donburi.NewComponentType[struct{}]()

// GetActionQueueComponent retrieves the ActionQueueComponentData from the world state entity.
// It expects a single entity with ActionQueueComponentType and worldStateTag to exist.
func GetActionQueueComponent(world donburi.World) *ActionQueueComponentData {
	entry, ok := query.NewQuery(query.Contains(ActionQueueComponentType)).First(world)
	if !ok {
		// This should not happen if initialized correctly in NewBattleScene
		log.Panicln("ActionQueueComponent not found in the world. It should be initialized on a world state entity.")
		return nil // Should be unreachable due to panic
	}
	return ActionQueueComponentType.Get(entry)
}

// EnsureWorldStateEntity ensures that an entity with ActionQueueComponentType and worldStateTag exists.
// If not, it creates one. This is typically called once during setup.
func EnsureActionQueueEntity(world donburi.World) *donburi.Entry {
	entry, ok := query.NewQuery(query.And(query.Contains(ActionQueueComponentType), query.Contains(worldStateTag))).First(world)
	if ok {
		return entry
	}

	log.Println("Creating ActionQueueEntity with ActionQueueComponent and worldStateTag.")
	newEntry := world.Entry(world.Create(ActionQueueComponentType, worldStateTag))
	ActionQueueComponentType.SetValue(newEntry, ActionQueueComponentData{
		Queue: make([]*donburi.Entry, 0),
	})
	return newEntry
}
