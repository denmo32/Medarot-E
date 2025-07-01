package main

import (
	"reflect" // Added for TypeOf

	"github.com/yohamta/donburi"
)

// ActionQueueResource stores the queue of entities ready to act.
type ActionQueueResource struct {
	Queue []*donburi.Entry
}

var actionQueueResourceType = reflect.TypeOf(ActionQueueResource{})

// GetActionQueue retrieves the ActionQueueResource from the world using world.Load().
// It initializes the resource if it doesn't exist.
func GetActionQueue(world donburi.World) *ActionQueueResource {
	if data, found := world.Load(actionQueueResourceType); found {
		if aqr, ok := data.(*ActionQueueResource); ok {
			return aqr
		}
	}

	// If not found or type mismatch, initialize and set it.
	newQueue := &ActionQueueResource{
		Queue: make([]*donburi.Entry, 0),
	}
	world.Store(actionQueueResourceType, newQueue)
	return newQueue
}
