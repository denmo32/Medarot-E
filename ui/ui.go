package ui

import (
	"image"
	"image/color"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

type UI struct {
	ebitenui          *ebitenui.UI
	battlefieldWidget *BattlefieldWidget
	infoPanelManager  *InfoPanelManager // infoPanelManager に変更
	commonBottomPanel *UIPanel          // 共通の下部パネル
	// イベント通知用チャネル
	eventChannel chan UIEvent
	// 依存性
	config                 *data.Config
	whitePixel             *ebiten.Image
	messageManager         *UIMessageDisplayManager
	actionModalManager     *UIActionModalManager
	targetIndicatorManager *UITargetIndicatorManager
	animationDrawer        *UIAnimationDrawer
	lastWidth, lastHeight  int            // レイアウト更新の最適化用
	battleUIState          *BattleUIState // UIの状態を保持
	uiFactory              *UIFactory     // uiFactoryを保持
}

// SetBattleUIState はUI全体のデータソースを一元的に設定します。
func (u *UI) SetBattleUIState(battleUIState *BattleUIState, config *data.Config, battlefieldRect image.Rectangle, uiFactory *UIFactory) {
	u.battleUIState = battleUIState // UI構造体に状態を保存

	// BattlefieldViewModel を設定
	u.battlefieldWidget.SetViewModel(battleUIState.BattlefieldViewModel)

	// InfoPanels を更新または再構築
	mainUIContainer := u.ebitenui.Container.Children()[0].(*widget.Container).Children()[0].(*widget.Container)

	// マップからスライスに変換
	infoPanelVMs := make([]core.InfoPanelViewModel, 0, len(battleUIState.InfoPanels))
	for _, vm := range battleUIState.InfoPanels {
		infoPanelVMs = append(infoPanelVMs, vm)
	}

	// InfoPanelManager に mainUIContainer と battlefieldRect、アイコンのY座標を渡す
	u.infoPanelManager.UpdatePanels(infoPanelVMs, mainUIContainer, battlefieldRect, battleUIState.BattlefieldViewModel.Icons)
}

// PostEvent はUIイベントをBattleSceneのキューに追加します。
func (u *UI) PostEvent(event UIEvent) {
	u.eventChannel <- event
}

// NewUI は新しいUIインスタンスを作成します。
func NewUI(config *data.Config, eventChannel chan UIEvent, uiFactory *UIFactory, gameDataManager *data.GameDataManager) *UI {
	whiteImg := ebiten.NewImage(1, 1)
	whiteImg.Fill(color.White)

	ui := &UI{
		infoPanelManager: NewInfoPanelManager(config, uiFactory), // InfoPanelManager を初期化
		eventChannel:     eventChannel,
		config:           config,
		whitePixel:       whiteImg,
		animationDrawer:  NewUIAnimationDrawer(config, uiFactory.Font, eventChannel), // uiFactory.Font を使用
		uiFactory:        uiFactory,                                                  // uiFactoryを保存
	}

	rootContainer := createRootContainer()
	baseLayoutContainer := createBaseLayoutContainer()
	rootContainer.AddChild(baseLayoutContainer)

	// 上部パネル（既存のmainUIContainer）
	mainUIContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(nil), // レイアウトをnilに設定し、手動でRectを設定
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{})),
	)
	baseLayoutContainer.AddChild(mainUIContainer)

	ui.commonBottomPanel = createCommonBottomPanel(config, uiFactory, gameDataManager)
	bottomPanelWrapper := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{})), // GridLayoutData を設定
	)
	ui.commonBottomPanel.RootContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	bottomPanelWrapper.AddChild(ui.commonBottomPanel.RootContainer)
	baseLayoutContainer.AddChild(bottomPanelWrapper)

	ui.battlefieldWidget = NewBattlefieldWidget(config)
	mainUIContainer.AddChild(ui.battlefieldWidget.Container)
	ui.messageManager = NewUIMessageDisplayManager(gameDataManager.Messages, config, uiFactory.MessageWindowFont, uiFactory, ui.commonBottomPanel)
	ui.ebitenui = &ebitenui.UI{
		Container: rootContainer,
	}
	ui.actionModalManager = NewUIActionModalManager(ui.ebitenui, eventChannel, uiFactory, ui.commonBottomPanel)
	ui.targetIndicatorManager = NewUITargetIndicatorManager()
	return ui
}

func createRootContainer() *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewStackedLayout()),
	)
}

func createBaseLayoutContainer() *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{true, false}),
			widget.GridLayoutOpts.Spacing(0, 10),
		)),
	)
}

func createCommonBottomPanel(config *data.Config, uiFactory *UIFactory, gameDataManager *data.GameDataManager) *UIPanel {
	return NewPanel(&PanelOptions{
		PanelWidth:      814,
		PanelHeight:     180,
		Padding:         widget.NewInsetsSimple(5),
		Spacing:         5,
		BackgroundColor: color.NRGBA{50, 50, 70, 200},
		BorderColor:     config.UI.Colors.Gray,
		BorderThickness: 5,
		CenterContent:   true,
	}, uiFactory.imageGenerator, gameDataManager.Font)
}

// IsActionModalVisible はアクションモーダルが表示されているかどうかを返します。
func (u *UI) IsActionModalVisible() bool {
	return u.actionModalManager.IsVisible()
}

// ShowActionModal はアクション選択モーダルを表示します。
func (u *UI) ShowActionModal(vm core.ActionModalViewModel) {
	u.actionModalManager.ShowActionModal(vm)
}

// HideActionModal はアクション選択モーダルを非表示にします。
func (u *UI) HideActionModal() {
	u.actionModalManager.HideActionModal()
}

// GetActionTargetMap は現在のアクションターゲットマップを返します。
func (u *UI) GetActionTargetMap() map[core.PartSlotKey]core.ActionTarget {
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

// updateLayout はUIのレイアウトを更新します。
func (u *UI) updateLayout() {
	// ルートコンテナのRectは、ウィンドウサイズとほぼ同じ
	rootRect := u.ebitenui.Container.GetWidget().Rect
	width, height := rootRect.Dx(), rootRect.Dy()

	// サイズが変わっていなければ何もしない
	if u.lastWidth == width && u.lastHeight == height {
		return
	}
	u.lastWidth, u.lastHeight = width, height

	// mainUIContainer (上部パネル) のRectを取得
	// baseLayoutContainer -> mainUIContainer
	mainUIContainer := u.ebitenui.Container.Children()[0].(*widget.Container).Children()[0].(*widget.Container)
	containerRect := mainUIContainer.GetWidget().Rect

	// バトルフィールドのRectを計算
	infoPanelWidth := int(u.config.UI.InfoPanel.BlockWidth)
	padding := int(u.config.UI.InfoPanel.Padding)

	// バトルフィールドの幅は、コンテナの幅から左右の情報パネルとパディングを引いたもの
	bfWidth := containerRect.Dx() - (infoPanelWidth+padding)*2
	bfHeight := containerRect.Dy()

	// バトルフィールドのX座標は、左の情報パネルの幅とパディングの合計
	bfX := infoPanelWidth + padding
	bfY := 0 // mainUIContainerのY座標は0から始まる

	battlefieldRect := image.Rect(bfX, bfY, bfX+bfWidth, bfY+bfHeight)
	u.battlefieldWidget.Container.GetWidget().Rect = battlefieldRect

	// 情報パネルの更新（battlefieldRectを渡す）
	// SetBattleUIStateから呼び出されるViewModelの更新時に、この情報を使ってパネルを配置する
	// ここでは直接呼び出さず、SetBattleUIStateが呼び出されたときにUpdatePanelsが実行されることを期待する
	// ただし、レイアウト変更時に情報パネルも再配置されるように、SetBattleUIStateを呼び出す必要がある
	// レイアウト更新時に情報パネルも再配置されるように、直接UpdatePanelsを呼び出す
	if u.battleUIState != nil {
		infoPanelVMs := make([]core.InfoPanelViewModel, 0, len(u.battleUIState.InfoPanels))
		for _, vm := range u.battleUIState.InfoPanels {
			infoPanelVMs = append(infoPanelVMs, vm)
		}
		u.infoPanelManager.UpdatePanels(infoPanelVMs, mainUIContainer, battlefieldRect, u.battleUIState.BattlefieldViewModel.Icons)
	}
}

// Update はUIの状態を更新します。
func (u *UI) Update(tick int) {
	u.updateLayout() // ここで呼び出す
	u.ebitenui.Update()
	u.animationDrawer.Update(float64(tick)) // アニメーションの更新をここで行う
	u.messageManager.Update()
}

// Draw はUIを描画します。
func (u *UI) Draw(screen *ebiten.Image, tick int, gameDataManager *data.GameDataManager) {
	// ターゲットインジケーターの描画に必要な IconViewModel を取得
	var indicatorTargetVM *core.IconViewModel
	if u.targetIndicatorManager.GetCurrentTarget() != 0 && u.battlefieldWidget.viewModel != nil {
		for _, iconVM := range u.battlefieldWidget.viewModel.Icons {
			if iconVM.EntryID == u.targetIndicatorManager.GetCurrentTarget() {
				indicatorTargetVM = iconVM
				break
			}
		}
	}

	// まずebitenuiを描画（背景とコンテナ）
	u.ebitenui.Draw(screen)

	// その後でBattlefieldWidgetの前景要素を描画
	// これにより背景画像の上にアイコンやラインが描画される
	u.battlefieldWidget.Draw(screen, indicatorTargetVM, tick)

	// アニメーションの描画（最前面）
	if u.battlefieldWidget.viewModel != nil {
		u.animationDrawer.Draw(screen, float64(tick), *u.battlefieldWidget.viewModel)
	}
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
func (u *UI) SetAnimation(anim *component.ActionAnimationData) {
	u.animationDrawer.SetAnimation(anim)
}

// IsAnimationFinished は現在のアニメーションが完了したかどうかを返します。
func (u *UI) IsAnimationFinished(tick int) bool {
	return u.animationDrawer.IsAnimationFinished(float64(tick))
}

// ClearAnimation は現在のアニメーションをクリアします。
func (u *UI) ClearAnimation() {
	u.animationDrawer.ClearAnimation()
}

// GetCurrentAnimationResult は現在のアニメーションの結果を返します。
func (u *UI) GetCurrentAnimationResult() component.ActionResult {
	return u.animationDrawer.GetCurrentAnimationResult()
}

// GetMessageDisplayManager はメッセージ表示マネージャーを返します。
func (u *UI) GetMessageDisplayManager() *UIMessageDisplayManager {
	return u.messageManager
}

func (u *UI) GetEventChannel() chan UIEvent {
	return u.eventChannel
}

func (u *UI) GetConfig() *data.Config {
	return u.config
}
