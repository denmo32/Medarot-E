package main

import "github.com/yohamta/donburi"

// ActionQueueResource stores the queue of entities ready to act.
type ActionQueueResource struct {
	Queue []*donburi.Entry
}

// ActionQueueResourceType is a unique identifier for the ActionQueueResource.
var ActionQueueResourceType = donburi.NewResourceType[ActionQueueResource]()

// GetActionQueue retrieves the ActionQueueResource from the world.
// It initializes the resource if it doesn't exist.
func GetActionQueue(world donburi.World) *ActionQueueResource {
	res, ok := donburi.FindResource[ActionQueueResource](world, ActionQueueResourceType)
	if !ok {
		// Should have been initialized in NewBattleScene, but as a fallback:
		newQueue := &ActionQueueResource{
			Queue: make([]*donburi.Entry, 0),
		}
		donburi.AddResource(world, ActionQueueResourceType, newQueue)
		return newQueue
	}
	return res
}
