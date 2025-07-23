package main

import (
	"github.com/yohamta/donburi"
)

type DebuffType string

const (
	DebuffTypeEvasion        DebuffType = "Evasion"
	DebuffTypeDefense        DebuffType = "Defense"
	DebuffTypeChargeStop     DebuffType = "ChargeStop"     // チャージ一時停止
	DebuffTypeDamageOverTime DebuffType = "DamageOverTime" // チャージ中ダメージ
	DebuffTypeTargetRandom   DebuffType = "TargetRandom"   // ターゲットのランダム化
)

// StatusEffect は、すべてのステータス効果（バフ・デバフ）が実装すべきインターフェースです。
type StatusEffect interface {
	Apply(world donburi.World, target *donburi.Entry)
	Remove(world donburi.World, target *donburi.Entry)
	Description() string
	Duration() int    // 効果の持続時間（ターン数や秒数など）。0の場合は永続、または即時解除。
	Type() DebuffType // 効果の種類を返すメソッドを追加
}

// ActiveStatusEffect は、エンティティに現在適用されている効果とその残り期間を追跡します。
type ActiveStatusEffect struct {
	Effect       StatusEffect
	RemainingDur int
}

// ChargeStopEffect はチャージを一時停止させるデバフです。
type ChargeStopEffect struct {
	DurationTurns int // ターン数での持続時間
}

// DamageOverTimeEffect は継続ダメージを与えるデバフです。
type DamageOverTimeEffect struct {
	DamagePerTurn int
	DurationTurns int
}

// TargetRandomEffect はターゲットをランダム化するデバフです。
type TargetRandomEffect struct {
	DurationTurns int
}

// EvasionDebuffEffect は回避率を低下させるデバフです。
type EvasionDebuffEffect struct {
	Multiplier float64
}

type DefenseDebuffEffect struct {
	Multiplier float64
}
