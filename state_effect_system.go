package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ProcessStateEffectsSystem processes entities that have just changed state
// by looking for temporary "JustBecame..." tags.
func ProcessStateEffectsSystem(world donburi.World) {
	// Process entities that just became Idle
	query.NewQuery(filter.Contains(JustBecameIdleTagComponent)).Each(world, func(entry *donburi.Entry) {
		if entry.HasComponent(SettingsComponent) { // Log context
			log.Printf("System: Processing JustBecameIdleTag for %s. Resetting gauge and action.", SettingsComponent.Get(entry).Name)
		}

		if gauge := GaugeComponent.Get(entry); gauge != nil {
			gauge.CurrentGauge = 0
			gauge.ProgressCounter = 0
			gauge.TotalDuration = 0
		}
		if action := ActionComponent.Get(entry); action != nil {
			action.SelectedPartKey = ""
			action.TargetPartSlot = ""
			action.TargetEntity = nil
		}

		// Remove the tag after processing
		entry.RemoveComponent(JustBecameIdleTagComponent)
	})

	// Process entities that just became Broken
	query.NewQuery(filter.Contains(JustBecameBrokenTagComponent)).Each(world, func(entry *donburi.Entry) {
		if entry.HasComponent(SettingsComponent) { // Log context
			log.Printf("System: Processing JustBecameBrokenTag for %s. Resetting gauge.", SettingsComponent.Get(entry).Name)
		}

		if gauge := GaugeComponent.Get(entry); gauge != nil {
			gauge.CurrentGauge = 0
			// Other resets for Broken state might be handled by other systems or directly in ChangeState if critical
		}

		// Remove the tag after processing
		entry.RemoveComponent(JustBecameBrokenTagComponent)
	})

	// Add processing for other "JustBecame..." tags here if created
}
