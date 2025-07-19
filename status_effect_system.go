package main

import (
	"fmt"
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
		donburi.Add(entry, ActiveEffectsComponent, &ActiveEffects{})
	}
	activeEffects := ActiveEffectsComponent.Get(entry)
	activeEffects.Effects = append(activeEffects.Effects, &ActiveStatusEffect{
		Effect:       effect,
		RemainingDur: effect.Duration(),
	})
}

// Remove はエンティティからステータス効果を解除します。
func (s *StatusEffectSystem) Remove(entry *donburi.Entry, effect StatusEffect) {
	log.Printf("Removing effect '%s' from %s", effect.Description(), SettingsComponent.Get(entry).Name)
	effect.Remove(s.world, entry)

	if entry.HasComponent(ActiveEffectsComponent) {
		activeEffects := ActiveEffectsComponent.Get(entry)
		newEffects := make([]*ActiveStatusEffect, 0)
		for _, activeEffect := range activeEffects.Effects {
			// Note: This might not work correctly if two identical effects are applied.
			// A more robust system would use unique IDs for each applied effect instance.
			if activeEffect.Effect != effect {
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
		newEffects := make([]*ActiveStatusEffect, 0)
		for _, activeEffect := range activeEffects.Effects {
			if activeEffect.RemainingDur > 0 {
				activeEffect.RemainingDur--
				if activeEffect.RemainingDur == 0 {
					log.Printf("Effect '%s' expired for %s", activeEffect.Effect.Description(), SettingsComponent.Get(entry).Name)
					activeEffect.Effect.Remove(s.world, entry)
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

// --- Concrete Status Effect Implementations ---

// EvasionDebuffEffect は回避率を低下させるデバフです。
type EvasionDebuffEffect struct {
	Multiplier float64
}

func (e *EvasionDebuffEffect) Apply(world donburi.World, target *donburi.Entry) {
	donburi.Add(target, EvasionDebuffComponent, &EvasionDebuff{Multiplier: e.Multiplier})
}

func (e *EvasionDebuffEffect) Remove(world donburi.World, target *donburi.Entry) {
	if target.HasComponent(EvasionDebuffComponent) {
		target.RemoveComponent(EvasionDebuffComponent)
	}
}

func (e *EvasionDebuffEffect) Description() string {
	return fmt.Sprintf("Evasion Debuff (x%.2f)", e.Multiplier)
}

func (e *EvasionDebuffEffect) Duration() int {
	// 0 means it will be removed manually (e.g., after an action).
	return 0
}

// DefenseDebuffEffect は防御力を低下させるデバフです。
type DefenseDebuffEffect struct {
	Multiplier float64
}

func (d *DefenseDebuffEffect) Apply(world donburi.World, target *donburi.Entry) {
	donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{Multiplier: d.Multiplier})
}

func (d *DefenseDebuffEffect) Remove(world donburi.World, target *donburi.Entry) {
	if target.HasComponent(DefenseDebuffComponent) {
		target.RemoveComponent(DefenseDebuffComponent)
	}
}

func (d *DefenseDebuffEffect) Description() string {
	return fmt.Sprintf("Defense Debuff (x%.2f)", d.Multiplier)
}

func (d *DefenseDebuffEffect) Duration() int {
	return 0
}
