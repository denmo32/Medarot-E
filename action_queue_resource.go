package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/resource" // Import the resource subpackage
)

// ActionQueueResource stores the queue of entities ready to act.
type ActionQueueResource struct {
	Queue []*donburi.Entry
}

// ActionQueueResourceType is not strictly needed if using resource.Set and resource.Get with the type itself.
// However, if NewResourceType was intended for something else or if a specific API required it,
// this would need to be revisited. Given the errors, we'll rely on type-based Set/Get.
// var ActionQueueResourceType = donburi.NewComponentType[ActionQueueResource]() // Corrected from NewResourceType if it was a typo for component style


// GetActionQueue retrieves the ActionQueueResource from the world.
// It initializes the resource if it doesn't exist.
func GetActionQueue(world donburi.World) *ActionQueueResource {
	// Use resource.Get to retrieve the resource by its type.
	// It returns the resource and a boolean indicating if it was found.
	res, found := resource.Get[ActionQueueResource](world)
	if !found {
		// Should have been initialized in NewBattleScene, but as a fallback:
		newQueue := &ActionQueueResource{
			Queue: make([]*donburi.Entry, 0),
		}
		// Use resource.Set to add/update the resource in the world.
		resource.Set(world, newQueue)
		return newQueue
	}
	return res
}
