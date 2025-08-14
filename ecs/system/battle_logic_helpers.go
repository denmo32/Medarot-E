package system

import (
	"log"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
)

// validateTarget はターゲットが有効かどうかをチェックします。
func validateTarget(targetEntry *donburi.Entry, targetPartSlot core.PartSlotKey) bool {
	if component.StateComponent.Get(targetEntry).CurrentState == core.StateBroken {
		return false
	}
	targetParts := component.PartsComponent.Get(targetEntry)
	if targetParts.Map[targetPartSlot] == nil || targetParts.Map[targetPartSlot].IsBroken {
		return false
	}
	return true
}

// performHitCheck は命中判定を実行します。
func performHitCheck(actingEntry, targetEntry *donburi.Entry, actingPartDef *core.PartDefinition, selectedPartKey core.PartSlotKey, hitCalculator *HitCalculator) bool {
	return hitCalculator.CalculateHit(actingEntry, targetEntry, actingPartDef, selectedPartKey)
}

// applyDamageAndDefense はダメージを適用し、防御処理を行います。
func applyDamageAndDefense(
	result *component.ActionResult,
	actingEntry *donburi.Entry,
	actingPartDef *core.PartDefinition,
	selectedPartKey core.PartSlotKey,
	damageCalculator *DamageCalculator,
	hitCalculator *HitCalculator,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
) {
	defendingPartInst := targetSelector.SelectDefensePart(result.TargetEntry)

	// 防御パーツが存在する場合にのみ防御判定を行う
	if defendingPartInst != nil {
		defendingPartDef, _ := partInfoProvider.GetGameDataManager().GetPartDefinition(defendingPartInst.DefinitionID)
		// 【修正点】CalculateDefenseに防御パーツの定義を渡すように修正。
		// これにより、battle_hit_calculator.goのシグネチャ変更に追従します。
		if hitCalculator.CalculateDefense(actingEntry, result.TargetEntry, actingPartDef, selectedPartKey, defendingPartDef) {
			result.ActionIsDefended = true
			result.DefendingPartType = string(defendingPartDef.Type)
			result.ActualHitPartSlot = partInfoProvider.FindPartSlot(result.TargetEntry, defendingPartInst)

			finalDamage := damageCalculator.CalculateReducedDamage(result.OriginalDamage, result.TargetEntry)
			result.DamageDealt = finalDamage
			result.DamageToApply = finalDamage
			result.TargetPartInstance = defendingPartInst
			return // 防御成功時はここで処理を終了
		}
	}

	// 防御失敗、または防御パーツがなかった場合の処理
	result.ActionIsDefended = false
	intendedTargetPartInstance := component.PartsComponent.Get(result.TargetEntry).Map[result.TargetPartSlot]
	result.DamageDealt = result.OriginalDamage
	result.ActualHitPartSlot = result.TargetPartSlot

	result.DamageToApply = result.OriginalDamage
	result.TargetPartInstance = intendedTargetPartInstance
}

// finalizeActionResult は ActionResult を最終化します。
func finalizeActionResult(result *component.ActionResult, partInfoProvider PartInfoProviderInterface) {
	actualHitPartInst := component.PartsComponent.Get(result.TargetEntry).Map[result.ActualHitPartSlot]
	actualHitPartDef, _ := partInfoProvider.GetGameDataManager().GetPartDefinition(actualHitPartInst.DefinitionID)

	result.TargetPartType = string(actualHitPartDef.Type)
}

// resolveAttackTarget は攻撃アクションのターゲットを解決します。
func resolveAttackTarget(
	actingEntry *donburi.Entry,
	world donburi.World,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	rand *rand.Rand,
) (targetEntry *donburi.Entry, targetPartSlot core.PartSlotKey) {
	targetComp := component.TargetComponent.Get(actingEntry)
	switch targetComp.Policy {
	case core.PolicyPreselected:
		if targetComp.TargetEntity == 0 {
			log.Printf("エラー: PolicyPreselected なのにターゲットが設定されていません。")
			return nil, ""
		}
		targetEntry := world.Entry(targetComp.TargetEntity)
		if targetEntry == nil {
			log.Printf("エラー: ターゲットエンティティID %d がワールドに見つかりません。", targetComp.TargetEntity)
			return nil, ""
		}
		return targetEntry, targetComp.TargetPartSlot
	case core.PolicyClosestAtExecution:
		closestEnemy := targetSelector.FindClosestEnemy(actingEntry, partInfoProvider)
		if closestEnemy == nil {
			return nil, ""
		}
		targetPart := targetSelector.SelectPartToDamage(closestEnemy, actingEntry, rand)
		if targetPart == nil {
			return nil, ""
		}
		slot := partInfoProvider.FindPartSlot(closestEnemy, targetPart)
		if slot == "" {
			return nil, ""
		}
		return closestEnemy, slot
	default:
		log.Printf("未対応のTargetingPolicyです: %s", targetComp.Policy)
		return nil, ""
	}
}

// initializeAttackResult は ActionResult を初期化します。
func initializeAttackResult(actingEntry *donburi.Entry, actingPartDef *core.PartDefinition) component.ActionResult {
	return component.ActionResult{
		ActingEntry:    actingEntry,
		AttackerName:   component.SettingsComponent.Get(actingEntry).Name,
		ActionName:     actingPartDef.PartName,
		ActionTrait:    actingPartDef.Trait,
		ActionCategory: actingPartDef.Category,
		WeaponType:     actingPartDef.WeaponType,
	}
}