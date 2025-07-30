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
	
	gameDataManager          *GameDataManager
	rand                     *rand.Rand
	uiEventChannel           chan UIEvent
	battleUIState            *BattleUIState
	statusEffectSystem       *StatusEffectSystem
	postActionEffectSystem   *PostActionEffectSystem
	viewModelFactory         ViewModelFactory
	uiFactory                *UIFactory

	lastActionResult *ActionResult
	battleLogic      *BattleLogic // 追加
}

func NewBattleScene(res *SharedResources, manager *SceneManager) *BattleScene {
	world := donburi.NewWorld()

	bs := &BattleScene{
		resources:                res,
		manager:                  manager,
		world:                    world,
		debugMode:                true,
		playerTeam:               Team1,
		winner:                   TeamNone,
		gameDataManager:          res.GameDataManager,
		rand:                     res.Rand,
		uiEventChannel:           make(chan UIEvent, 10),
		battleUIState:            &BattleUIState{}, // ← これを追加
	}

	InitializeBattleWorld(bs.world, bs.resources, bs.playerTeam)

	bs.battleLogic = NewBattleLogic(bs.world, &bs.resources.Config, bs.resources.GameDataManager, bs.rand)

	bs.uiFactory = NewUIFactory(&bs.resources.Config, bs.resources.GameDataManager.Font, bs.resources.GameDataManager.Messages)
	bs.ui = NewUI(&bs.resources.Config, bs.uiEventChannel, bs.uiFactory, bs.resources.GameDataManager)
	bs.viewModelFactory = NewViewModelFactory(bs.world, bs.battleLogic.GetPartInfoProvider(), bs.gameDataManager, bs.rand, bs.ui)
	bs.statusEffectSystem = NewStatusEffectSystem(bs.world, bs.battleLogic.GetDamageCalculator())
	bs.postActionEffectSystem = NewPostActionEffectSystem(bs.world, bs.statusEffectSystem, bs.gameDataManager, bs.battleLogic.GetPartInfoProvider())
	bs.messageManager = bs.ui.GetMessageDisplayManager() // uiからmessageManagerを取得

	bs.SetState(StatePlaying)

	return bs
}

func (bs *BattleScene) SetState(newState GameState) {
	gameStateEntry, ok := query.NewQuery(filter.Contains(GameStateComponent)).First(bs.world)
	if !ok {
		log.Panicln("GameStateComponent がワールドに見つかりません。")
	}
	GameStateComponent.Get(gameStateEntry).CurrentState = newState
}

func (bs *BattleScene) Update() error {
	bs.tickCount++
	gameStateEntry, ok := query.NewQuery(filter.Contains(GameStateComponent)).First(bs.world)
	if !ok {
		log.Panicln("GameStateComponent がワールドに見つかりません。")
	}
	currentGameStateComp := GameStateComponent.Get(gameStateEntry)

	bs.ui.Update()

	var uiGeneratedGameEvents []GameEvent
	playerActionQueue := GetPlayerActionQueueComponent(bs.world)
	playerActionQueue.Queue, uiGeneratedGameEvents = UpdateUIEventProcessorSystem(
		bs.world, bs.ui, bs.messageManager, bs.uiEventChannel, playerActionQueue.Queue,
	)
	bs.processGameEvents(uiGeneratedGameEvents)

	if currentGameStateComp.CurrentState == StateMessage {
		if bs.lastActionResult != nil {
			result := *bs.lastActionResult
			actingEntry := result.ActingEntry

			if actingEntry.Valid() && StateComponent.Get(actingEntry).CurrentState != StateBroken {
				StartCooldownSystem(actingEntry, bs.world, bs.battleLogic.GetPartInfoProvider())
			}
			bs.ui.PostEvent(ClearCurrentTargetUIEvent{})

			bs.messageManager.EnqueueMessageQueue(buildActionLogMessagesFromActionResult(result, bs.resources.GameDataManager), func() {
				UpdateHistorySystem(bs.world, &result)
			})

			bs.lastActionResult = nil
		}

		bs.messageManager.Update()
		if bs.messageManager.IsFinished() {
			gameEvents := []GameEvent{MessageDisplayFinishedGameEvent{}}
			bs.processGameEvents(gameEvents)
		}
	}

	// if currentGameState != StateMessage && currentGameState != StateAnimatingAction {
		battleContext := &BattleContext{
			World:                  bs.world,
			Config:                 &bs.resources.Config,
			GameDataManager:        bs.gameDataManager,
			Rand:                   bs.rand,
			Tick:                   bs.tickCount,
			ViewModelFactory:       bs.viewModelFactory,
			statusEffectSystem:     bs.statusEffectSystem,
			postActionEffectSystem: bs.postActionEffectSystem,
			BattleLogic:            bs.battleLogic,
		}

		// var tempGameEvents []GameEvent
		// switch currentGameState {
		// case StatePlaying:
		// 	tempGameEvents, _ = (&PlayingState{}).Update(battleContext)
		// case StatePlayerActionSelect:
		// 	tempGameEvents, _ = (&PlayerActionSelectState{}).Update(battleContext)
		// case StateAnimatingAction:
		// 	tempGameEvents, _ = (&AnimatingActionState{}).Update(battleContext)
		// case StateMessage:
		// 	tempGameEvents, _ = (&MessageState{}).Update(battleContext)
		// case StateGameOver:
		// 	tempGameEvents, _ = (&GameOverState{}).Update(battleContext)
		// }

		// currentStateがMessageStateでない場合のみUpdateを呼び出す
			if currentGameStateComp.CurrentState != StateMessage && currentGameStateComp.CurrentState != StateAnimatingAction {
			var tempGameEvents []GameEvent
			switch currentGameStateComp.CurrentState {
			case StatePlaying:
				tempGameEvents, _ = (&PlayingState{}).Update(battleContext)
			case StatePlayerActionSelect:
				tempGameEvents, _ = (&PlayerActionSelectState{}).Update(battleContext)
			case StateGameOver:
				tempGameEvents, _ = (&GameOverState{}).Update(battleContext)
			case StateAnimatingAction:
				tempGameEvents, _ = (&AnimatingActionState{}).Update(battleContext)
			case StateMessage:
				tempGameEvents, _ = (&MessageState{}).Update(battleContext)
			}
			bs.statusEffectSystem.Update()
			bs.processGameEvents(tempGameEvents)
		}
	// }

	// プレイヤーの行動選択状態からPlaying状態への遷移ロジックを移動
		if currentGameStateComp.CurrentState == StatePlayerActionSelect && len(playerActionQueue.Queue) == 0 {
		bs.SetState(StatePlaying)
	}

	battleUIStateEntry, ok := query.NewQuery(filter.Contains(BattleUIStateComponent)).First(bs.world)
	if !ok {
		log.Println("エラー: BattleUIStateComponent がワールドに見つかりません。UI更新をスキップします。")
		return nil
	}
	battleUIState := BattleUIStateComponent.Get(battleUIStateEntry)

	UpdateInfoPanelViewModelSystem(battleUIState, bs.world, bs.battleLogic.GetPartInfoProvider(), bs.viewModelFactory)

	battleUIState.BattlefieldViewModel = bs.viewModelFactory.BuildBattlefieldViewModel(bs.world, battleUIState, bs.battleLogic.GetPartInfoProvider(), &bs.resources.Config, bs.ui.GetBattlefieldWidgetRect())

	bs.ui.SetBattleUIState(battleUIState, &bs.resources.Config, bs.ui.GetBattlefieldWidgetRect(), bs.uiFactory)

	return nil
}

func (bs *BattleScene) Draw(screen *ebiten.Image) {
	screen.Fill(bs.resources.Config.UI.Colors.Background)
	bs.ui.DrawBackground(screen)
	bs.ui.Draw(screen, bs.tickCount, bs.resources.GameDataManager)

	// GameStateComponentから現在のゲーム状態を取得
	gameStateEntry, ok := query.NewQuery(filter.Contains(GameStateComponent)).First(bs.world)
	if !ok {
		log.Panicln("GameStateComponent がワールドに見つかりません。")
	}
	currentGameStateComp := GameStateComponent.Get(gameStateEntry)

	switch currentGameStateComp.CurrentState {
	case StatePlaying:
		(&PlayingState{}).Draw(screen)
	case StatePlayerActionSelect:
		(&PlayerActionSelectState{}).Draw(screen)
	case StateAnimatingAction:
		(&AnimatingActionState{}).Draw(screen)
	case StateMessage:
		(&MessageState{}).Draw(screen)
	case StateGameOver:
		(&GameOverState{}).Draw(screen)
	}

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
					// ChargeInitiationSystem を呼び出す
					successful := bs.battleLogic.GetChargeInitiationSystem().StartCharge(actingEntry, e.SelectedSlotKey, targetEntry, e.TargetPartSlot)
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
			playerActionQueue := GetPlayerActionQueueComponent(bs.world)
			if len(playerActionQueue.Queue) > 0 && playerActionQueue.Queue[0] == actingEntry {
				playerActionQueue.Queue = playerActionQueue.Queue[1:]
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
			bs.SetState(e.NextState)
		}
	}
}
