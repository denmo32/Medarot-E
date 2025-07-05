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

	// 新しい計算式でダメージを計算
	damage, isCritical := damageCalculator.CalculateDamage(actingEntry, actionResult.TargetEntry, actingPartDef)
	actionResult.IsCritical = isCritical
	actionResult.OriginalDamage = damage

	var finalDamageDealt int
	var actualHitPartInstance *PartInstanceData = intendedTargetPartInstance
	var actualHitPartSlot PartSlotKey = actionResult.TargetPartSlot
	var actualHitPartDef *PartDefinition = intendedTargetPartDef

	actionResult.ActionIsDefended = false

	// 自動防御の判定
	// 脚部パーツのDefense値が0より大きい場合のみ防御判定を行う
	defensePartDef, defFound := partInfoProvider.GetLegsPartDefinition(actionResult.TargetEntry)
	if defFound && defensePartDef.Defense > 0 && hitCalculator.CalculateDefense(actingEntry, actionResult.TargetEntry, actingPartDef) {
		actionResult.ActionIsDefended = true
		// 防御成功時、防御度に応じたダメージ軽減を行う
		defenseRate := partInfoProvider.GetDefenseRate(actionResult.TargetEntry)
		finalDamageDealt = int(float64(actionResult.OriginalDamage) - defenseRate)
		if finalDamageDealt < 1 {
			finalDamageDealt = 1
		}
		damageCalculator.ApplyDamage(actionResult.TargetEntry, actualHitPartInstance, finalDamageDealt)
		actionResult.LogMessage = damageCalculator.GenerateActionLogDefense(actionResult.TargetEntry, defensePartDef, finalDamageDealt, actionResult.OriginalDamage, isCritical)
	} else {
		// 防御失敗または防御パーツがない場合、元のダメージを適用
		damageCalculator.ApplyDamage(actionResult.TargetEntry, actualHitPartInstance, actionResult.OriginalDamage)
		finalDamageDealt = actionResult.OriginalDamage
		actionResult.LogMessage = damageCalculator.GenerateActionLog(actingEntry, actionResult.TargetEntry, actingPartDef, actualHitPartDef, finalDamageDealt, isCritical, true)
	}

	actionResult.DamageDealt = finalDamageDealt
	actionResult.TargetPartBroken = actualHitPartInstance.IsBroken
	actionResult.ActualHitPartSlot = actualHitPartSlot

	return actualHitPartDef
}
