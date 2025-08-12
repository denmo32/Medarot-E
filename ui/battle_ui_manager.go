package ui

import (
	"image"
	"log"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/event"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
)

// BattleUIManager はバトルシーンのUI要素の管理と描画を担当する唯一の司令塔です。
// 以前のサブマネージャー(ActionModal, MessageDisplay, TargetIndicator)の責務を統合しています。
type BattleUIManager struct {
	config    *data.Config
	world     donburi.World
	uiFactory *UIFactory

	// ebitenui root
	ebitenui *ebitenui.UI

	// Sub-managers (Complex ones that remain)
	infoPanelManager *InfoPanelManager
	animationDrawer  *UIAnimationDrawer

	// Widgets
	battlefieldWidget *BattlefieldWidget
	commonBottomPanel *UIPanel

	// --- State from UIActionModalManager ---
	actionModal          widget.PreferredSizeLocateableWidget
	isActionModalVisible bool

	// --- State from UIMessageDisplayManager ---
	messageWindow       widget.PreferredSizeLocateableWidget
	messageQueue        []string
	currentMessageIndex int
	postMessageCallback func()

	// --- State from UITargetIndicatorManager ---
	currentTarget donburi.Entity

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
		config:               config,
		world:                world,
		eventChannel:         make(chan event.GameEvent, 10),
		messageQueue:         make([]string, 0),
		isActionModalVisible: false,
	}

	bum.uiFactory = NewUIFactory(config, resources.Font, resources.ModalButtonFont, resources.MessageWindowFont, resources.GameDataManager.Messages)

	// Initialize sub-managers that remain
	bum.infoPanelManager = NewInfoPanelManager(config, bum.uiFactory)
	bum.animationDrawer = NewUIAnimationDrawer(config, bum.uiFactory.Font, bum.eventChannel)

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

	return bum
}

// Update はUI全体の状態を更新します。
func (bum *BattleUIManager) Update(tickCount int) []event.GameEvent {
	bum.ebitenui.Update()
	bum.animationDrawer.Update(float64(tickCount))

	// --- Logic from UIMessageDisplayManager.Update ---
	if len(bum.messageQueue) > 0 && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		bum.currentMessageIndex++
		if bum.currentMessageIndex < len(bum.messageQueue) {
			bum.showCurrentMessage()
		} else {
			bum.hideMessageWindow()
			if bum.postMessageCallback != nil {
				bum.postMessageCallback()
				bum.postMessageCallback = nil
			}
			bum.messageQueue = make([]string, 0) // メッセージキューをクリア
		}
	}
	// --- End of logic from UIMessageDisplayManager.Update ---

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
	if bum.currentTarget != 0 && bum.battlefieldWidget.viewModel != nil {
		for _, iconVM := range bum.battlefieldWidget.viewModel.Icons {
			if iconVM.EntryID == bum.currentTarget {
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

// --- Message Display Methods (from UIMessageDisplayManager) ---

func (bum *BattleUIManager) EnqueueMessage(msg string, callback func()) {
	bum.EnqueueMessageQueue([]string{msg}, callback)
}

func (bum *BattleUIManager) EnqueueMessageQueue(messages []string, callback func()) {
	bum.messageQueue = messages
	bum.currentMessageIndex = 0
	bum.postMessageCallback = callback
	bum.showCurrentMessage()
}

func (bum *BattleUIManager) IsMessageFinished() bool {
	return len(bum.messageQueue) == 0 && bum.messageWindow == nil
}

func (bum *BattleUIManager) showCurrentMessage() {
	if len(bum.messageQueue) > 0 {
		bum.showMessageWindow(bum.messageQueue[bum.currentMessageIndex])
	}
}

func (bum *BattleUIManager) showMessageWindow(message string) {
	if bum.messageWindow != nil {
		bum.hideMessageWindow()
	}
	win := createMessageWindow(message, bum.uiFactory)
	bum.messageWindow = win
	bum.commonBottomPanel.SetContent(bum.messageWindow)
}

func (bum *BattleUIManager) hideMessageWindow() {
	if bum.messageWindow != nil {
		bum.commonBottomPanel.SetContent(nil)
		bum.messageWindow = nil
	}
}

// --- Action Modal Methods (from UIActionModalManager) ---

func (bum *BattleUIManager) ShowActionModal(vm any) {
	bum.isActionModalVisible = true
	modal := createActionModalUI(vm.(*core.ActionModalViewModel), bum.uiFactory, bum.eventChannel, bum.world, bum)
	bum.actionModal = modal
	bum.commonBottomPanel.SetContent(bum.actionModal)
	log.Println("アクションモーダルを表示しました。")
}

func (bum *BattleUIManager) HideActionModal() {
	if !bum.isActionModalVisible {
		return
	}
	if bum.actionModal != nil {
		bum.commonBottomPanel.SetContent(nil)
		bum.actionModal = nil
	}
	bum.isActionModalVisible = false
	log.Println("アクションモーダルを非表示にしました。")
}

func (bum *BattleUIManager) IsActionModalVisible() bool {
	return bum.isActionModalVisible
}

// --- Target Indicator Methods (from UITargetIndicatorManager) ---

func (bum *BattleUIManager) SetCurrentTarget(entityID donburi.Entity) {
	bum.currentTarget = entityID
}

func (bum *BattleUIManager) ClearCurrentTarget() {
	bum.currentTarget = 0 // donburi.Entity のゼロ値
}

func (bum *BattleUIManager) GetCurrentTarget() donburi.Entity {
	return bum.currentTarget
}

// --- Animation Methods ---

func (bum *BattleUIManager) SetAnimation(anim *component.ActionAnimationData) {
	bum.animationDrawer.SetAnimation(anim)
}

func (bum *BattleUIManager) ClearAnimation() {
	bum.animationDrawer.ClearAnimation()
}

// --- Other Public Methods ---

func (bum *BattleUIManager) GetBattlefieldWidgetRect() image.Rectangle {
	return bum.battlefieldWidget.Container.GetWidget().Rect
}
