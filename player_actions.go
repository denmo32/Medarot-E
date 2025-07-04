package main

import (
	"log"
	"math/rand"

	"github.com/yohamta/donburi"
)

// playerSelectRandomTargetPart はプレイヤー専用の、ランダムな敵の有効なパーツをターゲットとして選択する関数です。
func playerSelectRandomTargetPart(
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider, // 将来の拡張に備えて追加
) (*donburi.Entry, PartSlotKey) {
	if targetSelector == nil {
		log.Printf("playerSelectRandomTargetPart: targetSelector がnilです。")
		return nil, ""
	}
	candidates := targetSelector.GetTargetableEnemies(actingEntry)
	log.Printf("playerSelectRandomTargetPart: ターゲット候補の敵: %d 体", len(candidates))
	if len(candidates) == 0 {
		log.Printf("playerSelectRandomTargetPart: ターゲット候補の敵がいません。")
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
	log.Printf("playerSelectRandomTargetPart: 攻撃可能なパーツを持つ敵の総数: %d", len(allTargetableParts))

	if len(allTargetableParts) == 0 {
		// 攻撃できるパーツがない場合でも、最低限敵エンティティは返します（スロットキーは空）。
		if len(candidates) > 0 {
			// 破壊されていない敵がいれば、そのうちの1体を返します (パーツは特定できません)。
			selectedEnemy := candidates[rand.Intn(len(candidates))]
			log.Printf("playerSelectRandomTargetPart: 攻撃可能なパーツがないため、敵エンティティ %s を選択 (パーツなし)", SettingsComponent.Get(selectedEnemy).Name)
			return selectedEnemy, ""
		}
		log.Printf("playerSelectRandomTargetPart: 攻撃可能な敵もパーツもいません。")
		return nil, ""
	}

	// ランダムに一つのパーツを選択します。
	selected := allTargetableParts[rand.Intn(len(allTargetableParts))]
	log.Printf("playerSelectRandomTargetPart: 選択されたターゲット: %s の %s", SettingsComponent.Get(selected.entity).Name, selected.slot)
	return selected.entity, selected.slot
}
