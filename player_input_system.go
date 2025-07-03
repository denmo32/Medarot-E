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
		if StateComponent.Get(entry).Current == StateTypeIdle {
			playersToAct = append(playersToAct, entry)
		}
	})

	// 行動順は推進力などでソートすることも可能ですが、ここでは単純に検出順とします。
	// 必要であれば、ここでソートロジックを追加します。
	// sort.Slice(playersToAct, func(i, j int) bool { ... })

	return PlayerInputSystemResult{PlayerMedarotsToAct: playersToAct}
}
