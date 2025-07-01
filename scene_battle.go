package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
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
	winner              TeamID
	restartRequested    bool
	playerMedarotToAct  *donburi.Entry
	currentTarget       *donburi.Entry
	attackingEntity     *donburi.Entry
	targetedEntity      *donburi.Entry

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
		playerMedarotToAct: nil,
		attackingEntity:    nil,
		targetedEntity:     nil,
		currentTarget:      nil,
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

	// Initialize and add ActionQueueResource to the world
	actionQueue := &ActionQueueResource{
		Queue: make([]*donburi.Entry, 0),
	}
	donburi.AddResource(bs.world, ActionQueueResourceType, actionQueue)

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
		UpdateGaugeSystem(bs.world) // Modified: Call the new system

		// Call the new ActionQueueSystem
		actionResults, err := UpdateActionQueueSystem(
			bs.world,
			bs.partInfoProvider,
			bs.damageCalculator,
			bs.hitCalculator,
			bs.targetSelector,
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
			}
		}

		// Process AI and Player inputs if no player is currently acting and game is playing
		if bs.playerMedarotToAct == nil && bs.state == StatePlaying {
			UpdateAIInputSystem(bs.world, bs.partInfoProvider, bs.targetSelector, &bs.resources.Config)

			playerInputResult := UpdatePlayerInputSystem(bs.world)
			if playerInputResult.PlayerMedarotToAct != nil {
				bs.playerMedarotToAct = playerInputResult.PlayerMedarotToAct
				bs.state = StatePlayerActionSelect
			}
		}

		if bs.state == StatePlaying { // Only check game end if still playing
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
