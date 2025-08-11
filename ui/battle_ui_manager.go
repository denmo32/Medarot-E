package ui

import (
	"image"
	"log"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ecs/system"
	"medarot-ebiten/event"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

// BattleUIManager はバトルシーンのUI要素の管理と描画を担当する唯一の司令塔です。
// UIInterfaceを実装します。
type BattleUIManager struct {
	config           *data.Config
	world            donburi.World
	uiFactory        *UIFactory
	viewModelFactory *viewModelFactoryImpl

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
	eventChannel          chan UIEvent
	battleUIState         *BattleUIState
	lastWidth, lastHeight int
}

// NewBattleUIManager は BattleUIManager の新しいインスタンスを作成します。
func NewBattleUIManager(
	config *data.Config,
	resources *data.SharedResources,
	world donburi.World,
	partInfoProvider system.PartInfoProviderInterface,
	rand *rand.Rand,
) *BattleUIManager {
	bum := &BattleUIManager{
		config:       config,
		world:        world,
		eventChannel: make(chan UIEvent, 10),
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

	// Initialize ViewModelFactory, which implements UIMediator
	// It needs a reference to the fully constructed BattleUIManager to fulfill the UIInterface contract.
	bum.viewModelFactory = NewViewModelFactory(world, partInfoProvider, resources.GameDataManager, rand, bum)

	// Initialize BattleUIStateComponent in the ECS world
	battleUIStateEntry := world.Entry(world.Create(BattleUIStateComponent))
	bum.battleUIState = BattleUIStateComponent.Get(battleUIStateEntry)
	bum.battleUIState.InfoPanels = make(map[string]core.InfoPanelViewModel)

	return bum
}

// Update はUI全体の状態を更新します。
func (bum *BattleUIManager) Update(tickCount int, world donburi.World, battleLogic system.BattleLogic) []event.GameEvent {
	bum.updateLayout()
	bum.ebitenui.Update()
	bum.animationDrawer.Update(float64(tickCount))
	bum.messageManager.Update()

	// UIイベントを処理し、対応するゲームイベントを取得
	uiGeneratedGameEvents := UpdateUIEventProcessorSystem(
		world, bum, bum.messageManager, bum.eventChannel,
	)

	// ViewModelを更新
	UpdateInfoPanelViewModelSystem(bum.battleUIState, world, battleLogic.GetPartInfoProvider(), bum.viewModelFactory)
	battlefieldViewModel, err := bum.viewModelFactory.BuildBattlefieldViewModel(world, bum.GetBattlefieldWidgetRect())
	if err != nil {
		log.Printf("Error building battlefield view model: %v", err)
	}
	bum.battleUIState.BattlefieldViewModel = battlefieldViewModel

	// 更新されたViewModelをUIウィジェットに設定
	bum.SetBattleUIState(bum.battleUIState)

	return uiGeneratedGameEvents
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

// updateLayout はUIのレイアウトを更新します。
func (bum *BattleUIManager) updateLayout() {
	rootRect := bum.ebitenui.Container.GetWidget().Rect
	width, height := rootRect.Dx(), rootRect.Dy()

	if bum.lastWidth == width && bum.lastHeight == height {
		return
	}
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

	if bum.battleUIState != nil {
		infoPanelVMs := make([]core.InfoPanelViewModel, 0, len(bum.battleUIState.InfoPanels))
		for _, vm := range bum.battleUIState.InfoPanels {
			infoPanelVMs = append(infoPanelVMs, vm)
		}
		bum.infoPanelManager.UpdatePanels(infoPanelVMs, mainUIContainer, battlefieldRect, bum.battleUIState.BattlefieldViewModel.Icons)
	}
}

// --- UIInterfaceの実装 ---

func (bum *BattleUIManager) GetViewModelFactory() *viewModelFactoryImpl {
	return bum.viewModelFactory
}

func (bum *BattleUIManager) SetBattleUIState(battleUIState *BattleUIState) {
	bum.battleUIState = battleUIState
	bum.battlefieldWidget.SetViewModel(battleUIState.BattlefieldViewModel)

	mainUIContainer := bum.ebitenui.Container.Children()[0].(*widget.Container).Children()[0].(*widget.Container)

	infoPanelVMs := make([]core.InfoPanelViewModel, 0, len(battleUIState.InfoPanels))
	for _, vm := range battleUIState.InfoPanels {
		infoPanelVMs = append(infoPanelVMs, vm)
	}

	bum.infoPanelManager.UpdatePanels(infoPanelVMs, mainUIContainer, bum.GetBattlefieldWidgetRect(), battleUIState.BattlefieldViewModel.Icons)
}

func (bum *BattleUIManager) PostEvent(event UIEvent) {
	bum.eventChannel <- event
}

func (bum *BattleUIManager) IsActionModalVisible() bool {
	return bum.actionModalManager.IsVisible()
}

func (bum *BattleUIManager) ShowActionModal(vm core.ActionModalViewModel) {
	bum.actionModalManager.ShowActionModal(vm)
}

func (bum *BattleUIManager) HideActionModal() {
	bum.actionModalManager.HideActionModal()
}

func (bum *BattleUIManager) GetActionTargetMap() map[core.PartSlotKey]core.ActionTarget {
	return bum.actionModalManager.GetActionTargetMap()
}

func (bum *BattleUIManager) SetCurrentTarget(entityID donburi.Entity) {
	bum.targetIndicatorManager.SetCurrentTarget(entityID)
}

func (bum *BattleUIManager) ClearCurrentTarget() {
	bum.targetIndicatorManager.ClearCurrentTarget()
}

func (bum *BattleUIManager) GetBattlefieldWidgetRect() image.Rectangle {
	return bum.battlefieldWidget.Container.GetWidget().Rect
}

func (bum *BattleUIManager) GetRootContainer() *widget.Container {
	return bum.ebitenui.Container
}

func (bum *BattleUIManager) SetAnimation(anim *component.ActionAnimationData) {
	bum.animationDrawer.SetAnimation(anim)
}

func (bum *BattleUIManager) IsAnimationFinished(tick int) bool {
	return bum.animationDrawer.IsAnimationFinished(float64(tick))
}

func (bum *BattleUIManager) ClearAnimation() {
	bum.animationDrawer.ClearAnimation()
}

func (bum *BattleUIManager) GetCurrentAnimationResult() component.ActionResult {
	return bum.animationDrawer.GetCurrentAnimationResult()
}

func (bum *BattleUIManager) GetMessageDisplayManager() *UIMessageDisplayManager {
	return bum.messageManager
}

func (bum *BattleUIManager) GetEventChannel() chan UIEvent {
	return bum.eventChannel
}

func (bum *BattleUIManager) GetConfig() *data.Config {
	return bum.config
}