package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdatePlayerInputSystem はアイドル状態のすべてのプレイヤー制御メダロットを見つけます。
// このシステムは BattleScene に直接依存しません。
// 行動が必要なプレイヤーエンティティのリストを返します。
func UpdatePlayerInputSystem(world donburi.World) []GameEvent {
	playerActionQueue := GetPlayerActionQueueComponent(world)
	var gameEvents []GameEvent

	// キューをクリアし、現在のアイドル状態のプレイヤーエンティティを再収集
	playerActionQueue.Queue = make([]*donburi.Entry, 0)
	query.NewQuery(filter.Contains(PlayerControlComponent)).Each(world, func(entry *donburi.Entry) {
		if StateComponent.Get(entry).CurrentState == StateIdle {
			playerActionQueue.Queue = append(playerActionQueue.Queue, entry)
		}
	})

	if len(playerActionQueue.Queue) > 0 {
		gameEvents = append(gameEvents, PlayerActionRequiredGameEvent{})
		// プレイヤーの行動が必要になったので、フェーズを更新
		battlePhaseEntry, ok := query.NewQuery(filter.Contains(BattlePhaseComponent)).First(world)
		if ok {
			BattlePhaseComponent.Get(battlePhaseEntry).CurrentPhase = PhasePlayerAction
		}
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
		filter.Not(filter.Contains(PlayerControlComponent)), // プレイヤー制御ではないエンティティ
	).Each(world, func(entry *donburi.Entry) {
		if !entry.HasComponent(StateComponent) || StateComponent.Get(entry).CurrentState != StateIdle {
			return
		}
		aiSelectAction(world, entry, battleLogic)
	})
}
