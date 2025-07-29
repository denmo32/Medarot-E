package main

import (
	"fmt"

	"github.com/yohamta/donburi"
)

// ApplyChargeStopEffect はChargeStopEffectDataを適用するロジックです。
func ApplyChargeStopEffect(world donburi.World, target *donburi.Entry, data *ChargeStopEffectData) {
	// この効果の適用ロジックはChargeInitiationSystemなどで処理される
}

// RemoveChargeStopEffect はChargeStopEffectDataを解除するロジックです。
func RemoveChargeStopEffect(world donburi.World, target *donburi.Entry, data *ChargeStopEffectData) {
	// この効果の解除ロジックはChargeInitiationSystemなどで処理される
}

// DescriptionChargeStopEffect はChargeStopEffectDataの説明を返します。
func DescriptionChargeStopEffect(data *ChargeStopEffectData) string {
	return fmt.Sprintf("チャージ停止 (Duration: %d)", data.DurationTurns)
}

// DurationChargeStopEffect はChargeStopEffectDataの持続時間を返します。
func DurationChargeStopEffect(data *ChargeStopEffectData) int { return data.DurationTurns }

// TypeChargeStopEffect はChargeStopEffectDataの種類を返します。
func TypeChargeStopEffect(data *ChargeStopEffectData) DebuffType { return DebuffTypeChargeStop }

// ApplyDamageOverTimeEffect はDamageOverTimeEffectDataを適用するロジックです。
func ApplyDamageOverTimeEffect(world donburi.World, target *donburi.Entry, data *DamageOverTimeEffectData) {
	// この効果の適用ロジックはStatusEffectSystemなどで処理される
}

// RemoveDamageOverTimeEffect はDamageOverTimeEffectDataを解除するロジックです。
func RemoveDamageOverTimeEffect(world donburi.World, target *donburi.Entry, data *DamageOverTimeEffectData) {
	// この効果の解除ロジックはStatusEffectSystemなどで処理される
}

// DescriptionDamageOverTimeEffect はDamageOverTimeEffectDataの説明を返します。
func DescriptionDamageOverTimeEffect(data *DamageOverTimeEffectData) string {
	return fmt.Sprintf("継続ダメージ (%d/ターン)", data.DamagePerTurn)
}

// DurationDamageOverTimeEffect はDamageOverTimeEffectDataの持続時間を返します。
func DurationDamageOverTimeEffect(data *DamageOverTimeEffectData) int { return data.DurationTurns }

// TypeDamageOverTimeEffect はDamageOverTimeEffectDataの種類を返します。
func TypeDamageOverTimeEffect(data *DamageOverTimeEffectData) DebuffType {
	return DebuffTypeDamageOverTime
}

// ApplyTargetRandomEffect はTargetRandomEffectDataを適用するロジックです。
func ApplyTargetRandomEffect(world donburi.World, target *donburi.Entry, data *TargetRandomEffectData) {
	// この効果の適用ロジックはBattleTargetSelectorなどで処理される
}

// RemoveTargetRandomEffect はTargetRandomEffectDataを解除するロジックです。
func RemoveTargetRandomEffect(world donburi.World, target *donburi.Entry, data *TargetRandomEffectData) {
	// この効果の解除ロジックはBattleTargetSelectorなどで処理される
}

// DescriptionTargetRandomEffect はTargetRandomEffectDataの説明を返します。
func DescriptionTargetRandomEffect(data *TargetRandomEffectData) string {
	return fmt.Sprintf("ターゲットランダム化 (Duration: %d)", data.DurationTurns)
}

// DurationTargetRandomEffect はTargetRandomEffectDataの持続時間を返します。
func DurationTargetRandomEffect(data *TargetRandomEffectData) int { return data.DurationTurns }

// TypeTargetRandomEffect はTargetRandomEffectDataの種類を返します。
func TypeTargetRandomEffect(data *TargetRandomEffectData) DebuffType { return DebuffTypeTargetRandom }

// ApplyEvasionDebuffEffect はEvasionDebuffEffectDataを適用するロジックです。
func ApplyEvasionDebuffEffect(world donburi.World, target *donburi.Entry, data *EvasionDebuffEffectData) {
	// EvasionDebuffComponentはActiveEffectsComponentに統合されたため、直接追加・削除は不要
	// log.Printf("EvasionDebuffEffect applied to %s", SettingsComponent.Get(target).Name)
}

// RemoveEvasionDebuffEffect はEvasionDebuffEffectDataを解除するロジックです。
func RemoveEvasionDebuffEffect(world donburi.World, target *donburi.Entry, data *EvasionDebuffEffectData) {
	// EvasionDebuffComponentはActiveEffectsComponentに統合されたため、直接追加・削除は不要
	// log.Printf("EvasionDebuffEffect removed from %s", SettingsComponent.Get(target).Name)
}

// DescriptionEvasionDebuffEffect はEvasionDebuffEffectDataの説明を返します。
func DescriptionEvasionDebuffEffect(data *EvasionDebuffEffectData) string {
	return fmt.Sprintf("Evasion Debuff (x%.2f)", data.Multiplier)
}

// DurationEvasionDebuffEffect はEvasionDebuffEffectDataの持続時間を返します。
func DurationEvasionDebuffEffect(data *EvasionDebuffEffectData) int {
	// 0 means it will be removed manually (e.g., after an action).
	return 0
}

// TypeEvasionDebuffEffect はEvasionDebuffEffectDataの種類を返します。
func TypeEvasionDebuffEffect(data *EvasionDebuffEffectData) DebuffType {
	return DebuffTypeEvasion
}

// ApplyDefenseDebuffEffect はDefenseDebuffEffectDataを適用するロジックです。
func ApplyDefenseDebuffEffect(world donburi.World, target *donburi.Entry, data *DefenseDebuffEffectData) {
	// DefenseDebuffComponentはActiveEffectsComponentに統合されたため、直接追加・削除は不要
	// log.Printf("DefenseDebuffEffect applied to %s", SettingsComponent.Get(target).Name)
}

// RemoveDefenseDebuffEffect はDefenseDebuffEffectDataを解除するロジックです。
func RemoveDefenseDebuffEffect(world donburi.World, target *donburi.Entry, data *DefenseDebuffEffectData) {
	// DefenseDebuffComponentはActiveEffectsComponentに統合されたため、直接追加・削除は不要
	// log.Printf("DefenseDebuffEffect removed from %s", SettingsComponent.Get(target).Name)
}

// DescriptionDefenseDebuffEffect はDefenseDebuffEffectDataの説明を返します。
func DescriptionDefenseDebuffEffect(data *DefenseDebuffEffectData) string {
	return fmt.Sprintf("Defense Debuff (x%.2f)", data.Multiplier)
}

// DurationDefenseDebuffEffect はDefenseDebuffEffectDataの持続時間を返します。
func DurationDefenseDebuffEffect(data *DefenseDebuffEffectData) int {
	return 0
}

// TypeDefenseDebuffEffect はDefenseDebuffEffectDataの種類を返します。
func TypeDefenseDebuffEffect(data *DefenseDebuffEffectData) DebuffType {
	return DebuffTypeDefense
}
