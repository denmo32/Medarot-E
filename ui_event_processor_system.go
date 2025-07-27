package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// UpdateUIEventProcessorSystem はUIイベントを処理し、対応するゲームイベントを発行します。
func UpdateUIEventProcessorSystem(
	world donburi.World,
	ui UIInterface,
	messageManager *UIMessageDisplayManager,
	eventChannel chan UIEvent,
	playerActionPendingQueue []*donburi.Entry,
	currentState GameState,
) ([]*donburi.Entry, GameState, []GameEvent) {
	var gameEvents []GameEvent
	var nextState = currentState

	select {
	case uiEvent := <-eventChannel:
		switch e := uiEvent.(type) {
		case PartSelectedUIEvent:
			actingEntry := world.Entry(e.ActingEntityID)
			if actingEntry == nil {
				log.Printf("Error: PartSelectedUIEvent - ActingEntry not found for ID %d", e.ActingEntityID)
				break
			}
			var targetEntry *donburi.Entry
			if e.TargetEntityID != 0 {
				targetEntry = world.Entry(e.TargetEntityID)
				if targetEntry == nil {
					log.Printf("Error: PartSelectedUIEvent - TargetEntry not found for ID %d", e.TargetEntityID)
					break
				}
			}
			// ターゲットインジケーターを表示
			ui.SetCurrentTarget(targetEntry)
			log.Printf("UI Event: PartSelectedUIEvent - %s selected part %s", SettingsComponent.Get(actingEntry).Name, e.SelectedPartDefID)
		case TargetSelectedUIEvent:
			actingEntry := world.Entry(e.ActingEntityID)
			if actingEntry == nil {
				log.Printf("Error: TargetSelectedUIEvent - ActingEntry not found for ID %d", e.ActingEntityID)
				break
			}
			targetEntry := world.Entry(e.TargetEntityID)
			if targetEntry == nil {
				log.Printf("Error: TargetSelectedUIEvent - TargetEntry not found for ID %d", e.TargetEntityID)
				break
			}
			ui.SetCurrentTarget(targetEntry)
			log.Printf("UI Event: TargetSelectedUIEvent - %s selected target %s", SettingsComponent.Get(actingEntry).Name, SettingsComponent.Get(targetEntry).Name)
		case ActionConfirmedUIEvent:
			actingEntry := world.Entry(e.ActingEntityID)
			if actingEntry == nil {
				log.Printf("Error: ActionConfirmedUIEvent - ActingEntry not found for ID %d", e.ActingEntityID)
				break
			}
			var targetEntry *donburi.Entry
			if e.TargetEntityID != 0 {
				targetEntry = world.Entry(e.TargetEntityID)
				if targetEntry == nil {
					log.Printf("Error: ActionConfirmedUIEvent - TargetEntry not found for ID %d", e.TargetEntityID)
					break
				}
			}
			// プレイヤーの行動が確定されたので、チャージ開始イベントを発行
			gameEvents = append(gameEvents, ChargeRequestedGameEvent{
				ActingEntry:     actingEntry,
				SelectedSlotKey: e.SelectedSlotKey,
				TargetEntry:     targetEntry,
				TargetPartSlot:  e.TargetPartSlot,
			})
			// アクションモーダルを非表示にする
			gameEvents = append(gameEvents, HideActionModalGameEvent{})
			// ターゲットインジケーターをクリア
			gameEvents = append(gameEvents, ClearCurrentTargetGameEvent{})
			gameEvents = append(gameEvents, PlayerActionProcessedGameEvent{ActingEntry: actingEntry})
			nextState = StatePlaying // 行動確定後はPlaying状態に戻る
			log.Printf("UI Event: ActionConfirmedUIEvent - %s confirmed action", SettingsComponent.Get(actingEntry).Name)
		case ActionCanceledUIEvent:
			actingEntry := world.Entry(e.ActingEntityID)
			if actingEntry == nil {
				log.Printf("Error: ActionCanceledUIEvent - ActingEntry not found for ID %d", e.ActingEntityID)
				break
			}
			// アクションモーダルを非表示にする
			gameEvents = append(gameEvents, HideActionModalGameEvent{})
			// ターゲットインジケーターをクリア
			gameEvents = append(gameEvents, ClearCurrentTargetGameEvent{})
			gameEvents = append(gameEvents, PlayerActionProcessedGameEvent{ActingEntry: actingEntry})
			nextState = StatePlaying           // 行動キャンセル後はPlaying状態に戻る
			log.Printf("UI Event: ActionCanceledUIEvent - %s canceled action", SettingsComponent.Get(actingEntry).Name)
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
		default:
			log.Printf("Unknown UI Event: %T", uiEvent)
		}
	default:
		// イベントがない場合は何もしない
	}

	return playerActionPendingQueue, nextState, gameEvents
}
