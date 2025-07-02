package main

import (
	"log"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/yohamta/donburi"
)

type UI struct {
	ebitenui          *ebitenui.UI
	actionModal       widget.PreferredSizeLocateableWidget
	messageWindow     widget.PreferredSizeLocateableWidget
	battlefieldWidget *BattlefieldWidget
	medarotInfoPanels map[string]*infoPanelUI
	actionTargetMap   map[PartSlotKey]ActionTarget
}

func NewUI(bs *BattleScene) *UI {
	ui := &UI{
		medarotInfoPanels: make(map[string]*infoPanelUI),
		actionTargetMap:   make(map[PartSlotKey]ActionTarget),
	}
	bs.ui = ui

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

	ui.battlefieldWidget = NewBattlefieldWidget(bs)
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

func (u *UI) ShowActionModal(bs *BattleScene, actingEntry *donburi.Entry) {
	if u.actionModal != nil {
		u.HideActionModal()
	}
	u.actionTargetMap = make(map[PartSlotKey]ActionTarget)

	modal := createActionModalUI(bs, actingEntry)
	u.actionModal = modal
	u.ebitenui.Container.AddChild(u.actionModal)
	log.Println("Action modal shown.")
}

func (u *UI) HideActionModal() {
	if u.actionModal != nil {
		u.ebitenui.Container.RemoveChild(u.actionModal)
		u.actionTargetMap = make(map[PartSlotKey]ActionTarget)
		u.actionModal = nil
		log.Println("Action modal hidden.")
	}
}

func (u *UI) ShowMessageWindow(bs *BattleScene) {
	if u.messageWindow != nil {
		u.HideMessageWindow()
	}
	win := createMessageWindow(bs)
	u.messageWindow = win
	u.ebitenui.Container.AddChild(u.messageWindow)
}

func (u *UI) HideMessageWindow() {
	if u.messageWindow != nil {
		u.ebitenui.Container.RemoveChild(u.messageWindow)
		u.messageWindow = nil
	}
}
