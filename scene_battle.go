package main

import (
	"fmt"
	// "reflect" // No longer needed here as actionQueueResourceType is used

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
	// "github.com/yohamta/donburi/resource" // Removed
)

// BattleScene は戦闘シーンのすべてを管理します
type BattleScene struct {
	resources           *SharedResources
	world               donburi.World
	tickCount           int
	debugMode           bool
	state               GameState
	playerTeam          TeamID
	// actionQueue         []*donburi.Entry // Removed: Will be managed as a world resource
	ui                  *UI
	message             string
	postMessageCallback func()
	winner              TeamID // Initialized to TeamNone
	restartRequested    bool
	playerMedarotToAct       *donburi.Entry
	playerActionPendingQueue []*donburi.Entry // Queue for player medarots waiting for action selection
	currentTarget            *donburi.Entry
	attackingEntity          *donburi.Entry
	targetedEntity           *donburi.Entry

	// リファクタリングで追加されたヘルパー
	damageCalculator *DamageCalculator
	hitCalculator    *HitCalculator
	targetSelector   *TargetSelector
	partInfoProvider *PartInfoProvider
}

// NewBattleScene は新しい戦闘シーンを初期化します
func NewBattleScene(res *SharedResources) *BattleScene {
	world := donburi.NewWorld()

	bs := &BattleScene{
		resources:          res,
		world:              world,
		tickCount:          0,
		debugMode:          true,
		state:              StatePlaying,
		playerTeam:         Team1,
		// actionQueue:        make([]*donburi.Entry, 0), // Removed
		playerMedarotToAct:       nil,
		playerActionPendingQueue: make([]*donburi.Entry, 0), // Initialize the new queue
		attackingEntity:          nil,
		targetedEntity:           nil,
		currentTarget:            nil,
		winner:                   TeamNone, // Initialize winner to TeamNone
		// ヘルパーは後で初期化
	}

	// ヘルパー構造体の初期化
	bs.partInfoProvider = NewPartInfoProvider(bs.world, &bs.resources.Config)
	bs.damageCalculator = NewDamageCalculator(bs.world, &bs.resources.Config)
	bs.hitCalculator = NewHitCalculator(bs.world, &bs.resources.Config)
	bs.targetSelector = NewTargetSelector(bs.world, &bs.resources.Config)

	// 依存性の注入
	// DamageCalculatorへの依存性設定
	if bs.damageCalculator != nil && bs.partInfoProvider != nil {
		bs.damageCalculator.SetPartInfoProvider(bs.partInfoProvider)
	} else {
		fmt.Println("Error: DamageCalculator or PartInfoProvider is nil during NewBattleScene setup.")
	}

	// HitCalculatorへの依存性設定
	if bs.hitCalculator != nil && bs.partInfoProvider != nil {
		bs.hitCalculator.SetPartInfoProvider(bs.partInfoProvider)
	} else {
		fmt.Println("Error: HitCalculator or PartInfoProvider is nil during NewBattleScene setup.")
	}

	// TargetSelectorへの依存性設定
	if bs.targetSelector != nil && bs.partInfoProvider != nil {
		bs.targetSelector.SetPartInfoProvider(bs.partInfoProvider)
	} else {
		fmt.Println("Error: TargetSelector or PartInfoProvider is nil during NewBattleScene setup.")
	}

	// Ensure the world state entity with ActionQueueComponent exists.
	// This will create the entity and the ActionQueueComponentData if they don't exist.
	EnsureActionQueueEntity(bs.world)
	RegisterStateChangeEventHandlers(bs.world) // Register event handlers

	CreateMedarotEntities(bs.world, bs.resources.GameData, bs.playerTeam)
	bs.ui = NewUI(bs) // UIの初期化はヘルパーの後でも良い

	return bs
}

// Update は戦闘シーンのロジックを更新します
func (bs *BattleScene) Update() (SceneType, error) {
	bs.ui.ebitenui.Update()

	switch bs.state {
	case StatePlaying:
		bs.tickCount++

		// Process inputs if no player is currently selecting an action AND the pending queue is empty.
		// This means we are not in the middle of a multi-unit player action sequence.
		if bs.playerMedarotToAct == nil && len(bs.playerActionPendingQueue) == 0 {
			UpdateAIInputSystem(bs.world, bs.partInfoProvider, bs.targetSelector, &bs.resources.Config)

			playerInputResult := UpdatePlayerInputSystem(bs.world)
			if len(playerInputResult.PlayerMedarotsToAct) > 0 {
				bs.playerActionPendingQueue = playerInputResult.PlayerMedarotsToAct
				// Don't set playerMedarotToAct or change state here yet,
				// let the logic below handle picking from the queue.
			}
		}

		// If there are players in the pending queue and no one is currently acting (modal not shown yet for current one)
		if bs.playerMedarotToAct == nil && len(bs.playerActionPendingQueue) > 0 {
			bs.playerMedarotToAct = bs.playerActionPendingQueue[0]
			// bs.playerActionPendingQueue = bs.playerActionPendingQueue[1:] // Dequeueing happens after selection in handleActionSelection
			bs.state = StatePlayerActionSelect
		}

		// Update gauges only if no player action is pending (neither currently selecting nor in queue)
		// AND the main action execution queue is also empty.
		actionQueueComp := GetActionQueueComponent(bs.world) // Get the action queue component
		if bs.playerMedarotToAct == nil && len(bs.playerActionPendingQueue) == 0 && len(actionQueueComp.Queue) == 0 {
			UpdateGaugeSystem(bs.world)
		}

		// If still in StatePlaying (e.g. AI might have queued an action, or no input was needed, and no player pending),
		// process the action queue. This needs to happen regardless of shouldSkipGaugeUpdateThisFrame
		// because AI actions or previously queued player actions should still execute.
		// However, if state changed to PlayerActionSelect, this part might be skipped or handled differently.
		if bs.state == StatePlaying { // Re-check state as it might have changed to PlayerActionSelect
			actionResults, err := UpdateActionQueueSystem(
				bs.world,
				bs.partInfoProvider,
				bs.damageCalculator,
				bs.hitCalculator,
				bs.targetSelector,
			&bs.resources.Config, // Pass gameConfig
			)
			if err != nil {
				// Handle error appropriately
				fmt.Println("Error processing action queue system:", err)
			}

			for _, result := range actionResults {
				if result.ActingEntry != nil && result.ActingEntry.Valid() {
					logComp := LogComponent.Get(result.ActingEntry)
					if logComp != nil {
						logComp.LastActionLog = result.LogMessage
					}
					bs.attackingEntity = result.ActingEntry // For UI indication
					bs.targetedEntity = result.TargetEntry   // For UI indication

					// Enqueue message and set callback for cooldown
					bs.enqueueMessage(result.LogMessage, func() {
						// Check validity and state again before starting cooldown
						if result.ActingEntry.Valid() && !result.ActingEntry.HasComponent(BrokenStateComponent) {
							// Call StartCooldownSystem (moved from systems.go)
							StartCooldownSystem(result.ActingEntry, bs.world, &bs.resources.Config)
						}
						bs.attackingEntity = nil // Clear after action processed
						bs.targetedEntity = nil  // Clear after action processed
					})
					// If a message was enqueued, state changes to StateMessage, so we might want to break/return early
					// to avoid further processing in StatePlaying this frame.
					if bs.state == StateMessage {
						break // Exit the actionResults loop as state has changed
					}
				}
			}
		}

		// Check for game end only if still in StatePlaying and no message is being shown
		// (as enqueueMessage from actionResults could change state to StateMessage)
		if bs.state == StatePlaying {
			gameEndResult := CheckGameEndSystem(bs.world)
			if gameEndResult.IsGameOver {
				bs.winner = gameEndResult.Winner
				bs.state = StateGameOver
				bs.enqueueMessage(gameEndResult.Message, nil)
			}
		}
		updateAllInfoPanels(bs)
		if bs.ui.battlefieldWidget != nil {
			bs.ui.battlefieldWidget.UpdatePositions()
		}
	case StatePlayerActionSelect:
		// Ensure battlefield icons are updated while player is selecting action
		if bs.ui.battlefieldWidget != nil {
			bs.ui.battlefieldWidget.UpdatePositions()
		}
		// If action modal is not yet shown and a player unit needs to act, show it.
		if bs.ui.actionModal == nil && bs.playerMedarotToAct != nil {
			// Ensure the selected medarot is still valid and in Idle state
			if bs.playerMedarotToAct.Valid() && bs.playerMedarotToAct.HasComponent(IdleStateComponent) {
				bs.ui.ShowActionModal(bs, bs.playerMedarotToAct)
			} else {
				// Medarot is no longer valid or not in Idle state, reset.
				bs.playerMedarotToAct = nil
				bs.state = StatePlaying // Go back to playing state to re-evaluate
			}
		}
	case StateMessage:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			bs.ui.HideMessageWindow()
			if bs.postMessageCallback != nil {
				bs.postMessageCallback()
				bs.postMessageCallback = nil
			}

			// Determine next state based on whether a winner was declared
			if bs.winner != TeamNone { // If a winner was set (game was over)
				bs.state = StateGameOver
			} else {
				bs.state = StatePlaying
				// Input systems (AI/Player) will be called in the next StatePlaying iteration
				// if bs.playerMedarotToAct is nil.
			}
			// SystemProcessIdleMedarots(bs) // Removed: Handled by StatePlaying logic
		}
	case StateGameOver:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			return SceneTypeTitle, nil // ゲームオーバー時にクリックでタイトルに戻る
		}
	}
	return SceneTypeBattle, nil
}

// Draw は戦闘シーンを描画します
func (bs *BattleScene) Draw(screen *ebiten.Image) {
	screen.Fill(bs.resources.Config.UI.Colors.Background)
	bs.ui.ebitenui.Draw(screen)

	bf := bs.ui.battlefieldWidget
	if bf != nil {
		bf.DrawBackground(screen)
		bf.DrawIcons(screen)

		var indicatorTarget *donburi.Entry
		if bs.state == StatePlayerActionSelect && bs.currentTarget != nil {
			indicatorTarget = bs.currentTarget
		} else if bs.state == StateMessage && bs.targetedEntity != nil {
			indicatorTarget = bs.targetedEntity
		}

		if indicatorTarget != nil {
			bf.DrawTargetIndicator(screen, indicatorTarget)
		}
		bf.DrawDebug(screen)
	}
	if bs.debugMode {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\nState: %s",
			ebiten.ActualTPS(), ebiten.ActualFPS(), bs.state))
	}
}

// enqueueMessage はバトルシーン内でのメッセージ表示を扱います
func (bs *BattleScene) enqueueMessage(msg string, callback func()) {
	bs.message = msg
	bs.postMessageCallback = callback
	bs.state = StateMessage
	bs.ui.ShowMessageWindow(bs)
}
