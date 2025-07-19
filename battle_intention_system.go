package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// PlayerInputSystemResult はプレイヤーの入力が必要なエンティティのリストを保持します。
type PlayerInputSystemResult struct {
	PlayerMedarotsToAct []*donburi.Entry
}

// UpdatePlayerInputSystem はアイドル状態のすべてのプレイヤー制御メダロットを見つけます。
// このシステムは BattleScene に直接依存しません。
// 行動が必要なプレイヤーエンティティのリストを返します。
func UpdatePlayerInputSystem(world donburi.World) PlayerInputSystemResult {
	var playersToAct []*donburi.Entry

	query.NewQuery(filter.Contains(PlayerControlComponent)).Each(world, func(entry *donburi.Entry) {
		if StateComponent.Get(entry).FSM.Is(string(StateIdle)) {
			playersToAct = append(playersToAct, entry)
		}
	})

	// 行動順は推進力などでソートすることも可能ですが、ここでは単純に検出順とします。
	// 必要であれば、ここでソートロジックを追加します。
	// sort.Slice(playersToAct, func(i, j int) bool { ... })

	return PlayerInputSystemResult{PlayerMedarotsToAct: playersToAct}
}

// UpdateAIInputSystem はAI制御のメダロットの行動選択を処理します。
// このシステムは BattleScene に直接依存しません。
// aiSelectAction は BattleScene ではなく、world と必要なヘルパーを引数に取るように変更されることを想定しています。
func UpdateAIInputSystem(
	world donburi.World,
	battleLogic *BattleLogic, // battleLogic を追加
) {
	// BattleScene の state や playerMedarotToAct に相当する条件をどのように扱うか検討が必要です。
	// 例えば、ワールドリソースでゲームの状態（PlayerTurn, AITurnなど）を管理します。
	// ここでは、単純にIdle状態の非プレイヤーエンティティを対象とします。

	query.NewQuery(
		filter.Not(filter.Contains(PlayerControlComponent)), // プレイヤー制御ではないエンティティ
	).Each(world, func(entry *donburi.Entry) {
		if !entry.HasComponent(StateComponent) || !StateComponent.Get(entry).FSM.Is(string(StateIdle)) {
			return
		}
		// aiSelectAction のシグネチャが (world donburi.World, actingEntry *donburi.Entry, pip *PartInfoProvider, ts *TargetSelector, conf *Config) のようになっていると仮定します。
		// もし aiSelectAction が BattleScene 全体を必要とする場合、そのリファクタリングも必要です。
		aiSelectAction(world, entry, battleLogic) // battleLogic を追加
	})
}
