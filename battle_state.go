package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
)

// BattleContext は戦闘シーンの各状態が共通して必要とする依存関係をまとめた構造体です。
type BattleContext struct {
	World        donburi.World
	BattleLogic  *BattleLogic
	UI           UIInterface
	Config       *Config
	SceneManager *SceneManager
	Tick         int
}

// BattleState は戦闘シーンの各状態が満たすべきインターフェースです。
type BattleState interface {
	Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []GameEvent, error)
	Draw(screen *ebiten.Image)
}

// --- PlayingState ---

type PlayingState struct{}

func (s *PlayingState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []GameEvent, error) {
	world := ctx.World
	battleLogic := ctx.BattleLogic
	ui := ctx.UI
	config := ctx.Config
	tick := ctx.Tick

	var gameEvents []GameEvent

	// AIの行動選択
	if !ui.IsActionModalVisible() && len(playerActionPendingQueue) == 0 {
		UpdateAIInputSystem(world, battleLogic)
	}

	// プレイヤーの行動選択が必要かチェック
	playerInputResult := UpdatePlayerInputSystem(world)
	if len(playerInputResult.PlayerMedarotsToAct) > 0 {
		playerActionPendingQueue = playerInputResult.PlayerMedarotsToAct
		gameEvents = append(gameEvents, PlayerActionRequiredGameEvent{})
		return playerActionPendingQueue, gameEvents, nil
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
			gameEvents = append(gameEvents, ActionAnimationStartedGameEvent{AnimationData: ActionAnimationData{Result: result, StartTime: tick}})
			return playerActionPendingQueue, gameEvents, nil
		}
	}

	// ゲーム終了判定
	gameEndResult := CheckGameEndSystem(world)
	if gameEndResult.IsGameOver {
		gameEvents = append(gameEvents, MessageDisplayRequestGameEvent{Messages: []string{gameEndResult.Message}, Callback: nil})
		gameEvents = append(gameEvents, GameOverGameEvent{Winner: gameEndResult.Winner})
		return playerActionPendingQueue, gameEvents, nil
	}

	return playerActionPendingQueue, gameEvents, nil // 状態は維持
}

func (s *PlayingState) Draw(screen *ebiten.Image) {
	// Playing状態固有の描画があればここに記述
}

// --- PlayerActionSelectState ---

type PlayerActionSelectState struct{}

func (s *PlayerActionSelectState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []GameEvent, error) {
	world := ctx.World
	battleLogic := ctx.BattleLogic
	ui := ctx.UI

	var gameEvents []GameEvent

	// モーダル表示中は何もしない
	if ui.IsActionModalVisible() {
		return playerActionPendingQueue, gameEvents, nil
	}

	// 待機中のプレイヤーがいるかチェック
	if len(playerActionPendingQueue) > 0 {
		actingEntry := playerActionPendingQueue[0]

		// 有効で待機状態ならモーダルを表示
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
					targetEntity, targetPartSlot = personality.TargetingStrategy.SelectTarget(world, actingEntry, battleLogic)
				}
				actionTargetMap[slotKey] = ActionTarget{Target: targetEntity, Slot: targetPartSlot}
			}

			// ここでViewModelを構築し、UIに渡す
			actionModalVM := BuildActionModalViewModel(actingEntry, actionTargetMap, battleLogic)
			gameEvents = append(gameEvents, ShowActionModalGameEvent{ViewModel: actionModalVM})
			return playerActionPendingQueue, gameEvents, nil
		} else {
			// 無効または待機状態でないならキューから削除して次のプレイヤーを処理
			playerActionPendingQueue = playerActionPendingQueue[1:]
			// 即座に次のプレイヤーを評価するため、再帰的に呼び出す
			return s.Update(ctx, playerActionPendingQueue)
		}
	}

	// キューが空なら処理完了
	return playerActionPendingQueue, gameEvents, nil
}

func (s *PlayerActionSelectState) Draw(screen *ebiten.Image) {}

// --- AnimatingActionState ---

type AnimatingActionState struct{}

func (s *AnimatingActionState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []GameEvent, error) {
	world := ctx.World
	ui := ctx.UI
	tick := ctx.Tick

	var gameEvents []GameEvent

	if ui.IsAnimationFinished(tick) {
		result := ui.GetCurrentAnimationResult()
		gameEvents = append(gameEvents, ClearAnimationGameEvent{})
		gameEvents = append(gameEvents, MessageDisplayRequestGameEvent{Messages: buildActionLogMessages(result), Callback: func() {
			UpdateHistorySystem(world, &result)
		}})
		gameEvents = append(gameEvents, ActionAnimationFinishedGameEvent{Result: result, ActingEntry: result.ActingEntry})
		return playerActionPendingQueue, gameEvents, nil
	}
	return playerActionPendingQueue, gameEvents, nil
}

func (s *AnimatingActionState) Draw(screen *ebiten.Image) {}

// --- MessageState ---

type MessageState struct{}

func (s *MessageState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []GameEvent, error) {
	// MessageStateはMessageDisplayFinishedGameEventを返すのみで、MessageManagerのUpdateはBattleSceneで行う
	var gameEvents []GameEvent

	// MessageManagerのIsFinished()はBattleSceneでチェックされるため、ここではイベントを生成するのみ
	// if ctx.MessageManager.IsFinished() { // このチェックはBattleSceneに移動
	// 	gameEvents = append(gameEvents, MessageDisplayFinishedGameEvent{})
	// }
	// MessageStateはメッセージ表示の完了を待つ状態なので、ここではイベントを生成しない
	// 完了はBattleSceneでMessageManager.IsFinished()をチェックして判断する
	return playerActionPendingQueue, gameEvents, nil
}

func (s *MessageState) Draw(screen *ebiten.Image) {}

// --- GameOverState ---

type GameOverState struct{}

func (s *GameOverState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []GameEvent, error) {
	sceneManager := ctx.SceneManager
	var gameEvents []GameEvent

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		sceneManager.GoToTitleScene()
	}
	return playerActionPendingQueue, gameEvents, nil
}

func (s *GameOverState) Draw(screen *ebiten.Image) {}
