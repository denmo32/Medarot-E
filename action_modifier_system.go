package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// ApplyActionModifiersSystem calculates and applies temporary action modifiers
// to an acting entity based on its traits, medal, etc.
// This system should be called before hit/damage calculations.
func ApplyActionModifiersSystem(
	world donburi.World, // World might be needed if effects depend on global state or other entities
	actingEntry *donburi.Entry,
	gameConfig *Config, // For accessing balance numbers
	partInfoProvider *PartInfoProvider, // For things like propulsion for Berserk
) {
	if actingEntry == nil || !actingEntry.Valid() {
		return
	}

	modifiers := ActionModifierComponentData{
		// Initialize with defaults or neutral values
		CriticalRateBonus:     0,
		CriticalMultiplier:    0, // 0 means use gameConfig.Balance.Damage.CriticalMultiplier
		PowerAdditiveBonus:    0,
		PowerMultiplierBonus:  1.0, // Multiplier starts at 1.0 (no change)
		DamageAdditiveBonus:   0,
		DamageMultiplierBonus: 1.0,
		AccuracyAdditiveBonus: 0,
	}

	settings := SettingsComponent.Get(actingEntry) // For logging

	// Apply modifiers from Traits
	if actingEntry.HasComponent(ActingWithAimTraitTagComponent) {
		// AIM trait specific for SHOOT category parts, but tag is added if part has AIM.
		// The check for SHOOT category for AIM's crit bonus is usually in DamageCalculator.
		// Here, we just apply the bonus if the AIM tag is present.
		// The config holds the actual bonus value.
		modifiers.CriticalRateBonus += gameConfig.Balance.Effects.Aim.CriticalRateBonus
		log.Printf("%s: AIM特性によりクリティカル率ボーナス+%d適用", settings.Name, gameConfig.Balance.Effects.Aim.CriticalRateBonus)
	}

	if actingEntry.HasComponent(ActingWithBerserkTraitTagComponent) {
		// BERSERK trait: adds propulsion to power.
		// This is an additive power bonus.
		if partInfoProvider != nil {
			propulsion := partInfoProvider.GetOverallPropulsion(actingEntry)
			powerBonusFromPropulsion := float64(propulsion) * gameConfig.Balance.Factors.BerserkPowerPropulsionFactor
			modifiers.PowerAdditiveBonus += int(powerBonusFromPropulsion) // Assuming BerserkPowerPropulsionFactor results in an int-scale bonus
			log.Printf("%s: BERSERK特性により推進力(%d)から威力ボーナス+%d適用", settings.Name, propulsion, int(powerBonusFromPropulsion))
		}
	}

	// Apply modifiers from MedalComponent (example: skill level affecting power or crit)
	if medalComp := MedalComponent.Get(actingEntry); medalComp != nil {
		// Example: Medal skill level adds to power (already in DamageCalculator,
		// but could be moved here if we want all modifiers centralized)
		// For now, let's assume medal skill factor is applied later in DamageCalculator or
		// it's an additive bonus here:
		// modifiers.PowerAdditiveBonus += medalComp.SkillLevel * gameConfig.Balance.Damage.MedalSkillFactor
		// log.Printf("%s: メダルスキルにより威力ボーナス+%d適用", settings.Name, medalComp.SkillLevel*gameConfig.Balance.Damage.MedalSkillFactor)

		// Example: Medal skill level adds to critical rate (already in DamageCalculator)
		// modifiers.CriticalRateBonus += medalComp.SkillLevel * 2 // Example factor
	}


	// Add or update the ActionModifierComponent on the entity
	if actingEntry.HasComponent(ActionModifierComponent) {
		ActionModifierComponent.SetValue(actingEntry, modifiers)
	} else {
		donburi.Add(actingEntry, ActionModifierComponent, &modifiers)
	}
	log.Printf("%s: ActionModifierComponent更新完了: %+v", settings.Name, modifiers)
}

// RemoveActionModifiersSystem removes the temporary ActionModifierComponent from an entity.
// This should be called after hit/damage calculations are complete (e.g., in StartCooldownSystem or end of executeActionLogic).
func RemoveActionModifiersSystem(actingEntry *donburi.Entry) {
	if actingEntry == nil || !actingEntry.Valid() {
		return
	}
	if actingEntry.HasComponent(ActionModifierComponent) {
		actingEntry.RemoveComponent(ActionModifierComponent)
		// Correctly get SettingsComponent for logging
		if actingEntry.HasComponent(SettingsComponent) {
			settingsComp := SettingsComponent.Get(actingEntry)
			log.Printf("%s: ActionModifierComponent解除", settingsComp.Name)
		} else {
			// Fallback log if SettingsComponent is somehow not present (should not happen for medarots)
			log.Println("ActionModifierComponent解除 (対象エンティティ名不明)")
		}
	}
}
