package battle

import (
	"fmt"
	"medarot-ebiten/internal/game"
	"medarot-ebiten/internal/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
)

// BattleContext は戦闘シーンの各状態が共通して必要とする依存関係をまとめた構造体です。
type BattleContext struct {
	World                  donburi.World
	BattleLogic            *BattleLogic
	UI                     ui.UIInterface
	Config                 *game.Config
	GameDataManager        *game.GameDataManager // 追加
	Tick                   int
	ViewModelFactory       ui.ViewModelFactory // 追加
	BattleAnimationManager *BattleAnimationManager
}

// BattleState は戦闘シーンの各状態が満たすべきインターフェースです。
type BattleState interface {
	Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []game.GameEvent, error)
	Draw(screen *ebiten.Image)
}

// --- PlayingState ---

type PlayingState struct{}

func (s *PlayingState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []game.GameEvent, error) {
	world := ctx.World
	battleLogic := ctx.BattleLogic
	ui := ctx.UI
	config := ctx.Config
	tick := ctx.Tick

	var gameEvents []game.GameEvent

	// AIの行動選択
	if !ui.IsActionModalVisible() && len(playerActionPendingQueue) == 0 {
		UpdateAIInputSystem(world, battleLogic)
	}

	// プレイヤーの行動選択が必要かチェック
	playerInputResult := UpdatePlayerInputSystem(world)
	if len(playerInputResult.PlayerMedarotsToAct) > 0 {
		playerActionPendingQueue = playerInputResult.PlayerMedarotsToAct
		gameEvents = append(gameEvents, game.PlayerActionRequiredGameEvent{})
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
			gameEvents = append(gameEvents, game.ActionAnimationStartedGameEvent{AnimationData: game.ActionAnimationData{Result: result, StartTime: tick}})
			return playerActionPendingQueue, gameEvents, nil
		}
	}

	// ゲーム終了判定
	gameEndResult := CheckGameEndSystem(world)
	if gameEndResult.IsGameOver {
		gameEvents = append(gameEvents, game.MessageDisplayRequestGameEvent{Messages: []string{gameEndResult.Message}, Callback: nil})
		gameEvents = append(gameEvents, game.GameOverGameEvent{Winner: gameEndResult.Winner})
		return playerActionPendingQueue, gameEvents, nil
	}

	return playerActionPendingQueue, gameEvents, nil // 状態は維持
}

func (s *PlayingState) Draw(screen *ebiten.Image) {
	// Playing状態固有の描画があればここに記述
}

// --- PlayerActionSelectState ---

type PlayerActionSelectState struct{}

func (s *PlayerActionSelectState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []game.GameEvent, error) {
	world := ctx.World
	battleLogic := ctx.BattleLogic
	ui := ctx.UI

	var gameEvents []game.GameEvent

	// モーダル表示中は何もしない
	if ui.IsActionModalVisible() {
		return playerActionPendingQueue, gameEvents, nil
	}

	// 待機中のプレイヤーがいるかチェック
	if len(playerActionPendingQueue) > 0 {
		actingEntry := playerActionPendingQueue[0]

		// 有効で待機状態ならモーダルを表示
		if actingEntry.Valid() && game.StateComponent.Get(actingEntry).FSM.Is(string(game.StateIdle)) {
			actionTargetMap := make(map[game.PartSlotKey]ActionTarget)
			// ViewModelFactoryを介して利用可能なパーツを取得
			availableParts := ctx.ViewModelFactory.GetAvailableAttackParts(actingEntry)
			for _, available := range availableParts {
				partDef := available.PartDef
				slotKey := available.Slot
				var targetEntity *donburi.Entry
				var targetPartSlot game.PartSlotKey
				if partDef.Category == game.CategoryRanged || partDef.Category == game.CategoryIntervention {
					medal := game.MedalComponent.Get(actingEntry)
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
			actionModalVM := ctx.ViewModelFactory.BuildActionModalViewModel(actingEntry, actionTargetMap, battleLogic)
			gameEvents = append(gameEvents, game.ShowActionModalGameEvent{ViewModel: actionModalVM})
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

func (s *AnimatingActionState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []game.GameEvent, error) {
	world := ctx.World
	ui := ctx.UI
	tick := ctx.Tick

	var gameEvents []game.GameEvent

	if ui.IsAnimationFinished(tick) {
		result := ctx.BattleAnimationManager.GetCurrentActionResult()
		if result != nil {
			gameEvents = append(gameEvents, game.ClearAnimationGameEvent{})
			gameEvents = append(gameEvents, game.MessageDisplayRequestGameEvent{Messages: buildActionLogMessages(*result, ctx.GameDataManager), Callback: func() {
				UpdateHistorySystem(world, result)
			}})
			gameEvents = append(gameEvents, game.ActionAnimationFinishedGameEvent{Result: *result, ActingEntry: result.ActingEntry})
		}
		return playerActionPendingQueue, gameEvents, nil
	}
	return playerActionPendingQueue, gameEvents, nil
}

func (s *AnimatingActionState) Draw(screen *ebiten.Image) {}

// --- MessageState ---

type MessageState struct{}

func (s *MessageState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []game.GameEvent, error) {
	// MessageStateはMessageDisplayFinishedGameEventを返すのみで、MessageManagerのUpdateはBattleSceneで行う
	var gameEvents []game.GameEvent

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

func (s *GameOverState) Update(ctx *BattleContext, playerActionPendingQueue []*donburi.Entry) ([]*donburi.Entry, []game.GameEvent, error) {
	var gameEvents []game.GameEvent

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// ここでシーン遷移を直接行うのではなく、イベントを発行するなどして
		// BattleSceneに通知するのがよりクリーンな設計ですが、
		// 今回は一旦、このファイルからsceneパッケージへの依存をなくすことを優先します。
		// このイベントを処理する側でシーンマネージャを呼び出す必要があります。
		gameEvents = append(gameEvents, game.GameOverTransitionGameEvent{})
	}
	return playerActionPendingQueue, gameEvents, nil
}

func (s *GameOverState) Draw(screen *ebiten.Image) {}

// buildActionLogMessages はアクションの結果ログを生成します。
func buildActionLogMessages(result ActionResult, gameDataManager *game.GameDataManager) []string {
	messages := []string{}

	// メッセージテンプレートのパラメータを準備
	// action_name には特性名(Trait)を渡す
	initiateParams := map[string]interface{}{
		"attacker_name": result.AttackerName,
		"action_name":   result.ActionTrait,
		"weapon_type":   result.WeaponType,
	}

	// カテゴリに応じてメッセージIDを切り替え
	messageID := ""
	switch result.ActionCategory {
	case game.CategoryRanged, game.CategoryMelee:
		messageID = "action_initiate_attack"
	case game.CategoryIntervention:
		messageID = "action_initiate_intervention"
	}

	if result.ActionDidHit {
		if messageID != "" {
			messages = append(messages, gameDataManager.Messages.FormatMessage(messageID, initiateParams))
		}

		// ダメージや防御のメッセージを追加
		switch result.ActionCategory {
		case game.CategoryRanged, game.CategoryMelee:
			if result.ActionIsDefended {
				defendParams := map[string]interface{}{"defender_name": result.DefenderName, "defending_part_type": result.DefendingPartType}
				messages = append(messages, gameDataManager.Messages.FormatMessage("action_defend", defendParams))
			}
			damageParams := map[string]interface{}{"defender_name": result.DefenderName, "target_part_type": result.TargetPartType, "damage": result.DamageDealt}
			messages = append(messages, gameDataManager.Messages.FormatMessage("action_damage", damageParams))
		case game.CategoryIntervention:
			// 介入アクションの成功メッセージ（例：「味方チーム全体の命中率が上昇した！」）
			// 必要であれば、ここで特性(Trait)に応じたメッセージを追加する
			if result.ActionTrait == game.TraitSupport { // string() を削除
				messages = append(messages, gameDataManager.Messages.FormatMessage("support_action_generic", nil))
			}
		}
	} else {
		// ミスした場合
		if messageID != "" {
			messages = append(messages, gameDataManager.Messages.FormatMessage(messageID, initiateParams))
		}
		missParams := map[string]interface{}{
			"target_name": result.DefenderName,
		}
		messages = append(messages, gameDataManager.Messages.FormatMessage("attack_miss", missParams))
	}
	return messages
}

// UpdateInfoPanelViewModelSystem は、すべてのメダロットエンティティからInfoPanelViewModelを構築し、BattleUIStateComponentに格納します。
func UpdateInfoPanelViewModelSystem(battleUIState *ui.BattleUIState, world donburi.World, battleLogic *BattleLogic, factory ui.ViewModelFactory) {

	query.NewQuery(filter.Contains(game.SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		settings := game.SettingsComponent.Get(entry)
		battleUIState.InfoPanels[settings.ID] = factory.BuildInfoPanelViewModel(entry, battleLogic)
	})
}
