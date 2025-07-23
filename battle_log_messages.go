package main

// buildActionLogMessagesFromActionResult はActionResultから表示用のメッセージスライスを構築します。
func buildActionLogMessagesFromActionResult(result ActionResult, gameDataManager *GameDataManager) []string {
	messages := []string{}

	// 攻撃開始メッセージ
	var actionInitiateMsg string
	if result.ActionCategory == CategoryRanged || result.ActionCategory == CategoryMelee {
		actionInitiateMsg = gameDataManager.Messages.FormatMessage("action_initiate_attack", map[string]interface{}{
			"attacker_name": result.AttackerName,
			"action_name":   result.ActionTrait, // ここを修正
			"weapon_type":   result.WeaponType,
		})
	} else if result.ActionCategory == CategoryIntervention {
		actionInitiateMsg = gameDataManager.Messages.FormatMessage("action_initiate_intervention", map[string]interface{}{
			"attacker_name": result.AttackerName,
			"action_name":   result.ActionTrait, // ここを修正
			"weapon_type":   result.WeaponType,
		})
	} else {
		actionInitiateMsg = gameDataManager.Messages.FormatMessage("action_generic", map[string]interface{}{
			"actor_name":  result.AttackerName,
			"action_name": result.ActionName,
		})
	}
	messages = append(messages, actionInitiateMsg)

	if !result.ActionDidHit {
		messages = append(messages, gameDataManager.Messages.FormatMessage("attack_miss", map[string]interface{}{
			"attacker_name": result.AttackerName,
			"skill_name":    result.ActionName,
			"target_name":   result.DefenderName,
		}))
	} else {
		// 防御メッセージ
		if result.ActionIsDefended {
			messages = append(messages, gameDataManager.Messages.FormatMessage("action_defend", map[string]interface{}{
				"defending_part_type": result.DefendingPartType,
			}))
		}

		// ダメージメッセージ
		if result.DamageDealt > 0 {
			messages = append(messages, gameDataManager.Messages.FormatMessage("action_damage", map[string]interface{}{
				"defender_name":    result.DefenderName,
				"target_part_type": result.TargetPartType,
				"damage":           result.DamageDealt,
			}))
		}

		// パーツ破壊メッセージ
		if result.TargetPartBroken {
			messages = append(messages, gameDataManager.Messages.FormatMessage("part_broken", map[string]interface{}{
				"target_name":      result.DefenderName,
				"target_part_name": result.TargetPartType,
			}))
		}
	}

	return messages
}
