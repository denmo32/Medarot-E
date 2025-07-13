package main

import (
	"image"
	"image/color"
	"log"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"

	// "github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yohamta/donburi"
)

func (u *UI) IsActionModalVisible() bool {
	return u.isActionModalVisible
}

func (u *UI) SetBattlefieldViewModel(vm BattlefieldViewModel) {
	u.battlefieldWidget.SetViewModel(vm)
}

type UI struct {
	ebitenui    *ebitenui.UI
	actionModal widget.PreferredSizeLocateableWidget

	battlefieldWidget *BattlefieldWidget
	medarotInfoPanels map[string]*infoPanelUI
	actionTargetMap   map[PartSlotKey]ActionTarget
	// UIの状態
	playerMedarotToAct   *donburi.Entry // 現在アクション選択中のプレイヤーメダロット
	currentTarget        *donburi.Entry // 現在ターゲットとして表示されているエンティティ
	isActionModalVisible bool           // アクションモーダルが表示されているか
	// イベント通知用チャネル
	eventChannel chan UIEvent
	// 依存性
	config          *Config
	whitePixel      *ebiten.Image
	animationDrawer *UIAnimationDrawer // 新しく追加
	messageManager  *UIMessageDisplayManager
}

// PostEvent はUIイベントをBattleSceneのキューに追加します。
func (u *UI) PostEvent(event UIEvent) {
	u.eventChannel <- event
}

// NewUI は新しいUIインスタンスを作成します。
func NewUI(world donburi.World, config *Config, eventChannel chan UIEvent) *UI {
	whiteImg := ebiten.NewImage(1, 1)
	whiteImg.Fill(color.White)
	animationManager := NewBattleAnimationManager(config)
	ui := &UI{
		medarotInfoPanels:    make(map[string]*infoPanelUI),
		actionTargetMap:      make(map[PartSlotKey]ActionTarget),
		isActionModalVisible: false,
		eventChannel:         eventChannel,
		config:               config,
		whitePixel:           whiteImg,
		animationDrawer:      NewUIAnimationDrawer(config, animationManager), // UIAnimationDrawerを初期化
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
	infoPanelResults := CreateInfoPanels(world, config, GlobalGameDataManager.Font)
	for _, result := range infoPanelResults {
		ui.medarotInfoPanels[result.ID] = result.PanelUI
		if result.Team == Team1 {
			team1PanelContainer.AddChild(result.PanelUI.rootContainer)
		} else {
			team2PanelContainer.AddChild(result.PanelUI.rootContainer)
		}
	}
	ui.messageManager = NewUIMessageDisplayManager(config, GlobalGameDataManager.Font, rootContainer)
	ui.ebitenui = &ebitenui.UI{
		Container: rootContainer,
	}
	return ui
}

// ShowActionModal はアクション選択モーダルを表示します。
func (u *UI) ShowActionModal(actingEntry *donburi.Entry, actionTargetMap map[PartSlotKey]ActionTarget) {
	if u.isActionModalVisible {
		u.HideActionModal()
	}
	u.playerMedarotToAct = actingEntry
	u.isActionModalVisible = true
	u.actionTargetMap = actionTargetMap // Set the pre-calculated map

	modal := createActionModalUI(actingEntry, u.config, u.actionTargetMap, u.eventChannel, GlobalGameDataManager.Font)
	u.actionModal = modal
	u.ebitenui.Container.AddChild(u.actionModal)
	log.Println("アクションモーダルを表示しました。")
}

// HideActionModal はアクション選択モーダルを非表示にします。
func (u *UI) HideActionModal() {
	if !u.isActionModalVisible {
		return
	}
	if u.actionModal != nil {
		u.ebitenui.Container.RemoveChild(u.actionModal)
		u.actionModal = nil
	}
	u.playerMedarotToAct = nil
	u.currentTarget = nil
	u.isActionModalVisible = false
	u.actionTargetMap = make(map[PartSlotKey]ActionTarget) // ターゲットマップをクリア
	log.Println("アクションモーダルを非表示にしました。")
}

// ShowMessageWindow はメッセージウィンドウを表示します。

func (u *UI) UpdateInfoPanels(world donburi.World, config *Config) {
	updateAllInfoPanels(world, config, u.medarotInfoPanels)
}

func (u *UI) GetActionTargetMap() map[PartSlotKey]ActionTarget {
	return u.actionTargetMap
}

func (u *UI) SetCurrentTarget(entry *donburi.Entry) {
	u.currentTarget = entry
}

func (u *UI) ClearCurrentTarget() {
	u.currentTarget = nil
}

func (u *UI) Update() {
	u.ebitenui.Update()
}

func (u *UI) Draw(screen *ebiten.Image, tick int) {
	// ターゲットインジケーターの描画に必要な IconViewModel を取得
	var indicatorTargetVM *IconViewModel
	if u.currentTarget != nil && u.battlefieldWidget.viewModel != nil {
		for _, iconVM := range u.battlefieldWidget.viewModel.Icons {
			if iconVM.EntryID == uint32(u.currentTarget.Id()) {
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

func (u *UI) DrawBackground(screen *ebiten.Image) {
	u.battlefieldWidget.DrawBackground(screen)
}

func (u *UI) GetBattlefieldWidgetRect() image.Rectangle {
	return u.battlefieldWidget.Container.GetWidget().Rect
}

func (u *UI) GetRootContainer() *widget.Container {
	return u.ebitenui.Container
}

func (u *UI) SetAnimation(anim *ActionAnimationData) {
	u.animationDrawer.animationManager.SetAnimation(anim)
}

func (u *UI) IsAnimationFinished(tick int) bool {
	return u.animationDrawer.animationManager.IsAnimationFinished(tick)
}

func (u *UI) ClearAnimation() {
	u.animationDrawer.animationManager.ClearAnimation()
}

func (u *UI) GetCurrentAnimationResult() ActionResult {
	return u.animationDrawer.animationManager.currentAnimation.Result
}

// drawPingAnimation は、指定された中心にレーダーのようなピングアニメーションを描画します。
// progress は 0.0 から 1.0 の値で、アニメーションの進行状況を示します。
// expandがtrueの場合は拡大、falseの場合は縮小アニメーションになります。
