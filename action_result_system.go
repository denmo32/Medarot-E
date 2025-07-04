package main

import (
	"fmt"
	"log"
	"strings"
)

// GenerateActionResultSystem は最終的なアクション結果を生成し、関連コンポーネントを更新します。
func GenerateActionResultSystem(ctx *ActionContext) {
	settings := SettingsComponent.Get(ctx.ActingEntry)
	targetSettings := SettingsComponent.Get(ctx.TargetEntry) // TargetEntryがnilでないことを前提とする

	// 攻撃アクションの場合の処理
	if ctx.ActingPartDef.Category == CategoryShoot || ctx.ActingPartDef.Category == CategoryMelee {
		if ctx.ActionResult.TargetPartBroken {
			targetPartName := "(不明部位)"
			if ctx.ActualHitPartDef != nil { // ActualHitPartDef は ApplyDamageSystem で設定される
				targetPartName = ctx.ActualHitPartDef.PartName
			}

			partBrokenParams := map[string]interface{}{
				"target_name":      targetSettings.Name,
				"target_part_name": targetPartName,
			}
			if ctx.ActionIsDefended {
				// LogMessage が既に防御成功メッセージで埋まっていると仮定
				additionalMsg := GlobalGameDataManager.Messages.FormatMessage("part_broken_on_defense", partBrokenParams)
				if !strings.Contains(ctx.ActionResult.LogMessage, additionalMsg) { // 重複追加を避ける
					ctx.ActionResult.LogMessage += " " + additionalMsg
				}
			} else {
				// LogMessage が既に攻撃ヒットメッセージで埋まっていると仮定
				additionalMsg := GlobalGameDataManager.Messages.FormatMessage("part_broken", partBrokenParams)
				if !strings.Contains(ctx.ActionResult.LogMessage, additionalMsg) { // 重複追加を避ける
					ctx.ActionResult.LogMessage += " " + additionalMsg
				}
			}
		}

		// --- 履歴コンポーネントの更新 ---
		if ctx.TargetEntry != nil && ctx.TargetEntry.HasComponent(AIComponent) {
			ai := AIComponent.Get(ctx.TargetEntry)
			ai.TargetHistory.LastAttacker = ctx.ActingEntry
			log.Printf("履歴更新: %s の LastAttacker を %s に設定", SettingsComponent.Get(ctx.TargetEntry).Name, settings.Name)
		}
		if ctx.ActingEntry.HasComponent(AIComponent) {
			ai := AIComponent.Get(ctx.ActingEntry)
			ai.LastActionHistory.LastHitTarget = ctx.TargetEntry
			ai.LastActionHistory.LastHitPartSlot = ctx.ActualHitPartSlot
			log.Printf("履歴更新: %s の LastHit を %s の %s に設定", settings.Name, SettingsComponent.Get(ctx.TargetEntry).Name, ctx.ActualHitPartSlot)
		}
	} else { // 非攻撃アクションの場合 (例: サポート、防御など)
		if ctx.ActionResult.LogMessage == "" { // 他のシステムでメッセージが設定されていない場合
			ctx.ActionResult.LogMessage = GlobalGameDataManager.Messages.FormatMessage("action_generic", map[string]interface{}{
				"actor_name":  settings.Name,
				"action_name": ctx.ActingPartDef.PartName,
			})
		}
	}
}
