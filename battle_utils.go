package main

import (
	"github.com/yohamta/donburi"
)

// CleanupActionDebuffs は行動後のデバフをクリーンアップします。
// この関数は、行動を行ったエンティティから回避デバフと防御デバフのコンポーネントを削除します。
func CleanupActionDebuffs(actingEntry *donburi.Entry) {
	if actingEntry.HasComponent(EvasionDebuffComponent) {
		actingEntry.RemoveComponent(EvasionDebuffComponent)
	}
	if actingEntry.HasComponent(DefenseDebuffComponent) {
		actingEntry.RemoveComponent(DefenseDebuffComponent)
	}
}
