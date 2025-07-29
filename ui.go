package main

import (
	"image"
	"image/color"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/yohamta/donburi"
)

type UI struct {
	ebitenui          *ebitenui.UI
	battlefieldWidget *BattlefieldWidget
	infoPanelManager  *InfoPanelManager // infoPanelManager に変更
	// イベント通知用チャネル
	eventChannel chan UIEvent
	// 依存性
	config                 *Config
	whitePixel             *ebiten.Image
	messageManager         *UIMessageDisplayManager
	actionModalManager     *UIActionModalManager
	targetIndicatorManager *UITargetIndicatorManager
	animationDrawer        *UIAnimationDrawer
}

// SetBattleUIState はUI全体のデータソースを一元的に設定します。
func (u *UI) SetBattleUIState(battleUIState *BattleUIState, config *Config, battlefieldRect image.Rectangle, uiFactory *UIFactory) {
	// BattlefieldViewModel を設定
	u.battlefieldWidget.SetViewModel(battleUIState.BattlefieldViewModel)

	// InfoPanels を更新または再構築
	mainUIContainer := u.ebitenui.Container.Children()[0].(*widget.Container)
	team1PanelContainer := mainUIContainer.Children()[0].(*widget.Container)
	team2PanelContainer := mainUIContainer.Children()[2].(*widget.Container)

	// マップからスライスに変換
	infoPanelVMs := make([]InfoPanelViewModel, 0, len(battleUIState.InfoPanels))
	for _, vm := range battleUIState.InfoPanels {
		infoPanelVMs = append(infoPanelVMs, vm)
	}

	u.infoPanelManager.UpdatePanels(infoPanelVMs, team1PanelContainer, team2PanelContainer)
}

// PostEvent はUIイベントをBattleSceneのキューに追加します。
func (u *UI) PostEvent(event UIEvent) {
	u.eventChannel <- event
}

// NewUI は新しいUIインスタンスを作成します。
func NewUI(config *Config, eventChannel chan UIEvent, uiFactory *UIFactory, gameDataManager *GameDataManager) *UI {
	whiteImg := ebiten.NewImage(1, 1)
	whiteImg.Fill(color.White)

	ui := &UI{
		infoPanelManager: NewInfoPanelManager(config, uiFactory), // InfoPanelManager を初期化
		eventChannel:     eventChannel,
		config:           config,
		whitePixel:       whiteImg,
		animationDrawer:  NewUIAnimationDrawer(config, gameDataManager.Font),
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
	// InfoPanelsの初期化はSetBattleUIStateで行われるため、ここでは行わない
	// ui.medarotInfoPanelsはSetBattleUIStateで動的に構築される
	ui.messageManager = NewUIMessageDisplayManager(gameDataManager.Messages, config, gameDataManager.Font, ui, uiFactory) // ui を渡す
	ui.ebitenui = &ebitenui.UI{
		Container: rootContainer,
	}
	ui.actionModalManager = NewUIActionModalManager(ui.ebitenui, eventChannel, uiFactory)
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
func (u *UI) SetCurrentTarget(entityID donburi.Entity) {
	u.targetIndicatorManager.SetCurrentTarget(entityID)
}

// ClearCurrentTarget は現在のターゲットをクリアします。
func (u *UI) ClearCurrentTarget() {
	u.targetIndicatorManager.ClearCurrentTarget()
}

// Update はUIの状態を更新します。
func (u *UI) Update() {
	u.ebitenui.Update()
	// アニメーション終了の検知とイベント発行は、Draw メソッド内で tick を使って行う
}

// Draw はUIを描画します。
func (u *UI) Draw(screen *ebiten.Image, tick int, gameDataManager *GameDataManager) {
	// アニメーションが再生中で、かつ終了している場合、イベントを発行
	if u.animationDrawer.currentAnimation != nil && u.animationDrawer.IsAnimationFinished(tick) {
		result := u.animationDrawer.GetCurrentAnimationResult()
		u.PostEvent(AnimationFinishedUIEvent{Result: result})
		u.animationDrawer.ClearAnimation()
	}

	// ターゲットインジケーターの描画に必要な IconViewModel を取得
	var indicatorTargetVM *IconViewModel
	if u.targetIndicatorManager.GetCurrentTarget() != 0 && u.battlefieldWidget.viewModel != nil { // 0はdonburi.Entityのゼロ値
		for _, iconVM := range u.battlefieldWidget.viewModel.Icons {
			if iconVM.EntryID == u.targetIndicatorManager.GetCurrentTarget() { // uint32 へのキャストを削除
				indicatorTargetVM = iconVM
				break
			}
		}
	}

	// BattlefieldWidget の Draw メソッドを先に呼び出す
	u.battlefieldWidget.Draw(screen, indicatorTargetVM, tick)

	// アニメーションの描画
	if u.battlefieldWidget.viewModel != nil {
		u.animationDrawer.Draw(screen, tick, *u.battlefieldWidget.viewModel) // gameDataManagerを削除
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
	u.animationDrawer.SetAnimation(anim)
}

// IsAnimationFinished は現在のアニメーションが完了したかどうかを返します。
func (u *UI) IsAnimationFinished(tick int) bool {
	return u.animationDrawer.IsAnimationFinished(tick)
}

// ClearAnimation は現在のアニメーションをクリアします。
func (u *UI) ClearAnimation() {
	u.animationDrawer.ClearAnimation()
}

// GetCurrentAnimationResult は現在のアニメーションの結果を返します。
func (u *UI) GetCurrentAnimationResult() ActionResult {
	return u.animationDrawer.GetCurrentAnimationResult()
}

// GetMessageDisplayManager はメッセージ表示マネージャーを返します。
func (u *UI) GetMessageDisplayManager() *UIMessageDisplayManager {
	return u.messageManager
}

// ShowMessagePanel はメッセージパネルをルートコンテナに追加します。
func (u *UI) ShowMessagePanel(panel widget.PreferredSizeLocateableWidget) {
	u.ebitenui.Container.AddChild(panel)
}

// HideMessagePanel はメッセージパネルをルートコンテナから削除します。
func (u *UI) HideMessagePanel() {
	if u.messageManager.messageWindow != nil {
		u.ebitenui.Container.RemoveChild(u.messageManager.messageWindow)
	}
}
