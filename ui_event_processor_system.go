package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// UIEventProcessorSystem はUIイベントを処理し、対応するGameEventを発行します。
// このシステムは直接ゲームロジックを呼び出しません。
func UpdateUIEventProcessorSystem(
	world donburi.World,
	battleLogic *BattleLogic,
	ui UIInterface,
	messageManager *UIMessageDisplayManager,
	uiEventChannel chan UIEvent,
	playerActionPendingQueue []*donburi.Entry,
	currentState GameState,
) (newPlayerActionPendingQueue []*donburi.Entry, newState GameState, gameEvents []GameEvent) {
	newPlayerActionPendingQueue = playerActionPendingQueue
	newState = currentState
	gameEvents = []GameEvent{}

	select {
	case event := <-uiEventChannel:
		switch e := event.(type) {
		case PartSelectedUIEvent:
			log.Printf("UIEventProcessor: PartSelectedUIEvent を処理中 - Actor: %s, Part: %s",
				SettingsComponent.Get(e.ActingEntry).Name,
				e.SelectedPartDef.PartName)

			actionTargetMap := ui.GetActionTargetMap() // UIからマップを取得

			var targetEntry *donburi.Entry
			var targetPartSlot PartSlotKey

			switch e.SelectedPartDef.Category {
			case CategoryRanged:
				actionTarget, ok := actionTargetMap[e.SelectedSlotKey]
				if !ok || actionTarget.Target == nil || actionTarget.Slot == "" {
					log.Printf("ターゲットがいません！")
					ui.ClearCurrentTarget()
					// ターゲットがいない場合は行動選択をキャンセルし、キューから削除
					if len(newPlayerActionPendingQueue) > 0 && newPlayerActionPendingQueue[0] == e.ActingEntry {
						newPlayerActionPendingQueue = newPlayerActionPendingQueue[1:]
					}
					return newPlayerActionPendingQueue, newState, gameEvents
				}
				targetEntry = actionTarget.Target
				targetPartSlot = actionTarget.Slot
			case CategoryMelee, CategoryIntervention:
				// 格闘や介入はターゲット選択が不要な場合があるため、nil, "" を渡す
				targetEntry = nil
				targetPartSlot = ""
			default:
				log.Printf("未対応のパーツカテゴリです: %s", e.SelectedPartDef.Category)
				// 未対応のカテゴリの場合もキューから削除
				if len(newPlayerActionPendingQueue) > 0 && newPlayerActionPendingQueue[0] == e.ActingEntry {
					newPlayerActionPendingQueue = newPlayerActionPendingQueue[1:]
				}
				return newPlayerActionPendingQueue, newState, gameEvents
			}

			// ChargeRequestedGameEvent を発行
			gameEvents = append(gameEvents, ChargeRequestedGameEvent{
				ActingEntry:     e.ActingEntry,
				SelectedSlotKey: e.SelectedSlotKey,
				TargetEntry:     targetEntry,
				TargetPartSlot:  targetPartSlot,
			})

			ui.HideActionModal()
			if len(newPlayerActionPendingQueue) > 0 && newPlayerActionPendingQueue[0] == e.ActingEntry {
				newPlayerActionPendingQueue = newPlayerActionPendingQueue[1:]
			}

		case TargetSelectedUIEvent:
			// TargetSelectedUIEvent は UI がターゲットを設定する際に使用される
			ui.SetCurrentTarget(e.TargetEntry)

		case ActionConfirmedUIEvent:
			// UIEventProcessorSystem は直接ゲームロジックを呼び出さず、GameEventを発行する
			gameEvents = append(gameEvents, ChargeRequestedGameEvent{
				ActingEntry:     e.ActingEntry,
				SelectedSlotKey: e.SelectedSlotKey,
				TargetEntry:     e.TargetEntry,
				TargetPartSlot:  e.TargetPartSlot,
			})

		case ActionCanceledUIEvent:
			log.Printf("UIEventProcessor: ActionCanceledUIEvent を処理中 - Actor: %s",
				SettingsComponent.Get(e.ActingEntry).Name)
			newPlayerActionPendingQueue = make([]*donburi.Entry, 0)
			ui.ClearCurrentTarget()
			gameEvents = append(gameEvents, ActionCanceledGameEvent{ActingEntry: e.ActingEntry})
			newState = StatePlaying // キャンセル時は即座にPlaying状態に戻る

		case ShowActionModalUIEvent:
			ui.ShowActionModal(e.ViewModel)
		case HideActionModalUIEvent:
			ui.HideActionModal()
		case SetAnimationUIEvent:
			ui.SetAnimation(&e.AnimationData)
		case ClearAnimationUIEvent:
			ui.ClearAnimation()
		case ClearCurrentTargetUIEvent:
			ui.ClearCurrentTarget()
		case MessageDisplayRequestUIEvent:
			messageManager.EnqueueMessageQueue(e.Messages, e.Callback)
		}
	default:
	}
	return
}

// buildActionLogMessages はアクションの結果ログを生成します。
func buildActionLogMessages(result ActionResult, gameDataManager *GameDataManager) []string {
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
			messages = append(messages, gameDataManager.Messages.FormatMessage(messageID, initiateParams))
		}

		// ダメージや防御のメッセージを追加
		switch result.ActionCategory {
		case CategoryRanged, CategoryMelee:
			if result.ActionIsDefended {
				defendParams := map[string]interface{}{"defender_name": result.DefenderName, "defending_part_type": result.DefendingPartType}
				messages = append(messages, gameDataManager.Messages.FormatMessage("action_defend", defendParams))
			}
			damageParams := map[string]interface{}{"defender_name": result.DefenderName, "target_part_type": result.TargetPartType, "damage": result.DamageDealt}
			messages = append(messages, gameDataManager.Messages.FormatMessage("action_damage", damageParams))
		case CategoryIntervention:
			// 介入アクションの成功メッセージ（例：「味方チーム全体の命中率が上昇した！」）
			// 必要であれば、ここで特性(Trait)に応じたメッセージを追加する
			if result.ActionTrait == TraitSupport { // string() を削除
				messages = append(messages, gameDataManager.Messages.FormatMessage("support_action_generic", nil))
			}
		}
	} else {
		// ミスした場合
		if messageID != "" {
			messages = append(messages, gameDataManager.Messages.FormatMessage(messageID, initiateParams))
		}
		missParams := map[string]interface{}{
			"target_name": result.DefenderName,
		}
		messages = append(messages, gameDataManager.Messages.FormatMessage("attack_miss", missParams))
	}
	return messages
}
