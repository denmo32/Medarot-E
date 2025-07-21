package ui

import (
	"image"
	"image/color"
	"medarot-ebiten/internal/game"

	// "log"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"

	// "github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yohamta/donburi"
)

type UI struct {
	ebitenui               *ebitenui.UI
	battlefieldWidget      *BattlefieldWidget
	medarotInfoPanels      map[string]*infoPanelUI
	eventChannel           chan game.UIEvent
	config                 *game.Config
	gameDataManager        *game.GameDataManager
	whitePixel             *ebiten.Image
	messageManager         *UIMessageDisplayManager
	actionModalManager     *UIActionModalManager
	targetIndicatorManager *UITargetIndicatorManager
	uiFactory              *UIFactory
	animationDrawer        *UIAnimationDrawer
	MessageManager         *UIMessageDisplayManager
	AnimationDrawer        *UIAnimationDrawer
	BattlefieldWidget      *BattlefieldWidget
}

// SetBattleUIState はUI全体のデータソースを一元的に設定します。
func (u *UI) SetBattleUIState(battleUIState *BattleUIState, config *game.Config, battlefieldRect image.Rectangle) {
	// BattlefieldViewModel を設定
	u.battlefieldWidget.SetViewModel(battleUIState.BattlefieldViewModel)

	// InfoPanels を更新または再構築
	if len(battleUIState.InfoPanels) != len(u.medarotInfoPanels) {
		// パネルの数が変わった場合のみ再構築
		// 既存のパネルをクリア
		mainUIContainer := u.ebitenui.Container.Children()[0].(*widget.Container)
		team1PanelContainer := mainUIContainer.Children()[0].(*widget.Container)
		team2PanelContainer := mainUIContainer.Children()[2].(*widget.Container)

		for _, panel := range u.medarotInfoPanels {
			team1PanelContainer.RemoveChild(panel.rootContainer)
			team2PanelContainer.RemoveChild(panel.rootContainer)
		}
		u.medarotInfoPanels = make(map[string]*infoPanelUI) // マップをクリア

		// 新しいViewModelに基づいてパネルを再生成
		infoPanelVMs := make([]InfoPanelViewModel, 0, len(battleUIState.InfoPanels))
		for _, vm := range battleUIState.InfoPanels {
			infoPanelVMs = append(infoPanelVMs, vm)
		}

		infoPanelResults := CreateInfoPanels(config, u.uiFactory, infoPanelVMs)

		for _, result := range infoPanelResults {
			u.medarotInfoPanels[result.ID] = result.PanelUI
			if result.Team == game.Team1 {
				team1PanelContainer.AddChild(result.PanelUI.rootContainer)
			} else {
				team2PanelContainer.AddChild(result.PanelUI.rootContainer)
			}
		}
	}

	// 各パネルのデータを更新
	for id, vm := range battleUIState.InfoPanels {
		if panel, ok := u.medarotInfoPanels[id]; ok {
			updateSingleInfoPanel(panel, vm, config)
		}
	}
}

// PostEvent はUIイベントをBattleSceneのキューに追加します。
func (u *UI) PostEvent(event game.UIEvent) {
	u.eventChannel <- event
}

// NewUI は新しいUIインスタンスを作成します。
func NewUI(config *game.Config, uiConfig *UIConfig, eventChannel chan game.UIEvent, gameDataManager *game.GameDataManager, animationManager game.AnimationManager) *UI {
	whiteImg := ebiten.NewImage(1, 1)
	whiteImg.Fill(color.White)

	uiFactory := NewUIFactory(uiConfig, gameDataManager.Font, gameDataManager.Messages, gameDataManager) // UIFactoryを初期化

	ui := &UI{
		medarotInfoPanels: make(map[string]*infoPanelUI),
		eventChannel:      eventChannel,
		config:            config,
		gameDataManager:   gameDataManager, // 追加
		whitePixel:        whiteImg,
		uiFactory:         uiFactory,                                                 // 追加
		animationDrawer:   NewUIAnimationDrawer(uiConfig, animationManager, gameDataManager), // UIAnimationDrawerを初期化
	}

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewStackedLayout()),
	)
	mainUIContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{false, true, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(uiConfig.InfoPanel.Padding, 0),
		)),
	)
	rootContainer.AddChild(mainUIContainer)
	team1PanelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(config.UI.InfoPanel.Padding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(config.UI.InfoPanel.BlockWidth), 0),
		),
	)
	mainUIContainer.AddChild(team1PanelContainer)
	ui.battlefieldWidget = NewBattlefieldWidget(&uiConfig)
	ui.battlefieldWidget.Container.GetWidget().LayoutData = widget.GridLayoutData{}
	mainUIContainer.AddChild(ui.battlefieldWidget.Container)
	team2PanelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(config.UI.InfoPanel.Padding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(config.UI.InfoPanel.BlockWidth), 0),
		),
	)
	mainUIContainer.AddChild(team2PanelContainer)
	// InfoPanelsの初期化はSetBattleUIStateで行われるため、ここでは行わない
	// ui.medarotInfoPanelsはSetBattleUIStateで動的に構築される
	ui.messageManager = NewUIMessageDisplayManager(uiFactory, rootContainer)
	ui.ebitenui = &ebitenui.UI{
		Container: rootContainer,
	}
	ui.actionModalManager = NewUIActionModalManager(ui.ebitenui, eventChannel, uiFactory) // uiFactoryを渡す
	ui.targetIndicatorManager = NewUITargetIndicatorManager()
	return ui
}

// IsActionModalVisible はアクションモーダルが表示されているかどうかを返します。
func (u *UI) IsActionModalVisible() bool {
	return u.actionModalManager.IsVisible()
}

// ShowActionModal はアクション選択モーダルを表示します。
func (u *UI) ShowActionModal(vm ActionModalViewModel) {
	u.actionModalManager.ShowActionModal(vm)
}

// HideActionModal はアクション選択モーダルを非表示にします。
func (u *UI) HideActionModal() {
	u.actionModalManager.HideActionModal()
}

// GetActionTargetMap は現在のアクションターゲットマップを返します。
func (u *UI) GetActionTargetMap() map[game.PartSlotKey]game.ActionTarget {
	return u.actionModalManager.GetActionTargetMap()
}

// SetCurrentTarget は現在のターゲットを設定します。
func (u *UI) SetCurrentTarget(entry *donburi.Entry) {
	u.targetIndicatorManager.SetCurrentTarget(entry)
}

// ClearCurrentTarget は現在のターゲットをクリアします。
func (u *UI) ClearCurrentTarget() {
	u.targetIndicatorManager.ClearCurrentTarget()
}

// Update はUIの状態を更新します。
func (u *UI) Update() {
	u.ebitenui.Update()
}

// Draw はUIを描画します。
func (u *UI) Draw(screen *ebiten.Image, tick int) {
	// ターゲットインジケーターの描画に必要な IconViewModel を取得
	var indicatorTargetVM *IconViewModel
	if u.targetIndicatorManager.GetCurrentTarget() != nil && u.battlefieldWidget.viewModel != nil {
		for _, iconVM := range u.battlefieldWidget.viewModel.Icons {
			if iconVM.EntryID == uint32(u.targetIndicatorManager.GetCurrentTarget().Id()) {
				indicatorTargetVM = iconVM
				break
			}
		}
	}

	// BattlefieldWidget の Draw メソッドを先に呼び出す
	u.battlefieldWidget.Draw(screen, indicatorTargetVM, tick)

	// アニメーションの描画
	if u.battlefieldWidget.viewModel != nil {
		u.animationDrawer.Draw(screen, tick, *u.battlefieldWidget.viewModel, u.battlefieldWidget)
	}

	// その後でebitenuiを描画する
	u.ebitenui.Draw(screen)
}

// DrawBackground は背景を描画します。
func (u *UI) DrawBackground(screen *ebiten.Image) {
	u.battlefieldWidget.DrawBackground(screen)
}

// GetBattlefieldWidgetRect はバトルフィールドウィジェットの矩形を返します。
func (u *UI) GetBattlefieldWidgetRect() image.Rectangle {
	return u.battlefieldWidget.Container.GetWidget().Rect
}

// GetRootContainer はルートコンテナを返します。
func (u *UI) GetRootContainer() *widget.Container {
	return u.ebitenui.Container
}

// SetAnimation はアニメーションを設定します。
func (u *UI) SetAnimation(anim *game.ActionAnimationData) {
	u.animationDrawer.animationManager.SetAnimation(anim)
}

// IsAnimationFinished は現在のアニメーションが完了したかどうかを返します。
func (u *UI) IsAnimationFinished(tick int) bool {
	return u.animationDrawer.animationManager.IsAnimationFinished(tick)
}

// ClearAnimation は現在のアニメーションをクリアします。
func (u *UI) ClearAnimation() {
	u.animationDrawer.animationManager.ClearAnimation()
}

// GetCurrentAnimationResult は現在のアニメーションの結果を返します。
func (u *UI) GetCurrentAnimationResult() game.AnimationResultViewModel {
	return u.animationDrawer.animationManager.GetCurrentAnimationResult()
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

type infoPanelUI struct {
	rootContainer *widget.Container
	nameText      *widget.Text
	stateText     *widget.Text
	partSlots     map[game.PartSlotKey]*infoPanelPartUI
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
	ID        string
	Name      string
	Team      game.TeamID
	DrawIndex int
	StateStr  string
	IsLeader  bool
	Parts     map[game.PartSlotKey]PartViewModel
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
	PartCategory    game.PartCategory
	SlotKey         game.PartSlotKey
	IsBroken        bool
	TargetEntry     *donburi.Entry // 射撃などのターゲットが必要な場合
	SelectedPartDef *game.PartDefinition
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
	State         game.StateType
	GaugeProgress float64 // 0.0 to 1.0
	DebugText     string
}

// BattleUIStateComponent holds all the ViewModels for the UI.
var BattleUIStateComponent = donburi.NewComponentType[BattleUIState]()

// BattleUIState is a singleton component that stores UI-specific data (ViewModels).
type BattleUIState struct {
	InfoPanels           map[string]InfoPanelViewModel // Map from Medarot ID to its ViewModel
	BattlefieldViewModel BattlefieldViewModel          // Add BattlefieldViewModel here
}

// UIInterface defines the interface for the game's user interface.
// BattleScene will interact with the UI through this interface.
type UIInterface interface {
	Update()
	Draw(screen *ebiten.Image, tick int)
	DrawBackground(screen *ebiten.Image)
	GetRootContainer() *widget.Container
	SetAnimation(anim *game.ActionAnimationData)
	IsAnimationFinished(tick int) bool
	ClearAnimation()
	GetCurrentAnimationResult() battle.ActionResult
	ShowActionModal(vm ActionModalViewModel)
	HideActionModal()
	SetBattleUIState(battleUIState *BattleUIState, config *UIConfig, battlefieldRect image.Rectangle) // 単一のデータ設定メソッド
	PostEvent(event game.UIEvent)                                                                        // This will be implemented by the concrete UI struct
	IsActionModalVisible() bool
	GetActionTargetMap() map[game.PartSlotKey]game.ActionTarget
	SetCurrentTarget(entry *donburi.Entry)
	ClearCurrentTarget()
	GetBattlefieldWidgetRect() image.Rectangle
}
