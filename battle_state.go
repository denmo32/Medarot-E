package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
)

// BattleContext は戦闘シーンの各状態が共通して必要とする依存関係をまとめた構造体です。
type BattleContext struct {
	World          donburi.World
	BattleLogic    *BattleLogic
	UI             UIInterface
	MessageManager *UIMessageDisplayManager
	Config         *Config
	SceneManager   *SceneManager
	Tick           int
}

// BattleState は戦闘シーンの各状態が満たすべきインターフェースです。
type BattleState interface {
	Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, *StateUpdateResult, error)
	Draw(screen *ebiten.Image)
}

// --- PlayingState ---

type PlayingState struct{}

func (s *PlayingState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, *StateUpdateResult, error) {
	world := ctx.World
	battleLogic := ctx.BattleLogic
	ui := ctx.UI
	messageManager := ctx.MessageManager
	config := ctx.Config
	tick := ctx.Tick
	// sceneManager := ctx.SceneManager // PlayingStateでは未使用

	// AIの行動選択
	if !ui.IsActionModalVisible() && len(playerActionPendingQueue) == 0 {
		UpdateAIInputSystem(world, battleLogic)
	}

	// プレイヤーの行動選択が必要かチェック
	playerInputResult := UpdatePlayerInputSystem(world)
	if len(playerInputResult.PlayerMedarotsToAct) > 0 {
		playerActionPendingQueue = playerInputResult.PlayerMedarotsToAct
		return playerActionPendingQueue, &StateUpdateResult{PlayerActionRequired: true}, nil
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
			return playerActionPendingQueue, &StateUpdateResult{ActionStarted: true}, nil
		}
	}

	// ゲーム終了判定
	gameEndResult := CheckGameEndSystem(world)
	if gameEndResult.IsGameOver {
		messageManager.EnqueueMessage(gameEndResult.Message, nil)
		return playerActionPendingQueue, &StateUpdateResult{GameOver: true, Winner: gameEndResult.Winner, MessageQueued: true}, nil
	}

	return playerActionPendingQueue, &StateUpdateResult{}, nil // 状態は維持
}

func (s *PlayingState) Draw(screen *ebiten.Image) {
	// Playing状態固有の描画があればここに記述
}

// --- PlayerActionSelectState ---

type PlayerActionSelectState struct{}

func (s *PlayerActionSelectState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, *StateUpdateResult, error) {
	world := ctx.World
	battleLogic := ctx.BattleLogic
	ui := ctx.UI
	// messageManager := ctx.MessageManager // PlayerActionSelectStateでは未使用
	// config := ctx.Config // PlayerActionSelectStateでは未使用
	// tick := ctx.Tick // PlayerActionSelectStateでは未使用
	// sceneManager := ctx.SceneManager // PlayerActionSelectStateでは未使用

	// モーダル表示中は何もしない
	if ui.IsActionModalVisible() {
		return playerActionPendingQueue, &StateUpdateResult{}, nil
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
			actionModalVM := BuildActionModalViewModel(actingEntry, actionTargetMap) // 新規追加
			ui.ShowActionModal(actionModalVM)                                        // 変更: ViewModelを渡す
			return playerActionPendingQueue, &StateUpdateResult{}, nil               // モーダル表示中は状態維持
		} else {
			// 無効または待機状態でないならキューから削除して次のプレイヤーを処理
			playerActionPendingQueue = playerActionPendingQueue[1:]
			// 即座に次のプレイヤーを評価するため、再帰的に呼び出す
			return s.Update(ctx, playerActionPendingQueue)
		}
	}

	// キューが空なら処理完了
	return playerActionPendingQueue, &StateUpdateResult{}, nil
}

func (s *PlayerActionSelectState) Draw(screen *ebiten.Image) {}

// --- AnimatingActionState ---

type AnimatingActionState struct{}

func (s *AnimatingActionState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, *StateUpdateResult, error) {
	world := ctx.World
	battleLogic := ctx.BattleLogic
	ui := ctx.UI
	messageManager := ctx.MessageManager
	tick := ctx.Tick
	// config := ctx.Config // AnimatingActionStateでは未使用
	// sceneManager := ctx.SceneManager // AnimatingActionStateでは未使用

	if ui.IsAnimationFinished(tick) {
		result := ui.GetCurrentAnimationResult() // まず結果を取得
		ui.ClearAnimation()                      // その後でアニメーションをクリア

		messages := buildActionLogMessages(result)

		messageManager.EnqueueMessageQueue(messages, func() {
			UpdateHistorySystem(world, &result)

			actingEntry := result.ActingEntry
			if actingEntry.Valid() && !StateComponent.Get(actingEntry).FSM.Is(string(StateBroken)) {
				StartCooldownSystem(actingEntry, world, battleLogic)
			}
			ui.ClearCurrentTarget()
		})
		return playerActionPendingQueue, &StateUpdateResult{MessageQueued: true}, nil
	}
	return playerActionPendingQueue, &StateUpdateResult{}, nil
}

func (s *AnimatingActionState) Draw(screen *ebiten.Image) {}

// --- MessageState ---

type MessageState struct{}

func (s *MessageState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, *StateUpdateResult, error) {
	messageManager := ctx.MessageManager
	// world := ctx.World // MessageStateでは未使用
	// battleLogic := ctx.BattleLogic // MessageStateでは未使用
	// ui := ctx.UI // MessageStateでは未使用
	// config := ctx.Config // MessageStateでは未使用
	// tick := ctx.Tick // MessageStateでは未使用
	// sceneManager := ctx.SceneManager // MessageStateでは未使用

	messageManager.Update()
	messageFinished := messageManager.IsFinished()
	// メッセージ表示が完了したかどうかをBattleSceneに通知
	return playerActionPendingQueue, &StateUpdateResult{MessageFinished: messageFinished}, nil
}

func (s *MessageState) Draw(screen *ebiten.Image) {}

// --- GameOverState ---

type GameOverState struct{}

func (s *GameOverState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, *StateUpdateResult, error) {
	sceneManager := ctx.SceneManager
	// world := ctx.World // GameOverStateでは未使用
	// battleLogic := ctx.BattleLogic // GameOverStateでは未使用
	// ui := ctx.UI // GameOverStateでは未使用
	// messageManager := ctx.MessageManager // GameOverStateでは未使用
	// config := ctx.Config // GameOverStateでは未使用
	// tick := ctx.Tick // GameOverStateでは未使用

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		sceneManager.GoToTitleScene()
	}
	return playerActionPendingQueue, &StateUpdateResult{}, nil
}

func (s *GameOverState) Draw(screen *ebiten.Image) {}
