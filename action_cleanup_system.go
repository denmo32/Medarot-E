package main

import "log"

// CleanupActionSystem はアクション実行後のクリーンアップ処理を行います。
func CleanupActionSystem(ctx *ActionContext) {
	settings := SettingsComponent.Get(ctx.ActingEntry)

	if ctx.ActingEntry.HasComponent(ActingWithBerserkTraitTagComponent) {
		log.Printf("%s がBERSERK特性効果（行動後全効果リセット）を発動。", settings.Name)
		ResetAllEffects(ctx.World)
	}

	if ctx.ActingEntry.HasComponent(ActingWithBerserkTraitTagComponent) {
		ctx.ActingEntry.RemoveComponent(ActingWithBerserkTraitTagComponent)
	}
	if ctx.ActingEntry.HasComponent(ActingWithAimTraitTagComponent) {
		ctx.ActingEntry.RemoveComponent(ActingWithAimTraitTagComponent)
	}
	RemoveActionModifiersSystem(ctx.ActingEntry)
}
