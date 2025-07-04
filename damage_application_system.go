package main

import (
	"fmt"

	"github.com/yohamta/donburi"
)

// ApplyDamageSystem はダメージを計算し、ターゲットに適用します。
func ApplyDamageSystem(
	actingEntry *donburi.Entry,
	world donburi.World,
	actionResult *ActionResult,
	damageCalculator *DamageCalculator,
	hitCalculator *HitCalculator,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
	actingPartDef *PartDefinition,
	intendedTargetPartInstance *PartInstanceData,
	intendedTargetPartDef *PartDefinition,
) *PartDefinition {
	settings := SettingsComponent.Get(actingEntry)

	if actionResult.TargetEntry == nil || intendedTargetPartInstance == nil {
		actionResult.LogMessage = fmt.Sprintf("%s の攻撃は対象または対象パーツが不明です (内部エラー)。", settings.Name)
		return nil
	}

	damage, isCritical := damageCalculator.CalculateDamage(actingEntry, actingPartDef)
	actionResult.IsCritical = isCritical
	actionResult.OriginalDamage = damage

	var finalDamageDealt int
	var actualHitPartInstance *PartInstanceData = intendedTargetPartInstance
	var actualHitPartSlot PartSlotKey = actionResult.TargetPartSlot
	var actualHitPartDef *PartDefinition = intendedTargetPartDef

	actionResult.ActionIsDefended = false
	defensePartInstance := targetSelector.SelectDefensePart(actionResult.TargetEntry)

	if defensePartInstance != nil && defensePartInstance != intendedTargetPartInstance {
		defensePartDef, defFound := GlobalGameDataManager.GetPartDefinition(defensePartInstance.DefinitionID)
		if defFound && hitCalculator.CalculateDefense(actionResult.TargetEntry, defensePartDef) {
			actionResult.ActionIsDefended = true
			actualHitPartInstance = defensePartInstance
			actualHitPartDef = defensePartDef
			actualHitPartSlot = partInfoProvider.FindPartSlot(actionResult.TargetEntry, actualHitPartInstance)

			finalDamageAfterDefense := actionResult.OriginalDamage - defensePartDef.Defense
			if finalDamageAfterDefense < 0 {
				finalDamageAfterDefense = 0
			}
			finalDamageDealt = finalDamageAfterDefense
			damageCalculator.ApplyDamage(actionResult.TargetEntry, actualHitPartInstance, finalDamageDealt)
			actionResult.LogMessage = damageCalculator.GenerateActionLogDefense(actionResult.TargetEntry, actualHitPartDef, finalDamageDealt, actionResult.OriginalDamage, isCritical)
		}
	}

	if !actionResult.ActionIsDefended {
		actualHitPartInstance = intendedTargetPartInstance
		actualHitPartDef = intendedTargetPartDef
		actualHitPartSlot = actionResult.TargetPartSlot
		damageCalculator.ApplyDamage(actionResult.TargetEntry, actualHitPartInstance, actionResult.OriginalDamage)
		finalDamageDealt = actionResult.OriginalDamage
		actionResult.LogMessage = damageCalculator.GenerateActionLog(actingEntry, actionResult.TargetEntry, actingPartDef, actualHitPartDef, finalDamageDealt, isCritical, true)
	}

	actionResult.DamageDealt = finalDamageDealt
	actionResult.TargetPartBroken = actualHitPartInstance.IsBroken
	actionResult.ActualHitPartSlot = actualHitPartSlot

	return actualHitPartDef
}
