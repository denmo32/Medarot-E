package core

// --- Enums and Constants ---

type TeamID int
type GameState string
type PartSlotKey string
type PartType string
type StateType string
type PartCategory string
type Trait string
type WeaponType string
type TargetingPolicyType string
type BuffType string
type DebuffType string
type PartParameter string
type CustomizeCategory string

const (
	CustomizeCategoryMedal CustomizeCategory = "Medal"
	CustomizeCategoryHead  CustomizeCategory = "Head"
	CustomizeCategoryRArm  CustomizeCategory = "Right Arm"
	CustomizeCategoryLArm  CustomizeCategory = "Left Arm"
	CustomizeCategoryLegs  CustomizeCategory = "Legs"
)

const (
	Team1    TeamID = 0
	Team2    TeamID = 1
	TeamNone TeamID = -1
)

const (
	StateGaugeProgress      GameState = "GaugeProgress"
	StatePlayerActionSelect GameState = "PlayerActionSelect"
	StateActionExecution    GameState = "ActionExecution"
	StateAnimatingAction    GameState = "AnimatingAction"
	StatePostAction         GameState = "PostAction"
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

const (
	PolicyPreselected        TargetingPolicyType = "Preselected"
	PolicyClosestAtExecution TargetingPolicyType = "ClosestAtExecution"
)

const (
	BuffTypeAccuracy BuffType = "Accuracy"
)

const (
	DebuffTypeEvasion        DebuffType = "Evasion"
	DebuffTypeDefense        DebuffType = "Defense"
	DebuffTypeChargeStop     DebuffType = "ChargeStop"
	DebuffTypeDamageOverTime DebuffType = "DamageOverTime"
	DebuffTypeTargetRandom   DebuffType = "TargetRandom"
)

const (
	Power      PartParameter = "Power"
	Accuracy   PartParameter = "Accuracy"
	Mobility   PartParameter = "Mobility"
	Propulsion PartParameter = "Propulsion"
	Stability  PartParameter = "Stability"
	Defense    PartParameter = "Defense"
)

const PlayersPerTeam = 3

// --- Data Structures ---

// GameEndResult はゲーム終了チェックの結果を保持します。
type GameEndResult struct {
	IsGameOver bool
	Winner     TeamID
	Message    string
}

// MessageTemplate defines the structure for a single message in the JSON file.
type MessageTemplate struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

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

type PartDefinition struct {
	ID         string
	PartName   string
	Type       PartType
	Category   PartCategory
	Trait      Trait
	MaxArmor   int
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

type PartInstanceData struct {
	DefinitionID string
	CurrentArmor int
	IsBroken     bool
}

type Medal struct {
	ID          string
	Name        string
	Personality string
	SkillLevel  int
}

// --- Component Data Structs (donburi-independent) ---

type GameStateData struct {
	CurrentState GameState
}

type ChargeStopEffectData struct {
	DurationTurns int
}

type DamageOverTimeEffectData struct {
	DamagePerTurn int
	DurationTurns int
}

type TargetRandomEffectData struct {
	DurationTurns int
}

type EvasionDebuffEffectData struct {
	Multiplier float64
}

type DefenseDebuffEffectData struct {
	Multiplier float64
}

type Settings struct {
	ID        string
	Name      string
	Team      TeamID
	IsLeader  bool
	DrawIndex int
}

type PartsComponentData struct {
	Map map[PartSlotKey]*PartInstanceData
}

type State struct {
	CurrentState StateType
}

type Gauge struct {
	ProgressCounter float64
	TotalDuration   float64
	CurrentGauge    float64
}

type ActionIntent struct {
	SelectedPartKey PartSlotKey
	PendingEffects  []interface{}
}

type Log struct {
	LastActionLog string
}

type PlayerControl struct{}

// ActiveStatusEffectData は、エンティティに現在適用されている効果のデータとその残り期間を追跡します。
type ActiveStatusEffectData struct {
	EffectData   interface{}
	RemainingDur int
}

// ActiveEffects は、エンティティに現在適用されている効果のリストを保持します。
type ActiveEffects struct {
	Effects []*ActiveStatusEffectData
}

// --- Formula-related Structs ---

type BonusTerm struct {
	SourceParam PartParameter
	Multiplier  float64
}

type DebuffEffect struct {
	Type       DebuffType
	Multiplier float64
}

type ActionFormula struct {
	ID                 string
	SuccessRateBonuses []BonusTerm
	PowerBonuses       []BonusTerm
	CriticalRateBonus  float64
	UserDebuffs        []DebuffEffect
}

type ActionFormulaConfig struct {
	SuccessRateBonuses []BonusTerm
	PowerBonuses       []BonusTerm
	CriticalRateBonus  float64
	UserDebuffs        []DebuffEffect
}
