package main

import (
	"log"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/yohamta/donburi"
)

type UI struct {
	ebitenui               *ebitenui.UI
	actionModal            widget.PreferredSizeLocateableWidget
	messageWindow          widget.PreferredSizeLocateableWidget
	battlefieldWidget      *BattlefieldWidget
	medarotInfoPanels      map[string]*infoPanelUI
	actionTargetMap        map[PartSlotKey]ActionTarget
	scene                  *BattleScene // イベント発行のためにシーンへの参照を保持

	// UIの状態
	playerMedarotToAct *donburi.Entry // 現在アクション選択中のプレイヤーメダロット
	currentTarget      *donburi.Entry // 現在ターゲットとして表示されているエンティティ
	isActionModalVisible bool           // アクションモーダルが表示されているか
}

// PostEvent はUIイベントをBattleSceneのキューに追加します。
func (u *UI) PostEvent(event UIEvent) {
	if u.scene != nil {
		u.scene.uiEvents = append(u.scene.uiEvents, event)
	}
}

// NewUI は新しいUIインスタンスを作成します。
func NewUI(bs *BattleScene) *UI {
	ui := &UI{
		medarotInfoPanels:    make(map[string]*infoPanelUI),
		actionTargetMap:      make(map[PartSlotKey]ActionTarget),
		scene:                bs, // sceneへの参照を保存
		isActionModalVisible: false,
	}
	bs.ui = ui // BattleSceneにUIインスタンスを登録

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewStackedLayout()),
	)
	mainUIContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{false, true, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(bs.resources.Config.UI.InfoPanel.Padding, 0),
		)),
	)
	rootContainer.AddChild(mainUIContainer)

	team1PanelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(bs.resources.Config.UI.InfoPanel.Padding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(bs.resources.Config.UI.InfoPanel.BlockWidth), 0),
		),
	)
	mainUIContainer.AddChild(team1PanelContainer)

	ui.battlefieldWidget = NewBattlefieldWidget(&bs.resources.Config)
	ui.battlefieldWidget.Container.GetWidget().LayoutData = widget.GridLayoutData{}
	mainUIContainer.AddChild(ui.battlefieldWidget.Container)

	team2PanelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(bs.resources.Config.UI.InfoPanel.Padding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(bs.resources.Config.UI.InfoPanel.BlockWidth), 0),
		),
	)
	mainUIContainer.AddChild(team2PanelContainer)

	setupInfoPanels(bs, team1PanelContainer, team2PanelContainer)

	ui.ebitenui = &ebitenui.UI{
		Container: rootContainer,
	}
	return ui
}

// ShowActionModal はアクション選択モーダルを表示します。
func (u *UI) ShowActionModal(actingEntry *donburi.Entry) {
	if u.isActionModalVisible {
		u.HideActionModal()
	}
	u.playerMedarotToAct = actingEntry
	u.isActionModalVisible = true
	u.actionTargetMap = make(map[PartSlotKey]ActionTarget) // ターゲットマップを初期化

	modal := createActionModalUI(u.scene, actingEntry)
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
func (u *UI) ShowMessageWindow(bs *BattleScene) {
	if u.messageWindow != nil {
		u.HideMessageWindow()
	}
	win := createMessageWindow(bs)
	u.messageWindow = win
	u.ebitenui.Container.AddChild(u.messageWindow)
}

// HideMessageWindow はメッセージウィンドウを非表示にします。
func (u *UI) HideMessageWindow() {
	if u.messageWindow != nil {
		u.ebitenui.Container.RemoveChild(u.messageWindow)
		u.messageWindow = nil
	}
}
