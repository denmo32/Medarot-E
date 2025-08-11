package ui

import (
	"log"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/system"
	"medarot-ebiten/event"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// BattleUIManager はバトルシーンのUI要素の管理と描画を担当します。
type BattleUIManager struct {
	ui               UIInterface
	messageManager   *UIMessageDisplayManager
	viewModelFactory *viewModelFactoryImpl
	uiFactory        *UIFactory
	uiEventChannel   chan UIEvent
	config           *data.Config // UIFactoryから取得するため追加
}

// NewBattleUIManager は BattleUIManager の新しいインスタンスを作成します。
func NewBattleUIManager(
	config *data.Config,
	resources *data.SharedResources,
	gameDataManager *data.GameDataManager,
	world donburi.World,
	partInfoProvider system.PartInfoProvider,
	rand *rand.Rand,
	uiEventChannel chan UIEvent,
) *BattleUIManager {
	uiFactory := NewUIFactory(config, resources.Font, resources.ModalButtonFont, resources.MessageWindowFont, gameDataManager.Messages)
	ui := NewUI(config, uiEventChannel, uiFactory, gameDataManager)
	messageManager := ui.GetMessageDisplayManager()
	viewModelFactory := NewViewModelFactory(world, &partInfoProvider, gameDataManager, rand, ui)

	// Initialize BattleUIStateComponent
	battleUIStateEntry := world.Entry(world.Create(BattleUIStateComponent))
	if battleUIStateEntry.Valid() {
		BattleUIStateComponent.SetValue(battleUIStateEntry, BattleUIState{
			InfoPanels: make(map[string]core.InfoPanelViewModel),
		})
	}

	return &BattleUIManager{
		ui:               ui,
		messageManager:   messageManager,
		viewModelFactory: viewModelFactory,
		uiFactory:        uiFactory,
		uiEventChannel:   uiEventChannel,
		config:           config,
	}
}

// Update はUIの状態を更新します。
func (bum *BattleUIManager) Update(tickCount int, world donburi.World, battleLogic system.BattleLogic) []event.GameEvent {
	bum.ui.Update(tickCount)

	// UIイベントの処理
	uiGeneratedGameEvents := UpdateUIEventProcessorSystem(
		world, bum.ui, bum.messageManager, bum.uiEventChannel,
	)

	// InfoPanelViewModel の更新
	battleUIStateEntry, ok := query.NewQuery(filter.Contains(BattleUIStateComponent)).First(world)
	if !ok {
		log.Println("エラー: BattleUIStateComponent がワールドに見つかりません。UI更新をスキップします。")
		return uiGeneratedGameEvents // UIイベントは返す
	}
	battleUIState := BattleUIStateComponent.Get(battleUIStateEntry)
	UpdateInfoPanelViewModelSystem(battleUIState, world, battleLogic.GetPartInfoProvider(), bum.viewModelFactory)

	// BattlefieldViewModel の更新
	battlefieldViewModel, err := bum.viewModelFactory.BuildBattlefieldViewModel(world, bum.ui.GetBattlefieldWidgetRect())
	if err != nil {
		log.Printf("Error building battlefield view model: %v", err)
		// エラーハンドリング
	}
	battleUIState.BattlefieldViewModel = battlefieldViewModel

	bum.ui.SetBattleUIState(battleUIState, bum.config, bum.ui.GetBattlefieldWidgetRect(), bum.uiFactory)

	return uiGeneratedGameEvents
}

// Draw はUI要素を描画します。
func (bum *BattleUIManager) Draw(screen *ebiten.Image, tickCount int, gameDataManager *data.GameDataManager) {
	screen.Fill(bum.config.UI.Colors.Background) // 背景はUIマネージャーで描画
	bum.ui.DrawBackground(screen)
	bum.ui.Draw(screen, tickCount, gameDataManager)
}

// PostUIEvent はUIにイベントをポストします。
func (bum *BattleUIManager) PostUIEvent(evt UIEvent) {
	select {
	case bum.ui.GetEventChannel() <- evt:
	default:
		log.Println("警告: UIEvent の送信をスキップしました (チャネルがフルか重複)。")
	}
}

// GetMessageDisplayManager はメッセージ表示マネージャーを返します。
func (bum *BattleUIManager) GetMessageDisplayManager() *UIMessageDisplayManager {
	return bum.messageManager
}

// GetViewModelFactory は ViewModelFactory を返します。
func (bum *BattleUIManager) GetViewModelFactory() *viewModelFactoryImpl {
	return bum.viewModelFactory
}
