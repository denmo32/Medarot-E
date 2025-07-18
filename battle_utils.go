package main

import (
	"context"
	"log"

	"github.com/yohamta/donburi"
)

// ProcessPostActionEffects は、アクション実行後の共通処理（パーツ破壊、デバフ解除など）を適用します。
func ProcessPostActionEffects(result *ActionResult, world donburi.World) {
	if result == nil {
		return
	}

	// 1. パーツ破壊による状態遷移
	// 頭部パーツが破壊された場合、メダロットを機能停止状態に遷移させる
	if result.TargetEntry != nil && result.TargetPartBroken && result.ActualHitPartSlot == PartSlotHead {
		state := StateComponent.Get(result.TargetEntry)
		if state.FSM.Can("break") {
			err := state.FSM.Event(context.Background(), "break", result.TargetEntry)
			if err != nil {
				log.Printf("Error breaking medarot %s: %v", SettingsComponent.Get(result.TargetEntry).Name, err)
			}
		}
	}

	// 2. 行動後のデバフクリーンアップ
	// 行動を行ったエンティティから回避デバフと防御デバフのコンポーネントを削除します。
	if result.ActingEntry != nil {
		if result.ActingEntry.HasComponent(EvasionDebuffComponent) {
			result.ActingEntry.RemoveComponent(EvasionDebuffComponent)
		}
		if result.ActingEntry.HasComponent(DefenseDebuffComponent) {
			result.ActingEntry.RemoveComponent(DefenseDebuffComponent)
		}
	}
}
