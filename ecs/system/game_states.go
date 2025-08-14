package system

import (
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
// BattleLogicを廃止し、個別のシステムへの参照を直接保持することで、依存関係を明確にしています。
type BattleContext struct {
	World                  donburi.World
	Config                 *data.Config
	GameDataManager        *data.GameDataManager
	// Randの型を *core.Rand から正しい *rand.Rand に修正しました。
	Rand                   *rand.Rand
	Tick                   int
	BattleUIManager        UIUpdater
	ViewModelFactory       ViewModelBuilder
	StatusEffectSystem     *StatusEffectSystem
	PostActionEffectSystem *PostActionEffectSystem

	// BattleLogicの代わりに個別のシステムを保持
	PartInfoProvider       PartInfoProviderInterface
	ChargeInitiationSystem *ChargeInitiationSystem
	TargetSelector         *TargetSelector
	DamageCalculator       *DamageCalculator
	HitCalculator          *HitCalculator
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
		// 状態遷移イベントを発行
		gameEvents = append(gameEvents, event.StateChangeRequestedGameEvent{NextState: core.StatePlayerActionSelect})
		return gameEvents, nil
	}

	// AIの行動選択を試みる
	// BattleLogicの代わりに、必要なシステムを直接渡す
	UpdateAIInputSystem(
		ctx.World,
		ctx.PartInfoProvider,
		ctx.ChargeInitiationSystem,
		ctx.TargetSelector,
		ctx.Rand,
	)

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

type PlayerActionSelectState struct {
	processedEntry *donburi.Entry
}

func (s *PlayerActionSelectState) Update(ctx *BattleContext) ([]event.GameEvent, error) {
	var gameEvents []event.GameEvent

	playerActionQueue := entity.GetPlayerActionQueueComponent(ctx.World)

	// 行動選択待ちのプレイヤーがいるかチェック
	if len(playerActionQueue.Queue) > 0 {
		actingEntry := playerActionQueue.Queue[0]

		// まだモーダルを表示していないエントリの場合のみイベントを発行
		if s.processedEntry != actingEntry {
			if actingEntry.Valid() && component.StateComponent.Get(actingEntry).CurrentState == core.StateIdle {
				actionTargetMap := make(map[core.PartSlotKey]core.ActionTarget)
				availableParts := ctx.PartInfoProvider.GetAvailableAttackParts(actingEntry)

				for _, available := range availableParts {
					partDef := available.PartDef
					slotKey := available.Slot
					var targetEntity *donburi.Entry
					var targetPartSlot core.PartSlotKey

					// 射撃・介入パーツの場合、デフォルトのターゲットをAI戦略に基づいて提案
					if partDef.Category == core.CategoryRanged || partDef.Category == core.CategoryIntervention {
						medal := component.MedalComponent.Get(actingEntry)
						personality, ok := PersonalityRegistry[medal.Personality]
						if !ok {
							personality = PersonalityRegistry["リーダー"] // フォールバック
						}
						targetEntity, targetPartSlot = personality.TargetingStrategy.SelectTarget(ctx.World, actingEntry, ctx.TargetSelector, ctx.PartInfoProvider, ctx.Rand)
					}

					var targetID donburi.Entity
					if targetEntity != nil {
						targetID = targetEntity.Entity()
					}
					actionTargetMap[slotKey] = core.ActionTarget{TargetEntityID: targetID, Slot: targetPartSlot}
				}

				// モーダル表示イベントを発行
				gameEvents = append(gameEvents, event.ShowActionModalGameEvent{
					ActingEntry:     actingEntry,
					ActionTargetMap: actionTargetMap,
				})
				s.processedEntry = actingEntry // 処理済みとしてマーク
			} else {
				// 待機状態でないなど、何らかの理由で行動できない場合はキューから削除
				playerActionQueue.Queue = playerActionQueue.Queue[1:]
			}
		}
		// 既にモーダル表示済みの場合は、UIからのイベントを待つ
	} else {
		// キューが空なら行動選択フェーズ完了
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

	// アクションキューからアクションを実行
	// BattleLogicの代わりに、必要なシステムを直接渡す
	actionResults, err := UpdateActionQueueSystem(
		ctx.World,
		ctx.DamageCalculator,
		ctx.HitCalculator,
		ctx.TargetSelector,
		ctx.PartInfoProvider,
		ctx.Config,
		ctx.StatusEffectSystem,
		ctx.PostActionEffectSystem,
		ctx.Rand,
	)
	if err != nil {
		log.Printf("アクションキューシステムの処理中にエラーが発生しました: %v", err)
	}

	// 実行結果があればアニメーション状態に遷移
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
	// アニメーションの完了はUIからの ActionAnimationFinishedGameEvent によって通知されるため、
	// この状態では何もしない。
	return nil, nil
}

func (s *AnimatingActionState) Draw(screen *ebiten.Image) {}

// --- PostActionState ---

type PostActionState struct{}

func (s *PostActionState) Update(ctx *BattleContext) ([]event.GameEvent, error) {
	var gameEvents []event.GameEvent

	// 直前のアクション結果を取得
	lastActionResultEntry, ok := query.NewQuery(filter.Contains(component.LastActionResultComponent)).First(ctx.World)
	if !ok {
		log.Panicln("LastActionResultComponent がワールドに見つかりません。")
	}
	result := component.LastActionResultComponent.Get(lastActionResultEntry)

	// クールダウン開始
	actingEntry := result.ActingEntry
	if actingEntry != nil && actingEntry.Valid() && component.StateComponent.Get(actingEntry).CurrentState != core.StateBroken {
		StartCooldownSystem(actingEntry, ctx.World, ctx.PartInfoProvider)
	}

	// UIマネージャーにメッセージ表示を依頼
	ctx.BattleUIManager.DisplayMessagesForResult(result, func() {
		// メッセージ表示後のコールバックでAIの行動履歴を更新
		UpdateHistorySystem(ctx.World, result)
	})

	// 処理が終わったらLastActionResultをクリア
	*result = component.ActionResult{}

	// メッセージ表示状態へ遷移
	gameEvents = append(gameEvents, event.StateChangeRequestedGameEvent{NextState: core.StateMessage})
	return gameEvents, nil
}

func (s *PostActionState) Draw(screen *ebiten.Image) {}

// --- MessageState ---

type MessageState struct{}

func (s *MessageState) Update(ctx *BattleContext) ([]event.GameEvent, error) {
	var gameEvents []event.GameEvent

	// UI側でメッセージ表示が完了したかどうかをチェック
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
	// クリックでタイトル画面へ
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		gameEvents = append(gameEvents, event.GoToTitleSceneGameEvent{})
	}
	return gameEvents, nil
}

func (s *GameOverState) Draw(screen *ebiten.Image) {}