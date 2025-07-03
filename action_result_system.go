package main

import (
	"fmt"
	"log"
	"strings"
)

// GenerateActionResultSystem は最終的なアクション結果を生成し、関連コンポーネントを更新します。
func GenerateActionResultSystem(ctx *ActionContext) {
	settings := SettingsComponent.Get(ctx.ActingEntry)

	// 攻撃アクションの場合の処理
	if ctx.ActingPartDef.Category == CategoryShoot || ctx.ActingPartDef.Category == CategoryMelee {
		if ctx.ActionResult.TargetPartBroken {
			if ctx.ActionIsDefended {
				if !strings.Contains(ctx.ActionResult.LogMessage, "しかし、パーツは破壊された！") {
					ctx.ActionResult.LogMessage += " しかし、パーツは破壊された！"
				}
			} else {
				if !strings.Contains(ctx.ActionResult.LogMessage, "パーツを破壊した！") {
					ctx.ActionResult.LogMessage += " パーツを破壊した！"
				}
			}
		}

		// --- 履歴コンポーネントの更新 ---
		if ctx.TargetEntry.HasComponent(TargetHistoryComponent) {
			targetHistory := TargetHistoryComponent.Get(ctx.TargetEntry)
			targetHistory.LastAttacker = ctx.ActingEntry
			log.Printf("履歴更新: %s の LastAttacker を %s に設定", SettingsComponent.Get(ctx.TargetEntry).Name, settings.Name)
		}
		if ctx.ActingEntry.HasComponent(LastActionHistoryComponent) {
			lastActionHistory := LastActionHistoryComponent.Get(ctx.ActingEntry)
			lastActionHistory.LastHitTarget = ctx.TargetEntry
			lastActionHistory.LastHitPartSlot = ctx.ActualHitPartSlot
			log.Printf("履歴更新: %s の LastHit を %s の %s に設定", settings.Name, SettingsComponent.Get(ctx.TargetEntry).Name, ctx.ActualHitPartSlot)
		}
	} else { // 非攻撃アクションの場合
		if ctx.ActionResult.LogMessage == "" {
			ctx.ActionResult.LogMessage = fmt.Sprintf("%s は %s を実行した。", settings.Name, ctx.ActingPartDef.PartName)
		}
	}
}
