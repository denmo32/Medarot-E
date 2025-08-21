package system

import (
	"log"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/donburi"
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

// applyDamageAndDefense はダメージ計算と防御判定のロジックをカプセル化します。
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
	// 1. 防御に使用するパーツを選択
	defendingPartInst := targetSelector.SelectDefensePart(result.TargetEntry)
	var isDefended bool

	// 2. 防御判定
	if defendingPartInst != nil {
		defendingPartDef, _ := partInfoProvider.GetGameDataManager().GetPartDefinition(defendingPartInst.DefinitionID)
		isDefended = hitCalculator.CalculateDefense(actingEntry, result.TargetEntry, actingPartDef, selectedPartKey, defendingPartDef)
		result.ActionIsDefended = isDefended
		result.DefendingPartType = string(defendingPartDef.Type)
	} else {
		isDefended = false
		result.ActionIsDefended = false
	}

	// 3. ダメージ計算
	// 防御の成否を引数に渡し、計算式を切り替える
	damage, isCritical := damageCalculator.CalculateDamage(actingEntry, result.TargetEntry, actingPartDef, selectedPartKey, isDefended)
	result.IsCritical = isCritical
	result.OriginalDamage = damage // 計算後のダメージをOriginalDamageとして記録（UI表示用）
	result.DamageDealt = damage
	result.DamageToApply = damage

	// 4. 実際にダメージを受けるパーツを決定
	if isDefended {
		// 防御成功時は防御パーツがダメージを受ける
		result.ActualHitPartSlot = partInfoProvider.FindPartSlot(result.TargetEntry, defendingPartInst)
		result.TargetPartInstance = defendingPartInst
	} else {
		// 防御失敗時は元々狙われたパーツがダメージを受ける
		result.ActualHitPartSlot = result.TargetPartSlot
		result.TargetPartInstance = component.PartsComponent.Get(result.TargetEntry).Map[result.TargetPartSlot]
	}
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