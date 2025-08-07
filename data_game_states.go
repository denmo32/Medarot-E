package main

import (
	"fmt"
	"log"
	"math/rand"

	"medarot-ebiten/domain"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
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
	MessageManager         *UIMessageDisplayManager // 追加
	UI                     UIInterface              // 追加
}

// BattleState は戦闘シーンの各状態が満たすべきインターフェースです。
type BattleState interface {
	Update(ctx *BattleContext) ([]GameEvent, error)
	Draw(screen *ebiten.Image)
}

// --- GaugeProgressState ---

type GaugeProgressState struct{}

func (s *GaugeProgressState) Update(ctx *BattleContext) ([]GameEvent, error) {
	var gameEvents []GameEvent

	// ゲージ進行
	UpdateGaugeSystem(ctx.World)

	// プレイヤーの行動選択が必要かチェック
	playerInputEvents := UpdatePlayerInputSystem(ctx.World)
	if len(playerInputEvents) > 0 {
		gameEvents = append(gameEvents, playerInputEvents...)
		gameEvents = append(gameEvents, StateChangeRequestedGameEvent{NextState: domain.StatePlayerActionSelect})
		return gameEvents, nil
	}

	// AIの行動選択を試みる
	UpdateAIInputSystem(ctx.World, ctx.BattleLogic)

	// アクション実行キューをチェック
	actionQueueComp := GetActionQueueComponent(ctx.World)
	if len(actionQueueComp.Queue) > 0 {
		gameEvents = append(gameEvents, StateChangeRequestedGameEvent{NextState: domain.StateActionExecution})
		return gameEvents, nil
	}

	// ステータス効果の更新
	ctx.statusEffectSystem.Update()

	// ゲーム終了判定
	gameEndResult := CheckGameEndSystem(ctx.World)
	if gameEndResult.IsGameOver {
		gameEvents = append(gameEvents, MessageDisplayRequestGameEvent{Messages: []string{gameEndResult.Message}, Callback: nil})
		gameEvents = append(gameEvents, GameOverGameEvent{Winner: gameEndResult.Winner})
	}

	return gameEvents, nil
}

func (s *GaugeProgressState) Draw(screen *ebiten.Image) {
	// GaugeProgress状態固有の描画があればここに記述
}

// --- PlayerActionSelectState ---

type PlayerActionSelectState struct{}

func (s *PlayerActionSelectState) Update(ctx *BattleContext) ([]GameEvent, error) {
	viewModelFactory := ctx.ViewModelFactory

	battleLogic := ctx.BattleLogic

	var gameEvents []GameEvent

	playerActionQueue := GetPlayerActionQueueComponent(ctx.World)

	// 待機中のプレイヤーがいるかチェック
	if len(playerActionQueue.Queue) > 0 {
		actingEntry := playerActionQueue.Queue[0]

		// 有効で待機状態ならモーダルを表示
		if actingEntry.Valid() && StateComponent.Get(actingEntry).CurrentState == domain.StateIdle {
			actionTargetMap := make(map[domain.PartSlotKey]domain.ActionTarget)
			// ViewModelFactoryを介して利用可能なパーツを取得
			availableParts := viewModelFactory.GetAvailableAttackParts(actingEntry)
			for _, available := range availableParts {
				partDef := available.PartDef
				slotKey := available.Slot
				var targetEntity *donburi.Entry
				var targetPartSlot domain.PartSlotKey
				if partDef.Category == domain.CategoryRanged || partDef.Category == domain.CategoryIntervention {
					medal := MedalComponent.Get(actingEntry)
					personality, ok := PersonalityRegistry[medal.Personality]
					if !ok {
						personality = PersonalityRegistry["リーダー"]
					}
					targetEntity, targetPartSlot = personality.TargetingStrategy.SelectTarget(ctx.World, actingEntry, battleLogic)
				}
				var targetID donburi.Entity
				if targetEntity != nil {
					targetID = targetEntity.Entity()
				}
				actionTargetMap[slotKey] = domain.ActionTarget{TargetEntityID: targetID, Slot: targetPartSlot}
			}

			// ここでViewModelを構築し、UIに渡す
			actionModalVM := viewModelFactory.BuildActionModalViewModel(actingEntry, actionTargetMap, battleLogic.GetPartInfoProvider(), ctx.GameDataManager)
			// モーダルが既に表示されていない場合のみイベントを発行
			if !ctx.UI.IsActionModalVisible() { // Use ctx.UI
				gameEvents = append(gameEvents, ShowActionModalGameEvent{ViewModel: actionModalVM})
			}
		} else {
			// 無効または待機状態でないならキューから削除
			playerActionQueue.Queue = playerActionQueue.Queue[1:]
			// 次のフレームで再度Updateが呼ばれるのを待つ
		}
	} else {
		// キューが空なら処理完了
		gameEvents = append(gameEvents, PlayerActionSelectFinishedGameEvent{})
		gameEvents = append(gameEvents, StateChangeRequestedGameEvent{NextState: domain.StateGaugeProgress})
	}

	return gameEvents, nil
}

func (s *PlayerActionSelectState) Draw(screen *ebiten.Image) {}

// --- ActionExecutionState ---

type ActionExecutionState struct{}

func (s *ActionExecutionState) Update(ctx *BattleContext) ([]GameEvent, error) {
	var gameEvents []GameEvent

	actionResults, err := UpdateActionQueueSystem(
		ctx.World,
		ctx.BattleLogic.GetDamageCalculator(),
		ctx.BattleLogic.GetHitCalculator(),
		ctx.BattleLogic.GetTargetSelector(),
		ctx.BattleLogic.GetPartInfoProvider(),
		ctx.Config,
		ctx.statusEffectSystem,
		ctx.postActionEffectSystem,
		ctx.Rand,
	)
	if err != nil {
		fmt.Println("アクションキューシステムの処理中にエラーが発生しました:", err)
	}

	for _, result := range actionResults {
		if result.ActingEntry != nil && result.ActingEntry.Valid() {
			gameEvents = append(gameEvents, ActionAnimationStartedGameEvent{AnimationData: ActionAnimationData{Result: result, StartTime: ctx.Tick}})
			gameEvents = append(gameEvents, StateChangeRequestedGameEvent{NextState: domain.StateAnimatingAction})
			return gameEvents, nil
		}
	}

	// キューが空になったらゲージ進行に戻る
	actionQueueComp := GetActionQueueComponent(ctx.World)
	if len(actionQueueComp.Queue) == 0 {
		gameEvents = append(gameEvents, StateChangeRequestedGameEvent{NextState: domain.StateGaugeProgress})
	}

	return gameEvents, nil
}

func (s *ActionExecutionState) Draw(screen *ebiten.Image) {}

// --- AnimatingActionState ---

type AnimatingActionState struct{}

func (s *AnimatingActionState) Update(ctx *BattleContext) ([]GameEvent, error) {
	var gameEvents []GameEvent
	// 何もしない。状態遷移は ActionAnimationFinishedGameEvent によってトリガーされる。
	return gameEvents, nil
}

func (s *AnimatingActionState) Draw(screen *ebiten.Image) {}

// --- PostActionState ---

type PostActionState struct{}

func (s *PostActionState) Update(ctx *BattleContext) ([]GameEvent, error) {
	var gameEvents []GameEvent

	lastActionResultEntry, ok := query.NewQuery(filter.Contains(LastActionResultComponent)).First(ctx.World)
	if !ok {
		log.Panicln("LastActionResultComponent がワールドに見つかりません。")
	}
	result := LastActionResultComponent.Get(lastActionResultEntry)

	// クールダウン開始
	actingEntry := result.ActingEntry
	if actingEntry != nil && actingEntry.Valid() && StateComponent.Get(actingEntry).CurrentState != domain.StateBroken {
		StartCooldownSystem(actingEntry, ctx.World, ctx.BattleLogic.GetPartInfoProvider())
	}

	// メッセージ生成とエンキュー
	ctx.MessageManager.EnqueueMessageQueue(buildActionLogMessagesFromActionResult(*result, ctx.GameDataManager), func() {
		UpdateHistorySystem(ctx.World, result)
	})

	// 処理が終わったらLastActionResultをクリア
	*result = ActionResult{}

	gameEvents = append(gameEvents, StateChangeRequestedGameEvent{NextState: domain.StateMessage})
	return gameEvents, nil
}

func (s *PostActionState) Draw(screen *ebiten.Image) {}

// --- MessageState ---

type MessageState struct{}

func (s *MessageState) Update(ctx *BattleContext) ([]GameEvent, error) {
	var gameEvents []GameEvent

	ctx.MessageManager.Update() // Use ctx.MessageManager
	if ctx.MessageManager.IsFinished() {
		gameEvents = append(gameEvents, MessageDisplayFinishedGameEvent{})
	}

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
