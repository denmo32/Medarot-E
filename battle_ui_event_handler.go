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
