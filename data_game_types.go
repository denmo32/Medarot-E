package main

import (
	"strconv"
	"strings"

	"github.com/ebitenui/ebitenui/widget"
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
type WeaponType string
type CustomizeCategory string

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

type BuffType string

const (
	BuffTypeAccuracy BuffType = "Accuracy"
)

// ActionTarget はUIで使用するための一時的なターゲット情報です。
type ActionTarget struct {
	Target *donburi.Entry
	Slot   PartSlotKey
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
	PolicyPreselected TargetingPolicyType = "Preselected"
	PolicyClosestAtExecution TargetingPolicyType = "ClosestAtExecution"
)

type Config struct {
	Balance BalanceConfig
	UI      UIConfig
}

type SharedResources struct {
	GameData        *GameData
	Config          Config
	Font            text.Face
	GameDataManager *GameDataManager
	ButtonImage     *widget.ButtonImage
}

