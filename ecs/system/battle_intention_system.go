package system

import (
	"medarot-ebiten/core"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ecs/entity"
	"medarot-ebiten/event"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdatePlayerInputSystem はアイドル状態のすべてのプレイヤー制御メダロットを見つけます。
// このシステムは BattleScene に直接依存しません。
// 行動が必要なプレイヤーエンティティのリストを返します。
func UpdatePlayerInputSystem(world donburi.World) []event.GameEvent {
	playerActionQueue := entity.GetPlayerActionQueueComponent(world)
	var gameEvents []event.GameEvent

	// キューをクリアし、現在のアイドル状態のプレイヤーエンティティを再収集
	playerActionQueue.Queue = make([]*donburi.Entry, 0)
	query.NewQuery(filter.Contains(component.PlayerControlComponent)).Each(world, func(entry *donburi.Entry) {
		if component.StateComponent.Get(entry).CurrentState == core.StateIdle {
			playerActionQueue.Queue = append(playerActionQueue.Queue, entry)
		}
	})

	if len(playerActionQueue.Queue) > 0 {
		gameEvents = append(gameEvents, event.PlayerActionRequiredGameEvent{})

	}

	return gameEvents
}

// UpdateAIInputSystem はAI制御のメダロットの行動選択を処理します。
// このシステムは BattleScene に直接依存しません。
// aiSelectAction は BattleScene ではなく、world と必要なヘルパーを引数に取るように変更されることを想定しています。
func UpdateAIInputSystem(
	world donburi.World,
	battleLogic *BattleLogic,
) {
	query.NewQuery(
		filter.Not(filter.Contains(component.PlayerControlComponent)), // プレイヤー制御ではないエンティティ
	).Each(world, func(entry *donburi.Entry) {
		if !entry.HasComponent(component.StateComponent) || component.StateComponent.Get(entry).CurrentState != core.StateIdle {
			return
		}
		aiSelectAction(world, entry, battleLogic)
	})
}

// ProcessPlayerIntent はプレイヤーの行動意図を解釈し、具体的なアクションを開始します。
func ProcessPlayerIntent(world donburi.World, battleLogic *BattleLogic, event event.PlayerActionIntentEvent) {
	actingEntry := world.Entry(event.ActingEntityID)
	if actingEntry == nil || !actingEntry.Valid() {
		return
	}

	var targetEntry *donburi.Entry
	if event.TargetEntityID != 0 {
		targetEntry = world.Entry(event.TargetEntityID)
		if targetEntry == nil || !targetEntry.Valid() {
			return // ターゲットが無効なら中断
		}
	}

	successful := battleLogic.GetChargeInitiationSystem().StartCharge(
		actingEntry,
		event.SelectedSlotKey,
		targetEntry,
		event.TargetPartSlot,
	)

	if !successful {
		// エラーログはStartCharge内で行われるため、ここでは不要
	}
}

