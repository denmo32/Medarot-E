package main

import (
	"log"

	"github.com/yohamta/donburi"
)

func processPlayerActionSelected(
	world donburi.World,
	config *Config,
	battleLogic *BattleLogic,
	playerActionPendingQueue []*donburi.Entry,
	ui UIInterface,
	event PlayerActionSelectedEvent,
) (newPlayerActionPendingQueue []*donburi.Entry, newState GameState, newTargetedEntity *donburi.Entry, newAttackingEntity *donburi.Entry, message string, postMessageCallback func()) {
	log.Printf("UIEventSystem: PlayerActionSelectedEvent を処理中 - Actor: %s, Part: %s",
		SettingsComponent.Get(event.ActingEntry).Name,
		event.SelectedPartDef.PartName)

	var successful bool
	switch event.SelectedPartDef.Category {
	case CategoryShoot:
		log.Printf("processPlayerActionSelected: event.SelectedSlotKey = %s", event.SelectedSlotKey)
		log.Printf("processPlayerActionSelected: ui.GetActionTargetMap() = %+v", ui.GetActionTargetMap())
		actionTarget, ok := ui.GetActionTargetMap()[event.SelectedSlotKey]
		if !ok || actionTarget.Target == nil || actionTarget.Slot == "" {
			message = "ターゲットがいません！"
			postMessageCallback = func() {
				ui.ClearCurrentTarget()
			}
			return playerActionPendingQueue, StatePlaying, nil, nil, message, postMessageCallback
		}
		successful = StartCharge(event.ActingEntry, event.SelectedSlotKey, actionTarget.Target, actionTarget.Slot, world, config, battleLogic.PartInfoProvider)
	case CategoryMelee:
		// 格闘の場合はターゲットがnilでも問題ない
		successful = StartCharge(event.ActingEntry, event.SelectedSlotKey, nil, "", world, config, battleLogic.PartInfoProvider)
	default:
		log.Printf("未対応のパーツカテゴリです: %s", event.SelectedPartDef.Category)
		successful = false
	}

	if successful {
		ui.HideActionModal() // アクション成功時にモーダルを非表示にする

		// 現在のメダロットをデキューします
		if len(playerActionPendingQueue) > 0 && playerActionPendingQueue[0] == event.ActingEntry {
			playerActionPendingQueue = playerActionPendingQueue[1:]
		}

		newState = StatePlaying
		if len(playerActionPendingQueue) > 0 {
			newState = StatePlayerActionSelect
		}
		return playerActionPendingQueue, newState, nil, nil, "", nil
	} else {
		log.Printf("エラー: %s の行動選択に失敗しました。", SettingsComponent.Get(event.ActingEntry).Name)
		// 失敗した場合もキューを進めます
		if len(playerActionPendingQueue) > 0 && playerActionPendingQueue[0] == event.ActingEntry {
			playerActionPendingQueue = playerActionPendingQueue[1:]
		}
		newState = StatePlaying
		if len(playerActionPendingQueue) > 0 {
			newState = StatePlayerActionSelect
		}
		return playerActionPendingQueue, newState, nil, nil, "", nil
	}
}

// processPlayerActionCancel は、プレイヤーのキャンセルイベントを処理します。
func processPlayerActionCancel(
	playerActionPendingQueue []*donburi.Entry,
	ui UIInterface,
	event PlayerActionCancelEvent,
) (newPlayerActionPendingQueue []*donburi.Entry, newState GameState) {
	log.Printf("UIEventSystem: PlayerActionCancelEvent を処理中 - Actor: %s",
		SettingsComponent.Get(event.ActingEntry).Name)

	newPlayerActionPendingQueue = make([]*donburi.Entry, 0) // 保留キューをクリア
	ui.ClearCurrentTarget()
	newState = StatePlaying
	return
}