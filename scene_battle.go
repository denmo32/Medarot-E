package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
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
	battlefieldViewModel     BattlefieldViewModel
	statusEffectSystem       *StatusEffectSystem

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
	}

	bs.battleLogic = NewBattleLogic(bs.world, &bs.resources.Config, bs.resources.GameDataManager)
	bs.statusEffectSystem = NewStatusEffectSystem(bs.world)
	EnsureActionQueueEntity(bs.world)

	teamBuffsEntry := bs.world.Entry(bs.world.Create(TeamBuffsComponent))
	TeamBuffsComponent.SetValue(teamBuffsEntry, TeamBuffs{
		Buffs: make(map[TeamID]map[BuffType][]*BuffSource),
	})

	CreateMedarotEntities(bs.world, res.GameData, bs.playerTeam, bs.battleLogic)
	animationManager := NewBattleAnimationManager(&bs.resources.Config)
	bs.ui = NewUI(bs.world, &bs.resources.Config, bs.uiEventChannel, bs.resources.GameDataManager, animationManager)
	// ui.goでuiFactoryが初期化され、ui.messageManagerもuiFactoryを使って初期化されるため、
	// ここでbs.messageManagerを直接初期化する必要はない。
	// bs.messageManager = NewUIMessageDisplayManager(&bs.resources.Config, bs.resources.GameDataManager.Font, bs.resources.GameDataManager.Messages, bs.ui.GetRootContainer())
	bs.messageManager = bs.ui.(*UI).messageManager // uiからmessageManagerを取得

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

	// UIイベントプロセッサシステムを更新
	bs.playerActionPendingQueue, bs.state = UpdateUIEventProcessorSystem(
		bs.world, bs.battleLogic, bs.ui, bs.messageManager, bs.uiEventChannel, bs.playerActionPendingQueue, bs.state,
	)

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
		World:           bs.world,
		BattleLogic:     bs.battleLogic,
		UI:              bs.ui,
		Config:          &bs.resources.Config,
		SceneManager:    bs.manager,
		GameDataManager: bs.resources.GameDataManager, // 追加
		Tick:            bs.tickCount,
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
	} else if bs.state == StateAnimatingAction && bs.ui.IsAnimationFinished(bs.tickCount) {
		// この条件はActionAnimationFinishedGameEventで処理されるため、削除
	}

	// Transition to new state if changed (processGameEventsでbs.stateが更新されるため、このブロックは不要になる)
	// if nextState != bs.state {
	// 	bs.state = nextState
	// 	bs.currentState = bs.states[nextState]
	// }

	// Update UI components that depend on world state
	bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config, bs.battleLogic)
	bs.battlefieldViewModel = BuildBattlefieldViewModel(bs.world, bs.battleLogic.GetPartInfoProvider(), &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
	bs.ui.SetBattlefieldViewModel(bs.battlefieldViewModel)

	return nil
}

func (bs *BattleScene) Draw(screen *ebiten.Image) {
	screen.Fill(bs.resources.Config.UI.Colors.Background)
	bs.ui.DrawBackground(screen)
	bs.ui.Draw(screen, bs.tickCount)
	bs.ui.(*UI).animationDrawer.Draw(screen, bs.tickCount, bs.battlefieldViewModel, bs.ui.(*UI).battlefieldWidget)

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
			if actingEntry.Valid() && !StateComponent.Get(actingEntry).FSM.Is(string(StateBroken)) {
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
			bs.ui.PostEvent(ShowActionModalUIEvent(e))
		case ClearAnimationGameEvent:
			bs.ui.PostEvent(ClearAnimationUIEvent{})
		case ClearCurrentTargetGameEvent:
			bs.ui.PostEvent(ClearCurrentTargetUIEvent{})
		}
	}
	// イベント処理後に現在の状態を更新
	bs.currentState = bs.states[bs.state]
}
