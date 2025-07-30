package main

import (
	"log"
	"math/rand"

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
	playerTeam               TeamID
	ui                       UIInterface
	messageManager           *UIMessageDisplayManager
	winner                   TeamID
	playerActionPendingQueue []*donburi.Entry
	gameDataManager          *GameDataManager
	rand                     *rand.Rand
	uiEventChannel           chan UIEvent
	battleUIState            *BattleUIState
	statusEffectSystem       *StatusEffectSystem
	postActionEffectSystem   *PostActionEffectSystem
	viewModelFactory         ViewModelFactory
	uiFactory                *UIFactory

	lastActionResult *ActionResult

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
		playerTeam:               Team1,
		playerActionPendingQueue: make([]*donburi.Entry, 0),
		winner:                   TeamNone,
		gameDataManager:          res.GameDataManager,
		rand:                     res.Rand,
		uiEventChannel:           make(chan UIEvent, 10),
		battleUIState:            &BattleUIState{}, // ← これを追加
	}

	InitializeBattleWorld(bs.world, bs.resources, bs.playerTeam)

	// BattleLogicComponentからBattleLogicを取得
	battleLogicEntry, ok := query.NewQuery(filter.Contains(BattleLogicComponent)).First(bs.world)
	if !ok {
		log.Panicln("BattleLogicComponent がワールドに見つかりません。")
	}
	battleLogic := BattleLogicComponent.Get(battleLogicEntry)

	bs.uiFactory = NewUIFactory(&bs.resources.Config, bs.resources.GameDataManager.Font, bs.resources.GameDataManager.Messages)
	bs.ui = NewUI(&bs.resources.Config, bs.uiEventChannel, bs.uiFactory, bs.resources.GameDataManager)
	bs.viewModelFactory = NewViewModelFactory(bs.world, battleLogic.GetPartInfoProvider(), bs.gameDataManager, bs.rand, bs.ui)
	bs.statusEffectSystem = NewStatusEffectSystem(bs.world, battleLogic.GetDamageCalculator())
	bs.postActionEffectSystem = NewPostActionEffectSystem(bs.world, bs.statusEffectSystem, bs.gameDataManager, battleLogic.GetPartInfoProvider())
	bs.messageManager = bs.ui.GetMessageDisplayManager() // uiからmessageManagerを取得

	// Initialize state machine
	bs.states = map[GameState]BattleState{
		StatePlaying:            &PlayingState{},
		StatePlayerActionSelect: &PlayerActionSelectState{},
		StateAnimatingAction:    &AnimatingActionState{},
		StateMessage:            &MessageState{},
		StateGameOver:           &GameOverState{},
	}
	// 初期状態はGameStateComponentから取得
	gameStateEntry, ok := query.NewQuery(filter.Contains(GameStateComponent)).First(bs.world)
	if !ok {
		log.Panicln("GameStateComponent がワールドに見つかりません。")
	}
	initialGameState := GameStateComponent.Get(gameStateEntry).CurrentState
	bs.currentState = bs.states[initialGameState]

	return bs
}

func (bs *BattleScene) Update() error {
	bs.tickCount++
	bs.ui.Update()

	// GameStateComponentから現在のゲーム状態を取得
	gameStateEntry, ok := query.NewQuery(filter.Contains(GameStateComponent)).First(bs.world)
	if !ok {
		log.Panicln("GameStateComponent がワールドに見つかりません。")
	}
	currentGameState := GameStateComponent.Get(gameStateEntry).CurrentState

	// BattleLogicComponentからBattleLogicを取得 (スコープを広げる)
	battleLogicEntry, ok := query.NewQuery(filter.Contains(BattleLogicComponent)).First(bs.world)
	if !ok {
		log.Panicln("BattleLogicComponent がワールドに見つかりません。")
	}
	battleLogic := BattleLogicComponent.Get(battleLogicEntry)

	// UIイベントプロセッサシステムを更新
	var uiGeneratedGameEvents []GameEvent
	bs.playerActionPendingQueue, uiGeneratedGameEvents = UpdateUIEventProcessorSystem(
		bs.world, bs.ui, bs.messageManager, bs.uiEventChannel, bs.playerActionPendingQueue,
	)
	// UIイベントプロセッサから発行されたGameEventを処理
	bs.processGameEvents(uiGeneratedGameEvents)

	// メッセージ表示状態の場合、メッセージマネージャーを更新
	if currentGameState == StateMessage {
		// メッセージ状態に遷移した直後のフレームで、メッセージ表示の準備と関連ロジックを実行
		if bs.lastActionResult != nil {
			result := *bs.lastActionResult
			actingEntry := result.ActingEntry

			// クールダウン開始とターゲットクリア
			if actingEntry.Valid() && StateComponent.Get(actingEntry).CurrentState != StateBroken {
				StartCooldownSystem(actingEntry, bs.world, battleLogic.GetPartInfoProvider())
			}
			bs.ui.PostEvent(ClearCurrentTargetUIEvent{})

			// メッセージ表示を要求
			bs.messageManager.EnqueueMessageQueue(buildActionLogMessagesFromActionResult(result, bs.resources.GameDataManager), func() {
				UpdateHistorySystem(bs.world, &result)
			})

			bs.lastActionResult = nil // 処理が完了したのでクリア
		}

		bs.messageManager.Update()
		if bs.messageManager.IsFinished() {
			// メッセージ表示が完了したら、MessageDisplayFinishedGameEvent を発行
			gameEvents := []GameEvent{MessageDisplayFinishedGameEvent{}}
			bs.processGameEvents(gameEvents) // 新しいヘルパー関数でイベントを処理
		}
	}

	// アニメーション中またはメッセージ表示中はゲーム進行ロジックを停止
	if currentGameState != StateMessage && currentGameState != StateAnimatingAction {
		// 現在のバトルステートを更新
		battleContext := &BattleContext{
			World:                  bs.world,
			Config:                 &bs.resources.Config,
			GameDataManager:        bs.gameDataManager,
			Rand:                   bs.rand,
			Tick:                   bs.tickCount,
			ViewModelFactory:       bs.viewModelFactory,
			statusEffectSystem:     bs.statusEffectSystem,
			postActionEffectSystem: bs.postActionEffectSystem,
			BattleLogic:            battleLogic,
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
	}

	// Additional state transitions not directly tied to a single GameEvent
	if currentGameState == StatePlayerActionSelect && len(bs.playerActionPendingQueue) == 0 {
		// GameStateComponentを更新
		GameStateComponent.Get(gameStateEntry).CurrentState = StatePlaying
		bs.currentState = bs.states[StatePlaying]
	}

	// Update UI components that depend on world state
	battleUIStateEntry, ok := query.NewQuery(filter.Contains(BattleUIStateComponent)).First(bs.world)
	if !ok {
		log.Println("エラー: BattleUIStateComponent がワールドに見つかりません。UI更新をスキップします。")
		return nil
	}
	battleUIState := BattleUIStateComponent.Get(battleUIStateEntry)

	UpdateInfoPanelViewModelSystem(battleUIState, bs.world, battleLogic.GetPartInfoProvider(), bs.viewModelFactory) // InfoPanelのViewModelを更新

	// BattlefieldViewModelを構築し、BattleUIStateに設定
	battleUIState.BattlefieldViewModel = bs.viewModelFactory.BuildBattlefieldViewModel(bs.world, battleUIState, battleLogic.GetPartInfoProvider(), &bs.resources.Config, bs.ui.GetBattlefieldWidgetRect()) // worldを渡す

	// UIにBattleUIState全体を渡して更新を委譲
	bs.ui.SetBattleUIState(battleUIState, &bs.resources.Config, bs.ui.GetBattlefieldWidgetRect(), bs.uiFactory)

	return nil
}

func (bs *BattleScene) Draw(screen *ebiten.Image) {
	screen.Fill(bs.resources.Config.UI.Colors.Background)
	bs.ui.DrawBackground(screen)
	bs.ui.Draw(screen, bs.tickCount, bs.resources.GameDataManager)

	// 現在のステートに描画を委譲
	bs.currentState.Draw(screen)

}

func (bs *BattleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return bs.resources.Config.UI.Screen.Width, bs.resources.Config.UI.Screen.Height
}

// processGameEvents はGameEventのリストを処理し、BattleSceneの状態を更新します。
func (bs *BattleScene) processGameEvents(gameEvents []GameEvent) {
	gameStateEntry, ok := query.NewQuery(filter.Contains(GameStateComponent)).First(bs.world)
	if !ok {
		log.Panicln("GameStateComponent がワールドに見つかりません。")
	}
	currentGameStateComp := GameStateComponent.Get(gameStateEntry)

	for _, event := range gameEvents {
		switch e := event.(type) {
		case PlayerActionRequiredGameEvent:
			currentGameStateComp.CurrentState = StatePlayerActionSelect
		case ActionAnimationStartedGameEvent:
			bs.ui.PostEvent(SetAnimationUIEvent(e))
			currentGameStateComp.CurrentState = StateAnimatingAction
		case ActionAnimationFinishedGameEvent:
			// アニメーション終了後、結果を一時的に保持し、メッセージ状態へ遷移
			bs.lastActionResult = &e.Result
			currentGameStateComp.CurrentState = StateMessage
		case MessageDisplayRequestGameEvent:
			bs.messageManager.EnqueueMessageQueue(e.Messages, e.Callback)
			currentGameStateComp.CurrentState = StateMessage
		case MessageDisplayFinishedGameEvent:
			if bs.winner != TeamNone {
				currentGameStateComp.CurrentState = StateGameOver
			} else {
				currentGameStateComp.CurrentState = StatePlaying
			}
		case GameOverGameEvent:
			bs.winner = e.Winner
			currentGameStateComp.CurrentState = StateMessage // ゲームオーバーメッセージ表示のため
		case HideActionModalGameEvent:
			bs.ui.PostEvent(HideActionModalUIEvent{})
		case ShowActionModalGameEvent:
			bs.ui.HideActionModal() // 明示的に非表示にする
			// eventChannel に ShowActionModalUIEvent が既に存在しないかを確認してから送信
			select {
			case bs.ui.GetEventChannel() <- ShowActionModalUIEvent(e):
				// 送信成功
			default:
				// チャネルがフルで送信できなかった、または既に同じイベントが存在する
				log.Println("警告: ShowActionModalUIEvent の送信をスキップしました (チャネルがフルか重複)。")
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
					// BattleLogicComponentからBattleLogicを取得
					battleLogicEntry, ok := query.NewQuery(filter.Contains(BattleLogicComponent)).First(bs.world)
					if !ok {
						log.Panicln("BattleLogicComponent がワールドに見つかりません。")
					}
					battleLogic := BattleLogicComponent.Get(battleLogicEntry)
					// ChargeInitiationSystem を呼び出す
					successful := StartCharge(actingEntry, e.SelectedSlotKey, targetEntry, e.TargetPartSlot, bs.world, battleLogic.GetPartInfoProvider(), battleLogic.GetPartInfoProvider().GetGameDataManager())
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
			if len(bs.playerActionPendingQueue) > 0 && bs.playerActionPendingQueue[0] == actingEntry {
				bs.playerActionPendingQueue = bs.playerActionPendingQueue[1:]
			} else {
				log.Printf("警告: 処理されたエンティティ %s がキューの先頭にありませんでした。", SettingsComponent.Get(actingEntry).Name)
			}
			currentGameStateComp.CurrentState = StatePlaying // 行動処理後はPlaying状態に戻る
		case ActionCanceledGameEvent:
			actingEntry := e.ActingEntry
			if actingEntry == nil || !actingEntry.Valid() {
				log.Printf("Error: ActionCanceledGameEvent - ActingEntry is invalid or nil")
				break
			}
			// 行動キャンセル時の処理（PlayerActionProcessedGameEventでキュー操作は行われる）
			currentGameStateComp.CurrentState = StatePlaying // キャンセル時は即座にPlaying状態に戻る
		case GoToTitleSceneGameEvent:
			bs.manager.GoToTitleScene()
		case StateChangeRequestedGameEvent: // 新しいイベントの処理
			currentGameStateComp.CurrentState = e.NextState
		}
	}
	// イベント処理後に現在の状態を更新
	bs.currentState = bs.states[currentGameStateComp.CurrentState]
}
