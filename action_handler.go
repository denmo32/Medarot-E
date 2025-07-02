package main

import (
	"github.com/yohamta/donburi"
)

// ActionResult (再掲または共通化が必要か検討)
// action_queue_system.go から移動または共通パッケージにすることも検討。
// ここでは一旦、ActionHandler が返す結果として簡易的に定義。
// 実際には action_queue_system.go の ActionResult を利用/拡張する形が良い。
type HandlerTargetingResult struct {
	TargetEntity   *donburi.Entry
	TargetPartSlot PartSlotKey
	Success        bool
	LogMessage     string
}

// ActionHandler はカテゴリ固有のアクション処理のためのインターフェースを定義します。
type ActionHandler interface {
	// ResolveTarget はカテゴリに基づいてアクションのターゲットを決定します。
	// actingEntry: アクションを実行するエンティティ。
	// world: ゲームワールド。
	// actionComp: 実行エンティティのActionComponent。プレイヤー操作の射撃など、事前に選択されたターゲット情報を含む場合があります。
	// targetSelector: ターゲットを見つけるためのヘルパー。
	// PartInfoProvider: 一部のターゲットロジックで必要になる場合があります。
	// 選択されたターゲットエンティティとターゲットパーツスロット、またはエラー/ログを返します。
	ResolveTarget(
		actingEntry *donburi.Entry,
		world donburi.World,
		actionComp *Action, // ActionComponent.Get(actingEntry)
		targetSelector *TargetSelector,
		partInfoProvider *PartInfoProvider, // 一部のターゲットロジックで必要になる場合があります
	) HandlerTargetingResult

	// CanExecute (オプション): 行動実行が可能かどうかの事前チェック。
	// (例: 射撃なら弾数があるか、など。今回はスコープ外とする可能性あり)
	// CanExecute(actingEntry *donburi.Entry, world donburi.World, actingPart *Part) bool

	// Execute (オプション): カテゴリ固有の実行ロジックの主要部分。
	// 多くのロジックは executeActionLogic に集約されているため、
	// このメソッドは限定的な役割になるか、あるいは不要かもしれません。
	// executeActionLogic がこのハンドラから情報を得る形にします。
	// 現状では ResolveTarget に集中します。
}

// --- 具体的なハンドラ ---

// ShootActionHandler は射撃カテゴリのパーツのアクションを処理します。
type ShootActionHandler struct{}

func (h *ShootActionHandler) ResolveTarget(
	actingEntry *donburi.Entry,
	world donburi.World,
	actionComp *Action,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) HandlerTargetingResult {
	settings := SettingsComponent.Get(actingEntry)
	// 射撃の場合、ターゲットエンティティとパーツスロットは通常、事前に選択されているべきです
	// (プレイヤーまたはAIの初期ターゲティング戦略によって)。
	if actionComp.TargetEntity == nil || actionComp.TargetPartSlot == "" {
		return HandlerTargetingResult{Success: false, LogMessage: settings.Name + "は射撃ターゲットが未選択です。"}
	}
	targetEntry := actionComp.TargetEntity
	targetPartSlot := actionComp.TargetPartSlot

	if targetEntry.HasComponent(BrokenStateComponent) {
		return HandlerTargetingResult{TargetEntity: targetEntry, TargetPartSlot: targetPartSlot, Success: false, LogMessage: settings.Name + "はターゲット(" + SettingsComponent.Get(targetEntry).Name + ")を狙ったが、既に行動不能だった！"}
	}
	targetParts := PartsComponent.Get(targetEntry)
	if targetParts.Map[targetPartSlot] == nil || targetParts.Map[targetPartSlot].IsBroken {
		return HandlerTargetingResult{TargetEntity: targetEntry, TargetPartSlot: targetPartSlot, Success: false, LogMessage: settings.Name + "は" + SettingsComponent.Get(targetEntry).Name + "の" + string(targetPartSlot) + "を狙ったが、パーツは既に破壊されていた！"}
	}

	return HandlerTargetingResult{
		TargetEntity:   targetEntry,
		TargetPartSlot: targetPartSlot,
		Success:        true,
		LogMessage:     settings.Name + "は" + SettingsComponent.Get(targetEntry).Name + "の" + string(targetPartSlot) + "を狙う！",
	}
}

// MeleeActionHandler は格闘カテゴリのパーツのアクションを処理します。
type MeleeActionHandler struct{}

func (h *MeleeActionHandler) ResolveTarget(
	actingEntry *donburi.Entry,
	world donburi.World,
	actionComp *Action, // 格闘はActionComponentで事前に選択されたターゲットを無視する場合があります
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) HandlerTargetingResult {
	settings := SettingsComponent.Get(actingEntry)
	closestEnemy := targetSelector.FindClosestEnemy(actingEntry)
	if closestEnemy == nil {
		return HandlerTargetingResult{Success: false, LogMessage: settings.Name + "は格闘攻撃しようとしたが、相手がいなかった。"}
	}
	if closestEnemy.HasComponent(BrokenStateComponent) { // FindClosestEnemyでフィルタリングされるべきだが、念のため確認
		return HandlerTargetingResult{TargetEntity: closestEnemy, Success: false, LogMessage: settings.Name + "はターゲット(" + SettingsComponent.Get(closestEnemy).Name + ")を狙ったが、既に行動不能だった！"}
	}

	targetPart := targetSelector.SelectRandomPartToDamage(closestEnemy)
	if targetPart == nil {
		return HandlerTargetingResult{TargetEntity: closestEnemy, Success: false, LogMessage: settings.Name + "は" + SettingsComponent.Get(closestEnemy).Name + "を狙ったが、攻撃できる部位がなかった！"}
	}
	// Part構造体からPartSlotKeyを取得するにはFindPartSlotが必要です
	targetPartSlot := partInfoProvider.FindPartSlot(closestEnemy, targetPart)
	if targetPartSlot == "" {
		// SelectRandomPartToDamageがエンティティから有効なパーツを返せば、これは理想的には起こりません
		return HandlerTargetingResult{TargetEntity: closestEnemy, Success: false, LogMessage: settings.Name + "の" + SettingsComponent.Get(closestEnemy).Name + "への攻撃でパーツスロット特定失敗。"}
	}

	return HandlerTargetingResult{
		TargetEntity:   closestEnemy,
		TargetPartSlot: targetPartSlot,
		Success:        true,
		LogMessage:     settings.Name + "は" + SettingsComponent.Get(closestEnemy).Name + "の" + string(targetPartSlot) + "に格闘攻撃！",
	}
}

// ハンドラのグローバルインスタンス（またはファクトリ/レジストリを使用することもできます）
var (
	shootHandler = &ShootActionHandler{}
	meleeHandler = &MeleeActionHandler{}
)

// GetActionHandlerForCategory はパーツカテゴリに基づいて適切なActionHandlerを返します。
func GetActionHandlerForCategory(category PartCategory) ActionHandler {
	switch category {
	case CategoryShoot:
		return shootHandler
	case CategoryMelee:
		return meleeHandler
	default:
		// 未処理の場合はデフォルトハンドラまたはnilを返します
		// 現状、nilを返すとexecuteActionLogicでフォールバックまたはエラー処理が必要になります
		return nil
	}
}
