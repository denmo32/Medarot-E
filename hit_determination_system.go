package main

// DetermineHitSystem は攻撃の命中判定を行います。
// 攻撃が命中した場合は true を、外れた場合は false を返します。
func DetermineHitSystem(ctx *ActionContext) bool {
	var didHit = true
	if ctx.TargetEntry != nil && (ctx.ActingPartDef.Category == CategoryShoot || ctx.ActingPartDef.Category == CategoryMelee) {
		didHit = ctx.HitCalculator.CalculateHit(ctx.ActingEntry, ctx.TargetEntry, ctx.ActingPartDef)
	}
	ctx.ActionDidHit = didHit
	ctx.ActionResult.ActionDidHit = didHit

	if !didHit {
		ctx.ActionResult.LogMessage = ctx.DamageCalculator.GenerateActionLog(ctx.ActingEntry, ctx.TargetEntry, nil, 0, false, false)
		if ctx.ActingEntry.HasComponent(ActingWithBerserkTraitTagComponent) {
			ResetAllEffects(ctx.World)
		}
	}

	return didHit
}
