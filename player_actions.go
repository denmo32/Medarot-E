package main

import (
	"math/rand"
	"sort"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// playerGetTargetCandidates はプレイヤー専用の、相手チームの有効なメダロットリストを返す関数
// ★★★ 修正点: 引数を *Game から *BattleScene に変更 ★★★
func playerGetTargetCandidates(bs *BattleScene, actingEntry *donburi.Entry) []*donburi.Entry {
	playerTeam := SettingsComponent.Get(actingEntry).Team

	var opponentTeamID TeamID
	if playerTeam == Team1 {
		opponentTeamID = Team2
	} else {
		opponentTeamID = Team1
	}

	candidates := []*donburi.Entry{}
	query.NewQuery(filter.And(
		filter.Contains(SettingsComponent),
		filter.Contains(StateComponent),
	)).Each(bs.world, func(entry *donburi.Entry) { // ★★★ 修正点: g.World -> bs.world ★★★
		settings := SettingsComponent.Get(entry)
		state := StateComponent.Get(entry)
		if settings.Team == opponentTeamID && state.State != StateBroken {
			candidates = append(candidates, entry)
		}
	})

	sort.Slice(candidates, func(i, j int) bool {
		iSettings := SettingsComponent.Get(candidates[i])
		jSettings := SettingsComponent.Get(candidates[j])
		return iSettings.DrawIndex < jSettings.DrawIndex
	})

	return candidates
}

// playerSelectRandomTarget はプレイヤー専用の、ランダムな敵パーツをターゲットとして選択する関数
// ★★★ 修正点: 引数を *Game から *BattleScene に変更 ★★★
func playerSelectRandomTarget(bs *BattleScene, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	// プレイヤー専用の候補取得関数を使う
	candidates := playerGetTargetCandidates(bs, actingEntry) // ★★★ 修正点: g -> bs ★★★
	if len(candidates) == 0 {
		return nil, ""
	}

	// 攻撃可能なパーツを持つ敵のリストをフラットに作成
	type targetablePart struct {
		entity *donburi.Entry
		slot   PartSlotKey
	}
	var allTargetableParts []targetablePart

	for _, enemyEntry := range candidates {
		partsMap := PartsComponent.Get(enemyEntry).Map
		for slotKey, part := range partsMap {
			// 脚部パーツは攻撃対象外とする
			if !part.IsBroken && slotKey != PartSlotLegs {
				allTargetableParts = append(allTargetableParts, targetablePart{
					entity: enemyEntry,
					slot:   slotKey,
				})
			}
		}
	}

	if len(allTargetableParts) == 0 {
		// 攻撃できるパーツがない場合でも、最低限敵エンティティは返す
		if len(candidates) > 0 {
			return candidates[0], ""
		}
		return nil, ""
	}

	// ランダムに一つのパーツを選択
	selected := allTargetableParts[rand.Intn(len(allTargetableParts))]
	return selected.entity, selected.slot
}
