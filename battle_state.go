package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
)

// BattleState は戦闘シーンの各状態が満たすべきインターフェースです。
type BattleState interface {
	Update(world donburi.World, battleLogic *BattleLogic, ui UIInterface, messageManager *UIMessageDisplayManager, config *Config, tick int, winner TeamID, sceneManager *SceneManager, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, GameState, TeamID, error)
	Draw(screen *ebiten.Image)
}

// --- PlayingState ---

type PlayingState struct{}

func (s *PlayingState) Update(world donburi.World, battleLogic *BattleLogic, ui UIInterface, messageManager *UIMessageDisplayManager, config *Config, tick int, winner TeamID, sceneManager *SceneManager, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, GameState, TeamID, error) {
	// AIの行動選択
	if !ui.IsActionModalVisible() && len(playerActionPendingQueue) == 0 {
		UpdateAIInputSystem(world, battleLogic.PartInfoProvider, battleLogic.TargetSelector)
	}

	// プレイヤーの行動選択が必要かチェック
	playerInputResult := UpdatePlayerInputSystem(world)
	if len(playerInputResult.PlayerMedarotsToAct) > 0 {
		playerActionPendingQueue = playerInputResult.PlayerMedarotsToAct
		return playerActionPendingQueue, StatePlayerActionSelect, winner, nil
	}

	// ゲージ進行
	actionQueueComp := GetActionQueueComponent(world)
	if !ui.IsActionModalVisible() && len(playerActionPendingQueue) == 0 && len(actionQueueComp.Queue) == 0 {
		UpdateGaugeSystem(world)
	}

	// アクション実行
	actionResults, err := UpdateActionQueueSystem(world, battleLogic, config)
	if err != nil {
		fmt.Println("アクションキューシステムの処理中にエラーが発生しました:", err)
	}

	for _, result := range actionResults {
		if result.ActingEntry != nil && result.ActingEntry.Valid() {
			ui.SetAnimation(&ActionAnimationData{Result: result, StartTime: tick})
			return playerActionPendingQueue, StateAnimatingAction, winner, nil
		}
	}

	// ゲーム終了判定
	gameEndResult := CheckGameEndSystem(world)
	if gameEndResult.IsGameOver {
		winner = gameEndResult.Winner
		messageManager.EnqueueMessage(gameEndResult.Message, nil)
		return playerActionPendingQueue, StateMessage, winner, nil
	}

	return playerActionPendingQueue, StatePlaying, winner, nil // 状態は維持
}

func (s *PlayingState) Draw(screen *ebiten.Image) {
	// Playing状態固有の描画があればここに記述
}

// --- PlayerActionSelectState ---

type PlayerActionSelectState struct{}

func (s *PlayerActionSelectState) Update(world donburi.World, battleLogic *BattleLogic, ui UIInterface, messageManager *UIMessageDisplayManager, config *Config, tick int, winner TeamID, sceneManager *SceneManager, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, GameState, TeamID, error) {
	// If the action modal is currently visible, the player is still making a choice.
	if ui.IsActionModalVisible() {
		return playerActionPendingQueue, StatePlayerActionSelect, winner, nil
	}

	// If the modal is not visible, it means a choice was made (or cancelled)
	// in processUIEvents, or it's the first time entering this state.

	// Check if there are players still waiting to act.
	if len(playerActionPendingQueue) > 0 {
		actingEntry := playerActionPendingQueue[0]

		// If the current acting entry is valid and idle, show the modal for them.
		if actingEntry.Valid() && StateComponent.Get(actingEntry).FSM.Is(string(StateIdle)) {
			actionTargetMap := make(map[PartSlotKey]ActionTarget)
			availableParts := battleLogic.PartInfoProvider.GetAvailableAttackParts(actingEntry)
			for _, available := range availableParts {
				partDef := available.PartDef
				slotKey := available.Slot
				var targetEntity *donburi.Entry
				var targetPartSlot PartSlotKey
				if partDef.Category == CategoryRanged || partDef.Category == CategoryIntervention {
					medal := MedalComponent.Get(actingEntry)
					personality, ok := PersonalityRegistry[medal.Personality]
					if !ok {
						personality = PersonalityRegistry["リーダー"]
					}
					targetEntity, targetPartSlot = personality.TargetingStrategy.SelectTarget(world, actingEntry, battleLogic.TargetSelector, battleLogic.PartInfoProvider)
				}
				actionTargetMap[slotKey] = ActionTarget{Target: targetEntity, Slot: targetPartSlot}
			}
			ui.ShowActionModal(actingEntry, actionTargetMap)
			return playerActionPendingQueue, StatePlayerActionSelect, winner, nil // Stay in this state while modal is shown
		} else {
			// If the current acting entry is invalid or not idle, remove it from the queue
			// and re-evaluate for the next player.
			playerActionPendingQueue = playerActionPendingQueue[1:]
			// Recursively call Update to process the next player in the queue immediately
			// or transition to PlayingState if the queue is now empty.
			return s.Update(world, battleLogic, ui, messageManager, config, tick, winner, sceneManager, playerActionPendingQueue)
		}
	}

	// If the queue is empty, all player actions have been processed.
	return playerActionPendingQueue, StatePlaying, winner, nil
}

func (s *PlayerActionSelectState) Draw(screen *ebiten.Image) {}

// --- AnimatingActionState ---

type AnimatingActionState struct{}

func (s *AnimatingActionState) Update(world donburi.World, battleLogic *BattleLogic, ui UIInterface, messageManager *UIMessageDisplayManager, config *Config, tick int, winner TeamID, sceneManager *SceneManager, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, GameState, TeamID, error) {
	if ui.IsAnimationFinished(tick) {
		result := ui.GetCurrentAnimationResult()
		messages := buildActionLogMessages(result)

		messageManager.EnqueueMessageQueue(messages, func() {
			UpdateHistorySystem(world, &result)

			actingEntry := result.ActingEntry
			if actingEntry.Valid() && !StateComponent.Get(actingEntry).FSM.Is(string(StateBroken)) {
				StartCooldownSystem(actingEntry, world, battleLogic.PartInfoProvider)
			}
			ui.ClearCurrentTarget()
		})
		return playerActionPendingQueue, StateMessage, winner, nil
	}
	return playerActionPendingQueue, StateAnimatingAction, winner, nil
}

func (s *AnimatingActionState) Draw(screen *ebiten.Image) {}

// --- MessageState ---

type MessageState struct{}

func (s *MessageState) Update(world donburi.World, battleLogic *BattleLogic, ui UIInterface, messageManager *UIMessageDisplayManager, config *Config, tick int, winner TeamID, sceneManager *SceneManager, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, GameState, TeamID, error) {
	newState, finished := messageManager.Update(StateMessage) // Pass StateMessage as current state
	if finished {
		ui.ClearAnimation()
		if winner != TeamNone {
			return playerActionPendingQueue, StateGameOver, winner, nil
		}
		return playerActionPendingQueue, StatePlaying, winner, nil
	}
	return playerActionPendingQueue, newState, winner, nil
}

func (s *MessageState) Draw(screen *ebiten.Image) {}

// --- GameOverState ---

type GameOverState struct{}

func (s *GameOverState) Update(world donburi.World, battleLogic *BattleLogic, ui UIInterface, messageManager *UIMessageDisplayManager, config *Config, tick int, winner TeamID, sceneManager *SceneManager, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, GameState, TeamID, error) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		sceneManager.GoToTitleScene()
	}
	return playerActionPendingQueue, StateGameOver, winner, nil
}

func (s *GameOverState) Draw(screen *ebiten.Image) {}
