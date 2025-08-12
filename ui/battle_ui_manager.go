package ui

import (
	"image"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/event"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

// BattleUIManager はバトルシーンのUI要素の管理と描画を担当する唯一の司令塔です。
type BattleUIManager struct {
	config    *data.Config
	world     donburi.World
	uiFactory *UIFactory

	// ebitenui root
	ebitenui *ebitenui.UI

	// Sub-managers
	infoPanelManager       *InfoPanelManager
	messageManager         *UIMessageDisplayManager
	actionModalManager     *UIActionModalManager
	targetIndicatorManager *UITargetIndicatorManager
	animationDrawer        *UIAnimationDrawer

	// Widgets
	battlefieldWidget *BattlefieldWidget
	commonBottomPanel *UIPanel

	// State & Events
	eventChannel          chan event.GameEvent
	lastWidth, lastHeight int
}

// NewBattleUIManager は BattleUIManager の新しいインスタンスを作成します。
func NewBattleUIManager(
	config *data.Config,
	resources *data.SharedResources,
	world donburi.World,
) *BattleUIManager {
	bum := &BattleUIManager{
		config:       config,
		world:        world,
		eventChannel: make(chan event.GameEvent, 10),
	}

	bum.uiFactory = NewUIFactory(config, resources.Font, resources.ModalButtonFont, resources.MessageWindowFont, resources.GameDataManager.Messages)

	// Initialize sub-managers
	bum.infoPanelManager = NewInfoPanelManager(config, bum.uiFactory)
	bum.animationDrawer = NewUIAnimationDrawer(config, bum.uiFactory.Font, bum.eventChannel)
	bum.targetIndicatorManager = NewUITargetIndicatorManager()

	// Build UI layout
	rootContainer := createRootContainer()
	baseLayoutContainer := createBaseLayoutContainer()
	rootContainer.AddChild(baseLayoutContainer)

	mainUIContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(nil),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{})),
	)
	baseLayoutContainer.AddChild(mainUIContainer)

	bum.commonBottomPanel = createCommonBottomPanel(config, bum.uiFactory, resources.GameDataManager)
	bottomPanelWrapper := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{})),
	)
	bum.commonBottomPanel.RootContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	bottomPanelWrapper.AddChild(bum.commonBottomPanel.RootContainer)
	baseLayoutContainer.AddChild(bottomPanelWrapper)

	bum.battlefieldWidget = NewBattlefieldWidget(config)
	mainUIContainer.AddChild(bum.battlefieldWidget.Container)

	bum.ebitenui = &ebitenui.UI{
		Container: rootContainer,
	}

	// Initialize managers that depend on the layout
	bum.messageManager = NewUIMessageDisplayManager(resources.GameDataManager.Messages, config, bum.uiFactory.MessageWindowFont, bum.uiFactory, bum.commonBottomPanel)
	bum.actionModalManager = NewUIActionModalManager(bum.ebitenui, bum.eventChannel, bum.uiFactory, bum.commonBottomPanel)

	return bum
}

// Update はUI全体の状態を更新します。
func (bum *BattleUIManager) Update(tickCount int) []event.GameEvent {
	bum.ebitenui.Update()
	bum.animationDrawer.Update(float64(tickCount))
	bum.messageManager.Update()

	// UIイベントチャネルからゲームイベントを収集
	var uiGeneratedGameEvents []event.GameEvent
	for len(bum.eventChannel) > 0 {
		uiGeneratedGameEvents = append(uiGeneratedGameEvents, <-bum.eventChannel)
	}

	return uiGeneratedGameEvents
}

// SetViewModels は、渡されたViewModelに基づいてUI全体を更新します。
func (bum *BattleUIManager) SetViewModels(infoPanelVMs []core.InfoPanelViewModel, battlefieldVM core.BattlefieldViewModel) {
	// Battlefield
	bum.battlefieldWidget.SetViewModel(battlefieldVM)

	// Info Panels
	mainUIContainer := bum.ebitenui.Container.Children()[0].(*widget.Container).Children()[0].(*widget.Container)
	bum.infoPanelManager.UpdatePanels(infoPanelVMs, mainUIContainer, bum.GetBattlefieldWidgetRect(), battlefieldVM.Icons)

	// Ensure layout is up to date
	rootRect := bum.ebitenui.Container.GetWidget().Rect
	width, height := rootRect.Dx(), rootRect.Dy()
	if bum.lastWidth != width || bum.lastHeight != height {
		bum.lastWidth, bum.lastHeight = width, height
		mainUIContainer := bum.ebitenui.Container.Children()[0].(*widget.Container).Children()[0].(*widget.Container)
		containerRect := mainUIContainer.GetWidget().Rect

		infoPanelWidth := int(bum.config.UI.InfoPanel.BlockWidth)
		padding := int(bum.config.UI.InfoPanel.Padding)

		bfWidth := containerRect.Dx() - (infoPanelWidth+padding)*2
		bfHeight := containerRect.Dy()
		bfX := infoPanelWidth + padding
		bfY := 0

		battlefieldRect := image.Rect(bfX, bfY, bfX+bfWidth, bfY+bfHeight)
		bum.battlefieldWidget.Container.GetWidget().Rect = battlefieldRect
	}
}

// Draw はUI全体を描画します。
func (bum *BattleUIManager) Draw(screen *ebiten.Image, tickCount int, gameDataManager *data.GameDataManager) {
	// 背景色で塗りつぶし
	screen.Fill(bum.config.UI.Colors.Background)

	// ターゲットインジケーターの描画に必要なIconViewModelを取得
	var indicatorTargetVM *core.IconViewModel
	if bum.targetIndicatorManager.GetCurrentTarget() != 0 && bum.battlefieldWidget.viewModel != nil {
		for _, iconVM := range bum.battlefieldWidget.viewModel.Icons {
			if iconVM.EntryID == bum.targetIndicatorManager.GetCurrentTarget() {
				indicatorTargetVM = iconVM
				break
			}
		}
	}

	// ebitenuiのコンテナ（背景、パネルなど）を描画
	bum.ebitenui.Draw(screen)

	// BattlefieldWidgetの前景要素（アイコン、ラインなど）を描画
	bum.battlefieldWidget.Draw(screen, indicatorTargetVM, tickCount)

	// アニメーションを最前面に描画
	if bum.battlefieldWidget.viewModel != nil {
		bum.animationDrawer.Draw(screen, float64(tickCount), *bum.battlefieldWidget.viewModel)
	}
}

// --- Public Methods for Scene interaction ---

func (bum *BattleUIManager) EnqueueMessage(msg string, callback func()) {
	bum.messageManager.EnqueueMessage(msg, callback)
}

func (bum *BattleUIManager) EnqueueMessageQueue(messages []string, callback func()) {
	bum.messageManager.EnqueueMessageQueue(messages, callback)
}

func (bum *BattleUIManager) IsMessageFinished() bool {
	return bum.messageManager.IsFinished()
}

func (bum *BattleUIManager) ShowActionModal(vm core.ActionModalViewModel) {
	bum.actionModalManager.ShowActionModal(vm, bum.world, bum)
}

func (bum *BattleUIManager) HideActionModal() {
	bum.actionModalManager.HideActionModal()
}

func (bum *BattleUIManager) IsActionModalVisible() bool {
	return bum.actionModalManager.IsVisible()
}

func (bum *BattleUIManager) SetCurrentTarget(entityID donburi.Entity) {
	bum.targetIndicatorManager.SetCurrentTarget(entityID)
}

func (bum *BattleUIManager) ClearCurrentTarget() {
	bum.targetIndicatorManager.ClearCurrentTarget()
}

func (bum *BattleUIManager) SetAnimation(anim *component.ActionAnimationData) {
	bum.animationDrawer.SetAnimation(anim)
}

func (bum *BattleUIManager) ClearAnimation() {
	bum.animationDrawer.ClearAnimation()
}

func (bum *BattleUIManager) GetBattlefieldWidgetRect() image.Rectangle {
	return bum.battlefieldWidget.Container.GetWidget().Rect
}
