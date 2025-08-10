package ui

import (
	"log"

	"medarot-ebiten/ecs/component"
	"medarot-ebiten/event"

	"github.com/yohamta/donburi"
)

// UpdateUIEventProcessorSystem はUIイベントを処理し、対応するゲームイベントを発行します。
func UpdateUIEventProcessorSystem(
	world donburi.World,
	ui UIInterface,
	messageManager *UIMessageDisplayManager,
	eventChannel chan UIEvent,
) []event.GameEvent {
	var gameEvents []event.GameEvent
	// var nextState = currentState // nextStateの宣言を削除

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
			ui.SetCurrentTarget(e.TargetEntityID) // donburi.Entity を渡す
			log.Printf("UI Event: PartSelectedUIEvent - %s selected part %s", component.SettingsComponent.Get(actingEntry).Name, e.SelectedSlotKey)
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
			ui.SetCurrentTarget(e.TargetEntityID) // donburi.Entity を渡す
			log.Printf("UI Event: TargetSelectedUIEvent - %s selected target %s", component.SettingsComponent.Get(actingEntry).Name, component.SettingsComponent.Get(targetEntry).Name)
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
			gameEvents = append(gameEvents, event.ChargeRequestedGameEvent{
				ActingEntry:     actingEntry,
				SelectedSlotKey: e.SelectedSlotKey,
				TargetEntry:     targetEntry,
				TargetPartSlot:  e.TargetPartSlot,
			})
			// アクションモーダルを非表示にする
			gameEvents = append(gameEvents, event.HideActionModalGameEvent{})
			// ターゲットインジケーターをクリア
			gameEvents = append(gameEvents, event.ClearCurrentTargetGameEvent{})
			// プレイヤーの行動選択フェーズが完了したことを通知
			gameEvents = append(gameEvents, event.PlayerActionProcessedGameEvent{
				ActingEntry: actingEntry,
			})
			log.Printf("UI Event: ActionConfirmedUIEvent - %s confirmed action", component.SettingsComponent.Get(actingEntry).Name)
		case ActionCanceledUIEvent:
			actingEntry := world.Entry(e.ActingEntityID)
			if actingEntry == nil {
				log.Printf("Error: ActionCanceledUIEvent - ActingEntry not found for ID %d", e.ActingEntityID)
				break
			}
			// アクションモーダルを非表示にする
			gameEvents = append(gameEvents, event.HideActionModalGameEvent{})
			// ターゲットインジケーターをクリア
			gameEvents = append(gameEvents, event.ClearCurrentTargetGameEvent{})
			// プレイヤーの行動選択フェーズが完了したことを通知
			gameEvents = append(gameEvents, event.PlayerActionProcessedGameEvent{ActingEntry: actingEntry})
			log.Printf("UI Event: ActionCanceledUIEvent - %s canceled action", component.SettingsComponent.Get(actingEntry).Name)
		case ShowActionModalUIEvent:
			if !ui.IsActionModalVisible() { // モーダルが既に表示されていない場合のみ表示
				ui.ShowActionModal(e.ViewModel)
			}
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
		case AnimationFinishedUIEvent: // 新しいUIイベントの処理
			gameEvents = append(gameEvents, event.ActionAnimationFinishedGameEvent{Result: e.Result, ActingEntry: e.Result.ActingEntry})
		default:
			log.Printf("Unknown UI Event: %T", uiEvent)
		}
	default:
		// イベントがない場合は何もしない
	}

	return gameEvents // nextStateの戻り値を削除
}
