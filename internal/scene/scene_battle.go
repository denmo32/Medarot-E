package scene

import (
	"log"

	"medarot-ebiten/internal/battle"
	"medarot-ebiten/internal/game"
	"medarot-ebiten/internal/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

type BattleScene struct {
	resources                *SharedResources
	manager                  SceneManagerChanger
	world                    donburi.World
	tickCount                int
	debugMode                bool
	state                    game.GameState
	playerTeam               game.TeamID
	ui                       ui.UIInterface
	messageManager           *ui.UIMessageDisplayManager
	winner                   game.TeamID
	playerActionPendingQueue []*donburi.Entry
	battleLogic              *battle.BattleLogic
	uiEventChannel           chan game.UIEvent
	battleUIState            *ui.BattleUIState // 追加
	statusEffectSystem       *battle.StatusEffectSystem
	viewModelFactory         battle.ViewModelFactory // 追加
	animationManager         *battle.BattleAnimationManager

	// State Machine
	states       map[game.GameState]battle.BattleState
	currentState battle.BattleState
}

func NewBattleScene(res *SharedResources, manager SceneManagerChanger) *BattleScene {
	world := donburi.NewWorld()

	bs := &BattleScene{
		resources:                res,
		manager:                  manager,
		world:                    world,
		debugMode:                true,
		state:                    game.StatePlaying,
		playerTeam:               game.Team1,
		playerActionPendingQueue: make([]*donburi.Entry, 0),
		winner:                   game.TeamNone,
		uiEventChannel:           make(chan game.UIEvent, 10),
	}

	bs.battleLogic = battle.NewBattleLogic(bs.world, &bs.resources.Config, bs.resources.GameDataManager)
	bs.viewModelFactory = battle.NewViewModelFactory(&world, bs.battleLogic) // ViewModelFactoryを初期化
	bs.statusEffectSystem = battle.NewStatusEffectSystem(bs.world)
	battle.EnsureActionQueueEntity(bs.world)

	teamBuffsEntry := bs.world.Entry(bs.world.Create(game.TeamBuffsComponent))
	game.TeamBuffsComponent.SetValue(teamBuffsEntry, game.TeamBuffs{
		Buffs: make(map[game.TeamID]map[game.BuffType][]*game.BuffSource),
	})

	// Initialize BattleUIStateComponent
	battleUIStateEntry := bs.world.Entry(bs.world.Create(ui.BattleUIStateComponent))
	if battleUIStateEntry.Valid() {
		bs.battleUIState = &ui.BattleUIState{
			InfoPanels: make(map[string]ui.InfoPanelViewModel),
		}
		ui.BattleUIStateComponent.SetValue(battleUIStateEntry, *bs.battleUIState)
		log.Println("BattleUIStateComponent successfully created and initialized.")
	} else {
		log.Println("ERROR: Failed to create BattleUIStateComponent entry.")
	}

	game.CreateMedarotEntities(bs.world, res.GameData, bs.playerTeam, bs.resources.GameDataManager)
	bs.animationManager = battle.NewBattleAnimationManager(&bs.resources.Config)
	bs.ui = ui.NewUI(&bs.resources.Config, &bs.resources.UIConfig, bs.uiEventChannel, bs.resources.GameDataManager, bs.animationManager)
	// ui.goでuiFactoryが初期化され、ui.messageManagerもuiFactoryを使って初期化されるため、
	// ここでbs.messageManagerを直接初期化する必要はない。
	// bs.messageManager = NewUIMessageDisplayManager(&bs.resources.Config, bs.resources.GameDataManager.Font, bs.resources.GameDataManager.Messages, bs.ui.GetRootContainer())
	bs.messageManager = bs.ui.(*ui.UI).MessageManager // uiからmessageManagerを取得

	// Initialize state machine
	bs.states = map[game.GameState]battle.BattleState{
		game.StatePlaying:            &battle.PlayingState{},
		game.StatePlayerActionSelect: &battle.PlayerActionSelectState{},
		game.StateAnimatingAction:    &battle.AnimatingActionState{},
		game.StateMessage:            &battle.MessageState{},
		game.StateGameOver:           &battle.GameOverState{},
	}
	bs.currentState = bs.states[game.StatePlaying]

	return bs
}

func (bs *BattleScene) Update() error {
	bs.tickCount++
	bs.ui.Update()

	// UIイベントプロセッサシステムを更新
	var uiGeneratedGameEvents []game.GameEvent
	bs.playerActionPendingQueue, bs.state, uiGeneratedGameEvents = ui.UpdateUIEventProcessorSystem(
		bs.world, bs.battleLogic, bs.ui, bs.messageManager, bs.uiEventChannel, bs.playerActionPendingQueue, bs.state,
	)
	// UIイベントプロセッサから発行されたGameEventを処理
	bs.processGameEvents(uiGeneratedGameEvents)

	// メッセージ表示状態の場合、メッセージマネージャーを更新
	if bs.state == game.StateMessage {
		bs.messageManager.Update()
		if bs.messageManager.IsFinished() {
			// メッセージ表示が完了したら、MessageDisplayFinishedGameEvent を発行
			gameEvents := []game.GameEvent{game.MessageDisplayFinishedGameEvent{}}
			bs.processGameEvents(gameEvents) // 新しいヘルパー関数でイベントを処理
		}
	}

	// 現在のバトルステートを更新
	battleContext := &battle.BattleContext{
		World:                  bs.world,
		BattleLogic:            bs.battleLogic,
		UI:                     bs.ui,
		Config:                 &bs.resources.Config,
		GameDataManager:        bs.resources.GameDataManager, // 追加
		Tick:                   bs.tickCount,
		ViewModelFactory:       bs.viewModelFactory,
		BattleAnimationManager: bs.animationManager,
	}

	newPlayerActionPendingQueue, gameEvents, err := bs.currentState.Update(
		battleContext,
		bs.playerActionPendingQueue,
	)
	if err != nil {
		return err
	}
	bs.playerActionPendingQueue = newPlayerActionPendingQueue

	// Update status effect durations
	bs.statusEffectSystem.Update()

	// Process game events and transition state
	// nextState は processGameEvents 内で更新される
	bs.processGameEvents(gameEvents)

	// Additional state transitions not directly tied to a single GameEvent
	if bs.state == game.StatePlayerActionSelect && len(bs.playerActionPendingQueue) == 0 {
		bs.state = game.StatePlaying
		bs.currentState = bs.states[bs.state]
	} else if bs.state == game.StateAnimatingAction && bs.ui.IsAnimationFinished(bs.tickCount) {
		// この条件はActionAnimationFinishedGameEventで処理されるため、削除
	}

	// Transition to new state if changed (processGameEventsでbs.stateが更新されるため、このブロックは不要になる)
	// if nextState != bs.state {
	// 	bs.state = nextState
	// 	bs.currentState = bs.states[nextState]
	// }

	// Update UI components that depend on world state
	battleUIStateEntry, ok := query.NewQuery(filter.Contains(ui.BattleUIStateComponent)).First(bs.world)
	if !ok {
		log.Println("エラー: BattleUIStateComponent がワールドに見つかりません。UI更新をスキップします。")
		return nil
	}
	battleUIState := ui.BattleUIStateComponent.Get(battleUIStateEntry)

	battle.UpdateInfoPanelViewModelSystem(battleUIState, bs.world, bs.battleLogic, bs.viewModelFactory) // InfoPanelのViewModelを更新

	// BattlefieldViewModelを構築し、BattleUIStateに設定
	battleUIState.BattlefieldViewModel = bs.viewModelFactory.BuildBattlefieldViewModel(battleUIState, bs.battleLogic, &bs.resources.Config, bs.ui.GetBattlefieldWidgetRect())

	// UIにBattleUIState全体を渡して更新を委譲
	uiConfig := bs.resources.Config.UI.(ui.UIConfig)
	bs.ui.SetBattleUIState(battleUIState, &uiConfig, bs.ui.GetBattlefieldWidgetRect())

	return nil
}

func (bs *BattleScene) Draw(screen *ebiten.Image) {
	screen.Fill(bs.resources.Config.UI.Colors.Background)
	bs.ui.DrawBackground(screen)
	bs.ui.Draw(screen, bs.tickCount)
	// bs.battlefieldViewModel は不要になるため、直接 battleUIState.BattlefieldViewModel を渡す
	bs.ui.(*ui.UI).AnimationDrawer.Draw(screen, bs.tickCount, bs.battleUIState.BattlefieldViewModel, bs.ui.(*ui.UI).BattlefieldWidget)

	// 現在のステートに描画を委譲
	bs.currentState.Draw(screen)

}

func (bs *BattleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return bs.resources.Config.UI.Screen.Width, bs.resources.Config.UI.Screen.Height
}

// processGameEvents はGameEventのリストを処理し、BattleSceneの状態を更新します。
func (bs *BattleScene) processGameEvents(gameEvents []game.GameEvent) {
	for _, event := range gameEvents {
		switch e := event.(type) {
		case game.PlayerActionRequiredGameEvent:
			bs.state = game.StatePlayerActionSelect
		case game.ActionAnimationStartedGameEvent:
			bs.ui.PostEvent(game.SetAnimationUIEvent(e))
			bs.state = game.StateAnimatingAction
		case game.ActionAnimationFinishedGameEvent:
			// アニメーション終了後、クールダウン開始とターゲットクリア
			actingEntry := e.ActingEntry
			if actingEntry.Valid() && !game.StateComponent.Get(actingEntry).FSM.Is(string(game.StateBroken)) {
				battle.StartCooldownSystem(actingEntry, bs.world, bs.battleLogic)
			}
			bs.ui.PostEvent(game.ClearCurrentTargetUIEvent{})
			bs.state = game.StateMessage
		case game.MessageDisplayRequestGameEvent:
			bs.messageManager.EnqueueMessageQueue(e.Messages, e.Callback)
			bs.state = game.StateMessage
		case game.MessageDisplayFinishedGameEvent:
			if bs.winner != game.TeamNone {
				bs.state = game.StateGameOver
			} else {
				bs.state = game.StatePlaying
			}
		case game.GameOverGameEvent:
			bs.winner = e.Winner
			bs.state = game.StateMessage // ゲームオーバーメッセージ表示のため
		case game.HideActionModalGameEvent:
			bs.ui.PostEvent(game.HideActionModalUIEvent{})
		case game.ShowActionModalGameEvent:
			bs.ui.PostEvent(game.ShowActionModalUIEvent(e))
		case game.ClearAnimationGameEvent:
			bs.ui.PostEvent(game.ClearAnimationUIEvent{})
		case game.ClearCurrentTargetGameEvent:
			bs.ui.PostEvent(game.ClearCurrentTargetUIEvent{})
		case game.ChargeRequestedGameEvent:
			// ChargeInitiationSystem を呼び出す
			successful := battle.StartCharge(e.ActingEntry, e.SelectedSlotKey, e.TargetEntry, e.TargetPartSlot, bs.world, bs.battleLogic)
			if !successful {
				log.Printf("エラー: %s の行動開始に失敗しました。", game.SettingsComponent.Get(e.ActingEntry).Name)
				// 必要であれば、ここでエラーメッセージをキューに入れるなどの処理を追加
			}
		case game.ActionCanceledGameEvent:
			// 行動キャンセル時の処理
			bs.state = game.StatePlaying // キャンセル時は即座にPlaying状態に戻る
		case game.GameOverTransitionGameEvent:
			bs.manager.GoToTitleScene()
		}
	}
	// イベント処理後に現在の状態を更新
	bs.currentState = bs.states[bs.state]
}
