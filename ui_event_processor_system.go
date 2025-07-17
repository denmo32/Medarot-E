package main

import (
	

	"github.com/yohamta/donburi"
)

// UpdateUIEventProcessorSystem はUIイベントを処理し、ゲームの状態を更新します。
func UpdateUIEventProcessorSystem(
	world donburi.World,
	battleLogic *BattleLogic,
	ui UIInterface,
	messageManager *UIMessageDisplayManager,
	uiEventChannel chan UIEvent,
	playerActionPendingQueue []*donburi.Entry,
	currentState GameState,
) (newPlayerActionPendingQueue []*donburi.Entry, newState GameState) {
	newPlayerActionPendingQueue = playerActionPendingQueue
	newState = currentState

	// このメソッドは、PlayerActionSelectState のみが関心を持つイベントを処理します。
	if currentState != StatePlayerActionSelect {
		return
	}

	select {
	case event := <-uiEventChannel:
		switch e := event.(type) {
		case PlayerActionSelectedEvent:
			var message string
			var postMessageCallback func()
			newPlayerActionPendingQueue, message, postMessageCallback = ProcessPlayerActionSelected(
				world, battleLogic, playerActionPendingQueue, ui, e)
			if message != "" {
				messageManager.EnqueueMessage(message, postMessageCallback)
				newState = StateMessage // メッセージ表示状態に遷移
			}
		case PlayerActionCancelEvent:
			newPlayerActionPendingQueue = ProcessPlayerActionCancel(playerActionPendingQueue, ui, e)
			newState = StatePlaying // キャンセル時は即座にPlaying状態に戻る
		case SetCurrentTargetEvent:
			ui.SetCurrentTarget(e.Target)
		case ClearCurrentTargetEvent:
			ui.ClearCurrentTarget()
		}
	default:
	}
	return
}

// buildActionLogMessages はアクションの結果ログを生成します。
// targetPartDef はダメージを受けたパーツの定義 (nilの場合あり)
// actingPartDef は攻撃に使用されたパーツの定義
func buildActionLogMessages(result ActionResult) []string {
	messages := []string{}
	if result.ActionDidHit {
		initiateParams := map[string]interface{}{"attacker_name": result.AttackerName, "action_name": result.ActionName, "weapon_type": result.WeaponType}
		messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_initiate", initiateParams))

		actingPartDef, _ := GlobalGameDataManager.GetPartDefinition(PartsComponent.Get(result.ActingEntry).Map[ActionIntentComponent.Get(result.ActingEntry).SelectedPartKey].DefinitionID)
		switch actingPartDef.Category {
		case CategoryRanged, CategoryMelee:
			if result.ActionIsDefended {
				defendParams := map[string]interface{}{"defender_name": result.DefenderName, "defending_part_type": result.DefendingPartType}
				messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_defend", defendParams))
			}
			damageParams := map[string]interface{}{"defender_name": result.DefenderName, "target_part_type": result.TargetPartType, "damage": result.DamageDealt}
			messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_damage", damageParams))
		case CategoryIntervention:
			messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("support_action_generic", nil))
		}
	} else {
		initiateParams := map[string]interface{}{"attacker_name": result.AttackerName, "action_name": result.ActionName, "weapon_type": result.WeaponType}
		messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_initiate", initiateParams))
		missParams := map[string]interface{}{
			"target_name": result.DefenderName,
		}
		messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("attack_miss", missParams))
	}
	return messages
}
