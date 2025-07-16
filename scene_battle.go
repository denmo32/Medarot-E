package main

import (
	"fmt"

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

	// Process UI events first
	bs.processUIEvents()

	// Update current state
	newState, err := bs.currentState.Update(bs)
	if err != nil {
		return err
	}

	// Transition to new state if changed
	if newState != bs.state {
		bs.state = newState
		bs.currentState = bs.states[newState]
	}

	// Update UI components that depend on world state
	bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
	bs.battlefieldViewModel = BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
	bs.ui.SetBattlefieldViewModel(bs.battlefieldViewModel)

	return nil
}

func (bs *BattleScene) processUIEvents() {
	// このメソッドは、PlayerActionSelectState のみが関心を持つイベントを処理します。
	if bs.state != StatePlayerActionSelect {
		return
	}

	select {
	case event := <-bs.uiEventChannel:
		switch e := event.(type) {
		case PlayerActionSelectedEvent:
			var message string
			var postMessageCallback func()
			bs.playerActionPendingQueue, message, postMessageCallback = ProcessPlayerActionSelected(
				bs.world, bs.battleLogic, bs.playerActionPendingQueue, bs.ui, e, bs.ui.GetActionTargetMap())
			if message != "" {
				bs.messageManager.EnqueueMessage(message, postMessageCallback)
				bs.state = StateMessage // メッセージ表示状態に遷移
			}
		case PlayerActionCancelEvent:
			bs.playerActionPendingQueue = ProcessPlayerActionCancel(bs.playerActionPendingQueue, bs.ui, e)
			bs.state = StatePlaying // キャンセル時は即座にPlaying状態に戻る
		case SetCurrentTargetEvent:
			bs.ui.SetCurrentTarget(e.Target)
		case ClearCurrentTargetEvent:
			bs.ui.ClearCurrentTarget()
		}
	default:
	}
}

func (bs *BattleScene) buildActionLogMessages(result ActionResult) []string {
	messages := []string{}
	if result.ActionDidHit {
		initiateParams := map[string]interface{}{"attacker_name": result.AttackerName, "action_name": result.ActionName, "weapon_type": result.WeaponType}
		messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_initiate", initiateParams))

		actingPartDef, _ := GlobalGameDataManager.GetPartDefinition(PartsComponent.Get(result.ActingEntry).Map[ActionIntentComponent.Get(result.ActingEntry).SelectedPartKey].DefinitionID)
		switch actingPartDef.Category {
		case CategoryRanged, CategoryMelee:
			if result.ActionIsDefended {
				defendParams := map[string]interface{}{"defender_name": result.DefenderName, "defending_part_type": result.DefendingPartType}
				messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_defend", defendParams))
			}
			damageParams := map[string]interface{}{"defender_name": result.DefenderName, "target_part_type": result.TargetPartType, "damage": result.DamageDealt}
			messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_damage", damageParams))
		case CategoryIntervention:
			messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("support_action_generic", nil))
		}
	} else {
		initiateParams := map[string]interface{}{"attacker_name": result.AttackerName, "action_name": result.ActionName, "weapon_type": result.WeaponType}
		messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_initiate", initiateParams))
		missParams := map[string]interface{}{
			"target_name": result.DefenderName,
		}
		messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("attack_miss", missParams))
	}
	return messages
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