package main

import (
	"fmt"

	"github.com/yohamta/donburi"
)

// ChargeStopEffect はチャージを一時停止させるデバフです。
func (e *ChargeStopEffect) Apply(world donburi.World, target *donburi.Entry) {
	// この効果の適用ロジックはChargeInitiationSystemなどで処理される
}
func (e *ChargeStopEffect) Remove(world donburi.World, target *donburi.Entry) {
	// この効果の解除ロジックはChargeInitiationSystemなどで処理される
}
func (e *ChargeStopEffect) Description() string { return fmt.Sprintf("チャージ停止 (Duration: %d)", e.DurationTurns) }
func (e *ChargeStopEffect) Duration() int       { return e.DurationTurns }
func (e *ChargeStopEffect) Type() DebuffType    { return DebuffTypeChargeStop }

// DamageOverTimeEffect は継続ダメージを与えるデバフです。
func (e *DamageOverTimeEffect) Apply(world donburi.World, target *donburi.Entry) {
	// この効果の適用ロジックはStatusEffectSystemなどで処理される
}
func (e *DamageOverTimeEffect) Remove(world donburi.World, target *donburi.Entry) {
	// この効果の解除ロジックはStatusEffectSystemなどで処理される
}
func (e *DamageOverTimeEffect) Description() string {
	return fmt.Sprintf("継続ダメージ (%d/ターン)", e.DamagePerTurn)
}
func (e *DamageOverTimeEffect) Duration() int    { return e.DurationTurns }
func (e *DamageOverTimeEffect) Type() DebuffType { return DebuffTypeDamageOverTime }

// TargetRandomEffect はターゲットをランダム化するデバフです。
func (e *TargetRandomEffect) Apply(world donburi.World, target *donburi.Entry) {
	// この効果の適用ロジックはBattleTargetSelectorなどで処理される
}
func (e *TargetRandomEffect) Remove(world donburi.World, target *donburi.Entry) {
	// この効果の解除ロジックはBattleTargetSelectorなどで処理される
}
func (e *TargetRandomEffect) Description() string { return fmt.Sprintf("ターゲットランダム化 (Duration: %d)", e.DurationTurns) }
func (e *TargetRandomEffect) Duration() int       { return e.DurationTurns }
func (e *TargetRandomEffect) Type() DebuffType    { return DebuffTypeTargetRandom }

// EvasionDebuffEffect は回避率を低下させるデバフです。
func (e *EvasionDebuffEffect) Apply(world donburi.World, target *donburi.Entry) {
	// EvasionDebuffComponentはActiveEffectsComponentに統合されたため、直接追加・削除は不要
	// log.Printf("EvasionDebuffEffect applied to %s", SettingsComponent.Get(target).Name)
}

func (e *EvasionDebuffEffect) Remove(world donburi.World, target *donburi.Entry) {
	// EvasionDebuffComponentはActiveEffectsComponentに統合されたため、直接追加・削除は不要
	// log.Printf("EvasionDebuffEffect removed from %s", SettingsComponent.Get(target).Name)
}

func (e *EvasionDebuffEffect) Description() string {
	return fmt.Sprintf("Evasion Debuff (x%.2f)", e.Multiplier)
}

func (e *EvasionDebuffEffect) Duration() int {
	// 0 means it will be removed manually (e.g., after an action).
	return 0
}

func (e *EvasionDebuffEffect) Type() DebuffType {
	return DebuffTypeEvasion
}

// DefenseDebuffEffect は防御力を低下させるデバフです。
func (d *DefenseDebuffEffect) Apply(world donburi.World, target *donburi.Entry) {
	// DefenseDebuffComponentはActiveEffectsComponentに統合されたため、直接追加・削除は不要
	// log.Printf("DefenseDebuffEffect applied to %s", SettingsComponent.Get(target).Name)
}

func (d *DefenseDebuffEffect) Remove(world donburi.World, target *donburi.Entry) {
	// DefenseDebuffComponentはActiveEffectsComponentに統合されたため、直接追加・削除は不要
	// log.Printf("DefenseDebuffEffect removed from %s", SettingsComponent.Get(target).Name)
}

func (d *DefenseDebuffEffect) Description() string {
	return fmt.Sprintf("Defense Debuff (x%.2f)", d.Multiplier)
}

func (d *DefenseDebuffEffect) Duration() int {
	return 0
}

func (d *DefenseDebuffEffect) Type() DebuffType {
	return DebuffTypeDefense
}
