package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

type BattleScene struct {
	resources                *SharedResources
	manager                  *SceneManager
	world                    donburi.World
	tickCount                int
	debugMode                bool
	state                    GameState
	playerTeam               TeamID
	ui                       UIInterface
	messageManager           *UIMessageDisplayManager
	winner                   TeamID
	playerActionPendingQueue []*donburi.Entry
	battleLogic              *BattleLogic
	uiEventChannel           chan UIEvent
	battleUIState            *BattleUIState // 追加
	statusEffectSystem       *StatusEffectSystem
	postActionEffectSystem   *PostActionEffectSystem
	viewModelFactory         ViewModelFactory // 追加
	uiFactory                *UIFactory       // 追加

	// State Machine
	states       map[GameState]BattleState
	currentState BattleState
}

func NewBattleScene(res *SharedResources, manager *SceneManager) *BattleScene {
	world := donburi.NewWorld()

	bs := &BattleScene{
		resources:                res,
		manager:                  manager,
		world:                    world,
		debugMode:                true,
		state:                    StatePlaying,
		playerTeam:               Team1,
		playerActionPendingQueue: make([]*donburi.Entry, 0),
		winner:                   TeamNone,
		uiEventChannel:           make(chan UIEvent, 10),
		battleUIState:            &BattleUIState{}, // ← これを追加
	}

	bs.battleLogic = NewBattleLogic(bs.world, &bs.resources.Config, bs.resources.GameDataManager)
	bs.viewModelFactory = NewViewModelFactory(bs.world, bs.battleLogic)
	bs.uiFactory = NewUIFactory(&bs.resources.Config, bs.resources.GameDataManager.Font, bs.resources.GameDataManager.Messages)
	bs.statusEffectSystem = NewStatusEffectSystem(bs.world)
bs.postActionEffectSystem = NewPostActionEffectSystem(bs.world, bs.statusEffectSystem, bs.resources.GameDataManager, bs.battleLogic.GetPartInfoProvider())

	InitializeBattleWorld(bs.world, bs.resources, bs.playerTeam)

	animationManager := NewBattleAnimationManager(&bs.resources.Config)
	bs.ui = NewUI(&bs.resources.Config, bs.uiEventChannel, animationManager, bs.uiFactory, bs.resources.GameDataManager, bs.world)
	bs.messageManager = bs.ui.GetMessageDisplayManager() // uiからmessageManagerを取得

	// Initialize state machine
	bs.states = map[GameState]BattleState{
		StatePlaying:            &PlayingState{},
		StatePlayerActionSelect: &PlayerActionSelectState{},
		StateAnimatingAction:    &AnimatingActionState{},
		StateMessage:            &MessageState{},
		StateGameOver:           &GameOverState{},
	}
	bs.currentState = bs.states[StatePlaying]

	return bs
}

func (bs *BattleScene) Update() error {
	bs.tickCount++
	bs.ui.Update()

	// アニメーション再生中の場合、アニメーションの終了をチェック
	if bs.state == StateAnimatingAction && bs.ui.IsAnimationFinished(bs.tickCount) {
		// currentAnimationがnilでないことを確認してから結果を取得
		if bs.ui.(*UI).animationDrawer.animationManager.currentAnimation != nil {
			result := bs.ui.GetCurrentAnimationResult()
			gameEvents := []GameEvent{
				ClearAnimationGameEvent{},
				MessageDisplayRequestGameEvent{Messages: buildActionLogMessagesFromActionResult(result, bs.resources.GameDataManager), Callback: func() {
					UpdateHistorySystem(bs.world, &result)
				}},
				ActionAnimationFinishedGameEvent{Result: result, ActingEntry: result.ActingEntry},
			}

			bs.processGameEvents(gameEvents)
			return nil // アニメーション終了イベントを処理したので、このフレームの更新は終了
		}
	}

	// UIイベントプロセッサシステムを更新
	var uiGeneratedGameEvents []GameEvent
	bs.playerActionPendingQueue, bs.state, uiGeneratedGameEvents = UpdateUIEventProcessorSystem(
		bs.world, bs.ui, bs.messageManager, bs.uiEventChannel, bs.playerActionPendingQueue, bs.state,
	)
	// UIイベントプロセッサから発行されたGameEventを処理
	bs.processGameEvents(uiGeneratedGameEvents)

	// メッセージ表示状態の場合、メッセージマネージャーを更新
	if bs.state == StateMessage {
		bs.messageManager.Update()
		if bs.messageManager.IsFinished() {
			// メッセージ表示が完了したら、MessageDisplayFinishedGameEvent を発行
			gameEvents := []GameEvent{MessageDisplayFinishedGameEvent{}}
			bs.processGameEvents(gameEvents) // 新しいヘルパー関数でイベントを処理
		}
	}

	// 現在のバトルステートを更新
	battleContext := &BattleContext{
		World:            bs.world,
		BattleLogic:      bs.battleLogic,
		Config:           &bs.resources.Config,
		GameDataManager:  bs.resources.GameDataManager,
		Tick:             bs.tickCount,
		ViewModelFactory: bs.viewModelFactory,
		statusEffectSystem: bs.statusEffectSystem,
		postActionEffectSystem: bs.postActionEffectSystem,
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
	if bs.state == StatePlayerActionSelect && len(bs.playerActionPendingQueue) == 0 {
		bs.state = StatePlaying
		bs.currentState = bs.states[bs.state]
	}

	// Transition to new state if changed (processGameEventsでbs.stateが更新されるため、このブロックは不要になる)
	// if nextState != bs.state {
	// 	bs.state = nextState
	// 	bs.currentState = bs.states[nextState]
	// }

	// Update UI components that depend on world state
	battleUIStateEntry, ok := query.NewQuery(filter.Contains(BattleUIStateComponent)).First(bs.world)
	if !ok {
		log.Println("エラー: BattleUIStateComponent がワールドに見つかりません。UI更新をスキップします。")
		return nil
	}
	battleUIState := BattleUIStateComponent.Get(battleUIStateEntry)

	UpdateInfoPanelViewModelSystem(battleUIState, bs.world, bs.battleLogic, bs.viewModelFactory) // InfoPanelのViewModelを更新

	// BattlefieldViewModelを構築し、BattleUIStateに設定
	battleUIState.BattlefieldViewModel = bs.viewModelFactory.BuildBattlefieldViewModel(bs.world, battleUIState, bs.battleLogic, &bs.resources.Config, bs.ui.GetBattlefieldWidgetRect()) // worldを渡す

	// UIにBattleUIState全体を渡して更新を委譲
	bs.ui.SetBattleUIState(battleUIState, &bs.resources.Config, bs.ui.GetBattlefieldWidgetRect(), bs.uiFactory)

	return nil
}

func (bs *BattleScene) Draw(screen *ebiten.Image) {
	screen.Fill(bs.resources.Config.UI.Colors.Background)
	bs.ui.DrawBackground(screen)
	bs.ui.Draw(screen, bs.tickCount, bs.resources.GameDataManager)
	// bs.battlefieldViewModel は不要になるため、直接 battleUIState.BattlefieldViewModel を渡す
	bs.ui.(*UI).animationDrawer.Draw(screen, bs.tickCount, bs.battleUIState.BattlefieldViewModel, bs.ui.(*UI).battlefieldWidget)

	// 現在のステートに描画を委譲
	bs.currentState.Draw(screen)

}

func (bs *BattleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return bs.resources.Config.UI.Screen.Width, bs.resources.Config.UI.Screen.Height
}

// processGameEvents はGameEventのリストを処理し、BattleSceneの状態を更新します。
func (bs *BattleScene) processGameEvents(gameEvents []GameEvent) {
	for _, event := range gameEvents {
		switch e := event.(type) {
		case PlayerActionRequiredGameEvent:
			bs.state = StatePlayerActionSelect
		case ActionAnimationStartedGameEvent:
			bs.ui.PostEvent(SetAnimationUIEvent(e))
			bs.state = StateAnimatingAction
		case ActionAnimationFinishedGameEvent:
			// アニメーション終了後、クールダウン開始とターゲットクリア
			actingEntry := e.ActingEntry
			if actingEntry.Valid() && StateComponent.Get(actingEntry).CurrentState != StateBroken {
				StartCooldownSystem(actingEntry, bs.world, bs.battleLogic)
			}
			bs.ui.PostEvent(ClearCurrentTargetUIEvent{})
			bs.state = StateMessage
		case MessageDisplayRequestGameEvent:
			bs.messageManager.EnqueueMessageQueue(e.Messages, e.Callback)
			bs.state = StateMessage
		case MessageDisplayFinishedGameEvent:
			if bs.winner != TeamNone {
				bs.state = StateGameOver
			} else {
				bs.state = StatePlaying
			}
		case GameOverGameEvent:
			bs.winner = e.Winner
			bs.state = StateMessage // ゲームオーバーメッセージ表示のため
		case HideActionModalGameEvent:
			bs.ui.PostEvent(HideActionModalUIEvent{})
		case ShowActionModalGameEvent:
			// アクションモーダルが既に表示されている場合は無視
			if !bs.ui.IsActionModalVisible() {
				bs.ui.PostEvent(ShowActionModalUIEvent(e))
			}
		case ClearAnimationGameEvent:
			bs.ui.PostEvent(ClearAnimationUIEvent{})
		case ClearCurrentTargetGameEvent:
			bs.ui.PostEvent(ClearCurrentTargetUIEvent{})
		case ChargeRequestedGameEvent:
			actingEntry := e.ActingEntry
			if actingEntry == nil || !actingEntry.Valid() {
				log.Printf("Error: ChargeRequestedGameEvent - ActingEntry is invalid or nil")
				break
			}
			var targetEntry *donburi.Entry
			if e.TargetEntry != nil && e.TargetEntry.Valid() {
				targetEntry = e.TargetEntry
			}
			if targetEntry == nil && e.TargetPartSlot != "" { // TargetPartSlotがあるのにTargetEntryがない場合はエラー
				log.Printf("Error: ChargeRequestedGameEvent - TargetEntry is nil but TargetPartSlot is provided")
				break
			}
			// ChargeInitiationSystem を呼び出す
			successful := StartCharge(actingEntry, e.SelectedSlotKey, targetEntry, e.TargetPartSlot, bs.world, bs.battleLogic, bs.resources.GameDataManager)
			if !successful {
				log.Printf("エラー: %s の行動開始に失敗しました。", SettingsComponent.Get(actingEntry).Name)
				// 必要であれば、ここでエラーメッセージをキューに入れるなどの処理を追加
			}
		case PlayerActionProcessedGameEvent:
			actingEntry := e.ActingEntry
			if actingEntry == nil || !actingEntry.Valid() {
				log.Printf("Error: PlayerActionProcessedGameEvent - ActingEntry is invalid or nil")
				break
			}
			// プレイヤーの行動キューから現在のエンティティを削除
			bs.playerActionPendingQueue = bs.playerActionPendingQueue[1:]
			bs.state = StatePlaying // 行動処理後はPlaying状態に戻る
		case ActionCanceledGameEvent:
			actingEntry := e.ActingEntry
			if actingEntry == nil || !actingEntry.Valid() {
				log.Printf("Error: ActionCanceledGameEvent - ActingEntry is invalid or nil")
				break
			}
			// 行動キャンセル時の処理（PlayerActionProcessedGameEventでキュー操作は行われる）
			bs.state = StatePlaying // キャンセル時は即座にPlaying状態に戻る
		case GoToTitleSceneGameEvent:
			bs.manager.GoToTitleScene()
		}
	}
	// イベント処理後に現在の状態を更新
	bs.currentState = bs.states[bs.state]
}
