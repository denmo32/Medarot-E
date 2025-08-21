package scene

import (
	"log"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ecs/entity"
	"medarot-ebiten/ecs/system"
	"medarot-ebiten/event"
	"medarot-ebiten/ui"

	"medarot-ebiten/donburi"

	"github.com/hajimehoshi/ebiten/v2"
)

// BattleScene は戦闘シーン全体の管理を行います。
// BattleLogicファサードを廃止し、個別のシステムを直接保持することで、依存関係を明確にしました。
type BattleScene struct {
	resources       *data.SharedResources
	manager         *SceneManager
	world           donburi.World
	tickCount       int
	debugMode       bool
	playerTeam      core.TeamID
	winner          core.TeamID
	battleUIManager system.UIUpdater

	// 状態管理
	battleStates map[core.GameState]system.BattleState

	// --- 依存性注入されるシステム群 ---
	// BattleLogicの代わりに、必要なシステムを直接フィールドとして保持します。
	gameDataManager *data.GameDataManager
	// randの型を *core.Rand から正しい *rand.Rand に修正しました。
	rand                   *rand.Rand
	statusEffectSystem     *system.StatusEffectSystem
	postActionEffectSystem *system.PostActionEffectSystem
	partInfoProvider       system.PartInfoProviderInterface
	chargeInitiationSystem *system.ChargeInitiationSystem
	targetSelector         *system.TargetSelector
	damageCalculator       *system.DamageCalculator
	hitCalculator          *system.HitCalculator
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
		// SharedResourcesから正しい型のrandを取得します。
		rand: res.Rand,
	}

	// ワールドの初期化
	entity.InitializeBattleWorld(bs.world, bs.resources, bs.playerTeam)

	// UI専用の状態コンポーネントをワールドに登録
	uiStateEntry := bs.world.Entry(bs.world.Create(ui.BattleUIStateComponent, component.WorldStateTag))
	ui.BattleUIStateComponent.SetValue(uiStateEntry, ui.BattleUIState{IsActionModalVisible: false})

	// --- 各システムの初期化と依存性の注入 ---
	// BattleLogicを介さず、必要なコンポーネントを直接生成・注入します。
	logger := data.NewBattleLogger(bs.gameDataManager)
	bs.partInfoProvider = system.NewPartInfoProvider(bs.world, &bs.resources.Config, bs.gameDataManager)
	bs.damageCalculator = system.NewDamageCalculator(bs.world, &bs.resources.Config, bs.partInfoProvider, bs.gameDataManager, bs.rand, logger)
	bs.hitCalculator = system.NewHitCalculator(bs.world, &bs.resources.Config, bs.partInfoProvider, bs.rand, logger)
	bs.targetSelector = system.NewTargetSelector(bs.world, &bs.resources.Config, bs.partInfoProvider)
	bs.chargeInitiationSystem = system.NewChargeInitiationSystem(bs.world, bs.partInfoProvider)
	bs.statusEffectSystem = system.NewStatusEffectSystem(bs.world, bs.damageCalculator)
	bs.postActionEffectSystem = system.NewPostActionEffectSystem(bs.world, bs.statusEffectSystem, bs.gameDataManager, bs.partInfoProvider)

	// UIとViewModelFactoryの初期化
	// ViewModelFactoryは、UIが必要とする情報（パーツ情報など）を提供するためのインターフェース(PartInfoProvider)に依存します。
	viewModelFactory := ui.NewViewModelFactory(bs.partInfoProvider, bs.gameDataManager, bs.rand)
	bs.battleUIManager = ui.NewBattleUIManager(&bs.resources.Config, bs.resources, viewModelFactory)

	// 戦闘の進行を管理するステートマシンの初期化
	bs.battleStates = map[core.GameState]system.BattleState{
		core.StateGaugeProgress:      &system.GaugeProgressState{},
		core.StatePlayerActionSelect: &system.PlayerActionSelectState{},
		core.StateActionExecution:    &system.ActionExecutionState{},
		core.StateAnimatingAction:    &system.AnimatingActionState{},
		core.StatePostAction:         &system.PostActionState{},
		core.StateMessage:            &system.MessageState{},
		core.StateGameOver:           &system.GameOverState{},
	}

	// 初期状態を設定
	bs.SetState(core.StateGaugeProgress)

	return bs
}

func (bs *BattleScene) SetState(newState core.GameState) {
	gameStateEntry, ok := donburi.NewQuery(donburi.Contains(component.GameStateComponent)).First(bs.world)
	if !ok {
		log.Panicln("GameStateComponent がワールドに見つかりません。")
	}
	component.GameStateComponent.Get(gameStateEntry).CurrentState = newState
}

func (bs *BattleScene) Update() error {
	bs.tickCount++

	// 1. UIを更新し、UIから発行されたゲームイベントを収集
	uiEvents := bs.battleUIManager.Update(bs.tickCount, bs.world)

	// 2. 現在のゲーム状態に対応するロジックを実行
	gameStateEntry, _ := donburi.NewQuery(donburi.Contains(component.GameStateComponent)).First(bs.world)
	currentGameStateComp := component.GameStateComponent.Get(gameStateEntry)

	// BattleContextに必要なシステムをすべて渡す
	battleContext := &system.BattleContext{
		World:                  bs.world,
		Config:                 &bs.resources.Config,
		GameDataManager:        bs.gameDataManager,
		Rand:                   bs.rand,
		Tick:                   bs.tickCount,
		BattleUIManager:        bs.battleUIManager,
		StatusEffectSystem:     bs.statusEffectSystem,
		PostActionEffectSystem: bs.postActionEffectSystem,
		PartInfoProvider:       bs.partInfoProvider,
		ChargeInitiationSystem: bs.chargeInitiationSystem,
		TargetSelector:         bs.targetSelector,
		DamageCalculator:       bs.damageCalculator,
		HitCalculator:          bs.hitCalculator,
	}

	var stateEvents []event.GameEvent
	if currentStateImpl, ok := bs.battleStates[currentGameStateComp.CurrentState]; ok {
		var err error
		stateEvents, err = currentStateImpl.Update(battleContext)
		if err != nil {
			log.Printf("Error updating game state %s: %v", currentGameStateComp.CurrentState, err)
		}
	} else {
		log.Printf("Unknown game state: %s", currentGameStateComp.CurrentState)
	}

	// 3. すべてのイベントを処理
	allGameEvents := append(uiEvents, stateEvents...)
	bs.battleUIManager.ProcessEvents(bs.world, allGameEvents)
	stateChangeRequests := bs.processGameEvents(allGameEvents)

	// 4. 状態遷移要求を適用
	for _, req := range stateChangeRequests {
		if stateChangeReq, ok := req.(event.StateChangeRequestedGameEvent); ok {
			bs.SetState(stateChangeReq.NextState)
		}
	}

	return nil
}

func (bs *BattleScene) Draw(screen *ebiten.Image) {
	// UIマネージャーがUI全体の描画を担当
	bs.battleUIManager.Draw(screen, bs.tickCount, bs.resources.GameDataManager)
}

func (bs *BattleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return bs.resources.Config.UI.Screen.Width, bs.resources.Config.UI.Screen.Height
}

// processGameEvents はGameEventのリストを処理し、BattleSceneの状態を更新します。
func (bs *BattleScene) processGameEvents(gameEvents []event.GameEvent) []event.GameEvent {
	var stateChangeEvents []event.GameEvent

	lastActionResultEntry, ok := donburi.NewQuery(donburi.Contains(component.LastActionResultComponent)).First(bs.world)
	if !ok {
		log.Panicln("LastActionResultComponent がワールドに見つかりません。")
	}
	lastActionResultComp := component.LastActionResultComponent.Get(lastActionResultEntry)

	for _, evt := range gameEvents {
		switch e := evt.(type) {
		case event.PlayerActionRequiredGameEvent:
			// プレイヤーの行動選択が必要になったので、対応する状態へ遷移
			if pss, ok := bs.battleStates[core.StatePlayerActionSelect].(*system.PlayerActionSelectState); ok {
				pss.Reset()
			}
			stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StatePlayerActionSelect})
		case event.PlayerActionIntentEvent:
			// プレイヤーの行動意図を処理
			system.ProcessPlayerIntent(bs.world, bs.chargeInitiationSystem, e)
		case event.PlayerActionProcessedGameEvent:
			// プレイヤーの行動選択が1人分完了した
			actingEntry := bs.world.Entry(e.ActingEntityID)
			if actingEntry == nil {
				break
			}
			playerActionQueue := entity.GetPlayerActionQueueComponent(bs.world)
			// キューの先頭が処理されたエントリと一致するか確認
			if len(playerActionQueue.Queue) > 0 && playerActionQueue.Queue[0].Entity() == e.ActingEntityID {
				playerActionQueue.Queue = playerActionQueue.Queue[1:]
			}
			// 次のプレイヤーの選択へ、または全員の選択が終わったらゲージ進行へ
			if len(playerActionQueue.Queue) > 0 {
				stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StatePlayerActionSelect})
			} else {
				stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StateGaugeProgress})
			}
		case event.ActionAnimationStartedGameEvent:
			// アニメーションを開始し、状態を遷移
			bs.battleUIManager.SetAnimation(&e.AnimationData)
			stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StateAnimatingAction})
		case event.ActionAnimationFinishedGameEvent:
			// アニメーションが終了したので、結果を保存し、事後処理状態へ
			*lastActionResultComp = e.Result
			stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StatePostAction})
		case event.MessageDisplayFinishedGameEvent:
			// メッセージ表示が完了。ゲームオーバーでなければゲージ進行へ
			if bs.winner != core.TeamNone {
				stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StateGameOver})
			} else {
				stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StateGaugeProgress})
			}
		case event.GameOverGameEvent:
			// ゲームオーバーフラグを立て、メッセージ表示状態へ
			bs.winner = e.Winner
			stateChangeEvents = append(stateChangeEvents, event.StateChangeRequestedGameEvent{NextState: core.StateMessage})
		case event.GoToTitleSceneGameEvent:
			// タイトルシーンへ遷移
			bs.manager.GoToTitleScene()
		case event.StateChangeRequestedGameEvent:
			// 他のシステムから直接発行された状態遷移要求
			stateChangeEvents = append(stateChangeEvents, e)
		}
	}
	return stateChangeEvents
}
