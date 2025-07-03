package main

import (
	"log"
	"math/rand"
	// "sort"

	"github.com/yohamta/donburi"
	// "github.com/yohamta/donburi/filter"
	// "github.com/yohamta/donburi/query"
)

// playerSelectRandomTarget はプレイヤー専用の、ランダムな敵パーツをターゲットとして選択する関数です。
func playerSelectRandomTarget(bs *BattleScene, actingEntry *donburi.Entry) (*donburi.Entry, PartSlotKey) {
	if bs.targetSelector == nil {
		log.Println("エラー: playerSelectRandomTarget - bs.targetSelector がnilです。")
		return nil, ""
	}
	candidates := bs.targetSelector.GetTargetableEnemies(actingEntry)
	if len(candidates) == 0 {
		return nil, ""
	}

	// 攻撃可能なパーツを持つ敵のリストをフラットに作成します。
	type targetablePartInfo struct { // 内部でのみ使用するため、より具体的な名前に変更
		entity *donburi.Entry
		slot   PartSlotKey
	}
	var allTargetableParts []targetablePartInfo

	for _, enemyEntry := range candidates {
		partsMap := PartsComponent.Get(enemyEntry).Map
		for slotKey, part := range partsMap {
			// 破壊されていないパーツは全て攻撃対象候補とします。
			if !part.IsBroken {
				allTargetableParts = append(allTargetableParts, targetablePartInfo{
					entity: enemyEntry,
					slot:   slotKey,
				})
			}
		}
	}

	if len(allTargetableParts) == 0 {
		// 攻撃できるパーツがない場合でも、最低限敵エンティティは返します（スロットキーは空）。
		if len(candidates) > 0 {
			// 破壊されていない敵がいれば、そのうちの1体を返します (パーツは特定できません)。
			return candidates[rand.Intn(len(candidates))], ""
		}
		return nil, ""
	}

	// ランダムに一つのパーツを選択します。
	selected := allTargetableParts[rand.Intn(len(allTargetableParts))]
	return selected.entity, selected.slot
}
