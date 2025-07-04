package main

import "github.com/yohamta/donburi"

// DetermineHitSystem は攻撃の命中判定を行います。
// 攻撃が命中した場合は true を、外れた場合は false を返します。
func DetermineHitSystem(
	actingEntry *donburi.Entry,
	world donburi.World,
	actionResult *ActionResult,
	hitCalculator *HitCalculator,
	damageCalculator *DamageCalculator,
	actingPartDef *PartDefinition,
) bool {
	var didHit = true
	if actionResult.TargetEntry != nil && (actingPartDef.Category == CategoryShoot || actingPartDef.Category == CategoryMelee) {
		didHit = hitCalculator.CalculateHit(actingEntry, actionResult.TargetEntry, actingPartDef)
	}
	actionResult.ActionDidHit = didHit

	if !didHit {
		// 攻撃が外れた場合、actingPartDef を渡して skill_name を正しく表示できるようにする
		actionResult.LogMessage = damageCalculator.GenerateActionLog(actingEntry, actionResult.TargetEntry, actingPartDef, nil, 0, false, false)
		if actingEntry.HasComponent(ActingWithBerserkTraitTagComponent) {
			ResetAllEffects(world)
		}
	}

	return didHit
}
