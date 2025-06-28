package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
)

// ... (TeamID, MedarotState, GameState, etc. は変更なし) ...
type TeamID int
type MedarotState string
type GameState string
type PartSlotKey string
type PartType string
type PartCategory string
type Trait string

const (
	Team1 TeamID = 0
	Team2 TeamID = 1
)
const (
	StateIdle     MedarotState = "待機"
	StateCharging MedarotState = "チャージ中"
	StateReady    MedarotState = "実行準備"
	StateCooldown MedarotState = "クールダウン"
	StateBroken   MedarotState = "機能停止"
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
	PartTypeHead PartType = "HEAD"
	PartTypeRArm PartType = "R_ARM"
	PartTypeLArm PartType = "L_ARM"
	PartTypeLegs PartType = "LEG"
)
const (
	CategoryShoot PartCategory = "SHOOT"
	CategoryMelee PartCategory = "FIGHT"
	CategoryNone  PartCategory = "NONE"
)
const (
	TraitAim     Trait = "AIM"
	TraitStrike  Trait = "STRIKE"
	TraitBerserk Trait = "BERSERK"
	TraitNormal  Trait = "NORMAL"
	TraitNone    Trait = "NONE"
)
const PlayersPerTeam = 3

type Config struct {
	Balance BalanceConfig
	UI      UIConfig
}

type BalanceConfig struct {
	Time struct {
		PropulsionEffectRate float64
		// [REMOVED] 古いフィールドを削除
		// OverallTimeDivisor   float64
		// [NEW] 新しいフィールドを追加
		GameSpeedMultiplier float64
	}
	Hit struct {
		BaseChance         int
		TraitAimBonus      int
		TraitStrikeBonus   int
		TraitBerserkDebuff int
	}
	Damage struct {
		CriticalMultiplier float64
		MedalSkillFactor   int
	}
}

// ... (UIConfig, GameData, etc. は変更なし) ...
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
	State             MedarotState
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
	
	IsBroken   bool
}
type Medal struct {
	ID         string
	Name       string
	SkillLevel int
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

