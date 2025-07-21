package ui

import (
	"log"
	"medarot-ebiten/internal/game"

	"github.com/yohamta/donburi"
)

// UIEventProcessorSystem はUIイベントを処理し、対応するGameEventを発行します。
// このシステムは直接ゲームロジックを呼び出しません。
func UpdateUIEventProcessorSystem(
	world donburi.World,
	battleLogic interface{},
	ui UIInterface,
	messageManager *UIMessageDisplayManager,
	uiEventChannel chan game.UIEvent,
	playerActionPendingQueue []*donburi.Entry,
	currentState game.GameState,
) (newPlayerActionPendingQueue []*donburi.Entry, newState game.GameState, gameEvents []game.GameEvent) {
	newPlayerActionPendingQueue = playerActionPendingQueue
	newState = currentState
	gameEvents = []game.GameEvent{}

	select {
	case event := <-uiEventChannel:
		switch e := event.(type) {
		case game.PartSelectedUIEvent:
			log.Printf("UIEventProcessor: PartSelectedUIEvent を処理中 - Actor: %s, Part: %s",
				game.SettingsComponent.Get(e.ActingEntry).Name,
				e.SelectedPartDef.PartName)

			actionTargetMap := ui.GetActionTargetMap() // UIからマップを取得

			var targetEntry *donburi.Entry
			var targetPartSlot game.PartSlotKey

			switch e.SelectedPartDef.Category {
			case game.CategoryRanged:
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
			case game.CategoryMelee, game.CategoryIntervention:
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
			gameEvents = append(gameEvents, game.ChargeRequestedGameEvent{
				ActingEntry:     e.ActingEntry,
				SelectedSlotKey: e.SelectedSlotKey,
				TargetEntry:     targetEntry,
				TargetPartSlot:  targetPartSlot,
			})

			ui.HideActionModal()
			if len(newPlayerActionPendingQueue) > 0 && newPlayerActionPendingQueue[0] == e.ActingEntry {
				newPlayerActionPendingQueue = newPlayerActionPendingQueue[1:]
			}

		case game.TargetSelectedUIEvent:
			// TargetSelectedUIEvent は UI がターゲットを設定する際に使用される
			ui.SetCurrentTarget(e.TargetEntry)

		case game.ActionConfirmedUIEvent:
			// UIEventProcessorSystem は直接ゲームロジックを呼び出さず、GameEventを発行する
			gameEvents = append(gameEvents, game.ChargeRequestedGameEvent{
				ActingEntry:     e.ActingEntry,
				SelectedSlotKey: e.SelectedSlotKey,
				TargetEntry:     e.TargetEntry,
				TargetPartSlot:  e.TargetPartSlot,
			})

		case game.ActionCanceledUIEvent:
			log.Printf("UIEventProcessor: ActionCanceledUIEvent を処理中 - Actor: %s",
				game.SettingsComponent.Get(e.ActingEntry).Name)
			newPlayerActionPendingQueue = make([]*donburi.Entry, 0)
			ui.ClearCurrentTarget()
			gameEvents = append(gameEvents, game.ActionCanceledGameEvent{ActingEntry: e.ActingEntry})
			newState = game.StatePlaying // キャンセル時は即座にPlaying状態に戻る

		case game.ShowActionModalUIEvent:
			ui.ShowActionModal(e.ViewModel)
		case game.HideActionModalUIEvent:
			ui.HideActionModal()
		case game.SetAnimationUIEvent:
			ui.SetAnimation(&e.AnimationData)
		case game.ClearAnimationUIEvent:
			ui.ClearAnimation()
		case game.ClearCurrentTargetUIEvent:
			ui.ClearCurrentTarget()
		case game.MessageDisplayRequestUIEvent:
			messageManager.EnqueueMessageQueue(e.Messages, e.Callback)
		}
	default:
	}
	return
}
