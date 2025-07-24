package main

type DebuffType string

const (
	DebuffTypeEvasion        DebuffType = "Evasion"
	DebuffTypeDefense        DebuffType = "Defense"
	DebuffTypeChargeStop     DebuffType = "ChargeStop"     // チャージ一時停止
	DebuffTypeDamageOverTime DebuffType = "DamageOverTime" // チャージ中ダメージ
	DebuffTypeTargetRandom   DebuffType = "TargetRandom"   // ターゲットのランダム化
)

// ChargeStopEffect はチャージを一時停止させるデバフのデータです。
type ChargeStopEffect struct {
	DurationTurns int // ターン数での持続時間
}

// DamageOverTimeEffect は継続ダメージを与えるデバフのデータです。
type DamageOverTimeEffect struct {
	DamagePerTurn int
	DurationTurns int
}

// TargetRandomEffect はターゲットをランダム化するデバフのデータです。
type TargetRandomEffect struct {
	DurationTurns int
}

// EvasionDebuffEffect は回避率を低下させるデバフのデータです。
type EvasionDebuffEffect struct {
	Multiplier float64
}

// DefenseDebuffEffect は防御力を低下させるデバフのデータです。
type DefenseDebuffEffect struct {
	Multiplier float64
}

// ActiveStatusEffectData は、エンティティに現在適用されている効果のデータとその残り期間を追跡します。
type ActiveStatusEffectData struct {
	EffectData   interface{} // ChargeStopEffect, DamageOverTimeEffect などのインスタンス
	RemainingDur int
}
