package main

import (
	"log"
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
)

// UI はゲームの全UI要素を管理する構造体
type UI struct {
	ebitenui          *ebitenui.UI
	actionModal       widget.PreferredSizeLocateableWidget
	messageWindow     widget.PreferredSizeLocateableWidget
	battlefieldWidget *BattlefieldWidget
	medarotInfoPanels map[string]*infoPanelUI
}

// NewUI はUIを構築し、管理構造体を返す
func NewUI(game *Game) *UI {
	ui := &UI{
		// フィールドをここで初期化する
		medarotInfoPanels: make(map[string]*infoPanelUI),
	}

	// --- UIの構築 ---
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

	// チーム1の情報パネルのコンテナ
	team1PanelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(game.Config.UI.InfoPanel.Padding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(game.Config.UI.InfoPanel.BlockWidth), 0),
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionCenter,
				VerticalPosition:   widget.GridLayoutPositionCenter,
			}),
		),
	)
	mainUIContainer.AddChild(team1PanelContainer)

	// バトルフィールドウィジェット
	ui.battlefieldWidget = NewBattlefieldWidget(game)
	ui.battlefieldWidget.Container.GetWidget().LayoutData = widget.GridLayoutData{
		HorizontalPosition: widget.GridLayoutPositionCenter,
		VerticalPosition:   widget.GridLayoutPositionCenter,
	}
	mainUIContainer.AddChild(ui.battlefieldWidget.Container)

	// チーム2の情報パネルのコンテナ
	team2PanelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(game.Config.UI.InfoPanel.Padding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(game.Config.UI.InfoPanel.BlockWidth), 0),
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionCenter,
				VerticalPosition:   widget.GridLayoutPositionCenter,
			}),
		),
	)
	mainUIContainer.AddChild(team2PanelContainer)

	// メダロット情報パネルを生成して配置
	for _, m := range game.Medarots {
		panelUI := createSingleMedarotInfoPanel(game, m)
		// グローバル変数ではなく、UI構造体のフィールドに格納する
		ui.medarotInfoPanels[m.ID] = panelUI
		if m.Team == Team1 {
			team1PanelContainer.AddChild(panelUI.rootContainer)
		} else {
			team2PanelContainer.AddChild(panelUI.rootContainer)
		}
	}

	ui.ebitenui = &ebitenui.UI{
		Container: rootContainer,
	}
	return ui
}

// ShowActionModal は行動選択モーダルを表示する
func (u *UI) ShowActionModal(game *Game, actingMedarot *Medarot) {
	if u.actionModal != nil {
		u.HideActionModal()
	}
	modal := createActionModalUI(game, actingMedarot)
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
