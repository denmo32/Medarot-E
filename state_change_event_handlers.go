package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// RegisterStateChangeEventHandlers registers all systems that listen to StateChangedEvent.
// This function should be called once during BattleScene setup.
func RegisterStateChangeEventHandlers(world donburi.World) {
	donburi.Subscribe(world, StateChangedEventType, handleStateChangeForGaugeReset)
	donburi.Subscribe(world, StateChangedEventType, handleStateChangeForIdleAI)
	// Add other handlers here
}

// handleStateChangeForGaugeReset resets gauge and action components when an entity becomes Idle.
func handleStateChangeForGaugeReset(event interface{}) {
	ev, ok := event.(StateChangedEvent)
	if !ok {
		return
	}

	if ev.NewState == StateTypeIdle {
		if ev.Entity.HasComponent(SettingsComponent) { // Log context
			log.Printf("Handler: %s became Idle, resetting gauge and action.", SettingsComponent.Get(ev.Entity).Name)
		}

		if gauge := GaugeComponent.Get(ev.Entity); gauge != nil {
			gauge.CurrentGauge = 0
			gauge.ProgressCounter = 0
			gauge.TotalDuration = 0
		}
		if action := ActionComponent.Get(ev.Entity); action != nil {
			action.SelectedPartKey = ""
			action.TargetPartSlot = ""
			action.TargetEntity = nil
		}
	}
	// Example: Also reset gauge when Broken
	if ev.NewState == StateTypeBroken {
		if ev.Entity.HasComponent(SettingsComponent) {
			log.Printf("Handler: %s became Broken, resetting gauge.", SettingsComponent.Get(ev.Entity).Name)
		}
		if gauge := GaugeComponent.Get(ev.Entity); gauge != nil {
			gauge.CurrentGauge = 0
			// Other resets for Broken state might be handled elsewhere or by other event handlers
		}
	}
}

// handleStateChangeForIdleAI is an example handler that might trigger AI processing
// when an AI-controlled entity becomes Idle.
// Note: Current AI processing is in UpdateAIInputSystem which queries for IdleStateComponent.
// This event-driven approach could be an alternative or complementary way.
func handleStateChangeForIdleAI(event interface{}) {
	ev, ok := event.(StateChangedEvent)
	if !ok {
		return
	}

	if ev.NewState == StateTypeIdle &&
		!ev.Entity.HasComponent(PlayerControlComponent) && // Not player controlled
		ev.Entity.HasComponent(SettingsComponent) { // Is a medarot

		// Potential place to trigger an immediate AI action evaluation for this entity.
		// However, the current AI system (UpdateAIInputSystem) polls idle AIs each frame.
		// Mixing polling and event-driven AI triggers needs careful design to avoid duplicate actions
		// or race conditions.
		// For now, this can just be a log or a placeholder.
		// log.Printf("Handler: AI entity %s became Idle. (AI system will poll)", SettingsComponent.Get(ev.Entity).Name)

		// If we wanted to make AI fully event-driven for action selection:
		// aiSelectAction(ev.Entity.World, ev.Entity, partInfo, targetSelector, config)
		// This would require passing necessary dependencies (partInfo, etc.) to event handlers,
		// which can be complex. Sticking to polling for AI for now is simpler.
	}
}
