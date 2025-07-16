package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
)

// BattleState は戦闘シーンの各状態が満たすべきインターフェースです。
type BattleState interface {
	Update(scene *BattleScene) (GameState, error)
	Draw(screen *ebiten.Image)
}

// --- PlayingState ---

type PlayingState struct{}

func (s *PlayingState) Update(scene *BattleScene) (GameState, error) {
	// AIの行動選択
	if !scene.ui.IsActionModalVisible() && len(scene.playerActionPendingQueue) == 0 {
		UpdateAIInputSystem(scene.world, scene.battleLogic.PartInfoProvider, scene.battleLogic.TargetSelector)
	}

	// プレイヤーの行動選択が必要かチェック
	playerInputResult := UpdatePlayerInputSystem(scene.world)
	if len(playerInputResult.PlayerMedarotsToAct) > 0 {
		scene.playerActionPendingQueue = playerInputResult.PlayerMedarotsToAct
		return StatePlayerActionSelect, nil
	}

	// ゲージ進行
	actionQueueComp := GetActionQueueComponent(scene.world)
	if !scene.ui.IsActionModalVisible() && len(scene.playerActionPendingQueue) == 0 && len(actionQueueComp.Queue) == 0 {
		UpdateGaugeSystem(scene.world)
	}

	// アクション実行
	actionResults, err := UpdateActionQueueSystem(scene.world, scene.battleLogic, &scene.resources.Config)
	if err != nil {
		fmt.Println("アクションキューシステムの処理中にエラーが発生しました:", err)
	}

	for _, result := range actionResults {
		if result.ActingEntry != nil && result.ActingEntry.Valid() {
			scene.ui.SetAnimation(&ActionAnimationData{Result: result, StartTime: scene.tickCount})
			return StateAnimatingAction, nil
		}
	}

	// ゲーム終了判定
	gameEndResult := CheckGameEndSystem(scene.world)
	if gameEndResult.IsGameOver {
		scene.winner = gameEndResult.Winner
		scene.messageManager.EnqueueMessage(gameEndResult.Message, nil)
		return StateMessage, nil
	}

	return StatePlaying, nil // 状態は維持
}

func (s *PlayingState) Draw(screen *ebiten.Image) {
	// Playing状態固有の描画があればここに記述
}

// --- PlayerActionSelectState ---

type PlayerActionSelectState struct{}

func (s *PlayerActionSelectState) Update(scene *BattleScene) (GameState, error) {
	// If the action modal is currently visible, the player is still making a choice.
	if scene.ui.IsActionModalVisible() {
		return StatePlayerActionSelect, nil
	}

	// If the modal is not visible, it means a choice was made (or cancelled)
	// in processUIEvents, or it's the first time entering this state.

	// Check if there are players still waiting to act.
	if len(scene.playerActionPendingQueue) > 0 {
		actingEntry := scene.playerActionPendingQueue[0]

		// If the current acting entry is valid and idle, show the modal for them.
		if actingEntry.Valid() && StateComponent.Get(actingEntry).FSM.Is(string(StateIdle)) {
			actionTargetMap := make(map[PartSlotKey]ActionTarget)
			availableParts := scene.battleLogic.PartInfoProvider.GetAvailableAttackParts(actingEntry)
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
					targetEntity, targetPartSlot = personality.TargetingStrategy.SelectTarget(scene.world, actingEntry, scene.battleLogic.TargetSelector, scene.battleLogic.PartInfoProvider)
				}
				actionTargetMap[slotKey] = ActionTarget{Target: targetEntity, Slot: targetPartSlot}
			}
			scene.ui.ShowActionModal(actingEntry, actionTargetMap)
			return StatePlayerActionSelect, nil // Stay in this state while modal is shown
		} else {
			// If the current acting entry is invalid or not idle, remove it from the queue
			// and re-evaluate for the next player.
			scene.playerActionPendingQueue = scene.playerActionPendingQueue[1:]
			// Recursively call Update to process the next player in the queue immediately
			// or transition to PlayingState if the queue is now empty.
			return s.Update(scene)
		}
	}

	// If the queue is empty, all player actions have been processed.
	return StatePlaying, nil
}

func (s *PlayerActionSelectState) Draw(screen *ebiten.Image) {}

// --- AnimatingActionState ---

type AnimatingActionState struct{}

func (s *AnimatingActionState) Update(scene *BattleScene) (GameState, error) {
	if scene.ui.IsAnimationFinished(scene.tickCount) {
		result := scene.ui.GetCurrentAnimationResult()
		messages := scene.buildActionLogMessages(result)

		scene.messageManager.EnqueueMessageQueue(messages, func() {
			UpdateHistorySystem(scene.world, &result)

			actingEntry := result.ActingEntry
			if actingEntry.Valid() && !StateComponent.Get(actingEntry).FSM.Is(string(StateBroken)) {
				StartCooldownSystem(actingEntry, scene.world, scene.battleLogic.PartInfoProvider)
			}
			scene.ui.ClearCurrentTarget()
		})
		return StateMessage, nil
	}
	return StateAnimatingAction, nil
}

func (s *AnimatingActionState) Draw(screen *ebiten.Image) {}

// --- MessageState ---

type MessageState struct{}

func (s *MessageState) Update(scene *BattleScene) (GameState, error) {
	newState, finished := scene.messageManager.Update(scene.state)
	if finished {
		scene.ui.ClearAnimation()
		if scene.winner != TeamNone {
			return StateGameOver, nil
		}
		return StatePlaying, nil
	}
	return newState, nil
}

func (s *MessageState) Draw(screen *ebiten.Image) {}

// --- GameOverState ---

type GameOverState struct{}

func (s *GameOverState) Update(scene *BattleScene) (GameState, error) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		scene.manager.GoToTitleScene()
	}
	return StateGameOver, nil
}

func (s *GameOverState) Draw(screen *ebiten.Image) {}
