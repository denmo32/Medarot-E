package system

import (
	"fmt"
	"log"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ecs/entity"
	"medarot-ebiten/event"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// BattleContext は戦闘シーンの各状態が共通して必要とする依存関係をまとめた構造体です。
type BattleContext struct {
	World                  donburi.World
	Config                 *data.Config
	GameDataManager        *data.GameDataManager
	Rand                   *rand.Rand
	Tick                   int
	BattleUIManager        UIUpdater
	ViewModelFactory       ViewModelBuilder
	StatusEffectSystem     *StatusEffectSystem
	PostActionEffectSystem *PostActionEffectSystem
	BattleLogic            *BattleLogic
}

// BattleState は戦闘シーンの各状態が満たすべきインターフェースです。
type BattleState interface {
	Update(ctx *BattleContext) ([]event.GameEvent, error)
	Draw(screen *ebiten.Image)
}

// --- GaugeProgressState ---

type GaugeProgressState struct{}

func (s *GaugeProgressState) Update(ctx *BattleContext) ([]event.GameEvent, error) {
	var gameEvents []event.GameEvent

	// ゲージ進行
	UpdateGaugeSystem(ctx.World)

	// プレイヤーの行動選択が必要かチェック
	playerInputEvents := UpdatePlayerInputSystem(ctx.World)
	if len(playerInputEvents) > 0 {
		gameEvents = append(gameEvents, playerInputEvents...)
		gameEvents = append(gameEvents, event.StateChangeRequestedGameEvent{NextState: core.StatePlayerActionSelect})
		return gameEvents, nil
	}

	// AIの行動選択を試みる
	UpdateAIInputSystem(ctx.World, ctx.BattleLogic)

	// アクション実行キューをチェック
	actionQueueComp := entity.GetActionQueueComponent(ctx.World)
	if len(actionQueueComp.Queue) > 0 {
		gameEvents = append(gameEvents, event.StateChangeRequestedGameEvent{NextState: core.StateActionExecution})
		return gameEvents, nil
	}

	// ステータス効果の更新
	ctx.StatusEffectSystem.Update()

	// ゲーム終了判定
	gameEndResult := CheckGameEndSystem(ctx.World)
	if gameEndResult.IsGameOver {
		gameEvents = append(gameEvents, event.MessageDisplayRequestGameEvent{Messages: []string{gameEndResult.Message}, Callback: nil})
		gameEvents = append(gameEvents, event.GameOverGameEvent{Winner: gameEndResult.Winner})
	}

	return gameEvents, nil
}

func (s *GaugeProgressState) Draw(screen *ebiten.Image) {
	// GaugeProgress状態固有の描画があればここに記述
}

// --- PlayerActionSelectState ---

type PlayerActionSelectState struct{}

func (s *PlayerActionSelectState) Update(ctx *BattleContext) ([]event.GameEvent, error) {
	// battleUIManager := ctx.BattleUIManager // UIMediator を使用

	battleLogic := ctx.BattleLogic

	var gameEvents []event.GameEvent

	playerActionQueue := entity.GetPlayerActionQueueComponent(ctx.World)

	// 待機中のプレイヤーがいるかチェック
	if len(playerActionQueue.Queue) > 0 {
		actingEntry := playerActionQueue.Queue[0]

		// 有効で待機状態ならモーダルを表示
		if actingEntry.Valid() && component.StateComponent.Get(actingEntry).CurrentState == core.StateIdle {
			actionTargetMap := make(map[core.PartSlotKey]core.ActionTarget) // core.ActionTarget を使用
			// ViewModelFactoryを介して利用可能なパーツを取得
			// UIMediator 経由で ViewModelFactory のメソッドを呼び出す
			availableParts := battleLogic.GetPartInfoProvider().GetAvailableAttackParts(actingEntry) // ViewModelFactory から直接取得しない
			for _, available := range availableParts {
				partDef := available.PartDef
				slotKey := available.Slot
				var targetEntity *donburi.Entry
				var targetPartSlot core.PartSlotKey
				if partDef.Category == core.CategoryRanged || partDef.Category == core.CategoryIntervention {
					medal := component.MedalComponent.Get(actingEntry)
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
				actionTargetMap[slotKey] = core.ActionTarget{TargetEntityID: targetID, Slot: targetPartSlot} // core.ActionTarget を使用
			}

			// ここでViewModelを構築し、UIに渡す
			actionModalVM, err := ctx.ViewModelFactory.BuildActionModalViewModel(actingEntry, actionTargetMap) // ViewModelFactory 経由で呼び出し
			if err != nil {
				return nil, fmt.Errorf("failed to build action modal view model: %w", err)
			}
			// モーダルが既に表示されていない場合のみイベントを発行
			if !ctx.BattleUIManager.IsActionModalVisible() {
				gameEvents = append(gameEvents, event.ShowActionModalGameEvent{ViewModel: &actionModalVM})
			}
		} else {
			// 無効または待機状態でないならキューから削除
			playerActionQueue.Queue = playerActionQueue.Queue[1:]
			// 次のフレームで再度Updateが呼ばれるのを待つ
		}
	} else {
		// キューが空なら処理完了
		gameEvents = append(gameEvents, event.PlayerActionSelectFinishedGameEvent{})
		gameEvents = append(gameEvents, event.StateChangeRequestedGameEvent{NextState: core.StateGaugeProgress})
	}

	return gameEvents, nil
}

func (s *PlayerActionSelectState) Draw(screen *ebiten.Image) {}

// --- ActionExecutionState ---

type ActionExecutionState struct{}

func (s *ActionExecutionState) Update(ctx *BattleContext) ([]event.GameEvent, error) {
	var gameEvents []event.GameEvent

	actionResults, err := UpdateActionQueueSystem(
		ctx.World,
		ctx.BattleLogic.GetDamageCalculator(),
		ctx.BattleLogic.GetHitCalculator(),
		ctx.BattleLogic.GetTargetSelector(),
		ctx.BattleLogic.GetPartInfoProvider(),
		ctx.Config,
		ctx.StatusEffectSystem,
		ctx.PostActionEffectSystem,
		ctx.Rand,
	)
	if err != nil {
		fmt.Println("アクションキューシステムの処理中にエラーが発生しました:", err)
	}

	for _, result := range actionResults {
		if result.ActingEntry != nil && result.ActingEntry.Valid() {
			gameEvents = append(gameEvents, event.ActionAnimationStartedGameEvent{AnimationData: component.ActionAnimationData{Result: result, StartTime: ctx.Tick}})
			gameEvents = append(gameEvents, event.StateChangeRequestedGameEvent{NextState: core.StateAnimatingAction})
			return gameEvents, nil
		}
	}

	// キューが空になったらゲージ進行に戻る
	actionQueueComp := entity.GetActionQueueComponent(ctx.World)
	if len(actionQueueComp.Queue) == 0 {
		gameEvents = append(gameEvents, event.StateChangeRequestedGameEvent{NextState: core.StateGaugeProgress})
	}

	return gameEvents, nil
}

func (s *ActionExecutionState) Draw(screen *ebiten.Image) {}

// --- AnimatingActionState ---

type AnimatingActionState struct{}

func (s *AnimatingActionState) Update(ctx *BattleContext) ([]event.GameEvent, error) {
	var gameEvents []event.GameEvent
	// 何もしない。状態遷移は ActionAnimationFinishedGameEvent によってトリガーされる。
	return gameEvents, nil
}

func (s *AnimatingActionState) Draw(screen *ebiten.Image) {}

// --- PostActionState ---

type PostActionState struct{}

func (s *PostActionState) Update(ctx *BattleContext) ([]event.GameEvent, error) {
	var gameEvents []event.GameEvent

	lastActionResultEntry, ok := query.NewQuery(filter.Contains(component.LastActionResultComponent)).First(ctx.World)
	if !ok {
		log.Panicln("LastActionResultComponent がワールドに見つかりません。")
	}
	result := component.LastActionResultComponent.Get(lastActionResultEntry)

	// クールダウン開始
	actingEntry := result.ActingEntry
	if actingEntry != nil && actingEntry.Valid() && component.StateComponent.Get(actingEntry).CurrentState != core.StateBroken {
		StartCooldownSystem(actingEntry, ctx.World, ctx.BattleLogic.GetPartInfoProvider())
	}

	// メッセージ生成とエンキュー
	ctx.BattleUIManager.EnqueueMessageQueue(data.BuildActionLogMessagesFromActionResult(*result, ctx.GameDataManager), func() {
		UpdateHistorySystem(ctx.World, result)
	})

	// 処理が終わったらLastActionResultをクリア
	*result = component.ActionResult{}

	gameEvents = append(gameEvents, event.StateChangeRequestedGameEvent{NextState: core.StateMessage})
	return gameEvents, nil
}

func (s *PostActionState) Draw(screen *ebiten.Image) {}

// --- MessageState ---

type MessageState struct{}

func (s *MessageState) Update(ctx *BattleContext) ([]event.GameEvent, error) {
	var gameEvents []event.GameEvent

	// UIMediator に Update メソッドがないため、ここでは直接呼び出さない。
	// UIのUpdateはBattleSceneのUpdateで一括して行われる想定。
	// メッセージ表示完了のチェックのみUIMediator経由で行う。
	if ctx.BattleUIManager.IsMessageFinished() {
		gameEvents = append(gameEvents, event.MessageDisplayFinishedGameEvent{})
	}

	return gameEvents, nil
}

func (s *MessageState) Draw(screen *ebiten.Image) {}

// --- GameOverState ---

type GameOverState struct{}

func (s *GameOverState) Update(ctx *BattleContext) ([]event.GameEvent, error) {
	var gameEvents []event.GameEvent
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		gameEvents = append(gameEvents, event.GoToTitleSceneGameEvent{})
	}
	return gameEvents, nil
}

func (s *GameOverState) Draw(screen *ebiten.Image) {}
