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
			// ターゲットインジケーターを表示
			ui.SetCurrentTarget(e.TargetEntry)
			// アクションモーダルを更新（ターゲット選択ボタンの有効化など）
			// 現状はViewModelを再構築してモーダルを再表示する
			// TODO: ViewModelFactoryを引数で渡すか、UIから取得できるようにする
			// gameEvents = append(gameEvents, ShowActionModalGameEvent{ViewModel: updatedVM})
			log.Printf("UI Event: PartSelectedUIEvent - %s selected part %s", SettingsComponent.Get(e.ActingEntry).Name, e.SelectedPartDef.PartName)
		case TargetSelectedUIEvent:
			ui.SetCurrentTarget(e.TargetEntry)
			log.Printf("UI Event: TargetSelectedUIEvent - %s selected target %s", SettingsComponent.Get(e.ActingEntry).Name, SettingsComponent.Get(e.TargetEntry).Name)
		case ActionConfirmedUIEvent:
			// プレイヤーの行動が確定されたので、チャージ開始イベントを発行
			gameEvents = append(gameEvents, ChargeRequestedGameEvent{
				ActingEntry:     e.ActingEntry,
				SelectedSlotKey: e.SelectedSlotKey,
				TargetEntry:     e.TargetEntry,
				TargetPartSlot:  e.TargetPartSlot,
			})
			// アクションモーダルを非表示にする
			gameEvents = append(gameEvents, HideActionModalGameEvent{})
			// プレイヤーの行動キューから現在のエンティティを削除
			playerActionPendingQueue = playerActionPendingQueue[1:]
			// ターゲットインジケーターをクリア
			gameEvents = append(gameEvents, ClearCurrentTargetGameEvent{})
			nextState = StatePlaying // 行動確定後はPlaying状態に戻る
			log.Printf("UI Event: ActionConfirmedUIEvent - %s confirmed action", SettingsComponent.Get(e.ActingEntry).Name)
		case ActionCanceledUIEvent:
			// アクションモーダルを非表示にする
			gameEvents = append(gameEvents, HideActionModalGameEvent{})
			// ターゲットインジケーターをクリア
			gameEvents = append(gameEvents, ClearCurrentTargetGameEvent{})
			// プレイヤーの行動キューから現在のエンティティを削除
			playerActionPendingQueue = playerActionPendingQueue[1:]
			gameEvents = append(gameEvents, e) // GameEventとしてActionCanceledGameEventを発行
			nextState = StatePlaying           // 行動キャンセル後はPlaying状態に戻る
			log.Printf("UI Event: ActionCanceledUIEvent - %s canceled action", SettingsComponent.Get(e.ActingEntry).Name)
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
