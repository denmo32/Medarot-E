package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// PlayerInputSystemResult holds the list of entities that require player input.
type PlayerInputSystemResult struct {
	PlayerMedarotsToAct []*donburi.Entry
}

// UpdatePlayerInputSystem finds all player-controlled medarots in the Idle state.
// This system does not depend directly on BattleScene.
// It returns a list of player entities that need to act.
func UpdatePlayerInputSystem(world donburi.World) PlayerInputSystemResult {
	var playersToAct []*donburi.Entry

	query.NewQuery(filter.And(
		filter.Contains(PlayerControlComponent),
		filter.Contains(IdleStateComponent),
	)).Each(world, func(entry *donburi.Entry) {
		playersToAct = append(playersToAct, entry)
	})

	// 行動順は推進力などでソートすることも可能だが、ここでは単純に検出順とする。
	// 必要であれば、ここでソートロジックを追加。
	// sort.Slice(playersToAct, func(i, j int) bool { ... })

	return PlayerInputSystemResult{PlayerMedarotsToAct: playersToAct}
}
