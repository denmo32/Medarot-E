package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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
	message                  string
	messageQueue             []string
	currentMessageIndex      int
	postMessageCallback      func()
	winner                   TeamID
	playerActionPendingQueue []*donburi.Entry
	attackingEntity          *donburi.Entry
	targetedEntity           *donburi.Entry
	whitePixel               *ebiten.Image
	currentActionAnimation   *ActionAnimationData
	battleLogic              *BattleLogic
	uiEventChannel           chan UIEvent
}

type ActionAnimationData struct {
	Result    ActionResult
	StartTime int
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

	return bs
}

func (bs *BattleScene) Update() error {
	bs.tickCount++
	bs.ui.Update()

	select {
	case event := <-bs.uiEventChannel:
		switch e := event.(type) {
		case PlayerActionSelectedEvent:
			bs.playerActionPendingQueue, bs.state, bs.message, bs.postMessageCallback = ProcessPlayerActionSelected(
				bs.world, &bs.resources.Config, bs.battleLogic, bs.playerActionPendingQueue, bs.ui, e, bs.ui.GetActionTargetMap())
			if bs.message != "" {
				bs.enqueueMessage(bs.message, bs.postMessageCallback)
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
					bs.currentActionAnimation = &ActionAnimationData{Result: result, StartTime: bs.tickCount}
					bs.state = StateAnimatingAction
					break
				}
			}
		}

		gameEndResult := CheckGameEndSystem(bs.world)
		if gameEndResult.IsGameOver {
			bs.winner = gameEndResult.Winner
			bs.state = StateGameOver
			bs.enqueueMessage(gameEndResult.Message, nil)
		}

		bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
		bfVM := BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
		bs.ui.SetBattlefieldViewModel(bfVM)
		// ProcessStateChangeSystem はFSMコールバックに統合されたため不要
		// ProcessStateChangeSystem(bs.world)

	case StateAnimatingAction:
		bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
		anim := bs.currentActionAnimation
		if anim == nil {
			bs.state = StatePlaying
			return nil
		}

		progress := float64(bs.tickCount - anim.StartTime)
		const totalAnimationDuration = 120 // Simplified duration

		if progress >= totalAnimationDuration {
			bs.currentActionAnimation = nil
			messages := []string{}
			result := anim.Result
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

			bs.enqueueMessageQueue(messages, func() {
				actingEntry := anim.Result.ActingEntry
				if actingEntry.Valid() && !StateComponent.Get(actingEntry).FSM.Is(string(StateBroken)) {
					StartCooldownSystem(actingEntry, bs.world, &bs.resources.Config, bs.battleLogic.PartInfoProvider)
				}
				bs.attackingEntity = nil
				bs.targetedEntity = nil
				bs.ui.ClearCurrentTarget()
			})
		}

	case StatePlayerActionSelect:
		bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
		bfVM := BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
		bs.ui.SetBattlefieldViewModel(bfVM)

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
		bfVM := BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
		bs.ui.SetBattlefieldViewModel(bfVM)

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			bs.currentMessageIndex++
			if bs.currentMessageIndex < len(bs.messageQueue) {
				bs.ui.ShowMessageWindow(bs.messageQueue[bs.currentMessageIndex])
			} else {
				bs.ui.HideMessageWindow()
				if bs.postMessageCallback != nil {
					bs.postMessageCallback()
					bs.postMessageCallback = nil
				}
				if bs.winner != TeamNone {
					bs.state = StateGameOver
				} else {
					bs.state = StatePlaying
				}
			}
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
	bs.ui.Draw(screen)

	if bs.currentActionAnimation != nil {
		anim := bs.currentActionAnimation
		progress := float64(bs.tickCount - anim.StartTime)
		bfVM := BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
		var attackerVM, targetVM *IconViewModel
		for _, icon := range bfVM.Icons {
			if icon.EntryID == uint32(anim.Result.ActingEntry.Id()) {
				attackerVM = icon
			}
			if icon.EntryID == uint32(anim.Result.TargetEntry.Id()) {
				targetVM = icon
			}
		}

		if attackerVM != nil && targetVM != nil {
			const travelDuration = 30
			if progress <= travelDuration {
				lerpFactor := progress / travelDuration
				currentX := attackerVM.X + (targetVM.X-attackerVM.X)*float32(lerpFactor)
				currentY := attackerVM.Y + (targetVM.Y-attackerVM.Y)*float32(lerpFactor)
				indicatorColor := bs.resources.Config.UI.Colors.Yellow
				iconRadius := bs.resources.Config.UI.Battlefield.IconRadius
				indicatorHeight := bs.resources.Config.UI.Battlefield.TargetIndicator.Height
				indicatorWidth := bs.resources.Config.UI.Battlefield.TargetIndicator.Width
				margin := float32(5)
				p1x := currentX - indicatorWidth/2
				p1y := currentY - iconRadius - margin - indicatorHeight
				p2x := currentX + indicatorWidth/2
				p2y := p1y
				p3x := currentX
				p3y := currentY - iconRadius - margin
				vertices := []ebiten.Vertex{{DstX: p1x, DstY: p1y}, {DstX: p2x, DstY: p2y}, {DstX: p3x, DstY: p3y}}
				r, g, b, a := indicatorColor.RGBA()
				cr, cg, cb, ca := float32(r)/65535, float32(g)/65535, float32(b)/65535, float32(a)/65535
				for i := range vertices {
					vertices[i].ColorR, vertices[i].ColorG, vertices[i].ColorB, vertices[i].ColorA = cr, cg, cb, ca
				}
				screen.DrawTriangles(vertices, []uint16{0, 1, 2}, bs.whitePixel, &ebiten.DrawTrianglesOptions{})
			}

			const popupDelay = 30
			const popupDuration = 60
			if progress >= travelDuration+popupDelay && progress < travelDuration+popupDelay+popupDuration {
				popupProgress := (progress - (travelDuration + popupDelay)) / popupDuration
				x := targetVM.X
				y := targetVM.Y - 20 - (20 * float32(popupProgress))
				alpha := 1.0
				if popupProgress > 0.7 {
					alpha = (1.0 - popupProgress) / 0.3
				}
				geoM := ebiten.GeoM{}
				geoM.Translate(float64(x), float64(y))
				colorM := ebiten.ColorM{}
				colorM.Scale(1, 1, 1, alpha)
				text.Draw(screen, fmt.Sprintf("%d", anim.Result.OriginalDamage), bs.resources.Font, &text.DrawOptions{DrawImageOptions: ebiten.DrawImageOptions{GeoM: geoM, ColorM: colorM}, LayoutOptions: text.LayoutOptions{PrimaryAlign: text.AlignCenter, SecondaryAlign: text.AlignCenter}})
			}
		}
	}

	if bs.debugMode {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\nState: %s", ebiten.ActualTPS(), ebiten.ActualFPS(), bs.state))
	}
}

func (bs *BattleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return bs.resources.Config.UI.Screen.Width, bs.resources.Config.UI.Screen.Height
}

func (bs *BattleScene) enqueueMessage(msg string, callback func()) {
	bs.enqueueMessageQueue([]string{msg}, callback)
}

func (bs *BattleScene) enqueueMessageQueue(messages []string, callback func()) {
	bs.messageQueue = messages
	bs.currentMessageIndex = 0
	bs.postMessageCallback = callback
	bs.state = StateMessage
	if len(bs.messageQueue) > 0 {
		bs.ui.ShowMessageWindow(bs.messageQueue[0])
	}
}
