package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// StatusEffectSystem はステータス効果の適用、更新、解除を管理します。
type StatusEffectSystem struct {
	world donburi.World
}

// NewStatusEffectSystem は新しいStatusEffectSystemのインスタンスを生成します。
func NewStatusEffectSystem(world donburi.World) *StatusEffectSystem {
	return &StatusEffectSystem{
		world: world,
	}
}

// Apply はエンティティにステータス効果を適用します。
func (s *StatusEffectSystem) Apply(entry *donburi.Entry, effect StatusEffect) {
	log.Printf("Applying effect '%s' to %s", effect.Description(), SettingsComponent.Get(entry).Name)
	effect.Apply(s.world, entry)

	// 効果の持続時間を管理するコンポーネントを追加
	if !entry.HasComponent(ActiveEffectsComponent) {
		donburi.Add(entry, ActiveEffectsComponent, &ActiveEffects{
			Effects: make([]*ActiveStatusEffectData, 0),
		})
	}
	activeEffects := ActiveEffectsComponent.Get(entry)
	activeEffects.Effects = append(activeEffects.Effects, &ActiveStatusEffectData{
		EffectData:   effect,
		RemainingDur: effect.Duration(),
	})
}

// Remove はエンティティからステータス効果を解除します。
func (s *StatusEffectSystem) Remove(entry *donburi.Entry, effect StatusEffect) {
	log.Printf("Removing effect '%s' from %s", effect.Description(), SettingsComponent.Get(entry).Name)
	effect.Remove(s.world, entry)

	if entry.HasComponent(ActiveEffectsComponent) {
		activeEffects := ActiveEffectsComponent.Get(entry)
		newEffects := make([]*ActiveStatusEffectData, 0)
		for _, activeEffect := range activeEffects.Effects {
			if activeEffect.EffectData != effect {
				newEffects = append(newEffects, activeEffect)
			}
		}
		activeEffects.Effects = newEffects
	}
}

// Update は毎フレーム呼び出され、効果の持続時間を更新し、期限切れの効果を削除します。
func (s *StatusEffectSystem) Update() {
	query.NewQuery(filter.Contains(ActiveEffectsComponent)).Each(s.world, func(entry *donburi.Entry) {
		activeEffects := ActiveEffectsComponent.Get(entry)
		newEffects := make([]*ActiveStatusEffectData, 0)
		for _, activeEffect := range activeEffects.Effects {
			if activeEffect.RemainingDur > 0 {
				activeEffect.RemainingDur--
				if activeEffect.RemainingDur == 0 {
					effect := activeEffect.EffectData.(StatusEffect)
					log.Printf("Effect '%s' expired for %s", effect.Description(), SettingsComponent.Get(entry).Name)
					effect.Remove(s.world, entry)
				} else {
					newEffects = append(newEffects, activeEffect)
				}
			} else {
				// Duration 0 or less means it's not a timed effect, so keep it.
				newEffects = append(newEffects, activeEffect)
			}
		}
		activeEffects.Effects = newEffects
	})
}
