package main

import (
	"fmt"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
)

// BattleContext は戦闘シーンの各状態が共通して必要とする依存関係をまとめた構造体です。
type BattleContext struct {
	World                  donburi.World
	Config                 *Config
	GameDataManager        *GameDataManager
	Rand                   *rand.Rand
	Tick                   int
	ViewModelFactory       ViewModelFactory
	statusEffectSystem     *StatusEffectSystem
	postActionEffectSystem *PostActionEffectSystem
	BattleLogic            *BattleLogic
}

// BattleState は戦闘シーンの各状態が満たすべきインターフェースです。
type BattleState interface {
	Update(ctx *BattleContext) ([]GameEvent, error)
	Draw(screen *ebiten.Image)
}

// --- PlayingState ---

type PlayingState struct{}

func (s *PlayingState) Update(ctx *BattleContext) ([]GameEvent, error) {
	world := ctx.World
	config := ctx.Config
	tick := ctx.Tick

	battleLogic := ctx.BattleLogic

	var gameEvents []GameEvent

	playerActionQueue := GetPlayerActionQueueComponent(world)

	// AIの行動選択
	if len(playerActionQueue.Queue) == 0 {
		UpdateAIInputSystem(world, battleLogic)
	}

	// プレイヤーの行動選択が必要かチェック
	if UpdatePlayerInputSystem(world) {
		gameEvents = append(gameEvents, PlayerActionRequiredGameEvent{})
		return gameEvents, nil
	}

	// ゲージ進行
	actionQueueComp := GetActionQueueComponent(world)
	if len(playerActionQueue.Queue) == 0 && len(actionQueueComp.Queue) == 0 {
		UpdateGaugeSystem(world)
	}

	// アクション実行
	actionResults, err := UpdateActionQueueSystem(world, battleLogic.GetDamageCalculator(), battleLogic.GetHitCalculator(), battleLogic.GetTargetSelector(), battleLogic.GetPartInfoProvider(), config, ctx.statusEffectSystem, ctx.postActionEffectSystem, ctx.Rand)
	if err != nil {
		fmt.Println("アクションキューシステムの処理中にエラーが発生しました:", err)
	}

	for _, result := range actionResults {
		if result.ActingEntry != nil && result.ActingEntry.Valid() {
			gameEvents = append(gameEvents, ActionAnimationStartedGameEvent{AnimationData: ActionAnimationData{Result: result, StartTime: tick}})
			return gameEvents, nil
		}
	}

	// ゲーム終了判定
	gameEndResult := CheckGameEndSystem(world)
	if gameEndResult.IsGameOver {
		gameEvents = append(gameEvents, MessageDisplayRequestGameEvent{Messages: []string{gameEndResult.Message}, Callback: nil})
		gameEvents = append(gameEvents, GameOverGameEvent{Winner: gameEndResult.Winner})
		return gameEvents, nil
	}

	return gameEvents, nil
}

func (s *PlayingState) Draw(screen *ebiten.Image) {
	// Playing状態固有の描画があればここに記述
}

// --- PlayerActionSelectState ---

type PlayerActionSelectState struct{}

func (s *PlayerActionSelectState) Update(ctx *BattleContext) ([]GameEvent, error) {
	world := ctx.World
	viewModelFactory := ctx.ViewModelFactory

	battleLogic := ctx.BattleLogic

	var gameEvents []GameEvent

	playerActionQueue := GetPlayerActionQueueComponent(world)

	// 待機中のプレイヤーがいるかチェック
	if len(playerActionQueue.Queue) > 0 {
		actingEntry := playerActionQueue.Queue[0]

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
					targetEntity, targetPartSlot = personality.TargetingStrategy.SelectTarget(world, actingEntry, battleLogic)
				}
				var targetID donburi.Entity
				if targetEntity != nil {
					targetID = targetEntity.Entity()
				}
				actionTargetMap[slotKey] = ActionTarget{TargetEntityID: targetID, Slot: targetPartSlot}
			}

			// ここでViewModelを構築し、UIに渡す
			actionModalVM := viewModelFactory.BuildActionModalViewModel(actingEntry, actionTargetMap, battleLogic.GetPartInfoProvider(), ctx.GameDataManager)
			// モーダルが既に表示されていない場合のみイベントを発行
			if !ctx.ViewModelFactory.IsActionModalVisible() {
				gameEvents = append(gameEvents, ShowActionModalGameEvent{ViewModel: actionModalVM})
			}
		} else {
			// 無効または待機状態でないならキューから削除
			playerActionQueue.Queue = playerActionQueue.Queue[1:]
			// 次のフレームで再度Updateが呼ばれるのを待つ
		}
	} else {
		// キューが空なら処理完了
	}

	return gameEvents, nil
}

func (s *PlayerActionSelectState) Draw(screen *ebiten.Image) {}

// --- AnimatingActionState ---

type AnimatingActionState struct{}

func (s *AnimatingActionState) Update(ctx *BattleContext) ([]GameEvent, error) {
	var gameEvents []GameEvent
	return gameEvents, nil
}

func (s *AnimatingActionState) Draw(screen *ebiten.Image) {}

// --- MessageState ---

type MessageState struct{}

func (s *MessageState) Update(ctx *BattleContext) ([]GameEvent, error) {
	var gameEvents []GameEvent
	return gameEvents, nil
}

func (s *MessageState) Draw(screen *ebiten.Image) {}

// --- GameOverState ---

type GameOverState struct{}

func (s *GameOverState) Update(ctx *BattleContext) ([]GameEvent, error) {
	var gameEvents []GameEvent
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		gameEvents = append(gameEvents, GoToTitleSceneGameEvent{})
	}
	return gameEvents, nil
}

func (s *GameOverState) Draw(screen *ebiten.Image) {}
