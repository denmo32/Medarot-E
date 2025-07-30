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

	// 以前のフレームでプレイヤーの行動が処理済みで、キューが空になった場合
	// （PlayerActionProcessedGameEventはUIイベントプロセッサから発行されるため、ここでは不要）
	// ただし、もしここでキューが空になったことを検知して状態遷移を促す必要があるなら、
	// ここでPlayerActionProcessedGameEventを発行することも考えられる。
	// 今回の変更では、UIイベントプロセッサがActionConfirmedUIEvent/ActionCanceledUIEventを処理した際に
	// PlayerActionProcessedGameEventを発行するように変更済みなので、ここでは不要。

	// キューをクリアし、現在のアイドル状態のプレイヤーエンティティを再収集
	playerActionQueue.Queue = make([]*donburi.Entry, 0)
	query.NewQuery(filter.Contains(PlayerControlComponent)).Each(world, func(entry *donburi.Entry) {
		if StateComponent.Get(entry).CurrentState == StateIdle {
			playerActionQueue.Queue = append(playerActionQueue.Queue, entry)
		}
	})

	if len(playerActionQueue.Queue) > 0 {
		gameEvents = append(gameEvents, PlayerActionRequiredGameEvent{})
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
	// BattleScene の state や playerMedarotToAct に相当する条件をどのように扱うか検討が必要です。
	// 例えば、ワールドリソースでゲームの状態（PlayerTurn, AITurnなど）を管理します。
	// ここでは、単純にIdle状態の非プレイヤーエンティティを対象とします。

	query.NewQuery(
		filter.Not(filter.Contains(PlayerControlComponent)), // プレイヤー制御ではないエンティティ
	).Each(world, func(entry *donburi.Entry) {
		if !entry.HasComponent(StateComponent) || StateComponent.Get(entry).CurrentState != StateIdle {
			return
		}
		// aiSelectAction のシグネチャが (world donburi.World, actingEntry *donburi.Entry, pip *PartInfoProvider, ts *TargetSelector, conf *Config) のようになっていると仮定します。
		// もし aiSelectAction が BattleScene 全体を必要とする場合、そのリファクタリングも必要です。
		aiSelectAction(world, entry, battleLogic)
	})
}
