package main

import (
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
