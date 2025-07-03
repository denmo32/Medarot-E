package main

import "github.com/yohamta/donburi"

// ActionHandler はカテゴリ固有のアクション処理のためのインターフェースを定義します。
type ActionHandler interface {
	// ResolveTarget はカテゴリに基づいてアクションのターゲットを決定します。
	// actingEntry: アクションを実行するエンティティ。
	// world: ゲームワールド。
	// actionComp: 実行エンティティのActionComponent。プレイヤー操作の射撃など、事前に選択されたターゲット情報を含む場合があります。
	// targetSelector: ターゲットを見つけるためのヘルパー。
	// partInfoProvider: 一部のターゲットロジックで必要になる場合があります。
	// result: ターゲット情報を格納するためのActionResultへのポインタ。
	// 成功した場合は (ターゲットエンティティ, ターゲットパーツスロット, true) を返します。
	// 失敗した場合は (nil, "", false) を返し、result.LogMessageに理由が設定されます。
	ResolveTarget(
		actingEntry *donburi.Entry,
		world donburi.World,
		intent *ActionIntent,
		targetSelector *TargetSelector,
		partInfoProvider *PartInfoProvider,
		result *ActionResult,
	) (*donburi.Entry, PartSlotKey, bool)

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
	intent *ActionIntent,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
	result *ActionResult,
) (*donburi.Entry, PartSlotKey, bool) {
	settings := SettingsComponent.Get(actingEntry)
	// 射撃の場合、ターゲットは事前にPlayerInputSystemまたはAIInputSystemによってTargetComponentに設定されているはずです。
	target := TargetComponent.Get(actingEntry)
	if target.TargetEntity == nil || target.TargetPartSlot == "" {
		result.LogMessage = settings.Name + "は射撃ターゲットが未選択です。"
		return nil, "", false
	}
	targetEntry := target.TargetEntity
	targetPartSlot := target.TargetPartSlot
	result.TargetEntry = targetEntry // 失敗時もターゲット情報をログに残すために設定

	if StateComponent.Get(targetEntry).Current == StateTypeBroken {
		result.LogMessage = settings.Name + "はターゲット(" + SettingsComponent.Get(targetEntry).Name + ")を狙ったが、既に行動不能だった！"
		return nil, "", false
	}
	targetParts := PartsComponent.Get(targetEntry)
	if targetParts.Map[targetPartSlot] == nil || targetParts.Map[targetPartSlot].IsBroken {
		result.LogMessage = settings.Name + "は" + SettingsComponent.Get(targetEntry).Name + "の" + string(targetPartSlot) + "を狙ったが、パーツは既に破壊されていた！"
		return targetEntry, targetPartSlot, false
	}

	result.TargetPartSlot = targetPartSlot
	result.LogMessage = settings.Name + "は" + SettingsComponent.Get(targetEntry).Name + "の" + string(targetPartSlot) + "を狙う！"
	return targetEntry, targetPartSlot, true
}

// MeleeActionHandler は格闘カテゴリのパーツのアクションを処理します。
type MeleeActionHandler struct{}

func (h *MeleeActionHandler) ResolveTarget(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
	result *ActionResult,
) (*donburi.Entry, PartSlotKey, bool) {
	settings := SettingsComponent.Get(actingEntry)
	closestEnemy := targetSelector.FindClosestEnemy(actingEntry)
	if closestEnemy == nil {
		result.LogMessage = settings.Name + "は格闘攻撃しようとしたが、相手がいなかった。"
		return nil, "", false
	}
	result.TargetEntry = closestEnemy // 失敗時もターゲット情報をログに残すために設定

	if StateComponent.Get(closestEnemy).Current == StateTypeBroken { // FindClosestEnemyでフィルタリングされるべきだが、念のため確認
		result.LogMessage = settings.Name + "はターゲット(" + SettingsComponent.Get(closestEnemy).Name + ")を狙ったが、既に行動不能だった！"
		return closestEnemy, "", false
	}

	targetPart := targetSelector.SelectRandomPartToDamage(closestEnemy)
	if targetPart == nil {
		result.LogMessage = settings.Name + "は" + SettingsComponent.Get(closestEnemy).Name + "を狙ったが、攻撃できる部位がなかった！"
		return closestEnemy, "", false
	}
	// Part構造体からPartSlotKeyを取得するにはFindPartSlotが必要です
	targetPartSlot := partInfoProvider.FindPartSlot(closestEnemy, targetPart)
	if targetPartSlot == "" {
		// SelectRandomPartToDamageがエンティティから有効なパーツを返せば、これは理想的には起こりません
		result.LogMessage = settings.Name + "の" + SettingsComponent.Get(closestEnemy).Name + "への攻撃でパーツスロット特定失敗。"
		return closestEnemy, "", false
	}

	result.TargetPartSlot = targetPartSlot
	result.LogMessage = settings.Name + "は" + SettingsComponent.Get(closestEnemy).Name + "の" + string(targetPartSlot) + "に格闘攻撃！"
	return closestEnemy, targetPartSlot, true
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
