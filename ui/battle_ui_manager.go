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
	config           *data.Config // UIFactoryから取得するため追加
}

// NewBattleUIManager は BattleUIManager の新しいインスタンスを作成します。
func NewBattleUIManager(
	config *data.Config,
	resources *data.SharedResources,
	world donburi.World,
	partInfoProvider system.PartInfoProvider,
	rand *rand.Rand,
) *BattleUIManager {
	// Create a new internal UI event channel
	internalUIEventChannel := make(chan UIEvent, 10) // Buffer size 10

	uiFactory := NewUIFactory(config, resources.Font, resources.ModalButtonFont, resources.MessageWindowFont, resources.GameDataManager.Messages)
	ui := NewUI(config, internalUIEventChannel, uiFactory, resources.GameDataManager)
	messageManager := ui.GetMessageDisplayManager() // Get from ui instance
	viewModelFactory := NewViewModelFactory(world, &partInfoProvider, resources.GameDataManager, rand, ui)

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
		config:           config,
	}
}

// Update はUIの状態を更新します。
func (bum *BattleUIManager) Update(tickCount int, world donburi.World, battleLogic system.BattleLogic) []event.GameEvent {
	bum.ui.Update(tickCount)

	// UIイベントの処理
	uiGeneratedGameEvents := UpdateUIEventProcessorSystem(
		world, bum.ui, bum.messageManager, bum.ui.GetEventChannel(),
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

	bum.ui.SetBattleUIState(battleUIState) // Simplified signature

	return uiGeneratedGameEvents
}

// Draw はUI要素を描画します。
func (bum *BattleUIManager) Draw(screen *ebiten.Image, tickCount int, gameDataManager *data.GameDataManager) {
	screen.Fill(bum.config.UI.Colors.Background) // 背景はUIマネージャーで描画
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
