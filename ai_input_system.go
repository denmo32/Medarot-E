package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdateAIInputSystem はAI制御のメダロットの行動選択を処理します。
// このシステムは BattleScene に直接依存しません。
// aiSelectAction は BattleScene ではなく、world と必要なヘルパーを引数に取るように変更されることを想定しています。
func UpdateAIInputSystem(
	world donburi.World,
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
	gameConfig *Config,
) {
	// BattleScene の state や playerMedarotToAct に相当する条件をどのように扱うか検討が必要です。
	// 例えば、ワールドリソースでゲームの状態（PlayerTurn, AITurnなど）を管理します。
	// ここでは、単純にIdle状態の非プレイヤーエンティティを対象とします。

	query.NewQuery(filter.And(
		filter.Contains(IdleStateComponent),
		filter.Not(filter.Contains(PlayerControlComponent)), // プレイヤー制御ではないエンティティ
	)).Each(world, func(entry *donburi.Entry) {
		// aiSelectAction のシグネチャが (world donburi.World, actingEntry *donburi.Entry, pip *PartInfoProvider, ts *TargetSelector, conf *Config) のようになっていると仮定します。
		// もし aiSelectAction が BattleScene 全体を必要とする場合、そのリファクタリングも必要です。
		aiSelectAction(world, entry, partInfoProvider, targetSelector, gameConfig)
	})
}
