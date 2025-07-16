package main

import (
	// "fmt"
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

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


// --- attack action helpers ---

// validateTarget は攻撃対象が有効かどうかをチェックします。
func validateTarget(targetEntry *donburi.Entry, targetPartSlot PartSlotKey) bool {
	if StateComponent.Get(targetEntry).FSM.Is(string(StateBroken)) {
		return false
	}
	targetParts := PartsComponent.Get(targetEntry)
	if targetParts.Map[targetPartSlot] == nil || targetParts.Map[targetPartSlot].IsBroken {
		return false
	}
	return true
}

// performHitCheck は命中判定を実行し、結果を返します。
func performHitCheck(actingEntry, targetEntry *donburi.Entry, actingPartDef *PartDefinition, battleLogic *BattleLogic) bool {
	return battleLogic.HitCalculator.CalculateHit(actingEntry, targetEntry, actingPartDef)
}

// applyDamageAndDefense は防御判定と最終的なダメージ適用を行います。
func applyDamageAndDefense(
	result *ActionResult,
	actingEntry *donburi.Entry,
	actingPartDef *PartDefinition,
	battleLogic *BattleLogic,
) {
	// 防御パーツの選択
	defendingPartInst := battleLogic.TargetSelector.SelectDefensePart(result.TargetEntry)

	// 防御成功判定
	if defendingPartInst != nil && battleLogic.HitCalculator.CalculateDefense(actingEntry, result.TargetEntry, actingPartDef) {
		result.ActionIsDefended = true
		defendingPartDef, _ := GlobalGameDataManager.GetPartDefinition(defendingPartInst.DefinitionID)
		result.DefendingPartType = string(defendingPartDef.Type)
		result.ActualHitPartSlot = battleLogic.PartInfoProvider.FindPartSlot(result.TargetEntry, defendingPartInst)

		// ダメージ軽減計算と適用
		finalDamage := battleLogic.DamageCalculator.CalculateReducedDamage(result.OriginalDamage, defendingPartDef)
		result.DamageDealt = finalDamage
		battleLogic.DamageCalculator.ApplyDamage(result.TargetEntry, defendingPartInst, finalDamage)
		result.TargetPartBroken = defendingPartInst.IsBroken
	} else {
		// 防御失敗、または防御パーツがない場合
		result.ActionIsDefended = false
		intendedTargetPartInstance := PartsComponent.Get(result.TargetEntry).Map[result.TargetPartSlot]
		result.DamageDealt = result.OriginalDamage
		result.ActualHitPartSlot = result.TargetPartSlot // 意図したパーツにヒット

		// 意図したターゲットパーツにダメージを適用
		battleLogic.DamageCalculator.ApplyDamage(result.TargetEntry, intendedTargetPartInstance, result.OriginalDamage)
		result.TargetPartBroken = intendedTargetPartInstance.IsBroken
	}
}

// finalizeActionResult は、最終的なアクション結果を構築します。
func finalizeActionResult(result *ActionResult, actingEntry *donburi.Entry, actingPartDef *PartDefinition) {
	actualHitPartInst := PartsComponent.Get(result.TargetEntry).Map[result.ActualHitPartSlot]
	actualHitPartDef, _ := GlobalGameDataManager.GetPartDefinition(actualHitPartInst.DefinitionID)

	result.TargetPartType = string(actualHitPartDef.Type)
}

// executeAttackAction は射撃と格闘の共通攻撃ロジックをカプセル化します。
func executeAttackAction(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	battleLogic *BattleLogic,
	gameConfig *Config,
	targetEntry *donburi.Entry,
	targetPartSlot PartSlotKey,
) ActionResult {
	result := ActionResult{
		ActingEntry:    actingEntry,
		TargetEntry:    targetEntry,
		TargetPartSlot: targetPartSlot,
	}
	partsComp := PartsComponent.Get(actingEntry)
	actingPartInstance := partsComp.Map[intent.SelectedPartKey]
	actingPartDef, _ := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID)

	// 基本的な情報を結果に設定（命中判定の前に設定することで、ミス時も情報が残るようにする）
	result.AttackerName = SettingsComponent.Get(actingEntry).Name
	result.DefenderName = SettingsComponent.Get(targetEntry).Name
	result.ActionName = string(actingPartDef.Trait)
	result.WeaponType = actingPartDef.WeaponType

	// 1. ターゲットの有効性チェック
	if !validateTarget(targetEntry, targetPartSlot) {
		cleanupActionDebuffs(actingEntry)
		return result
	}

	// 2. 命中判定
	didHit := performHitCheck(actingEntry, targetEntry, actingPartDef, battleLogic)
	result.ActionDidHit = didHit
	if !didHit {
		cleanupActionDebuffs(actingEntry)
		return result
	}

	// 3. 初期ダメージ計算
	damage, isCritical := battleLogic.DamageCalculator.CalculateDamage(actingEntry, targetEntry, actingPartDef)
	result.IsCritical = isCritical
	result.OriginalDamage = damage

	// 4. 防御判定と最終ダメージ適用
	applyDamageAndDefense(&result, actingEntry, actingPartDef, battleLogic)

	// 5. 最終結果の構築
	finalizeActionResult(&result, actingEntry, actingPartDef)

	// 6. クリーンアップ
	cleanupActionDebuffs(actingEntry)

	return result
}


// ShootActionHandler は射撃カテゴリのパーツのアクションを処理します。
type ShootActionHandler struct{}

func (h *ShootActionHandler) Execute(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	battleLogic *BattleLogic,
	gameConfig *Config,
) ActionResult {
	targetComp := TargetComponent.Get(actingEntry)

	if targetComp.TargetEntity == nil || targetComp.TargetPartSlot == "" {
		return ActionResult{
			ActingEntry: actingEntry,
			// LogMessage:  settings.Name + "は射撃ターゲットが未選択です。",
		}
	}

	return executeAttackAction(
		actingEntry,
		world,
		intent,
		battleLogic,
		gameConfig,
		targetComp.TargetEntity,
		targetComp.TargetPartSlot,
	)
}

// cleanup は行動後のデバフをクリーンアップします。
func cleanupActionDebuffs(actingEntry *donburi.Entry) {
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
	// 1. ターゲット解決
	closestEnemy := battleLogic.TargetSelector.FindClosestEnemy(actingEntry)
	if closestEnemy == nil {
		return ActionResult{
			ActingEntry: actingEntry,
			// LogMessage:  settings.Name + "は格闘攻撃しようとしたが、相手がいなかった。",
		}
	}

	targetPart := battleLogic.TargetSelector.SelectRandomPartToDamage(closestEnemy)
	if targetPart == nil {
		return ActionResult{
			ActingEntry: actingEntry,
			// LogMessage:  settings.Name + "は" + SettingsComponent.Get(closestEnemy).Name + "を狙ったが、攻撃できる部位がなかった！",
		}
	}
	targetPartSlot := battleLogic.PartInfoProvider.FindPartSlot(closestEnemy, targetPart)
	if targetPartSlot == "" {
		return ActionResult{
			ActingEntry: actingEntry,
			// LogMessage:  settings.Name + "の" + SettingsComponent.Get(closestEnemy).Name + "への攻撃でパーツスロット特定失敗。",
		}
	}

	return executeAttackAction(
		actingEntry,
		world,
		intent,
		battleLogic,
		gameConfig,
		closestEnemy,
		targetPartSlot,
	)
}

// InterventionActionHandler は介入カテゴリのパーツのアクションを処理します。
type InterventionActionHandler struct{}

func (h *InterventionActionHandler) Execute(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	battleLogic *BattleLogic,
	gameConfig *Config,
) ActionResult {
	settings := SettingsComponent.Get(actingEntry)
	partsComp := PartsComponent.Get(actingEntry)
	actingPartInst := partsComp.Map[intent.SelectedPartKey]
	actingPartDef, _ := GlobalGameDataManager.GetPartDefinition(actingPartInst.DefinitionID)
	result := ActionResult{
		ActingEntry:  actingEntry,
		ActionDidHit: true, // 介入行動は基本「成功」とする
		AttackerName: settings.Name,
		ActionName:   string(actingPartDef.Trait),
		WeaponType:   actingPartDef.WeaponType,
	}

	switch actingPartDef.Trait {
	case TraitSupport:
		// --- 支援処理 ---
		// 1. TeamBuffsComponent を取得
		teamBuffsEntry, ok := query.NewQuery(filter.Contains(TeamBuffsComponent)).First(world)
		if !ok {
			log.Println("エラー: TeamBuffsComponent がワールドに見つかりません。")
			result.ActionDidHit = false // 失敗としてマーク
			return result
		}
		teamBuffs := TeamBuffsComponent.Get(teamBuffsEntry)

		// 2. バフ情報を生成
		buffValue := 1.0 + (float64(actingPartDef.Power) / 100.0)
		newBuffSource := &BuffSource{
			SourceEntry: actingEntry,
			SourcePart:  intent.SelectedPartKey,
			Value:       buffValue,
		}

		// 3. TeamBuffsComponent を更新
		teamID := settings.Team
		buffType := BuffTypeAccuracy // 現在は命中バフ固定

		if _, exists := teamBuffs.Buffs[teamID]; !exists {
			teamBuffs.Buffs[teamID] = make(map[BuffType][]*BuffSource)
		}
		if _, exists := teamBuffs.Buffs[teamID][buffType]; !exists {
			teamBuffs.Buffs[teamID][buffType] = make([]*BuffSource, 0)
		}

		existingBuffs := teamBuffs.Buffs[teamID][buffType]
		filteredBuffs := make([]*BuffSource, 0, len(existingBuffs))
		for _, buff := range existingBuffs {
			if buff.SourceEntry != actingEntry || buff.SourcePart != intent.SelectedPartKey {
				filteredBuffs = append(filteredBuffs, buff)
			}
		}
		teamBuffs.Buffs[teamID][buffType] = append(filteredBuffs, newBuffSource)
		log.Printf("チーム%dに命中バフを追加: %s (%.2f倍)", teamID, settings.Name, buffValue)

	case TraitObstruct:
		// --- 妨害処理 ---
		targetComp := TargetComponent.Get(actingEntry)
		if targetComp.TargetEntity == nil {
			log.Printf("%s は妨害ターゲットが未選択です。", settings.Name)
			result.ActionDidHit = false
			return result
		}
		targetEntry := targetComp.TargetEntity
		result.TargetEntry = targetEntry
		result.DefenderName = SettingsComponent.Get(targetEntry).Name

		// ここに将来的なデバフ処理を実装します。
		// 例: ターゲットの回避デバフコンポーネントを追加
		// donburi.Add(targetEntry, EvasionDebuffComponent, &EvasionDebuff{Multiplier: 0.8})
		log.Printf("%s が %s に妨害を実行しました（現在効果なし）。", settings.Name, result.DefenderName)

	default:
		log.Printf("未対応の介入Traitです: %s", actingPartDef.Trait)
		result.ActionDidHit = false // 不明なTraitは失敗とする
	}

	cleanupActionDebuffs(actingEntry)
	return result
}

// ハンドラのグローバルインスタンス（またはファクトリ/レジストリを使用することもできます）
var (
	shootHandler        = &ShootActionHandler{}
	meleeHandler        = &MeleeActionHandler{}
	interventionHandler = &InterventionActionHandler{}
)

// GetActionHandlerForCategory はパーツカテゴリに基づいて適切なActionHandlerを返します。
func GetActionHandlerForCategory(category PartCategory) ActionHandler {
	switch category {
	case CategoryRanged:
		return shootHandler
	case CategoryMelee:
		return meleeHandler
	case CategoryIntervention:
		return interventionHandler
	default:
		// 未処理の場合はデフォルトハンドラまたはnilを返します
		// 現状、nilを返すとexecuteActionLogicでフォールバックまたはエラー処理が必要になります
		return nil
	}
}
