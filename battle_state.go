package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
)

// BattleContext は戦闘シーンの各状態が共通して必要とする依存関係をまとめた構造体です。
type BattleContext struct {
	World            donburi.World
	BattleLogic      *BattleLogic
	Config           *Config
	GameDataManager  *GameDataManager
	Tick             int
	ViewModelFactory ViewModelFactory
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
	// ui := ctx.UI // 削除
	config := ctx.Config
	tick := ctx.Tick // ctx.Tick を使用

	var gameEvents []GameEvent

	// AIの行動選択
	if len(playerActionPendingQueue) == 0 { // UIの表示状態はBattleSceneで管理
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
	if len(playerActionPendingQueue) == 0 && len(actionQueueComp.Queue) == 0 { // UIの表示状態はBattleSceneで管理
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
	viewModelFactory := ctx.ViewModelFactory

	var gameEvents []GameEvent

	// 待機中のプレイヤーがいるかチェック
	if len(playerActionPendingQueue) > 0 {
		actingEntry := playerActionPendingQueue[0]

		// 有効で待機状態ならモーダルを表示
		if actingEntry.Valid() && StateComponent.Get(actingEntry).CurrentState == StateIdle {
			actionTargetMap := make(map[PartSlotKey]ActionTarget)
			// ViewModelFactoryを介して利用可能なパーツを取得
			availableParts := viewModelFactory.GetAvailableAttackParts(actingEntry)
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
					// ViewModelFactoryを介してターゲットを選択
					targetEntity, targetPartSlot = personality.TargetingStrategy.SelectTarget(world, actingEntry, battleLogic)
				}
				actionTargetMap[slotKey] = ActionTarget{Target: targetEntity, Slot: targetPartSlot}
			}

			// ここでViewModelを構築し、UIに渡す
			actionModalVM := viewModelFactory.BuildActionModalViewModel(actingEntry, actionTargetMap, battleLogic)
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
	// world := ctx.World // 削除
	// ui := ctx.UI // UIへの直接参照を削除
	// tick := ctx.Tick // tickはBattleSceneで管理

	var gameEvents []GameEvent

	// アニメーションの終了はBattleSceneで判定し、ActionAnimationFinishedGameEventを発行する
	// ここではアニメーションが進行中であることを前提とする
	// アニメーション終了時にBattleSceneがこの状態からPlaying状態へ遷移させる

	// アニメーション終了判定はBattleSceneで行われるため、ここでは何もしない
	// BattleSceneがActionAnimationFinishedGameEventを受け取った際に、
	// MessageDisplayRequestGameEventとClearAnimationGameEventを発行する

	return playerActionPendingQueue, gameEvents, nil
}

func (s *AnimatingActionState) Draw(screen *ebiten.Image) {}

// --- MessageState ---

type MessageState struct{}

func (s *MessageState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []GameEvent, error) {
	// MessageStateはメッセージ表示の完了を待つ状態なので、ここではイベントを生成しない
	// 完了はBattleSceneでMessageManager.IsFinished()をチェックして判断する
	var gameEvents []GameEvent
	return playerActionPendingQueue, gameEvents, nil
}

func (s *MessageState) Draw(screen *ebiten.Image) {}

// --- GameOverState ---

type GameOverState struct{}

func (s *GameOverState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []GameEvent, error) {
	// sceneManager := ctx.SceneManager // SceneManagerへの直接参照を削除
	var gameEvents []GameEvent

	// ここではクリックイベントを直接処理せず、GameEventとして発行する
	// BattleSceneがこのイベントを受け取り、シーン遷移を行う
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		gameEvents = append(gameEvents, GoToTitleSceneGameEvent{})
	}
	return playerActionPendingQueue, gameEvents, nil
}

func (s *GameOverState) Draw(screen *ebiten.Image) {}
