package main

import (
	"log"
	"math/rand"
	// "sort"

	"github.com/yohamta/donburi"
	// "github.com/yohamta/donburi/filter"
	// "github.com/yohamta/donburi/query"
)

// playerSelectRandomTarget はプレイヤー専用の、ランダムな敵パーツをターゲットとして選択する関数
// ★★★ 修正点: 引数を *Game から *BattleScene に変更 ★★★
func playerSelectRandomTarget(bs *BattleScene, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	if bs.targetSelector == nil {
		log.Println("Error: playerSelectRandomTarget - bs.targetSelector is nil")
		return nil, ""
	}
	candidates := bs.targetSelector.GetTargetableEnemies(actingEntry)
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
			// 破壊されていないパーツは全て攻撃対象候補とする
			if !part.IsBroken {
				allTargetableParts = append(allTargetableParts, targetablePart{
					entity: enemyEntry,
					slot:   slotKey,
				})
			}
		}
	}

	if len(allTargetableParts) == 0 {
		// 攻撃できるパーツがない場合でも、最低限敵エンティティは返す（スロットキーは空）
		if len(candidates) > 0 {
			// 破壊されていない敵がいれば、そのうちの1体を返す (パーツは特定できない)
			return candidates[rand.Intn(len(candidates))], ""
		}
		return nil, ""
	}

	// ランダムに一つのパーツを選択
	selected := allTargetableParts[rand.Intn(len(allTargetableParts))]
	return selected.entity, selected.slot
}
