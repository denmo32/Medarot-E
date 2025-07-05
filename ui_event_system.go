package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// HandleUIEvents はUIイベントのキューを処理し、適切なゲームロジックをトリガーします。
func HandleUIEvents(bs *BattleScene) {
	// BattleSceneが持つイベントキューを処理します（このキューは次のステップで追加します）。
	// この例では、イベントが直接渡されると仮定します。
	// 実際のコードでは、bs.uiEvents のようなスライスをループ処理することになります。

	// この関数は、BattleSceneのUpdateループから呼び出されます。
	// bs.ui.PollEvents() のようなメソッドを呼び出してイベントを取得し、
	// この関数で処理する、という流れを想定しています。
}

// processPlayerActionSelected は、プレイヤーのパーツ選択イベントを処理します。
func processPlayerActionSelected(bs *BattleScene, event PlayerActionSelectedEvent) {
	log.Printf("UIEventSystem: PlayerActionSelectedEvent を処理中 - Actor: %s, Part: %s",
		SettingsComponent.Get(event.ActingEntry).Name,
		event.SelectedPartDef.PartName)

	var successful bool
	switch event.SelectedPartDef.Category {
	case CategoryShoot:
		log.Printf("processPlayerActionSelected: event.SelectedSlotKey = %s", event.SelectedSlotKey)
		log.Printf("processPlayerActionSelected: bs.ui.actionTargetMap = %+v", bs.ui.actionTargetMap)
		actionTarget, ok := bs.ui.actionTargetMap[event.SelectedSlotKey]
		if !ok || actionTarget.Target == nil || actionTarget.Slot == "" {
			bs.enqueueMessage("ターゲットがいません！", func() {
				bs.ui.playerMedarotToAct = nil
				bs.ui.currentTarget = nil
				bs.state = StatePlaying
			})
			return
		}
		successful = StartCharge(event.ActingEntry, event.SelectedSlotKey, actionTarget.Target, actionTarget.Slot, bs.world, &bs.resources.Config, bs.battleLogic.PartInfoProvider)
	case CategoryMelee:
		// 格闘の場合はターゲットがnilでも問題ない
		successful = StartCharge(event.ActingEntry, event.SelectedSlotKey, nil, "", bs.world, &bs.resources.Config, bs.battleLogic.PartInfoProvider)
	default:
		log.Printf("未対応のパーツカテゴリです: %s", event.SelectedPartDef.Category)
		successful = false
	}

	if successful {
		bs.ui.HideActionModal() // アクション成功時にモーダルを非表示にする

		// 現在のメダロットをデキューします
		if len(bs.playerActionPendingQueue) > 0 && bs.playerActionPendingQueue[0] == event.ActingEntry {
			bs.playerActionPendingQueue = bs.playerActionPendingQueue[1:]
		}

		if len(bs.playerActionPendingQueue) > 0 {
			// bs.ui.playerMedarotToAct は次のモーダル表示時に設定される
			bs.state = StatePlayerActionSelect
		} else {
			bs.ui.playerMedarotToAct = nil
			bs.state = StatePlaying
		}
	} else {
		log.Printf("エラー: %s の行動選択に失敗しました。", SettingsComponent.Get(event.ActingEntry).Name)
		// 失敗した場合もキューを進めます
		if len(bs.playerActionPendingQueue) > 0 && bs.playerActionPendingQueue[0] == event.ActingEntry {
			bs.playerActionPendingQueue = bs.playerActionPendingQueue[1:]
		}
		if len(bs.playerActionPendingQueue) > 0 {
			// bs.ui.playerMedarotToAct は次のモーダル表示時に設定される
			bs.state = StatePlayerActionSelect
		} else {
			bs.ui.playerMedarotToAct = nil
			bs.state = StatePlaying
		}
	}
}

// processPlayerActionCancel は、プレイヤーのキャンセルイベントを処理します。
func processPlayerActionCancel(bs *BattleScene, event PlayerActionCancelEvent) {
	log.Printf("UIEventSystem: PlayerActionCancelEvent を処理中 - Actor: %s",
		SettingsComponent.Get(event.ActingEntry).Name)

	bs.playerActionPendingQueue = make([]*donburi.Entry, 0) // 保留キューをクリア
	bs.ui.playerMedarotToAct = nil
	bs.ui.currentTarget = nil
	bs.state = StatePlaying
}