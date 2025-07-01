package main

import (
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/yohamta/donburi"
	"log"
)

// UI はゲームの全UI要素を管理する構造体
type UI struct {
	ebitenui          *ebitenui.UI
	actionModal       widget.PreferredSizeLocateableWidget
	messageWindow     widget.PreferredSizeLocateableWidget
	battlefieldWidget *BattlefieldWidget
	medarotInfoPanels map[string]*infoPanelUI
	actionTargetMap   map[PartSlotKey]ActionTarget // [修正] 保持する型をActionTargetに変更
}

// NewUI はUIを構築し、管理構造体を返す
func NewUI(game *Game) *UI {
	ui := &UI{
		medarotInfoPanels: make(map[string]*infoPanelUI),
		actionTargetMap:   make(map[PartSlotKey]ActionTarget), // [修正] マップを初期化
	}

	game.ui = ui

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewStackedLayout()),
	)

	mainUIContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{false, true, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(game.Config.UI.InfoPanel.Padding, 0),
		)),
	)
	rootContainer.AddChild(mainUIContainer)

	team1PanelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(game.Config.UI.InfoPanel.Padding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(game.Config.UI.InfoPanel.BlockWidth), 0),
		),
	)
	mainUIContainer.AddChild(team1PanelContainer)

	ui.battlefieldWidget = NewBattlefieldWidget(game)
	ui.battlefieldWidget.Container.GetWidget().LayoutData = widget.GridLayoutData{}
	mainUIContainer.AddChild(ui.battlefieldWidget.Container)

	team2PanelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(game.Config.UI.InfoPanel.Padding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(game.Config.UI.InfoPanel.BlockWidth), 0),
		),
	)
	mainUIContainer.AddChild(team2PanelContainer)

	setupInfoPanels(game, team1PanelContainer, team2PanelContainer)

	ui.ebitenui = &ebitenui.UI{
		Container: rootContainer,
	}
	return ui
}

// ShowActionModal は行動選択モーダルを表示する
func (u *UI) ShowActionModal(game *Game, actingEntry *donburi.Entry) {
	if u.actionModal != nil {
		u.HideActionModal()
	}
	u.actionTargetMap = make(map[PartSlotKey]ActionTarget)

	modal := createActionModalUI(game, actingEntry)
	if modal == nil {
		game.State = StatePlaying
		return
	}
	u.actionModal = modal
	u.ebitenui.Container.AddChild(u.actionModal)
	log.Println("Action modal shown.")
}

// HideActionModal は行動選択モーダルを非表示にする
func (u *UI) HideActionModal() {
	if u.actionModal != nil {
		u.ebitenui.Container.RemoveChild(u.actionModal)
		u.actionTargetMap = make(map[PartSlotKey]ActionTarget)
		u.actionModal = nil
		log.Println("Action modal hidden.")
	}
}

// ShowMessageWindow はメッセージウィンドウを表示する
func (u *UI) ShowMessageWindow(game *Game) {
	if u.messageWindow != nil {
		u.HideMessageWindow()
	}
	win := createMessageWindow(game)
	u.messageWindow = win
	u.ebitenui.Container.AddChild(u.messageWindow)
}

// HideMessageWindow はメッセージウィンドウを非表示にする
func (u *UI) HideMessageWindow() {
	if u.messageWindow != nil {
		u.ebitenui.Container.RemoveChild(u.messageWindow)
		u.messageWindow = nil
	}
}
