package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdateAIInputSystem はAI制御のメダロットの行動選択を処理します。
// このシステムは BattleScene に直接依存しません。
// aiSelectAction は BattleScene ではなく、world と必要なヘルパーを引数に取るように変更される想定です。
func UpdateAIInputSystem(
	world donburi.World,
	partInfoProvider *PartInfoProvider,
	targetSelector *TargetSelector,
	gameConfig *Config, // aiSelectAction が config を必要とする場合
) {
	// BattleScene の state や playerMedarotToAct に相当する条件をどう扱うか検討が必要。
	// 例えば、ワールドリソースでゲームの状態 (PlayerTurn, AITurnなど) を管理する。
	// ここでは、単純にIdle状態の非プレイヤーエンティティを対象とする。

	query.NewQuery(filter.And(
		filter.Contains(IdleStateComponent),
		filter.Not(filter.Contains(PlayerControlComponent)), // プレイヤー制御でない
	)).Each(world, func(entry *donburi.Entry) {
		// aiSelectAction のシグネチャが (world donburi.World, actingEntry *donburi.Entry, pip *PartInfoProvider, ts *TargetSelector, conf *Config) のようになっていると仮定
		// もし aiSelectAction が BattleScene全体を必要とする場合、そのリファクタリングも必要
		aiSelectAction(world, entry, partInfoProvider, targetSelector, gameConfig)
	})
}
