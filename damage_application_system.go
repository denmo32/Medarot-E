package main

import "fmt"

// ApplyDamageSystem はダメージを計算し、ターゲットに適用します。
func ApplyDamageSystem(ctx *ActionContext) {
	settings := SettingsComponent.Get(ctx.ActingEntry)

	if ctx.TargetEntry == nil || ctx.IntendedTargetPartInstance == nil {
		ctx.ActionResult.LogMessage = fmt.Sprintf("%s の攻撃は対象または対象パーツが不明です (内部エラー)。", settings.Name)
		return
	}

	damage, isCritical := ctx.DamageCalculator.CalculateDamage(ctx.ActingEntry, ctx.ActingPartDef)
	ctx.IsCritical = isCritical
	ctx.OriginalDamage = damage
	ctx.ActionResult.IsCritical = isCritical

	var finalDamageDealt int
	var actualHitPartInstance *PartInstanceData = ctx.IntendedTargetPartInstance
	var actualHitPartSlot PartSlotKey = ctx.TargetPartSlot
	var actualHitPartDef *PartDefinition = ctx.IntendedTargetPartDef

	ctx.ActionIsDefended = false
	defensePartInstance := ctx.TargetSelector.SelectDefensePart(ctx.TargetEntry)

	if defensePartInstance != nil && defensePartInstance != ctx.IntendedTargetPartInstance {
		defensePartDef, defFound := GlobalGameDataManager.GetPartDefinition(defensePartInstance.DefinitionID)
		if defFound && ctx.HitCalculator.CalculateDefense(ctx.TargetEntry, defensePartDef) {
			ctx.ActionIsDefended = true
			actualHitPartInstance = defensePartInstance
			actualHitPartDef = defensePartDef
			actualHitPartSlot = ctx.PartInfoProvider.FindPartSlot(ctx.TargetEntry, actualHitPartInstance)

			finalDamageAfterDefense := ctx.OriginalDamage - defensePartDef.Defense
			if finalDamageAfterDefense < 0 {
				finalDamageAfterDefense = 0
			}
			finalDamageDealt = finalDamageAfterDefense
			ctx.DamageCalculator.ApplyDamage(ctx.TargetEntry, actualHitPartInstance, finalDamageDealt)
			ctx.ActionResult.LogMessage = ctx.DamageCalculator.GenerateActionLogDefense(ctx.TargetEntry, actualHitPartDef, finalDamageDealt, ctx.OriginalDamage, isCritical)
		}
	}

	if !ctx.ActionIsDefended {
		actualHitPartInstance = ctx.IntendedTargetPartInstance
		actualHitPartDef = ctx.IntendedTargetPartDef
		actualHitPartSlot = ctx.TargetPartSlot
		ctx.DamageCalculator.ApplyDamage(ctx.TargetEntry, actualHitPartInstance, ctx.OriginalDamage)
		finalDamageDealt = ctx.OriginalDamage
		ctx.ActionResult.LogMessage = ctx.DamageCalculator.GenerateActionLog(ctx.ActingEntry, ctx.TargetEntry, actualHitPartDef, finalDamageDealt, isCritical, true)
	}

	ctx.FinalDamageDealt = finalDamageDealt
	ctx.ActualHitPartInstance = actualHitPartInstance
	ctx.ActualHitPartSlot = actualHitPartSlot
	ctx.ActualHitPartDef = actualHitPartDef
	ctx.ActionResult.DamageDealt = finalDamageDealt
	ctx.ActionResult.ActionIsDefended = ctx.ActionIsDefended
	ctx.ActionResult.TargetPartBroken = actualHitPartInstance.IsBroken
}
