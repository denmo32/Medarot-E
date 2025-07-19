package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// processPlayerActionSelected handles the PlayerActionSelectedEvent, initiating game logic.
func ProcessPlayerActionSelected(
	world donburi.World,
	battleLogic *BattleLogic,
	playerActionPendingQueue []*donburi.Entry,
	ui UIInterface,
	event PlayerActionSelectedEvent,
) (newPlayerActionPendingQueue []*donburi.Entry, message string, postMessageCallback func()) {
	log.Printf("BattleActionProcessor: PlayerActionSelectedEvent を処理中 - Actor: %s, Part: %s",
		SettingsComponent.Get(event.ActingEntry).Name,
		event.SelectedPartDef.PartName)

	var successful bool
	actionTargetMap := ui.GetActionTargetMap() // UIからマップを取得

	switch event.SelectedPartDef.Category {
	case CategoryRanged:
		actionTarget, ok := actionTargetMap[event.SelectedSlotKey]
		if !ok || actionTarget.Target == nil || actionTarget.Slot == "" {
			message = "ターゲットがいません！"
			postMessageCallback = func() {
				ui.ClearCurrentTarget()
			}
			return playerActionPendingQueue, message, postMessageCallback
		}
		successful = StartCharge(event.ActingEntry, event.SelectedSlotKey, actionTarget.Target, actionTarget.Slot, world, battleLogic.PartInfoProvider)
	case CategoryMelee:
		successful = StartCharge(event.ActingEntry, event.SelectedSlotKey, nil, "", world, battleLogic.PartInfoProvider)
	case CategoryIntervention:
		successful = StartCharge(event.ActingEntry, event.SelectedSlotKey, nil, "", world, battleLogic.PartInfoProvider)
	default:
		log.Printf("未対応のパーツカテゴリです: %s", event.SelectedPartDef.Category)
		successful = false
	}

	if successful {
		ui.HideActionModal()
		if len(playerActionPendingQueue) > 0 && playerActionPendingQueue[0] == event.ActingEntry {
			playerActionPendingQueue = playerActionPendingQueue[1:]
		}
	} else {
		log.Printf("エラー: %s の行動選択に失敗しました。", SettingsComponent.Get(event.ActingEntry).Name)
		if len(playerActionPendingQueue) > 0 && playerActionPendingQueue[0] == event.ActingEntry {
			playerActionPendingQueue = playerActionPendingQueue[1:]
		}
	}
	return playerActionPendingQueue, "", nil
}

// ProcessPlayerActionCancel handles the PlayerActionCancelEvent, clearing the pending queue.
func ProcessPlayerActionCancel(
	playerActionPendingQueue []*donburi.Entry,
	ui UIInterface,
	event PlayerActionCancelEvent,
) []*donburi.Entry {
	log.Printf("BattleActionProcessor: PlayerActionCancelEvent を処理中 - Actor: %s",
		SettingsComponent.Get(event.ActingEntry).Name)

	newPlayerActionPendingQueue := make([]*donburi.Entry, 0)
	ui.ClearCurrentTarget()
	return newPlayerActionPendingQueue
}

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
func buildActionLogMessages(result ActionResult) []string {
	messages := []string{}

	// メッセージテンプレートのパラメータを準備
	// action_name には特性名(Trait)を渡す
	initiateParams := map[string]interface{}{
		"attacker_name": result.AttackerName,
		"action_name":   result.ActionTrait,
		"weapon_type":   result.WeaponType,
	}

	// カテゴリに応じてメッセージIDを切り替え
	messageID := ""
	switch result.ActionCategory {
	case CategoryRanged, CategoryMelee:
		messageID = "action_initiate_attack"
	case CategoryIntervention:
		messageID = "action_initiate_intervention"
	}

	if result.ActionDidHit {
		if messageID != "" {
			messages = append(messages, GlobalGameDataManager.Messages.FormatMessage(messageID, initiateParams))
		}

		// ダメージや防御のメッセージを追加
		switch result.ActionCategory {
		case CategoryRanged, CategoryMelee:
			if result.ActionIsDefended {
				defendParams := map[string]interface{}{"defender_name": result.DefenderName, "defending_part_type": result.DefendingPartType}
				messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_defend", defendParams))
			}
			damageParams := map[string]interface{}{"defender_name": result.DefenderName, "target_part_type": result.TargetPartType, "damage": result.DamageDealt}
			messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_damage", damageParams))
		case CategoryIntervention:
			// 介入アクションの成功メッセージ（例：「味方チーム全体の命中率が上昇した！」）
			// 必要であれば、ここで特性(Trait)に応じたメッセージを追加する
			if result.ActionTrait == TraitSupport { // string() を削除
				messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("support_action_generic", nil))
			}
		}
	} else {
		// ミスした場合
		if messageID != "" {
			messages = append(messages, GlobalGameDataManager.Messages.FormatMessage(messageID, initiateParams))
		}
		missParams := map[string]interface{}{
			"target_name": result.DefenderName,
		}
		messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("attack_miss", missParams))
	}
	return messages
}
