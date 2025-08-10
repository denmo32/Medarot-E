package ui

import (
	"log"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"
		"medarot-ebiten/ecs/entity"
	"medarot-ebiten/ecs/system"
	"medarot-ebiten/event"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

type BattleScene struct {
	resources      *data.SharedResources
	manager        *SceneManager
	world          donburi.World
	tickCount      int
	debugMode      bool
	playerTeam     core.TeamID
	ui             UIInterface
	messageManager *UIMessageDisplayManager
	winner         core.TeamID

	gameDataManager        *data.GameDataManager
	rand                   *rand.Rand
	uiEventChannel         chan UIEvent
	battleUIState          *BattleUIState
	statusEffectSystem     *system.StatusEffectSystem
	postActionEffectSystem *system.PostActionEffectSystem
	viewModelFactory       *viewModelFactoryImpl // Changed from ViewModelFactory
	uiFactory              *UIFactory

	battleLogic *system.BattleLogic

	// New: Map of BattleStates
	battleStates map[core.GameState]system.BattleState
}

func NewBattleScene(res *data.SharedResources, manager *SceneManager) *BattleScene {
	world := donburi.NewWorld()

	bs := &BattleScene{
		resources:       res,
		manager:         manager,
		world:           world,
		debugMode:       true,
		playerTeam:      core.Team1,
		winner:          core.TeamNone,
		gameDataManager: res.GameDataManager,
		rand:            res.Rand,
		uiEventChannel:  make(chan UIEvent, 10),
		battleUIState:   &BattleUIState{},
	}

	entity.InitializeBattleWorld(bs.world, bs.resources, bs.playerTeam)

	// Initialize BattleLogic and its dependencies
	bs.battleLogic = system.NewBattleLogic(bs.world, &bs.resources.Config, bs.gameDataManager, bs.rand)
	bs.statusEffectSystem = system.NewStatusEffectSystem(bs.world, bs.battleLogic.GetDamageCalculator())
	bs.postActionEffectSystem = system.NewPostActionEffectSystem(bs.world, bs.statusEffectSystem, bs.gameDataManager, bs.battleLogic.GetPartInfoProvider())

	// Initialize UI related components
	bs.uiFactory = NewUIFactory(&bs.resources.Config, bs.resources.Font, bs.resources.ModalButtonFont, bs.resources.MessageWindowFont, bs.gameDataManager.Messages)
	bs.ui = NewUI(&bs.resources.Config, bs.uiEventChannel, bs.uiFactory, bs.gameDataManager)
	bs.messageManager = bs.ui.GetMessageDisplayManager() // Get the manager from the initialized UI
	bs.viewModelFactory = NewViewModelFactory(bs.world, bs.battleLogic.GetPartInfoProvider(), bs.gameDataManager, bs.rand, bs.ui)

	// Initialize BattleUIStateComponent
	battleUIStateEntry := bs.world.Entry(bs.world.Create(BattleUIStateComponent))
	if battleUIStateEntry.Valid() {
		BattleUIStateComponent.SetValue(battleUIStateEntry, BattleUIState{
			InfoPanels: make(map[string]core.InfoPanelViewModel),
		})
	}

	// Initialize BattleStates map
	bs.battleStates = map[core.GameState]system.BattleState{
		core.StateGaugeProgress:      &system.GaugeProgressState{},
		core.StatePlayerActionSelect: &system.PlayerActionSelectState{},
		core.StateActionExecution:    &system.ActionExecutionState{},
		core.StateAnimatingAction:    &system.AnimatingActionState{},
		core.StatePostAction:         &system.PostActionState{},
		core.StateMessage:            &system.MessageState{},
		core.StateGameOver:           &system.GameOverState{},
	}

	bs.SetState(core.StateGaugeProgress)

	return bs
}

func (bs *BattleScene) SetState(newState core.GameState) {
	gameStateEntry, ok := query.NewQuery(filter.Contains(component.GameStateComponent)).First(bs.world)
	if !ok {
		log.Panicln("GameStateComponent がワールドに見つかりません。")
	}
	component.GameStateComponent.Get(gameStateEntry).CurrentState = newState
}

func (bs *BattleScene) Update() error {
	bs.tickCount++
	gameStateEntry, ok := query.NewQuery(filter.Contains(component.GameStateComponent)).First(bs.world)
	if !ok {
		log.Panicln("GameStateComponent がワールドに見つかりません。")
	}
	currentGameStateComp := component.GameStateComponent.Get(gameStateEntry)

	var err error // Declare err here

	bs.ui.Update(bs.tickCount)

	uiGeneratedGameEvents := UpdateUIEventProcessorSystem(
		bs.world, bs.ui, bs.messageManager, bs.uiEventChannel,
	)

	allGameEvents := make([]event.GameEvent, 0)
	allGameEvents = append(allGameEvents, uiGeneratedGameEvents...)

	// Create BattleContext
	battleContext := &system.BattleContext{ // system. を追加
		World:                  bs.world,
		Config:                 &bs.resources.Config,
		GameDataManager:        bs.gameDataManager,
		Rand:                   bs.rand,
		Tick:                   bs.tickCount,
		UIMediator:             bs.viewModelFactory,
		StatusEffectSystem:     bs.statusEffectSystem,
		PostActionEffectSystem: bs.postActionEffectSystem,
		BattleLogic:            bs.battleLogic,
	}

	// Get current BattleState implementation and update it
	if currentStateImpl, ok := bs.battleStates[currentGameStateComp.CurrentState]; ok {
		tempGameEvents, err := currentStateImpl.Update(battleContext)
		if err != nil {
			log.Printf("Error updating game state %s: %v", currentGameStateComp.CurrentState, err)
		}
		allGameEvents = append(allGameEvents, tempGameEvents...)
	} else {
		log.Printf("Unknown game state: %s", currentGameStateComp.CurrentState)
	}

	// Process all collected game events and get state change requests
	stateChangeRequests := bs.processGameEvents(allGameEvents)

	// Apply state changes
	for _, req := range stateChangeRequests {
		if stateChangeReq, ok := req.(event.StateChangeRequestedGameEvent); ok {
			bs.SetState(stateChangeReq.NextState)
		}
	}

	battleUIStateEntry, ok := query.NewQuery(filter.Contains(BattleUIStateComponent)).First(bs.world)
	if !ok {
		log.Println("エラー: BattleUIStateComponent がワールドに見つかりません。UI更新をスキップします。")
		return nil
	}
	battleUIState := BattleUIStateComponent.Get(battleUIStateEntry)

	UpdateInfoPanelViewModelSystem(battleUIState, bs.world, bs.battleLogic.GetPartInfoProvider(), bs.viewModelFactory)

	battleUIState.BattlefieldViewModel, err = bs.viewModelFactory.BuildBattlefieldViewModel(bs.world, bs.ui.GetBattlefieldWidgetRect())
	if err != nil {
		log.Printf("Error building battlefield view model: %v", err)
		return err
	}

	bs.ui.SetBattleUIState(battleUIState, &bs.resources.Config, bs.ui.GetBattlefieldWidgetRect(), bs.uiFactory)

	return nil
}

func (bs *BattleScene) Draw(screen *ebiten.Image) {
	screen.Fill(bs.resources.Config.UI.Colors.Background)
	bs.ui.DrawBackground(screen)
	bs.ui.Draw(screen, bs.tickCount, bs.resources.GameDataManager)

	// GameStateComponentから現在のゲーム状態を取得
	gameStateEntry, ok := query.NewQuery(filter.Contains(component.GameStateComponent)).First(bs.world)
	if !ok {
		log.Panicln("GameStateComponent がワールドに見つかりません。")
	}
	currentGameStateComp := component.GameStateComponent.Get(gameStateEntry)

	// Draw current BattleState implementation
	if currentStateImpl, ok := bs.battleStates[currentGameStateComp.CurrentState]; ok {
		currentStateImpl.Draw(screen)
	} else {
		log.Printf("Unknown game state for drawing: %s", currentGameStateComp.CurrentState)
	}

}

func (bs *BattleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return bs.resources.Config.UI.Screen.Width, bs.resources.Config.UI.Screen.Height
}

// processGameEvents はGameEventのリストを処理し、BattleSceneの状態を更新します。
func (bs *BattleScene) processGameEvents(gameEvents []event.GameEvent) []event.GameEvent {
	var stateChangeEvents []event.GameEvent // 状態変更イベントを収集する新しいスライス

	lastActionResultEntry, ok := query.NewQuery(filter.Contains(component.LastActionResultComponent)).First(bs.world)
	if !ok {
		log.Panicln("LastActionResultComponent がワールドに見つかりません。")
	}
	lastActionResultComp := component.LastActionResultComponent.Get(lastActionResultEntry)

	for _, evt := range gameEvents {
		switch e := evt.(type) {
		case event.PlayerActionRequiredGameEvent:
			stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StatePlayerActionSelect})
		case event.ActionAnimationStartedGameEvent:
			bs.ui.PostEvent(SetAnimationUIEvent(e))
			stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StateAnimatingAction})

		case event.ActionAnimationFinishedGameEvent:
			*lastActionResultComp = e.Result // Store the result
			stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StatePostAction})
		case event.MessageDisplayRequestGameEvent:
			bs.messageManager.EnqueueMessageQueue(e.Messages, e.Callback)
			stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StateMessage})
		case event.MessageDisplayFinishedGameEvent:
			if bs.winner != core.TeamNone {
				stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StateGameOver})
			} else {
				stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StateGaugeProgress})
			}
		case event.GameOverGameEvent:
			bs.winner = e.Winner
			stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StateMessage})
		case event.HideActionModalGameEvent:
			bs.ui.PostEvent(HideActionModalUIEvent{})
		case event.ShowActionModalGameEvent:
			bs.ui.HideActionModal()
			select {
			case bs.ui.GetEventChannel() <- ShowActionModalUIEvent{ViewModel: e.ViewModel.(core.ActionModalViewModel)}: // core.ActionModalViewModel に変更
			default:
				log.Println("警告: ShowActionModalUIEvent の送信をスキップしました (チャネルがフルか重複)。")
			}
		case event.ClearAnimationGameEvent:
			bs.ui.PostEvent(ClearAnimationUIEvent{})
		case event.ClearCurrentTargetGameEvent:
			bs.ui.PostEvent(ClearCurrentTargetUIEvent{})
		case event.ChargeRequestedGameEvent:
			actingEntry := e.ActingEntry
			if actingEntry == nil || !actingEntry.Valid() {
				log.Printf("Error: ChargeRequestedGameEvent - ActingEntry is invalid or nil")
				break
			}
			var targetEntry *donburi.Entry
			if e.TargetEntry != nil && e.TargetEntry.Valid() {
				targetEntry = e.TargetEntry
			}
			if targetEntry == nil && e.TargetPartSlot != "" {
				log.Printf("Error: ChargeRequestedGameEvent - TargetEntry is nil but TargetPartSlot is provided")
				break
			}
			successful := bs.battleLogic.GetChargeInitiationSystem().StartCharge(actingEntry, e.SelectedSlotKey, targetEntry, e.TargetPartSlot)
			if !successful {
				log.Printf("エラー: %s の行動開始に失敗しました。", component.SettingsComponent.Get(actingEntry).Name)
			}
		case event.PlayerActionProcessedGameEvent:
			playerActionQueue := entity.GetPlayerActionQueueComponent(bs.world)
			// キューの先頭が処理されたエントリと一致するか確認
			if len(playerActionQueue.Queue) > 0 && playerActionQueue.Queue[0] == e.ActingEntry {
				// キューから削除
				playerActionQueue.Queue = playerActionQueue.Queue[1:]
			} else {
				log.Printf("警告: PlayerActionProcessedGameEvent の対象エントリがキューの先頭と一致しません。")
			}
			// 次の行動へ
			if len(playerActionQueue.Queue) > 0 {
				// まだ選択待ちのプレイヤーがいるので、再度選択状態へ
				stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StatePlayerActionSelect})
			} else {
				// 全員の選択が終わったのでゲージ進行へ
				stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StateGaugeProgress})
			}

		case event.PlayerActionSelectFinishedGameEvent:
			playerActionQueue := entity.GetPlayerActionQueueComponent(bs.world)
			if len(playerActionQueue.Queue) > 0 {
				stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StatePlayerActionSelect})
			} else {
				stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StateGaugeProgress})
			}

		case event.GoToTitleSceneGameEvent:
			bs.manager.GoToTitleScene()
		case event.StateChangeRequestedGameEvent:
			stateChangeEvents = append(stateChangeEvents, e)
		}
	}
	return stateChangeEvents
}
