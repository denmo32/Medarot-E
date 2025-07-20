package main

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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
type WeaponType string // 新しく追加
type CustomizeCategory string
type BuffType string // 新しく追加

// TargetingPolicyType はターゲット決定方針を示す型です。
type TargetingPolicyType string

// --- 計算式データ構造 ---

// PartParameter はパーツのどの数値を参照するかを示す型です
type PartParameter string

const (
	Power      PartParameter = "Power"
	Accuracy   PartParameter = "Accuracy"
	Mobility   PartParameter = "Mobility"
	Propulsion PartParameter = "Propulsion"
	Stability  PartParameter = "Stability"
	Defense    PartParameter = "Defense"
)

type DebuffType string

const (
	DebuffTypeEvasion        DebuffType = "Evasion"
	DebuffTypeDefense        DebuffType = "Defense"
	DebuffTypeChargeStop     DebuffType = "ChargeStop"     // チャージ一時停止
	DebuffTypeDamageOverTime DebuffType = "DamageOverTime" // チャージ中ダメージ
	DebuffTypeTargetRandom   DebuffType = "TargetRandom"   // ターゲットのランダム化
)

const (
	BuffTypeAccuracy BuffType = "Accuracy"
)

// BonusTerm は計算式のボーナス項を定義します
type BonusTerm struct {
	SourceParam PartParameter // どのパラメータを参照するか
	Multiplier  float64       // 乗数
}

// DebuffEffect は発生するデバフ効果を定義します
type DebuffEffect struct {
	Type       DebuffType // デバフの種類
	Multiplier float64    // 効果量（乗数）
}

// ActionFormula はアクションの計算ルール全体を定義します
type ActionFormula struct {
	ID                 string
	SuccessRateBonuses []BonusTerm    // 成功度へのボーナスリスト
	PowerBonuses       []BonusTerm    // 威力度へのボーナスリスト
	CriticalRateBonus  float64        // クリティカル率へのボーナス
	UserDebuffs        []DebuffEffect // チャージ中に自身にかかるデバフのリスト
}

// --- ここまで ---

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

// GetStateDisplayName は StateType に対応する日本語の表示名を返します。
func GetStateDisplayName(state StateType) string {
	switch state {
	case StateIdle:
		return "待機"
	case StateCharging:
		return "チャージ中"
	case StateReady:
		return "実行準備"
	case StateCooldown:
		return "クールダウン"
	case StateBroken:
		return "機能停止"
	default:
		return "不明"
	}
}

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
	CustomizeCategoryMedal CustomizeCategory = "Medal"
	CustomizeCategoryHead  CustomizeCategory = "Head"
	CustomizeCategoryRArm  CustomizeCategory = "Right Arm"
	CustomizeCategoryLArm  CustomizeCategory = "Left Arm"
	CustomizeCategoryLegs  CustomizeCategory = "Legs"
)

const PlayersPerTeam = 3

// ActionTarget はUIで使用するための一時的なターゲット情報です。
type ActionTarget struct {
	Target *donburi.Entry
	Slot   PartSlotKey
}

type Config struct {
	Balance BalanceConfig
	UI      UIConfig
}

// ActionFormulaConfig は types.go の ActionFormula と似ていますが、設定ファイル用です
type ActionFormulaConfig struct {
	SuccessRateBonuses []BonusTerm
	PowerBonuses       []BonusTerm
	CriticalRateBonus  float64
	UserDebuffs        []DebuffEffect
}

// BalanceConfig 構造体を新しいルールに合わせて拡張します。
type BalanceConfig struct {
	Time struct {
		PropulsionEffectRate float64
		GameSpeedMultiplier  float64
	}
	HPAnimationSpeed float64 // HPゲージアニメーション速度 (1フレームあたりのHP変化量)
	Factors          struct {
		AccuracyStabilityFactor      float64
		EvasionStabilityFactor       float64
		DefenseStabilityFactor       float64
		PowerStabilityFactor         float64
		MeleeAccuracyMobilityFactor  float64
		BerserkPowerPropulsionFactor float64
	}
	Effects struct {
		Melee struct {
			DefenseRateDebuff float64
			CriticalRateBonus int
		}
		Berserk struct {
			DefenseRateDebuff float64
			EvasionRateDebuff float64
		}
		Shoot struct{}
		Aim   struct {
			EvasionRateDebuff float64
			CriticalRateBonus int
		}
	}
	Damage struct {
		CriticalMultiplier     float64
		MedalSkillFactor       int
		DamageAdjustmentFactor float64
		Critical               struct {
			BaseChance        float64
			SuccessRateFactor float64
			MinChance         float64
			MaxChance         float64
		}
	}
	Hit struct {
		BaseChance float64
		MinChance  float64
		MaxChance  float64
	}
	Defense struct {
		BaseChance float64
		MinChance  float64
		MaxChance  float64
	}
	Formulas map[Trait]ActionFormulaConfig // 新しく追加
}

type SharedResources struct {
	GameData        *GameData
	Config          Config
	Font            text.Face
	GameDataManager *GameDataManager // 追加
	ButtonImage     *widget.ButtonImage
}

type UIConfig struct {
	Screen struct {
		Width  int
		Height int
	}
	Battlefield struct {
		Rect                   *widget.Container
		Height                 float32
		Team1HomeX             float32
		Team2HomeX             float32
		Team1ExecutionLineX    float32
		Team2ExecutionLineX    float32
		IconRadius             float32
		HomeMarkerRadius       float32
		LineWidth              float32
		MedarotVerticalSpacing float32
		TargetIndicator        struct {
			Width  float32
			Height float32
		}
	}
	InfoPanel struct {
		Padding           int
		BlockWidth        float32
		BlockHeight       float32
		PartHPGaugeWidth  float32
		PartHPGaugeHeight float32
	}
	ActionModal struct {
		ButtonWidth   float32
		ButtonHeight  float32
		ButtonSpacing int
	}
	Colors struct {
		White      color.Color
		Red        color.Color
		Blue       color.Color
		Yellow     color.Color
		Gray       color.Color
		Team1      color.Color
		Team2      color.Color
		Leader     color.Color
		Broken     color.Color
		HP         color.Color
		HPCritical color.Color
		Background color.Color
	}
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
	WeaponType WeaponType  // CSVから必要であればここに追加
	PartSlot   PartSlotKey // このパーツが通常装着されるスロット
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

type infoPanelUI struct {
	rootContainer *widget.Container
	nameText      *widget.Text
	stateText     *widget.Text
	partSlots     map[PartSlotKey]*infoPanelPartUI
}

type infoPanelPartUI struct {
	partNameText *widget.Text
	hpText       *widget.Text
	hpBar        *widget.ProgressBar
	displayedHP  float64 // 現在表示されているHP
	targetHP     float64 // 目標とするHP
}

// --- ViewModels ---

// ActionResult はアクション実行の詳細な結果を保持します。
type ActionResult struct {
	ActingEntry       *donburi.Entry
	TargetEntry       *donburi.Entry
	TargetPartSlot    PartSlotKey // ターゲットのパーツスロット
	ActionDidHit      bool        // 命中したかどうか
	IsCritical        bool        // クリティカルだったか
	OriginalDamage    int         // 元のダメージ量
	DamageDealt       int         // 実際に与えたダメージ
	TargetPartBroken  bool        // ターゲットパーツが破壊されたか
	ActionIsDefended  bool        // 攻撃が防御されたか
	ActualHitPartSlot PartSlotKey // 実際にヒットしたパーツのスロット

	// 新しいメッセージ形式のための追加フィールド
	AttackerName      string
	DefenderName      string
	ActionName        string // e.g., "パーツ名"
	ActionTrait       Trait  // e.g., "撃つ", "狙い撃ち" (Trait)
	WeaponType        WeaponType
	ActionCategory    PartCategory
	TargetPartType    string // e.g., "頭部", "脚部"
	DefendingPartType string // e.g., "頭部", "脚部"

	AppliedEffects []StatusEffect // アクションによって適用されるステータス効果
}

// --- Status Effect System ---

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

func (e *ChargeStopEffect) Apply(world donburi.World, target *donburi.Entry) {
	// この効果の適用ロジックはChargeInitiationSystemなどで処理される
}
func (e *ChargeStopEffect) Remove(world donburi.World, target *donburi.Entry) {
	// この効果の解除ロジックはChargeInitiationSystemなどで処理される
}
func (e *ChargeStopEffect) Description() string { return "チャージ停止" }
func (e *ChargeStopEffect) Duration() int       { return e.DurationTurns }
func (e *ChargeStopEffect) Type() DebuffType    { return DebuffTypeChargeStop }

// DamageOverTimeEffect は継続ダメージを与えるデバフです。
type DamageOverTimeEffect struct {
	DamagePerTurn int
	DurationTurns int
}

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
type TargetRandomEffect struct {
	DurationTurns int
}

func (e *TargetRandomEffect) Apply(world donburi.World, target *donburi.Entry) {
	// この効果の適用ロジックはBattleTargetSelectorなどで処理される
}
func (e *TargetRandomEffect) Remove(world donburi.World, target *donburi.Entry) {
	// この効果の解除ロジックはBattleTargetSelectorなどで処理される
}
func (e *TargetRandomEffect) Description() string { return "ターゲットランダム化" }
func (e *TargetRandomEffect) Duration() int       { return e.DurationTurns }
func (e *TargetRandomEffect) Type() DebuffType    { return DebuffTypeTargetRandom }

// --- End Status Effect System ---

// ActionAnimationData はアニメーションの再生に必要なデータを保持します。
type ActionAnimationData struct {
	Result    ActionResult
	StartTime int
}

// InfoPanelViewModel は、単一の情報パネルUIが必要とするすべてのデータを保持します。
type InfoPanelViewModel struct {
	MedarotName string
	StateStr    string
	IsLeader    bool
	Parts       map[PartSlotKey]PartViewModel
}

// PartViewModel は、単一のパーツUIが必要とするデータを保持します。
type PartViewModel struct {
	PartName     string
	CurrentArmor int
	MaxArmor     int
	IsBroken     bool
}

// ActionModalButtonViewModel は、アクション選択モーダルのボタン一つ分のデータを保持します。
type ActionModalButtonViewModel struct {
	PartName        string
	PartCategory    PartCategory
	SlotKey         PartSlotKey
	IsBroken        bool
	TargetEntry     *donburi.Entry // 射撃などのターゲットが必要な場合
	SelectedPartDef *PartDefinition
}

// ActionModalViewModel は、アクション選択モーダル全体の表示に必要なデータを保持します。
type ActionModalViewModel struct {
	ActingMedarotName string
	ActingEntry       *donburi.Entry // イベント発行時に必要
	Buttons           []ActionModalButtonViewModel
}

// BattlefieldViewModel は、バトルフィールド全体の描画に必要なデータを保持します。
type BattlefieldViewModel struct {
	Icons     []*IconViewModel
	DebugMode bool
}

// IconViewModel は、個々のメダロットアイコンの描画に必要なデータを保持します。
type IconViewModel struct {
	EntryID       uint32 // 元のdonburi.Entryを特定するためのID
	X, Y          float32
	Color         color.Color
	IsLeader      bool
	State         StateType
	GaugeProgress float64 // 0.0 to 1.0
	DebugText     string
}

// GameEvent は、ゲームロジックから発行されるすべてのイベントを示すマーカーインターフェースです。
type GameEvent interface {
	isGameEvent()
}

// PlayerActionRequiredGameEvent は、プレイヤーの行動選択が必要になったことを示すイベントです。
type PlayerActionRequiredGameEvent struct{}

func (e PlayerActionRequiredGameEvent) isGameEvent() {}

// ActionAnimationStartedGameEvent は、アクションアニメーションが開始されたことを示すイベントです。
type ActionAnimationStartedGameEvent struct {
	AnimationData ActionAnimationData
}

func (e ActionAnimationStartedGameEvent) isGameEvent() {}

// ActionAnimationFinishedGameEvent は、アクションアニメーションが終了したことを示すイベントです。
type ActionAnimationFinishedGameEvent struct {
	Result      ActionResult
	ActingEntry *donburi.Entry // クールダウン開始のために追加
}

func (e ActionAnimationFinishedGameEvent) isGameEvent() {}

// MessageDisplayRequestGameEvent は、メッセージ表示が必要になったことを示すイベントです。
type MessageDisplayRequestGameEvent struct {
	Messages []string
	Callback func()
}

func (e MessageDisplayRequestGameEvent) isGameEvent() {}

// MessageDisplayFinishedGameEvent は、メッセージ表示が終了したことを示すイベントです。
type MessageDisplayFinishedGameEvent struct{}

func (e MessageDisplayFinishedGameEvent) isGameEvent() {}

// GameOverGameEvent は、ゲームオーバーになったことを示すイベントです。
type GameOverGameEvent struct {
	Winner TeamID
}

func (e GameOverGameEvent) isGameEvent() {}

// HideActionModalGameEvent は、アクションモーダルを隠す必要があることを示すイベントです。
type HideActionModalGameEvent struct{}

func (e HideActionModalGameEvent) isGameEvent() {}

// ShowActionModalGameEvent は、アクションモーダルを表示する必要があることを示すイベントです。
type ShowActionModalGameEvent struct {
	ViewModel ActionModalViewModel
}

func (e ShowActionModalGameEvent) isGameEvent() {}

// ClearAnimationGameEvent は、アニメーションをクリアする必要があることを示すイベントです。
type ClearAnimationGameEvent struct{}

func (e ClearAnimationGameEvent) isGameEvent() {}

// ClearCurrentTargetGameEvent は、現在のターゲットをクリアする必要があることを示すイベントです。
type ClearCurrentTargetGameEvent struct{}

func (e ClearCurrentTargetGameEvent) isGameEvent() {}

// UIEvent は、UIから発行されるすべてのイベントを示すマーカーインターフェースです。
type UIEvent interface {
	isUIEvent()
}

// PlayerActionSelectedEvent は、プレイヤーが使用するパーツを選択したときに発行されます。
type PlayerActionSelectedEvent struct {
	ActingEntry     *donburi.Entry
	SelectedPartDef *PartDefinition
	SelectedSlotKey PartSlotKey
}

func (e PlayerActionSelectedEvent) isUIEvent() {}

// GameActionRequestEvent は、ゲームロジック層に行動の実行を要求するために発行されます。
type GameActionRequestEvent struct {
	ActingEntry     *donburi.Entry
	SelectedPartKey PartSlotKey
	TargetEntry     *donburi.Entry
	TargetPartSlot  PartSlotKey
}

func (e GameActionRequestEvent) isUIEvent() {}

// PlayerActionCancelEvent は、プレイヤーが行動選択をキャンセルしたときに発行されます。
type PlayerActionCancelEvent struct {
	ActingEntry *donburi.Entry
}

func (e PlayerActionCancelEvent) isUIEvent() {}

// SetCurrentTargetEvent は、UIがターゲットエンティティを設定するよう要求するときに発行されます。
type SetCurrentTargetEvent struct {
	Target *donburi.Entry
}

func (e SetCurrentTargetEvent) isUIEvent() {}

// ClearCurrentTargetEvent は、UIが現在のターゲットをクリアするよう要求するときに発行されます。
type ClearCurrentTargetEvent struct{}

func (e ClearCurrentTargetEvent) isUIEvent() {}

// ShowActionModalUIEvent は、アクションモーダルを表示するUIイベントです。
type ShowActionModalUIEvent struct {
	ViewModel ActionModalViewModel
}

func (e ShowActionModalUIEvent) isUIEvent() {}

// HideActionModalUIEvent は、アクションモーダルを隠すUIイベントです。
type HideActionModalUIEvent struct{}

func (e HideActionModalUIEvent) isUIEvent() {}

// SetAnimationUIEvent は、アニメーションを設定するUIイベントです。
type SetAnimationUIEvent struct {
	AnimationData ActionAnimationData
}

func (e SetAnimationUIEvent) isUIEvent() {}

// ClearAnimationUIEvent は、アニメーションをクリアするUIイベントです。
type ClearAnimationUIEvent struct{}

func (e ClearAnimationUIEvent) isUIEvent() {}

// ClearCurrentTargetUIEvent は、現在のターゲットをクリアするUIイベントです。
type ClearCurrentTargetUIEvent struct{}

func (e ClearCurrentTargetUIEvent) isUIEvent() {}

// MessageDisplayRequestUIEvent は、メッセージ表示を要求するUIイベントです。
type MessageDisplayRequestUIEvent struct {
	Messages []string
	Callback func()
}

func (e MessageDisplayRequestUIEvent) isUIEvent() {}

// UIInterface defines the interface for the game's user interface.
// BattleScene will interact with the UI through this interface.
type UIInterface interface {
	Update()
	Draw(screen *ebiten.Image, tick int)
	DrawBackground(screen *ebiten.Image)
	GetRootContainer() *widget.Container
	SetAnimation(anim *ActionAnimationData)
	IsAnimationFinished(tick int) bool
	ClearAnimation()
	GetCurrentAnimationResult() ActionResult
	ShowActionModal(vm ActionModalViewModel)
	HideActionModal()
	SetBattlefieldViewModel(vm BattlefieldViewModel)
	UpdateInfoPanels(world donburi.World, config *Config, battleLogic *BattleLogic)
	PostEvent(event UIEvent) // This will be implemented by the concrete UI struct
	IsActionModalVisible() bool
	GetActionTargetMap() map[PartSlotKey]ActionTarget
	SetCurrentTarget(entry *donburi.Entry)
	ClearCurrentTarget()
	GetBattlefieldWidgetRect() image.Rectangle
}

// parseInt は文字列をintに変換します。変換できない場合はdefaultValueを返します。
func parseInt(s string, defaultValue int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return i
}

// parseBool は文字列をboolに変換します。"true" (大文字小文字を区別しない) の場合のみtrueを返します。
func parseBool(s string) bool {
	return strings.ToLower(strings.TrimSpace(s)) == "true"
}

const (
	// PolicyPreselected は、行動選択時にターゲットが事前に選択されることを示します。
	PolicyPreselected TargetingPolicyType = "Preselected"
	// PolicyClosestAtExecution は、行動実行時に最も近い敵をターゲットとすることを示します。
	PolicyClosestAtExecution TargetingPolicyType = "ClosestAtExecution"
)
