package system

import (
	"log"

	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
)

// UpdateHistorySystem は、アクションの結果に基づいてAIの行動履歴を更新します。
// このシステムは、アニメーションが完了し、アクションの結果が完全に確定した後に呼び出されるべきです。
func UpdateHistorySystem(world donburi.World, result *component.ActionResult) {
	if result == nil {
		return
	}

	// --- 攻撃者側の履歴更新 ---
	// 自分が最後に攻撃をヒットさせたターゲットとパーツを記録します。
	if result.ActingEntry != nil && result.ActingEntry.Valid() && result.ActingEntry.HasComponent(component.AIComponent) {
		if result.ActionDidHit {
			ai := component.AIComponent.Get(result.ActingEntry)
			ai.LastActionHistory.LastHitTarget = result.TargetEntry
			ai.LastActionHistory.LastHitPartSlot = result.ActualHitPartSlot
			log.Printf(
				"履歴更新 (Focus): %s が %s の %s にヒットさせたことを記録しました。",
				component.SettingsComponent.Get(result.ActingEntry).Name,
				component.SettingsComponent.Get(result.TargetEntry).Name,
				result.ActualHitPartSlot,
			)
		}
	}

	// --- 防御者側の履歴更新 ---
	// 自分を最後に攻撃してきた相手を記録します。
	if result.TargetEntry != nil && result.TargetEntry.Valid() && result.TargetEntry.HasComponent(component.AIComponent) {
		// 命中したかどうかに関わらず、攻撃してきた相手として記録します。
		// これにより、回避した場合でもカウンターの対象となります。
		ai := component.AIComponent.Get(result.TargetEntry)
		ai.TargetHistory.LastAttacker = result.ActingEntry
		log.Printf(
			"履歴更新 (Counter): %s が %s から攻撃されたことを記録しました。",
			component.SettingsComponent.Get(result.TargetEntry).Name,
			component.SettingsComponent.Get(result.ActingEntry).Name,
		)
	}

	// --- 味方への履歴伝播 (Guard/Assist戦略のため) ---
	// リーダーが攻撃された場合、その情報をチームメイトに伝播させることも考えられますが、
	// 現在のGuard戦略はリーダー自身の履歴を見るため、ここでは不要です。
	// Assist戦略は味方のLastHitTargetを見るため、個々のAIComponentが更新されていれば機能します。
}
