package main

import (
	"github.com/yohamta/donburi"
	// "github.com/yohamta/donburi/resource" // Removed as it does not exist
)

// ActionQueueResource stores the queue of entities ready to act.
type ActionQueueResource struct {
	Queue []*donburi.Entry
}

// GetActionQueue retrieves the ActionQueueResource from the world using world.Data().
// It initializes the resource if it doesn't exist.
func GetActionQueue(world donburi.World) *ActionQueueResource {
	if data := world.Data(); data != nil {
		if aqr, ok := data.(*ActionQueueResource); ok {
			return aqr
		}
	}

	// If not found or type mismatch, initialize and set it.
	// Note: This approach assumes ActionQueueResource is the *only* data
	// set via world.SetData(). If other data types are used, a more robust
	// mechanism (e.g., a map stored in world.Data()) would be needed.
	// For this specific resource, this direct approach should be fine.
	newQueue := &ActionQueueResource{
		Queue: make([]*donburi.Entry, 0),
	}
	world.SetData(newQueue)
	return newQueue
}
