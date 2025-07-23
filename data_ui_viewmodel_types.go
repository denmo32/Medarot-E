package main

import (
	"image"
	"image/color"

	ebitenuiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/yohamta/donburi"
)

type CustomizeCategory string

const (
	CustomizeCategoryMedal CustomizeCategory = "Medal"
	CustomizeCategoryHead  CustomizeCategory = "Head"
	CustomizeCategoryRArm  CustomizeCategory = "Right Arm"
	CustomizeCategoryLArm  CustomizeCategory = "Left Arm"
	CustomizeCategoryLegs  CustomizeCategory = "Legs"
)

// ActionTarget はUIで使用するための一時的なターゲット情報です。
type ActionTarget struct {
	Target *donburi.Entry
	Slot   PartSlotKey
}

// --- ViewModels ---

// InfoPanelViewModel は、単一の情報パネルUIが必要とするすべてのデータを保持します。
type InfoPanelViewModel struct {
	ID        string
	Name      string
	Team      TeamID
	DrawIndex int
	StateStr  string
	IsLeader  bool
	Parts     map[PartSlotKey]PartViewModel
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
	TargetPartSlot  PartSlotKey    // 追加
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

// BattlefieldWidget はバトルフィールドの描画に必要なデータを保持します。
type BattlefieldWidget struct {
	*widget.Container
	config          *Config
	whitePixel      *ebiten.Image
	viewModel       *BattlefieldViewModel
	backgroundImage *ebiten.Image
}

// CustomIconWidget は個々のメダロットアイコンの描画に必要なデータを保持します。
type CustomIconWidget struct {
	viewModel *IconViewModel
	config    *Config
	rect      image.Rectangle
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

// InfoPanelCreationResult は生成された情報パネルとそのチーム情報を持つ構造体です。
type InfoPanelCreationResult struct {
	PanelUI *infoPanelUI
	Team    TeamID
	ID      string
}

// PanelOptions は、汎用パネルを作成するための設定を保持します。
type PanelOptions struct {
	PanelWidth      int
	PanelHeight     int
	Title           string
	Padding         widget.Insets
	Spacing         int
	BackgroundColor color.Color
	BackgroundImage *ebitenuiimage.NineSlice
	TitleColor      color.Color
	TitleFont       text.Face
	BorderColor     color.Color
	BorderThickness float32
}

// BattleUIStateComponent holds all the ViewModels for the UI.
var BattleUIStateComponent = donburi.NewComponentType[BattleUIState]()

// BattleUIState is a singleton component that stores UI-specific data (ViewModels).
type BattleUIState struct {
	InfoPanels           map[string]InfoPanelViewModel // Map from Medarot ID to its ViewModel
	BattlefieldViewModel BattlefieldViewModel          // Add BattlefieldViewModel here
}
