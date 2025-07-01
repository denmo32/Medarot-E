package main

import (
	"reflect"

	"github.com/yohamta/donburi"
)

// EventType is a type alias for reflect.Type, used for subscribing to events.
type EventType reflect.Type

// --- Concrete Event Types ---

// StateChangedEvent is published when an entity's state (Idle, Charging, etc.) changes.
type StateChangedEvent struct {
	Entity   *donburi.Entry // The entity whose state changed
	OldState StateType      // The previous state
	NewState StateType      // The new state
}

// StateChangedEventType is the unique type for StateChangedEvent, used for subscribing.
// We use the actual type of the event struct for this.
var StateChangedEventType EventType = reflect.TypeOf(StateChangedEvent{})
