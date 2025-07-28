package main

import (
	"github.com/yohamta/donburi"
)

type TeamID int
type GameState string
type PartSlotKey string
type PartType string

// StateType はエンティティの状態を表す文字列です。
type StateType string
type PartCategory string
type Trait string
type WeaponType string

// GameEndResult はゲーム終了チェックの結果を保持します。
type GameEndResult struct {
	IsGameOver bool
	Winner     TeamID
	Message    string
}

// PlayerInputSystemResult はプレイヤーの入力が必要なエンティティのリストを保持します。
type PlayerInputSystemResult struct {
	PlayerMedarotsToAct []*donburi.Entry
}

// AvailablePart now holds PartDefinition for AI/UI to see base stats.
type AvailablePart struct {
	PartDef  *PartDefinition // Changed from Part to PartDefinition
	Slot     PartSlotKey
	IsBroken bool // パーツが破壊されているか
}

// MessageTemplate defines the structure for a single message in the JSON file.
type MessageTemplate struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// targetablePart はAIがターゲット可能なパーツの情報を保持します。
type targetablePart struct {
	entity   *donburi.Entry
	partInst *PartInstanceData
	partDef  *PartDefinition
	slot     PartSlotKey
}

// TargetingPolicyType はターゲット決定方針を示す型です。
type TargetingPolicyType string

const (
	Team1    TeamID = 0
	Team2    TeamID = 1
	TeamNone TeamID = -1 // 勝者なし、または引き分けを表します
)
const (
	StatePlaying            GameState = "Playing"
	StatePlayerActionSelect GameState = "PlayerActionSelect"
	StateAnimatingAction    GameState = "AnimatingAction" // アクションアニメーション再生中
	StateMessage            GameState = "Message"
	StateGameOver           GameState = "GameOver"
)

const (
	StateIdle     StateType = "idle"
	StateCharging StateType = "charging"
	StateReady    StateType = "ready"
	StateCooldown StateType = "cooldown"
	StateBroken   StateType = "broken"
)

const (
	PartSlotHead     PartSlotKey = "head"
	PartSlotRightArm PartSlotKey = "r_arm"
	PartSlotLeftArm  PartSlotKey = "l_arm"
	PartSlotLegs     PartSlotKey = "legs"
)
const (
	PartTypeHead PartType = "頭部"
	PartTypeRArm PartType = "右腕"
	PartTypeLArm PartType = "左腕"
	PartTypeLegs PartType = "脚部"
)
const (
	CategoryRanged       PartCategory = "射撃"
	CategoryMelee        PartCategory = "格闘"
	CategoryIntervention PartCategory = "介入"
	CategoryNone         PartCategory = "NONE"
)
const (
	TraitAim      Trait = "狙い撃ち"
	TraitStrike   Trait = "殴る"
	TraitBerserk  Trait = "我武者羅"
	TraitShoot    Trait = "撃つ"
	TraitSupport  Trait = "支援"
	TraitObstruct Trait = "妨害"
	TraitNone     Trait = "NONE"
)

const PlayersPerTeam = 3

type BuffType string

const (
	BuffTypeAccuracy BuffType = "Accuracy"
)

type GameData struct {
	Medarots []MedarotData
}

type MedarotData struct {
	ID         string
	Name       string
	IsLeader   bool
	Team       TeamID
	MedalID    string
	HeadID     string
	RightArmID string
	LeftArmID  string
	LegsID     string
	DrawIndex  int
}

// PartDefinition はCSVからロードされるパーツの静的で不変のデータを保持します。
type PartDefinition struct {
	ID         string
	PartName   string
	Type       PartType
	Category   PartCategory
	Trait      Trait
	MaxArmor   int // MaxArmor は定義の一部です
	Power      int
	Accuracy   int
	Charge     int
	Cooldown   int
	Propulsion int
	Mobility   int
	Defense    int
	Stability  int
	WeaponType WeaponType
}

// PartInstanceData (旧Part) は戦闘中のパーツインスタンスの動的な状態を保持します。
type PartInstanceData struct {
	DefinitionID string // PartDefinition を検索するためのID
	CurrentArmor int
	IsBroken     bool
	// このインスタンスに固有の他の一時的なバフ/デバフなどの動的状態はここに記述可能
}

type Medal struct {
	ID          string
	Name        string
	Personality string
	SkillLevel  int
}

const (
	PolicyPreselected        TargetingPolicyType = "Preselected"
	PolicyClosestAtExecution TargetingPolicyType = "ClosestAtExecution"
)

type DebuffType string

const (
	DebuffTypeEvasion        DebuffType = "Evasion"
	DebuffTypeDefense        DebuffType = "Defense"
	DebuffTypeChargeStop     DebuffType = "ChargeStop"     // チャージ一時停止
	DebuffTypeDamageOverTime DebuffType = "DamageOverTime" // チャージ中ダメージ
	DebuffTypeTargetRandom   DebuffType = "TargetRandom"   // ターゲットのランダム化
)

// ActiveStatusEffectData は、エンティティに現在適用されている効果のデータとその残り期間を追跡します。
type ActiveStatusEffectData struct {
	EffectData   interface{} // ChargeStopEffect, DamageOverTimeEffect などのインスタンス
	RemainingDur int
}

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

// ActionTarget はUIが選択したアクションのターゲット情報を保持します。
type ActionTarget struct {
	TargetEntityID donburi.Entity // ターゲットエンティティのID
	Slot           PartSlotKey    // ターゲットパーツのスロット
}
