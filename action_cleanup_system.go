package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// CleanupActionSystem はアクション実行後のクリーンアップ処理を行います。
func CleanupActionSystem(actingEntry *donburi.Entry, world donburi.World) {
	settings := SettingsComponent.Get(actingEntry)

	if actingEntry.HasComponent(ActingWithBerserkTraitTagComponent) {
		log.Printf("%s がBERSERK特性効果（行動後全効果リセット）を発動。", settings.Name)
		ResetAllEffects(world)
	}

	if actingEntry.HasComponent(ActingWithBerserkTraitTagComponent) {
		actingEntry.RemoveComponent(ActingWithBerserkTraitTagComponent)
	}
	if actingEntry.HasComponent(ActingWithAimTraitTagComponent) {
		actingEntry.RemoveComponent(ActingWithAimTraitTagComponent)
	}
	RemoveActionModifiersSystem(actingEntry)
}
