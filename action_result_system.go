package main

import (
	// "fmt"
	"log"
	"strings"

	"github.com/yohamta/donburi"
)

// GenerateActionResultSystem は最終的なアクション結果を生成し、関連コンポーネントを更新します。
func GenerateActionResultSystem(
	actingEntry *donburi.Entry,
	world donburi.World,
	actionResult *ActionResult,
	actingPartDef *PartDefinition,
	actualHitPartDef *PartDefinition,
) {
	settings := SettingsComponent.Get(actingEntry)
	targetSettings := SettingsComponent.Get(actionResult.TargetEntry) // TargetEntryがnilでないことを前提とする

	// 攻撃アクションの場合の処理
	if actingPartDef.Category == CategoryShoot || actingPartDef.Category == CategoryMelee {
		if actionResult.TargetPartBroken {
			targetPartName := "(不明部位)"
			if actualHitPartDef != nil { // ActualHitPartDef は ApplyDamageSystem で設定される
				targetPartName = actualHitPartDef.PartName
			}

			partBrokenParams := map[string]interface{}{
				"target_name":      targetSettings.Name,
				"target_part_name": targetPartName,
			}
			if actionResult.ActionIsDefended {
				// LogMessage が既に防御成功メッセージで埋まっていると仮定
				additionalMsg := GlobalGameDataManager.Messages.FormatMessage("part_broken_on_defense", partBrokenParams)
				if !strings.Contains(actionResult.LogMessage, additionalMsg) { // 重複追加を避ける
					actionResult.LogMessage += " " + additionalMsg
				}
			} else {
				// LogMessage が既に攻撃ヒットメッセージで埋まっていると仮定
				additionalMsg := GlobalGameDataManager.Messages.FormatMessage("part_broken", partBrokenParams)
				if !strings.Contains(actionResult.LogMessage, additionalMsg) { // 重複追加を避ける
					actionResult.LogMessage += " " + additionalMsg
				}
			}
		}

		// --- 履歴コンポーネントの更新 ---
		if actionResult.TargetEntry != nil && actionResult.TargetEntry.HasComponent(AIComponent) {
			ai := AIComponent.Get(actionResult.TargetEntry)
			ai.TargetHistory.LastAttacker = actingEntry
			log.Printf("履歴更新: %s の LastAttacker を %s に設定", SettingsComponent.Get(actionResult.TargetEntry).Name, settings.Name)
		}
		if actingEntry.HasComponent(AIComponent) {
			ai := AIComponent.Get(actingEntry)
			ai.LastActionHistory.LastHitTarget = actionResult.TargetEntry
			ai.LastActionHistory.LastHitPartSlot = actionResult.ActualHitPartSlot
			log.Printf("履歴更新: %s の LastHit を %s の %s に設定", settings.Name, SettingsComponent.Get(actionResult.TargetEntry).Name, actionResult.ActualHitPartSlot)
		}
	} else { // 非攻撃アクションの場合 (例: サポート、防御など)
		if actionResult.LogMessage == "" { // 他のシステムでメッセージが設定されていない場合
			actionResult.LogMessage = GlobalGameDataManager.Messages.FormatMessage("action_generic", map[string]interface{}{
				"actor_name":  settings.Name,
				"action_name": actingPartDef.PartName,
			})
		}
	}
}
