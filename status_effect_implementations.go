package main

import (
	"fmt"
	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
)

// ApplyChargeStopEffect はChargeStopEffectDataを適用するロジックです。
func ApplyChargeStopEffect(world donburi.World, target *donburi.Entry, data *component.ChargeStopEffectData) {
	// この効果の適用ロジックはChargeInitiationSystemなどで処理される
}

// RemoveChargeStopEffect はChargeStopEffectDataを解除するロジックです。
func RemoveChargeStopEffect(world donburi.World, target *donburi.Entry, data *component.ChargeStopEffectData) {
	// この効果の解除ロジックはChargeInitiationSystemなどで処理される
}

// DescriptionChargeStopEffect はChargeStopEffectDataの説明を返します。
func DescriptionChargeStopEffect(data *component.ChargeStopEffectData) string {
	return fmt.Sprintf("チャージ停止 (Duration: %d)", data.DurationTurns)
}

// DurationChargeStopEffect はChargeStopEffectDataの持続時間を返します。
func DurationChargeStopEffect(data *component.ChargeStopEffectData) int { return data.DurationTurns }

// TypeChargeStopEffect はChargeStopEffectDataの種類を返します。
func TypeChargeStopEffect(data *component.ChargeStopEffectData) component.DebuffType {
	return component.DebuffTypeChargeStop
}

// ApplyDamageOverTimeEffect はDamageOverTimeEffectDataを適用するロジックです。
func ApplyDamageOverTimeEffect(world donburi.World, target *donburi.Entry, data *component.DamageOverTimeEffectData) {
	// この効果の適用ロジックはStatusEffectSystemなどで処理される
}

// RemoveDamageOverTimeEffect はDamageOverTimeEffectDataを解除するロジックです。
func RemoveDamageOverTimeEffect(world donburi.World, target *donburi.Entry, data *component.DamageOverTimeEffectData) {
	// この効果の解除ロジックはStatusEffectSystemなどで処理される
}

// DescriptionDamageOverTimeEffect はDamageOverTimeEffectDataの説明を返します。
func DescriptionDamageOverTimeEffect(data *component.DamageOverTimeEffectData) string {
	return fmt.Sprintf("継続ダメージ (%d/ターン)", data.DamagePerTurn)
}

// DurationDamageOverTimeEffect はDamageOverTimeEffectDataの持続時間を返します。
func DurationDamageOverTimeEffect(data *component.DamageOverTimeEffectData) int {
	return data.DurationTurns
}

// TypeDamageOverTimeEffect はDamageOverTimeEffectDataの種類を返します。
func TypeDamageOverTimeEffect(data *component.DamageOverTimeEffectData) component.DebuffType {
	return component.DebuffTypeDamageOverTime
}

// ApplyTargetRandomEffect はTargetRandomEffectDataを適用するロジックです。
func ApplyTargetRandomEffect(world donburi.World, target *donburi.Entry, data *component.TargetRandomEffectData) {
	// この効果の適用ロジックはBattleTargetSelectorなどで処理される
}

// RemoveTargetRandomEffect はTargetRandomEffectDataを解除するロジックです。
func RemoveTargetRandomEffect(world donburi.World, target *donburi.Entry, data *component.TargetRandomEffectData) {
	// この効果の解除ロジックはBattleTargetSelectorなどで処理される
}

// DescriptionTargetRandomEffect はTargetRandomEffectDataの説明を返します。
func DescriptionTargetRandomEffect(data *component.TargetRandomEffectData) string {
	return fmt.Sprintf("ターゲットランダム化 (Duration: %d)", data.DurationTurns)
}

// DurationTargetRandomEffect はTargetRandomEffectDataの持続時間を返します。
func DurationTargetRandomEffect(data *component.TargetRandomEffectData) int {
	return data.DurationTurns
}

// TypeTargetRandomEffect はTargetRandomEffectDataの種類を返します。
func TypeTargetRandomEffect(data *component.TargetRandomEffectData) component.DebuffType {
	return component.DebuffTypeTargetRandom
}

// ApplyEvasionDebuffEffect はEvasionDebuffEffectDataを適用するロジックです。
func ApplyEvasionDebuffEffect(world donburi.World, target *donburi.Entry, data *component.EvasionDebuffEffectData) {
	// EvasionDebuffComponentはActiveEffectsComponentに統合されたため、直接追加・削除は不要
	// log.Printf("EvasionDebuffEffect applied to %s", SettingsComponent.Get(target).Name)
}

// RemoveEvasionDebuffEffect はEvasionDebuffEffectDataを解除するロジックです。
func RemoveEvasionDebuffEffect(world donburi.World, target *donburi.Entry, data *component.EvasionDebuffEffectData) {
	// EvasionDebuffComponentはActiveEffectsComponentに統合されたため、直接追加・削除は不要
	// log.Printf("EvasionDebuffEffect removed from %s", SettingsComponent.Get(target).Name)
}

// DescriptionEvasionDebuffEffect はEvasionDebuffEffectDataの説明を返します。
func DescriptionEvasionDebuffEffect(data *component.EvasionDebuffEffectData) string {
	return fmt.Sprintf("Evasion Debuff (x%.2f)", data.Multiplier)
}

// DurationEvasionDebuffEffect はEvasionDebuffEffectDataの持続時間を返します。
func DurationEvasionDebuffEffect(data *component.EvasionDebuffEffectData) int {
	// 0 means it will be removed manually (e.g., after an action).
	return 0
}

// TypeEvasionDebuffEffect はEvasionDebuffEffectDataの種類を返します。
func TypeEvasionDebuffEffect(data *component.EvasionDebuffEffectData) component.DebuffType {
	return component.DebuffTypeEvasion
}

// ApplyDefenseDebuffEffect はDefenseDebuffEffectDataを適用するロジックです。
func ApplyDefenseDebuffEffect(world donburi.World, target *donburi.Entry, data *component.DefenseDebuffEffectData) {
	// DefenseDebuffComponentはActiveEffectsComponentに統合されたため、直接追加・削除は不要
	// log.Printf("DefenseDebuffEffect applied to %s", SettingsComponent.Get(target).Name)
}

// RemoveDefenseDebuffEffect はDefenseDebuffEffectDataを解除するロジックです。
func RemoveDefenseDebuffEffect(world donburi.World, target *donburi.Entry, data *component.DefenseDebuffEffectData) {
	// DefenseDebuffComponentはActiveEffectsComponentに統合されたため、直接追加・削除は不要
	// log.Printf("DefenseDebuffEffect removed from %s", SettingsComponent.Get(target).Name)
}

// DescriptionDefenseDebuffEffect はDefenseDebuffEffectDataの説明を返します。
func DescriptionDefenseDebuffEffect(data *component.DefenseDebuffEffectData) string {
	return fmt.Sprintf("Defense Debuff (x%.2f)", data.Multiplier)
}

// DurationDefenseDebuffEffect はDefenseDebuffEffectDataの持続時間を返します。
func DurationDefenseDebuffEffect(data *component.DefenseDebuffEffectData) int {
	return 0
}

// TypeDefenseDebuffEffect はDefenseDebuffEffectDataの種類を返します。
func TypeDefenseDebuffEffect(data *component.DefenseDebuffEffectData) component.DebuffType {
	return component.DebuffTypeDefense
}
