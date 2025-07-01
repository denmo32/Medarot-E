package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// PlayerInputSystemResult holds the entity that requires player input.
type PlayerInputSystemResult struct {
	PlayerMedarotToAct *donburi.Entry
}

// UpdatePlayerInputSystem はプレイヤーが操作するメダロットの行動選択状態への遷移を処理します。
// このシステムは BattleScene に直接依存しません。
// 行動が必要なプレイヤーエンティティを返します。
func UpdatePlayerInputSystem(world donburi.World) PlayerInputSystemResult {
	var playerToAct *donburi.Entry

	// 設計上、一度にプレイヤーが操作するのは1体なので、最初に見つかったものを返す
	query.NewQuery(filter.And(
		filter.Contains(PlayerControlComponent),
		filter.Contains(IdleStateComponent),
	)).Each(world, func(entry *donburi.Entry) {
		if playerToAct == nil { // まだ行動可能なプレイヤーが見つかっていなければ設定
			playerToAct = entry
		}
		// Each は途中で抜けられないが、playerToAct が設定されていれば
		// 後続のループで上書きはしない。
	})

	return PlayerInputSystemResult{PlayerMedarotToAct: playerToAct}
}
