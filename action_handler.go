package main

import "github.com/yohamta/donburi"

// ActionHandler はカテゴリ固有のアクション処理全体をカプセル化します。
type ActionHandler interface {
	// Execute は、アクションの解決から結果生成までの一連の処理を実行します。
	// 成功した場合は、詳細な結果を含む ActionResult を返します。
	// 失敗した場合は、エラー情報を含む ActionResult を返します。
	Execute(
		actingEntry *donburi.Entry,
		world donburi.World,
		intent *ActionIntent,
		battleLogic *BattleLogic, // 必要な計算機へのアクセスを提供
		gameConfig *Config,
	) ActionResult
}

// --- 具体的なハンドラ ---

// ShootActionHandler は射撃カテゴリのパーツのアクションを処理します。
type ShootActionHandler struct{}

func (h *ShootActionHandler) Execute(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	battleLogic *BattleLogic,
	gameConfig *Config,
) ActionResult {
	result := ActionResult{ActingEntry: actingEntry}
	settings := SettingsComponent.Get(actingEntry)
	partsComp := PartsComponent.Get(actingEntry)
	actingPartInstance := partsComp.Map[intent.SelectedPartKey]
	actingPartDef, _ := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID)

	// 1. ターゲット解決
	targetComp := TargetComponent.Get(actingEntry)
	if targetComp.TargetEntity == nil || targetComp.TargetPartSlot == "" {
		result.LogMessage = settings.Name + "は射撃ターゲットが未選択です。"
		return result
	}
	result.TargetEntry = targetComp.TargetEntity
	result.TargetPartSlot = targetComp.TargetPartSlot

	if StateComponent.Get(result.TargetEntry).Current == StateTypeBroken {
		result.LogMessage = settings.Name + "はターゲット(" + SettingsComponent.Get(result.TargetEntry).Name + ")を狙ったが、既に行動不能だった！"
		return result
	}
	targetParts := PartsComponent.Get(result.TargetEntry)
	if targetParts.Map[result.TargetPartSlot] == nil || targetParts.Map[result.TargetPartSlot].IsBroken {
		result.LogMessage = settings.Name + "は" + SettingsComponent.Get(result.TargetEntry).Name + "の" + string(result.TargetPartSlot) + "を狙ったが、パーツは既に破壊されていた！"
		return result
	}
	result.LogMessage = settings.Name + "は" + SettingsComponent.Get(result.TargetEntry).Name + "の" + string(result.TargetPartSlot) + "を狙う！"

	// 2. 命中判定
	didHit := battleLogic.HitCalculator.CalculateHit(actingEntry, result.TargetEntry, actingPartDef)
	result.ActionDidHit = didHit
	if !didHit {
		result.LogMessage = battleLogic.DamageCalculator.GenerateActionLog(actingEntry, result.TargetEntry, actingPartDef, nil, 0, false, false)
		h.cleanup(actingEntry)
		return result
	}

	// 3. ダメージ適用
	damage, isCritical := battleLogic.DamageCalculator.CalculateDamage(actingEntry, result.TargetEntry, actingPartDef)
	result.IsCritical = isCritical
	result.OriginalDamage = damage

	intendedTargetPartInstance := targetParts.Map[result.TargetPartSlot]
	battleLogic.DamageCalculator.ApplyDamage(result.TargetEntry, intendedTargetPartInstance, result.OriginalDamage)
	result.DamageDealt = result.OriginalDamage
	result.TargetPartBroken = intendedTargetPartInstance.IsBroken
	result.ActualHitPartSlot = result.TargetPartSlot
	
	// 4. アクション結果生成
	targetPartDef, _ := GlobalGameDataManager.GetPartDefinition(intendedTargetPartInstance.DefinitionID)
	result.LogMessage = battleLogic.DamageCalculator.GenerateActionLog(actingEntry, result.TargetEntry, actingPartDef, targetPartDef, result.DamageDealt, result.IsCritical, true)

	if result.TargetPartBroken {
		partBrokenParams := map[string]interface{}{
			"target_name":      SettingsComponent.Get(result.TargetEntry).Name,
			"target_part_name": targetPartDef.PartName,
		}
		additionalMsg := GlobalGameDataManager.Messages.FormatMessage("part_broken", partBrokenParams)
		result.LogMessage += " " + additionalMsg
	}

	// 5. クリーンアップ
	h.cleanup(actingEntry)

	return result
}

func (h *ShootActionHandler) cleanup(actingEntry *donburi.Entry) {
	if actingEntry.HasComponent(EvasionDebuffComponent) {
		actingEntry.RemoveComponent(EvasionDebuffComponent)
	}
	if actingEntry.HasComponent(DefenseDebuffComponent) {
		actingEntry.RemoveComponent(DefenseDebuffComponent)
	}
}

// MeleeActionHandler は格闘カテゴリのパーツのアクションを処理します。
type MeleeActionHandler struct{}

func (h *MeleeActionHandler) Execute(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	battleLogic *BattleLogic,
	gameConfig *Config,
) ActionResult {
	result := ActionResult{ActingEntry: actingEntry}
	settings := SettingsComponent.Get(actingEntry)
	partsComp := PartsComponent.Get(actingEntry)
	actingPartInstance := partsComp.Map[intent.SelectedPartKey]
	actingPartDef, _ := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID)

	// 1. ターゲット解決
	closestEnemy := battleLogic.TargetSelector.FindClosestEnemy(actingEntry)
	if closestEnemy == nil {
		result.LogMessage = settings.Name + "は格闘攻撃しようとしたが、相手がいなかった。"
		return result
	}
	result.TargetEntry = closestEnemy

	if StateComponent.Get(closestEnemy).Current == StateTypeBroken {
		result.LogMessage = settings.Name + "はターゲット(" + SettingsComponent.Get(closestEnemy).Name + ")を狙ったが、既に行動不能だった！"
		return result
	}

	targetPart := battleLogic.TargetSelector.SelectRandomPartToDamage(closestEnemy)
	if targetPart == nil {
		result.LogMessage = settings.Name + "は" + SettingsComponent.Get(closestEnemy).Name + "を狙ったが、攻撃できる部位がなかった！"
		return result
	}
	targetPartSlot := battleLogic.PartInfoProvider.FindPartSlot(closestEnemy, targetPart)
	if targetPartSlot == "" {
		result.LogMessage = settings.Name + "の" + SettingsComponent.Get(closestEnemy).Name + "への攻撃でパーツスロット特定失敗。"
		return result
	}
	result.TargetPartSlot = targetPartSlot
	result.LogMessage = settings.Name + "は" + SettingsComponent.Get(closestEnemy).Name + "の" + string(targetPartSlot) + "に格闘攻撃！"

	// 2. 命中判定
	didHit := battleLogic.HitCalculator.CalculateHit(actingEntry, result.TargetEntry, actingPartDef)
	result.ActionDidHit = didHit
	if !didHit {
		result.LogMessage = battleLogic.DamageCalculator.GenerateActionLog(actingEntry, result.TargetEntry, actingPartDef, nil, 0, false, false)
		h.cleanup(actingEntry)
		return result
	}

	// 3. ダメージ適用
	damage, isCritical := battleLogic.DamageCalculator.CalculateDamage(actingEntry, result.TargetEntry, actingPartDef)
	result.IsCritical = isCritical
	result.OriginalDamage = damage

	targetParts := PartsComponent.Get(result.TargetEntry)
	intendedTargetPartInstance := targetParts.Map[result.TargetPartSlot]
	battleLogic.DamageCalculator.ApplyDamage(result.TargetEntry, intendedTargetPartInstance, result.OriginalDamage)
	result.DamageDealt = result.OriginalDamage
	result.TargetPartBroken = intendedTargetPartInstance.IsBroken
	result.ActualHitPartSlot = result.TargetPartSlot

	// 4. アクション結果生成
	targetPartDef, _ := GlobalGameDataManager.GetPartDefinition(intendedTargetPartInstance.DefinitionID)
	result.LogMessage = battleLogic.DamageCalculator.GenerateActionLog(actingEntry, result.TargetEntry, actingPartDef, targetPartDef, result.DamageDealt, result.IsCritical, true)

	if result.TargetPartBroken {
		partBrokenParams := map[string]interface{}{
			"target_name":      SettingsComponent.Get(result.TargetEntry).Name,
			"target_part_name": targetPartDef.PartName,
		}
		additionalMsg := GlobalGameDataManager.Messages.FormatMessage("part_broken", partBrokenParams)
		result.LogMessage += " " + additionalMsg
	}

	// 5. クリーンアップ
	h.cleanup(actingEntry)

	return result
}

func (h *MeleeActionHandler) cleanup(actingEntry *donburi.Entry) {
	if actingEntry.HasComponent(EvasionDebuffComponent) {
		actingEntry.RemoveComponent(EvasionDebuffComponent)
	}
	if actingEntry.HasComponent(DefenseDebuffComponent) {
		actingEntry.RemoveComponent(DefenseDebuffComponent)
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
