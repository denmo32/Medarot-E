package main

import (
	"context"
	"log"

	"github.com/yohamta/donburi"
)

// PostActionEffectSystem は、アクション実行後の効果を処理するECSシステムです。
type PostActionEffectSystem struct {
	world              donburi.World
	statusEffectSystem *StatusEffectSystem // StatusEffectSystemへの参照
}

// NewPostActionEffectSystem は新しいPostActionEffectSystemのインスタンスを生成します。
func NewPostActionEffectSystem(world donburi.World, statusEffectSystem *StatusEffectSystem) *PostActionEffectSystem {
	return &PostActionEffectSystem{
		world:              world,
		statusEffectSystem: statusEffectSystem,
	}
}

// Process は、ActionResultに基づいてアクション後の効果を処理します。
func (s *PostActionEffectSystem) Process(result *ActionResult) {
	if result == nil {
		return
	}

	// 1. 適用されるべきステータス効果を処理
	if len(result.AppliedEffects) > 0 {
		// 効果の適用対象を決定する（通常は行動者自身）
		// 将来的には効果ごとに対象を指定できるように拡張可能
		targetEntry := result.ActingEntry
		if targetEntry != nil {
			for _, effect := range result.AppliedEffects {
				s.statusEffectSystem.Apply(targetEntry, effect)
			}
		}
	}

	// 2. パーツ破壊による状態遷移
	if result.TargetEntry != nil && result.TargetPartBroken && result.ActualHitPartSlot == PartSlotHead {
		state := StateComponent.Get(result.TargetEntry)
		if state.FSM.Can("break") {
			err := state.FSM.Event(context.Background(), "break", result.TargetEntry)
			if err != nil {
				log.Printf("Error breaking medarot %s: %v", SettingsComponent.Get(result.TargetEntry).Name, err)
			}
		}
	}

	// 3. 行動後のクリーンアップ
	//    チャージ中に付与された効果で、持続時間が0のものを解除する
	if result.ActingEntry != nil && result.ActingEntry.HasComponent(ActiveEffectsComponent) {
		activeEffects := ActiveEffectsComponent.Get(result.ActingEntry)
		// このループ内でRemoveを呼ぶとスライスが変更されて危険なので、
		// 解除すべきエフェクトを一旦リストアップする
		effectsToRemove := []StatusEffect{}
		for _, activeEffect := range activeEffects.Effects {
			if activeEffect.RemainingDur == 0 {
				effectsToRemove = append(effectsToRemove, activeEffect.Effect)
			}
		}
		// リストアップしたエフェクトを安全に解除する
		for _, effect := range effectsToRemove {
			s.statusEffectSystem.Remove(result.ActingEntry, effect)
		}
	}
}
