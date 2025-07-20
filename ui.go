package main

import (
	"image"
	"image/color"

	// "log"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"

	// "github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yohamta/donburi"
)

type UI struct {
	ebitenui          *ebitenui.UI
	battlefieldWidget *BattlefieldWidget
	medarotInfoPanels map[string]*infoPanelUI
	// イベント通知用チャネル
	eventChannel chan UIEvent
	// 依存性
	config                 *Config
	gameDataManager        *GameDataManager // 追加
	whitePixel             *ebiten.Image
	messageManager         *UIMessageDisplayManager
	actionModalManager     *UIActionModalManager
	targetIndicatorManager *UITargetIndicatorManager
	uiFactory              *UIFactory         // 追加
	animationDrawer        *UIAnimationDrawer // 新しく追加
}

// SetBattleUIState はUI全体のデータソースを一元的に設定します。
func (u *UI) SetBattleUIState(battleUIState *BattleUIState, config *Config, battlefieldRect image.Rectangle) {
	// BattlefieldViewModel を設定
	u.battlefieldWidget.SetViewModel(battleUIState.BattlefieldViewModel)

	// InfoPanels を更新
	for id, vm := range battleUIState.InfoPanels {
		if panel, ok := u.medarotInfoPanels[id]; ok {
			updateSingleInfoPanel(panel, vm, config)
		}
	}
}

// PostEvent はUIイベントをBattleSceneのキューに追加します。
func (u *UI) PostEvent(event UIEvent) {
	u.eventChannel <- event
}

// NewUI は新しいUIインスタンスを作成します。
func NewUI(world donburi.World, config *Config, eventChannel chan UIEvent, gameDataManager *GameDataManager, animationManager *BattleAnimationManager) *UI {
	whiteImg := ebiten.NewImage(1, 1)
	whiteImg.Fill(color.White)

	uiFactory := NewUIFactory(config, gameDataManager.Font, gameDataManager.Messages, gameDataManager) // UIFactoryを初期化

	ui := &UI{
		medarotInfoPanels: make(map[string]*infoPanelUI),
		eventChannel:      eventChannel,
		config:            config,
		gameDataManager:   gameDataManager, // 追加
		whitePixel:        whiteImg,
		uiFactory:         uiFactory,                                                       // 追加
		animationDrawer:   NewUIAnimationDrawer(config, animationManager, gameDataManager), // UIAnimationDrawerを初期化
	}

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewStackedLayout()),
	)
	mainUIContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{false, true, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(config.UI.InfoPanel.Padding, 0),
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
	ui.battlefieldWidget = NewBattlefieldWidget(config)
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
	infoPanelResults := CreateInfoPanels(world, config, uiFactory)
	for _, result := range infoPanelResults {
		ui.medarotInfoPanels[result.ID] = result.PanelUI
		if result.Team == Team1 {
			team1PanelContainer.AddChild(result.PanelUI.rootContainer)
		} else {
			team2PanelContainer.AddChild(result.PanelUI.rootContainer)
		}
	}
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
func (u *UI) GetActionTargetMap() map[PartSlotKey]ActionTarget {
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
func (u *UI) SetAnimation(anim *ActionAnimationData) {
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
func (u *UI) GetCurrentAnimationResult() ActionResult {
	return u.animationDrawer.animationManager.currentAnimation.Result
}
