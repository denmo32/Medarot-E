package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// --- WeaponTypeEffectHandlers ---
// 以下は構想案であり、名称や効果は変更の可能性があります。
// ThunderEffectHandler はサンダー効果（チャージ停止）を付与します。
type ThunderEffectHandler struct{}

func (h *ThunderEffectHandler) ApplyEffect(result *ActionResult, world donburi.World, battleLogic *BattleLogic, actingPartDef *PartDefinition) {
	if result.ActionDidHit && result.TargetEntry != nil {
		log.Printf("%s にサンダー効果！チャージを停止させます。", result.DefenderName)
		// ActionResult.AppliedEffectsにChargeStopEffectを追加
		result.AppliedEffects = append(result.AppliedEffects, &ChargeStopEffect{DurationTurns: 1}) // 例として1ターン
	}
}

// MeltEffectHandler はメルト効果（継続ダメージ）を付与します。
type MeltEffectHandler struct{}

func (h *MeltEffectHandler) ApplyEffect(result *ActionResult, world donburi.World, battleLogic *BattleLogic, actingPartDef *PartDefinition) {
	if result.ActionDidHit && result.TargetEntry != nil {
		log.Printf("%s にメルト効果！継続ダメージを与えます。", result.DefenderName)
		// ActionResult.AppliedEffectsにDamageOverTimeEffectを追加
		result.AppliedEffects = append(result.AppliedEffects, &DamageOverTimeEffect{DamagePerTurn: 10, DurationTurns: 2}) // 例としてダメージ10、2ターン
	}
}

// VirusEffectHandler はウイルス効果（ターゲットのランダム化）を付与します。
type VirusEffectHandler struct{}

func (h *VirusEffectHandler) ApplyEffect(result *ActionResult, world donburi.World, battleLogic *BattleLogic, actingPartDef *PartDefinition) {
	if result.ActionDidHit && result.TargetEntry != nil {
		log.Printf("%s にウイルス効果！ターゲットをランダム化します。", result.DefenderName)
		// ActionResult.AppliedEffectsにTargetRandomEffectを追加
		result.AppliedEffects = append(result.AppliedEffects, &TargetRandomEffect{DurationTurns: 1}) // 例として1ターン
	}
}
