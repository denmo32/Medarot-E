package main

import (
	"image"
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

type TeamID int
type GameState string
type PartSlotKey string
type PartType string
type PartCategory string
type Trait string
type CustomizeCategory string

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

// DebuffType はデバフの種類を示す型です
type DebuffType string

const (
	DebuffTypeEvasion DebuffType = "Evasion"
	DebuffTypeDefense DebuffType = "Defense"
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
	StateMessage            GameState = "Message"
	StateGameOver           GameState = "GameOver"
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
	CategoryShoot   PartCategory = "射撃"
	CategoryMelee   PartCategory = "格闘" // CSVの FIGHT に対応します
	CategorySupport PartCategory = "支援"
	CategoryDefense PartCategory = "防御"
	CategoryNone    PartCategory = "NONE" // NONE はそのままです
)
const (
	TraitAim     Trait = "狙い撃ち"
	TraitStrike  Trait = "殴る"
	TraitBerserk Trait = "我武者羅"
	TraitNormal  Trait = "撃つ"
	TraitNone    Trait = "NONE" // NONE はそのままです
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

// BalanceConfig 構造体を新しいルールに合わせて拡張します。
type BalanceConfig struct {
	Time struct {
		PropulsionEffectRate float64
		GameSpeedMultiplier  float64
	}
	HPAnimationSpeed float64 // HPゲージアニメーション速度 (1フレームあたりのHP変化量)
	Factors struct {
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
	Medarots []MedarotData // ここにはメダロットのロードアウトのみが残ります。この構造体も廃止可能です
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
	// WeaponType string // CSVから必要であればここに追加
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

// BattlefieldViewModel は、バトルフィールド全体の描画に必要なデータを保持します。
type BattlefieldViewModel struct {
	Icons []*IconViewModel
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

// UIInterface defines the interface for the game's user interface.
// BattleScene will interact with the UI through this interface.
type UIInterface interface {
	Update()
	Draw(screen *ebiten.Image)
	ShowActionModal(actingEntry *donburi.Entry)
	HideActionModal()
	ShowMessageWindow(message string)
	HideMessageWindow()
	SetBattlefieldViewModel(vm BattlefieldViewModel)
	UpdateInfoPanels(world donburi.World, config *Config)
	PostEvent(event UIEvent) // This will be implemented by the concrete UI struct
	IsActionModalVisible() bool
	GetActionTargetMap() map[PartSlotKey]ActionTarget
	SetCurrentTarget(entry *donburi.Entry)
	ClearCurrentTarget()
	GetBattlefieldWidgetRect() image.Rectangle
}