package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/yohamta/donburi"
)

type TeamID int
type GameState string
type PartSlotKey string
type PartType string
type PartCategory string
type Trait string
type StateType int
type CustomizeCategory string

const (
	StateTypeIdle StateType = iota
	StateTypeCharging
	StateTypeReady
	StateTypeCooldown
	StateTypeBroken
)
const (
	Team1    TeamID = 0
	Team2    TeamID = 1
	TeamNone TeamID = -1 // Represents no winner or a draw
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
	CategoryMelee   PartCategory = "格闘" // CSVの FIGHT に対応
	CategorySupport PartCategory = "支援"
	CategoryDefense PartCategory = "防御"
	CategoryNone    PartCategory = "NONE" // NONE はそのまま
)
const (
	TraitAim     Trait = "狙い撃ち"
	TraitStrike  Trait = "殴る"
	TraitBerserk Trait = "我武者羅"
	TraitNormal  Trait = "撃つ"
	TraitNone    Trait = "NONE"   // NONE はそのまま
)

const (
	CustomizeCategoryMedal CustomizeCategory = "Medal"
	CustomizeCategoryHead  CustomizeCategory = "Head"
	CustomizeCategoryRArm  CustomizeCategory = "Right Arm"
	CustomizeCategoryLArm  CustomizeCategory = "Left Arm"
	CustomizeCategoryLegs  CustomizeCategory = "Legs"
)

const PlayersPerTeam = 3

// ActionTarget はUIで使うための一時的なターゲット情報
type ActionTarget struct {
	Target *donburi.Entry
	Slot   PartSlotKey
}

type Config struct {
	Balance BalanceConfig
	UI      UIConfig
}

// BalanceConfig 構造体を新しいルールに合わせて拡張
type BalanceConfig struct {
	Time struct {
		PropulsionEffectRate float64
		GameSpeedMultiplier  float64
	}
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
		CriticalMultiplier float64
		MedalSkillFactor   int
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
	Medals   []Medal
	AllParts map[string]*Part
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

type PartData struct {
	ID         string
	Name       string
	Type       PartType
	Category   PartCategory
	Trait      Trait
	Armor      int
	Power      int
	Accuracy   int
	Charge     int
	Cooldown   int
	Propulsion int
	Mobility   int
}

type MedalData struct {
	ID         string
	Name       string
	SkillLevel int
}

type Medarot struct {
	ID                string
	Name              string
	Team              TeamID
	Medal             *Medal
	Parts             map[PartSlotKey]*Part
	IsLeader          bool
	Gauge             float64
	SelectedPartKey   PartSlotKey
	TargetedMedarot   *Medarot
	LastActionLog     string
	IsEvasionDisabled bool
	IsDefenseDisabled bool
	DrawIndex         int
	ProgressCounter   float64
	TotalDuration     float64
}

type Part struct {
	ID         string
	PartName   string
	Type       PartType
	Category   PartCategory
	Trait      Trait
	Armor      int
	MaxArmor   int
	Power      int
	Accuracy   int
	Charge     int
	Cooldown   int
	Propulsion int
	Mobility   int
	Defense    int
	Stability  int
	IsBroken   bool
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
}
