package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
)

type BattleScene struct {
	resources                *SharedResources
	manager                  *SceneManager // bamennのシーンマネージャ
	world                    donburi.World
	tickCount                int
	debugMode                bool
	state                    GameState
	playerTeam               TeamID
	ui                       UIInterface
	messageManager           *UIMessageDisplayManager
	winner                   TeamID
	playerActionPendingQueue []*donburi.Entry
	attackingEntity          *donburi.Entry
	targetedEntity           *donburi.Entry
	whitePixel               *ebiten.Image
	battleLogic              *BattleLogic
	uiEventChannel           chan UIEvent
	battlefieldViewModel     BattlefieldViewModel
}

func NewBattleScene(res *SharedResources, manager *SceneManager) *BattleScene {
	world := donburi.NewWorld()
	whitePixel := ebiten.NewImage(1, 1)
	whitePixel.Fill(color.White)

	bs := &BattleScene{
		resources:                res,
		manager:                  manager,
		world:                    world,
		debugMode:                true,
		state:                    StatePlaying,
		playerTeam:               Team1,
		playerActionPendingQueue: make([]*donburi.Entry, 0),
		winner:                   TeamNone,
		whitePixel:               whitePixel,
	}

	bs.battleLogic = NewBattleLogic(bs.world, &bs.resources.Config)
	EnsureActionQueueEntity(bs.world)
	CreateMedarotEntities(bs.world, res.GameData, bs.playerTeam)
	bs.uiEventChannel = make(chan UIEvent, 10)
	bs.ui = NewUI(bs.world, &bs.resources.Config, bs.uiEventChannel)
	bs.messageManager = NewUIMessageDisplayManager(&bs.resources.Config, GlobalGameDataManager.Font, bs.ui.GetRootContainer())

	return bs
}

func (bs *BattleScene) Update() error {
	bs.tickCount++
	bs.ui.Update()

	select {
	case event := <-bs.uiEventChannel:
		switch e := event.(type) {
		case PlayerActionSelectedEvent:
			var message string
			var postMessageCallback func()
			bs.playerActionPendingQueue, bs.state, message, postMessageCallback = ProcessPlayerActionSelected(
				bs.world, &bs.resources.Config, bs.battleLogic, bs.playerActionPendingQueue, bs.ui, e, bs.ui.GetActionTargetMap())
			if message != "" {
				bs.messageManager.EnqueueMessage(message, postMessageCallback)
			}
		case PlayerActionCancelEvent:
			bs.playerActionPendingQueue, bs.state = ProcessPlayerActionCancel(bs.playerActionPendingQueue, bs.ui, e)
		case SetCurrentTargetEvent:
			bs.ui.SetCurrentTarget(e.Target)
		case ClearCurrentTargetEvent:
			bs.ui.ClearCurrentTarget()
		}
	default:
	}

	switch bs.state {
	case StatePlaying:
		if !bs.ui.IsActionModalVisible() && len(bs.playerActionPendingQueue) == 0 {
			UpdateAIInputSystem(bs.world, bs.battleLogic.PartInfoProvider, bs.battleLogic.TargetSelector, &bs.resources.Config)
			playerInputResult := UpdatePlayerInputSystem(bs.world)
			if len(playerInputResult.PlayerMedarotsToAct) > 0 {
				bs.playerActionPendingQueue = playerInputResult.PlayerMedarotsToAct
			}
		}

		if len(bs.playerActionPendingQueue) > 0 {
			bs.state = StatePlayerActionSelect
		}

		actionQueueComp := GetActionQueueComponent(bs.world)
		if !bs.ui.IsActionModalVisible() && len(bs.playerActionPendingQueue) == 0 && len(actionQueueComp.Queue) == 0 {
			UpdateGaugeSystem(bs.world)
		}

		if bs.state == StatePlaying {
			actionResults, err := UpdateActionQueueSystem(bs.world, bs.battleLogic, &bs.resources.Config)
			if err != nil {
				fmt.Println("アクションキューシステムの処理中にエラーが発生しました:", err)
			}

			for _, result := range actionResults {
				if result.ActingEntry != nil && result.ActingEntry.Valid() {
										bs.ui.SetAnimation(&ActionAnimationData{Result: result, StartTime: bs.tickCount})
					bs.state = StateAnimatingAction
					break
				}
			}
		}

		gameEndResult := CheckGameEndSystem(bs.world)
		if gameEndResult.IsGameOver {
			bs.winner = gameEndResult.Winner
			bs.state = StateGameOver
			bs.messageManager.EnqueueMessage(gameEndResult.Message, nil)
			bs.state = StateMessage
		}

		bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
		bs.battlefieldViewModel = BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
		bs.ui.SetBattlefieldViewModel(bs.battlefieldViewModel)
		// ProcessStateChangeSystem はFSMコールバックに統合されたため不要
		// ProcessStateChangeSystem(bs.world)

	case StateAnimatingAction:
		bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
						if bs.ui.IsAnimationFinished(bs.tickCount) {
			messages := []string{}
									result := bs.ui.GetCurrentAnimationResult()
			if result.ActionDidHit {
				initiateParams := map[string]interface{}{"attacker_name": result.AttackerName, "action_name": result.ActionName, "weapon_type": result.WeaponType}
				messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_initiate", initiateParams))
				if result.ActionIsDefended {
					defendParams := map[string]interface{}{"defender_name": result.DefenderName, "defending_part_type": result.DefendingPartType}
					messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_defend", defendParams))
				}
				damageParams := map[string]interface{}{"defender_name": result.DefenderName, "target_part_type": result.TargetPartType, "damage": result.DamageDealt}
				messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_damage", damageParams))
			} else {
				messages = append(messages, result.LogMessage)
			}

			bs.messageManager.EnqueueMessageQueue(messages, func() {
				actingEntry := result.ActingEntry
				if actingEntry.Valid() && !StateComponent.Get(actingEntry).FSM.Is(string(StateBroken)) {
					StartCooldownSystem(actingEntry, bs.world, &bs.resources.Config, bs.battleLogic.PartInfoProvider)
				}
				bs.attackingEntity = nil
				bs.targetedEntity = nil
				bs.ui.ClearCurrentTarget()
			})
			bs.state = StateMessage
		}

	case StatePlayerActionSelect:
		bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
		bs.battlefieldViewModel = BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
		bs.ui.SetBattlefieldViewModel(bs.battlefieldViewModel)

		if !bs.ui.IsActionModalVisible() && len(bs.playerActionPendingQueue) > 0 {
			actingEntry := bs.playerActionPendingQueue[0]
			if actingEntry.Valid() && StateComponent.Get(actingEntry).FSM.Is(string(StateIdle)) {
				actionTargetMap := make(map[PartSlotKey]ActionTarget)
				availableParts := bs.battleLogic.PartInfoProvider.GetAvailableAttackParts(actingEntry)
				for _, available := range availableParts {
					partDef := available.PartDef
					slotKey := available.Slot
					var targetEntity *donburi.Entry
					var targetPartSlot PartSlotKey
					if partDef.Category == CategoryShoot || partDef.Category == CategoryMelee {
						medal := MedalComponent.Get(actingEntry)
						personality, ok := PersonalityRegistry[medal.Personality]
						if !ok {
							personality = PersonalityRegistry["リーダー"]
						}
						targetEntity, targetPartSlot = personality.TargetingStrategy.SelectTarget(bs.world, actingEntry, bs.battleLogic.TargetSelector, bs.battleLogic.PartInfoProvider)
					}
					actionTargetMap[slotKey] = ActionTarget{Target: targetEntity, Slot: targetPartSlot}
				}
				bs.ui.ShowActionModal(actingEntry, actionTargetMap)
			} else {
				bs.playerActionPendingQueue = bs.playerActionPendingQueue[1:]
				if len(bs.playerActionPendingQueue) == 0 {
					bs.state = StatePlaying
				}
			}
		}
	case StateMessage:
		bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
		bs.battlefieldViewModel = BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
		bs.ui.SetBattlefieldViewModel(bs.battlefieldViewModel)

		newState, finished := bs.messageManager.Update(bs.state)
		if finished {
									bs.ui.ClearAnimation()
			if bs.winner != TeamNone {
				bs.state = StateGameOver
			} else {
				bs.state = StatePlaying
			}
		} else {
			bs.state = newState
		}
	case StateGameOver:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			bs.manager.GoToTitleScene() // マネージャ経由でタイトルに戻る
		}
	}
	return nil
}

func (bs *BattleScene) Draw(screen *ebiten.Image) {
	screen.Fill(bs.resources.Config.UI.Colors.Background)

	// UIに背景描画を委譲
	bs.ui.DrawBackground(screen)

	// UIのメイン描画
	bs.ui.Draw(screen, bs.tickCount)

	// UIにアニメーション描画を委譲
			bs.ui.DrawAnimation(screen, bs.tickCount, bs.battlefieldViewModel)

	if bs.debugMode {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\nState: %s", ebiten.ActualTPS(), ebiten.ActualFPS(), bs.state))
	}
}

func (bs *BattleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return bs.resources.Config.UI.Screen.Width, bs.resources.Config.UI.Screen.Height
}


