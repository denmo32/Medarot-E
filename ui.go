package main

import (
    "bytes"
    _ "embed"
	// "fmt"
	"image" // 追加
	"image/color"
	"log" // 追加

    "github.com/ebitenui/ebitenui"
    "github.com/ebitenui/ebitenui/widget"
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/text/v2"
    "github.com/yohamta/donburi"
)

//go:embed MPLUS1p-Regular.ttf
var mplusFontData []byte

func (u *UI) IsActionModalVisible() bool {
             return u.isActionModalVisible
}

func (u *UI) SetBattlefieldViewModel(vm BattlefieldViewModel) {
             u.battlefieldWidget.SetViewModel(vm)
}

// loadFont はEbitenUIが要求する text.Face を返します。
func loadFont() (text.Face, error) {
    s, err := text.NewGoTextFaceSource(bytes.NewReader(mplusFontData))
    if err != nil {
        return nil, err
    }
    face := &text.GoTextFace{
        Source: s,
        Size:   12,
    }
    log.Println("カスタムフォント（text/v2）の読み込みに成功しました。")
    return face, nil
}

type UI struct {
    ebitenui               *ebitenui.UI
    actionModal            widget.PreferredSizeLocateableWidget
    messageWindow          widget.PreferredSizeLocateableWidget
    battlefieldWidget      *BattlefieldWidget
    medarotInfoPanels      map[string]*infoPanelUI
    actionTargetMap        map[PartSlotKey]ActionTarget
    // UIの状態
    playerMedarotToAct *donburi.Entry // 現在アクション選択中のプレイヤーメダロット
    currentTarget      *donburi.Entry // 現在ターゲットとして表示されているエンティティ
    isActionModalVisible bool           // アクションモーダルが表示されているか
    // イベント通知用チャネル
    eventChannel chan UIEvent
    // 依存性
    world            donburi.World
    config           *Config
    partInfoProvider *PartInfoProvider
    targetSelector   *TargetSelector
    whitePixel       *ebiten.Image
}

// PostEvent はUIイベントをBattleSceneのキューに追加します。
func (u *UI) PostEvent(event UIEvent) {
    u.eventChannel <- event
}

// NewUI は新しいUIインスタンスを作成します。
func NewUI(world donburi.World, config *Config, eventChannel chan UIEvent) *UI {
    whiteImg := ebiten.NewImage(1, 1)
    whiteImg.Fill(color.White)
    ui := &UI{
        medarotInfoPanels:    make(map[string]*infoPanelUI),
        actionTargetMap:      make(map[PartSlotKey]ActionTarget),
        isActionModalVisible: false,
        eventChannel:         eventChannel,
        world:                world,
        config:               config,
        whitePixel:           whiteImg,
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
	setupInfoPanels(world, config, GlobalGameDataManager.Font, ui.medarotInfoPanels, team1PanelContainer,team2PanelContainer)
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
func (u *UI) ShowMessageWindow(message string) {
    if u.messageWindow != nil {
        u.HideMessageWindow()
    }
     win := createMessageWindow(message, u.config, GlobalGameDataManager.Font)
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

func (u *UI) Draw(screen *ebiten.Image) {
    u.ebitenui.Draw(screen)

    // 修正: バトルフィールドの描画は BattlefieldWidget に任せる
    // u.GetBattlefieldWidgetRect() は BattlefieldWidget の内部で利用されるため、ここでは不要
    // u.whitePixel も BattlefieldWidget の内部で利用されるため、ここでは不要

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

    // BattlefieldWidget の Draw メソッドを呼び出す
    u.battlefieldWidget.Draw(screen, indicatorTargetVM)
}

func (u *UI) GetBattlefieldWidgetRect() image.Rectangle {
    return u.battlefieldWidget.Container.GetWidget().Rect
}