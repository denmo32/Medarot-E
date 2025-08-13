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
// TargetManagerインターフェースを実装します。
type BattleUIManager struct {
	config    *data.Config
	world     donburi.World
	uiFactory *UIFactory

	// ebitenui root
	ebitenui *ebitenui.UI

	// Sub-managers and components
	infoPanelManager *InfoPanelManager
	animationDrawer  *UIAnimationDrawer
	actionModal      *ActionModal
	messageWindow    *MessageWindow // 構造体へのポインタに変更

	// Widgets
	battlefieldWidget *BattlefieldWidget
	commonBottomPanel *UIPanel

	// --- State for Message Queue ---
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
		config:       config,
		world:        world,
		eventChannel: make(chan event.GameEvent, 10),
		messageQueue: make([]string, 0),
	}

	bum.uiFactory = NewUIFactory(config, resources.Font, resources.ModalButtonFont, resources.MessageWindowFont, resources.GameDataManager.Messages)

	// Initialize sub-managers and components
	bum.infoPanelManager = NewInfoPanelManager(config, bum.uiFactory)
	bum.animationDrawer = NewUIAnimationDrawer(config, bum.uiFactory.Font, bum.eventChannel)
	bum.actionModal = NewActionModal(bum.uiFactory, bum.eventChannel, bum.world, bum)
	bum.messageWindow = NewMessageWindow(bum.uiFactory)

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

	// --- Message Queue Logic ---
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

// --- Message Display Methods ---

func (bum *BattleUIManager) EnqueueMessageQueue(messages []string, callback func()) {
	bum.messageQueue = messages
	bum.currentMessageIndex = 0
	bum.postMessageCallback = callback
	bum.showCurrentMessage()
}

func (bum *BattleUIManager) IsMessageFinished() bool {
	return len(bum.messageQueue) == 0 && !bum.messageWindow.IsVisible()
}

func (bum *BattleUIManager) showCurrentMessage() {
	if len(bum.messageQueue) > 0 {
		bum.showMessageWindow(bum.messageQueue[bum.currentMessageIndex])
	}
}

func (bum *BattleUIManager) showMessageWindow(message string) {
	bum.messageWindow.SetMessage(message)
	bum.commonBottomPanel.SetContent(bum.messageWindow.Widget())
}

func (bum *BattleUIManager) hideMessageWindow() {
	bum.messageWindow.Hide()
	bum.commonBottomPanel.SetContent(nil)
}

// --- Action Modal Methods ---

func (bum *BattleUIManager) ShowActionModal(vm any) {
	bum.actionModal.Show(vm.(*core.ActionModalViewModel))
	bum.commonBottomPanel.SetContent(bum.actionModal.Widget())
	log.Println("アクションモーダルを表示しました。")
}

func (bum *BattleUIManager) HideActionModal() {
	if !bum.actionModal.IsVisible() {
		return
	}
	bum.actionModal.Hide()
	bum.commonBottomPanel.SetContent(nil)
	log.Println("アクションモーダルを非表示にしました。")
}

func (bum *BattleUIManager) IsActionModalVisible() bool {
	return bum.actionModal.IsVisible()
}

// --- Target Indicator Methods (TargetManager interface implementation) ---

func (bum *BattleUIManager) SetCurrentTarget(entityID donburi.Entity) {
	bum.currentTarget = entityID
}

func (bum *BattleUIManager) ClearCurrentTarget() {
	bum.currentTarget = 0 // donburi.Entity のゼロ値
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
