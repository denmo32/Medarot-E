package main

import (
	"log"
	"medarot-ebiten/domain"
)

// BattleLoggerImpl は BattleLogger インターフェースの実装です。
type BattleLoggerImpl struct {
	gameDataManager *GameDataManager
}

// NewBattleLogger は新しい BattleLoggerImpl のインスタンスを生成します。
func NewBattleLogger(gdm *GameDataManager) BattleLogger {
	return &BattleLoggerImpl{gameDataManager: gdm}
}

func (l *BattleLoggerImpl) LogHitCheck(attackerName, targetName string, chance, successRate, evasion float64, roll int) {
	log.Print(l.gameDataManager.Messages.FormatMessage("log_hit_roll", map[string]interface{}{
		"ordered_args": []interface{}{attackerName, targetName, chance, successRate, evasion, roll},
	}))
}

func (l *BattleLoggerImpl) LogDefenseCheck(targetName string, defenseRate, successRate, chance float64, roll int) {
	log.Print(l.gameDataManager.Messages.FormatMessage("log_defense_roll", map[string]interface{}{
		"ordered_args": []interface{}{targetName, defenseRate, successRate, chance, roll},
	}))
}

func (l *BattleLoggerImpl) LogCriticalHit(attackerName string, chance float64) {
	log.Printf("%s の攻撃がクリティカル！ (確率: %.1f%%)", attackerName, chance)
}

func (l *BattleLoggerImpl) LogPartBroken(medarotName, partName, partID string) {
	log.Print(l.gameDataManager.Messages.FormatMessage("log_part_broken_notification", map[string]interface{}{
		"ordered_args": []interface{}{medarotName, partName, partID},
	}))
}

func (l *BattleLoggerImpl) LogActionInitiated(attackerName string, actionTrait domain.Trait, weaponType domain.WeaponType, category domain.PartCategory) {
	// このメッセージは ActionResult から構築されるため、ここでは直接ログ出力しない
}

func (l *BattleLoggerImpl) LogAttackMiss(attackerName, skillName, targetName string) {
	// このメッセージは ActionResult から構築されるため、ここでは直接ログ出力しない
}

func (l *BattleLoggerImpl) LogDamageDealt(defenderName, targetPartType string, damage int) {
	// このメッセージは ActionResult から構築されるため、ここでは直接ログ出力しない
}

func (l *BattleLoggerImpl) LogDefenseSuccess(targetName, defensePartName string, originalDamage, actualDamage int, isCritical bool) {
	// このメッセージは ActionResult から構築されるため、ここでは直接ログ出力しない
}

// buildActionLogMessagesFromActionResult はActionResultから表示用のメッセージスライスを構築します。
func buildActionLogMessagesFromActionResult(result ActionResult, gameDataManager *GameDataManager) []string {
	messages := []string{}

	// 攻撃開始メッセージ
	var actionInitiateMsg string
	switch result.ActionCategory {
	case domain.CategoryRanged, domain.CategoryMelee:
		actionInitiateMsg = gameDataManager.Messages.FormatMessage("action_initiate_attack", map[string]interface{}{
			"attacker_name": result.AttackerName,
			"action_name":   result.ActionTrait, // ここを修正
			"weapon_type":   result.WeaponType,
		})
	case domain.CategoryIntervention:
		actionInitiateMsg = gameDataManager.Messages.FormatMessage("action_initiate_intervention", map[string]interface{}{
			"attacker_name": result.AttackerName,
			"action_name":   result.ActionTrait, // ここを修正
			"weapon_type":   result.WeaponType,
		})
	default:
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
		if result.IsTargetPartBroken {
			messages = append(messages, gameDataManager.Messages.FormatMessage("part_broken", map[string]interface{}{
				"target_name":      result.DefenderName,
				"target_part_name": result.TargetPartType,
			}))
		}
	}

	return messages
}
