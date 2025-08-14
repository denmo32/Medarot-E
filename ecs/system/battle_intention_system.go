package system

import (
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ecs/entity"
	"medarot-ebiten/event"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdatePlayerInputSystem はアイドル状態のすべてのプレイヤー制御メダロットを見つけます。
// このシステムは、行動が必要なプレイヤーエンティティのリストを含むイベントを発行します。
func UpdatePlayerInputSystem(world donburi.World) []event.GameEvent {
	playerActionQueue := entity.GetPlayerActionQueueComponent(world)
	var gameEvents []event.GameEvent

	// キューをクリアし、現在のアイドル状態のプレイヤーエンティティを再収集
	playerActionQueue.Queue = make([]*donburi.Entry, 0)
	query.NewQuery(filter.Contains(component.PlayerControlComponent)).Each(world, func(entry *donburi.Entry) {
		if component.StateComponent.Get(entry).CurrentState == core.StateIdle {
			playerActionQueue.Queue = append(playerActionQueue.Queue, entry)
		}
	})

	if len(playerActionQueue.Queue) > 0 {
		// 行動選択が必要なプレイヤーがいることを示すイベントを発行
		gameEvents = append(gameEvents, event.PlayerActionRequiredGameEvent{})
	}

	return gameEvents
}

// UpdateAIInputSystem はAI制御のメダロットの行動選択を処理します。
// BattleLogicへの依存をなくし、必要なシステムを直接引数に取ります。
func UpdateAIInputSystem(
	world donburi.World,
	partInfoProvider PartInfoProviderInterface,
	chargeSystem *ChargeInitiationSystem,
	targetSelector *TargetSelector,
	// randの型を *core.Rand から正しい *rand.Rand に修正しました。
	rand *rand.Rand,
) {
	// 存在しない filter.In を削除し、Eachループ内でif文によるチェックを行うように修正しました。
	// これがdonburiの標準的な値によるフィルタリング方法です。
	query.NewQuery(
		filter.Not(filter.Contains(component.PlayerControlComponent)), // プレイヤー制御ではないエンティティ
	).Each(world, func(entry *donburi.Entry) {
		// アイドル状態のエンティティのみを処理
		if !entry.HasComponent(component.StateComponent) || component.StateComponent.Get(entry).CurrentState != core.StateIdle {
			return
		}
		aiSelectAction(world, entry, partInfoProvider, chargeSystem, targetSelector, rand)
	})
}

// ProcessPlayerIntent はプレイヤーの行動意図を解釈し、具体的なアクションを開始します。
// BattleLogicへの依存をなくし、ChargeInitiationSystemを直接受け取ります。
func ProcessPlayerIntent(
	world donburi.World,
	chargeSystem *ChargeInitiationSystem,
	intentEvent event.PlayerActionIntentEvent,
) {
	actingEntry := world.Entry(intentEvent.ActingEntityID)
	if actingEntry == nil || !actingEntry.Valid() {
		return
	}

	var targetEntry *donburi.Entry
	if intentEvent.TargetEntityID != 0 {
		targetEntry = world.Entry(intentEvent.TargetEntityID)
		if targetEntry == nil || !targetEntry.Valid() {
			return // ターゲットが無効なら中断
		}
	}

	// チャージ開始システムを呼び出す
	chargeSystem.StartCharge(
		actingEntry,
		intentEvent.SelectedSlotKey,
		targetEntry,
		intentEvent.TargetPartSlot,
	)
}