package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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

	bs.battleLogic = NewBattleLogic(bs.world, &bs.resources.Config)
	bs.statusEffectSystem = NewStatusEffectSystem(bs.world)
	EnsureActionQueueEntity(bs.world)

	teamBuffsEntry := bs.world.Entry(bs.world.Create(TeamBuffsComponent))
	TeamBuffsComponent.SetValue(teamBuffsEntry, TeamBuffs{
		Buffs: make(map[TeamID]map[BuffType][]*BuffSource),
	})

	CreateMedarotEntities(bs.world, res.GameData, bs.playerTeam)
	bs.ui = NewUI(bs.world, &bs.resources.Config, bs.uiEventChannel)
	bs.messageManager = NewUIMessageDisplayManager(&bs.resources.Config, GlobalGameDataManager.Font, bs.ui.GetRootContainer())

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

	log.Printf("BattleScene.Update: Current State: %s", bs.state)

	bs.playerActionPendingQueue, bs.state = UpdateUIEventProcessorSystem(
		bs.world, bs.battleLogic, bs.ui, bs.messageManager, bs.uiEventChannel, bs.playerActionPendingQueue, bs.state,
	)

	// Update current state
	newPlayerActionPendingQueue, result, err := bs.currentState.Update(
		bs.world,
		bs.battleLogic,
		bs.ui,
		bs.messageManager,
		&bs.resources.Config,
		bs.tickCount,
		bs.manager,
		bs.playerActionPendingQueue,
	)
	if err != nil {
		return err
	}
	bs.playerActionPendingQueue = newPlayerActionPendingQueue

	// Update status effect durations
	bs.statusEffectSystem.Update()

	// Process result and transition state
	nextState := bs.state // Default to current state

	if result.GameOver {
		bs.winner = result.Winner
		nextState = StateMessage
		log.Printf("BattleScene.Update: Transitioning to StateMessage (GameOver)")
	} else if result.ActionStarted {
		nextState = StateAnimatingAction
		log.Printf("BattleScene.Update: Transitioning to StateAnimatingAction")
	} else if result.MessageQueued {
		nextState = StateMessage
		log.Printf("BattleScene.Update: Transitioning to StateMessage (MessageQueued)")
	} else if result.PlayerActionRequired {
		nextState = StatePlayerActionSelect
		log.Printf("BattleScene.Update: Transitioning to StatePlayerActionSelect")
	} else if bs.state == StatePlayerActionSelect && len(bs.playerActionPendingQueue) == 0 {
		nextState = StatePlaying
		log.Printf("BattleScene.Update: Transitioning to StatePlaying (PlayerActionSelect finished)")
	} else if bs.state == StateAnimatingAction && bs.ui.IsAnimationFinished(bs.tickCount) {
		nextState = StateMessage
		log.Printf("BattleScene.Update: Transitioning to StateMessage (Animation finished)")
	} else if bs.state == StateMessage && result.MessageFinished {
		log.Printf("BattleScene.Update: MessageFinished is true. winner: %v", bs.winner)
		if bs.winner != TeamNone {
			nextState = StateGameOver
			log.Printf("BattleScene.Update: Transitioning to StateGameOver")
		} else {
			nextState = StatePlaying
			log.Printf("BattleScene.Update: Transitioning to StatePlaying (Message finished)")
		}
	}

	// Transition to new state if changed
	if nextState != bs.state {
		log.Printf("BattleScene.Update: State changed from %s to %s", bs.state, nextState)
		bs.state = nextState
		bs.currentState = bs.states[nextState]
	}

	// Update UI components that depend on world state
	bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
	bs.battlefieldViewModel = BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
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

	if bs.debugMode {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\nState: %s", ebiten.ActualTPS(), ebiten.ActualFPS(), bs.state))
	}
}

func (bs *BattleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return bs.resources.Config.UI.Screen.Width, bs.resources.Config.UI.Screen.Height
}
