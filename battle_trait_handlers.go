package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// --- BaseAttackHandler ---

// BaseAttackHandler は、すべての攻撃アクションに共通するロジックをカプセル化します。
type BaseAttackHandler struct{}

// Execute は TraitActionHandler インターフェースを実装します。
func (h *BaseAttackHandler) Execute(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	battleLogic *BattleLogic,
	gameConfig *Config,
	actingPartDef *PartDefinition,
) ActionResult {
	_ = battleLogic // リンターの未使用パラメータ警告を抑制
	// PerformAttack は、ターゲットの解決、命中判定、ダメージ計算、防御処理などの共通攻撃ロジックを実行します。
	// Execute メソッドから呼び出されるため、引数を調整します。
	return h.performAttackLogic(actingEntry, intent, battleLogic, actingPartDef)
}

// initializeAttackResult は ActionResult を初期化します。
func initializeAttackResult(actingEntry *donburi.Entry, actingPartDef *PartDefinition, battleLogic *BattleLogic) ActionResult {
	_ = battleLogic // リンターの未使用パラメータ警告を抑制
	return ActionResult{
		ActingEntry:    actingEntry,
		ActionDidHit:   false, // 初期値はfalse
		AttackerName:   SettingsComponent.Get(actingEntry).Name,
		ActionName:     actingPartDef.PartName,
		ActionTrait:    actingPartDef.Trait,
		ActionCategory: actingPartDef.Category,
		WeaponType:     actingPartDef.WeaponType,
	}
}

// performAttackLogic は攻撃アクションの主要なロジックを実行します。
func (h *BaseAttackHandler) performAttackLogic(
	actingEntry *donburi.Entry,
	intent *ActionIntent,
	battleLogic *BattleLogic,
	actingPartDef *PartDefinition,
) ActionResult {
	result := initializeAttackResult(actingEntry, actingPartDef, battleLogic)

	targetEntry, targetPartSlot := resolveAttackTarget(actingEntry, battleLogic)
	if targetEntry == nil {
		return result // ターゲットが見つからない場合は、ActionDidHit: false のまま返す
	}

	result.TargetEntry = targetEntry
	result.TargetPartSlot = targetPartSlot
	result.DefenderName = SettingsComponent.Get(targetEntry).Name
	result.ActionDidHit = true // ターゲットが見つかったので、初期値をtrueに設定

	if !validateTarget(targetEntry, targetPartSlot) {
		result.ActionDidHit = false
		return result
	}

	didHit := performHitCheck(actingEntry, targetEntry, actingPartDef, intent.SelectedPartKey, battleLogic)
	result.ActionDidHit = didHit
	if !didHit {
		return result
	}

	damage, isCritical := battleLogic.GetDamageCalculator().CalculateDamage(actingEntry, targetEntry, actingPartDef, intent.SelectedPartKey, battleLogic)
	result.IsCritical = isCritical
	result.OriginalDamage = damage

	applyDamageAndDefense(&result, actingEntry, actingPartDef, intent.SelectedPartKey, battleLogic)

	finalizeActionResult(&result, battleLogic)

	return result
}

// --- attack action helpers ---

func validateTarget(targetEntry *donburi.Entry, targetPartSlot PartSlotKey) bool {
	if StateComponent.Get(targetEntry).CurrentState == StateBroken {
		return false
	}
	targetParts := PartsComponent.Get(targetEntry)
	if targetParts.Map[targetPartSlot] == nil || targetParts.Map[targetPartSlot].IsBroken {
		return false
	}
	return true
}

func performHitCheck(actingEntry, targetEntry *donburi.Entry, actingPartDef *PartDefinition, selectedPartKey PartSlotKey, battleLogic *BattleLogic) bool {
	return battleLogic.GetHitCalculator().CalculateHit(actingEntry, targetEntry, actingPartDef, selectedPartKey, battleLogic)
}

func applyDamageAndDefense(
	result *ActionResult,
	actingEntry *donburi.Entry,
	actingPartDef *PartDefinition,
	selectedPartKey PartSlotKey,
	battleLogic *BattleLogic,
) {
	defendingPartInst := battleLogic.GetTargetSelector().SelectDefensePart(result.TargetEntry, battleLogic)

	if defendingPartInst != nil && battleLogic.GetHitCalculator().CalculateDefense(actingEntry, result.TargetEntry, actingPartDef, selectedPartKey, battleLogic) {
		result.ActionIsDefended = true
		defendingPartDef, _ := battleLogic.GetPartInfoProvider().gameDataManager.GetPartDefinition(defendingPartInst.DefinitionID)
		result.DefendingPartType = string(defendingPartDef.Type)
		result.ActualHitPartSlot = battleLogic.GetPartInfoProvider().FindPartSlot(result.TargetEntry, defendingPartInst)

		finalDamage := battleLogic.GetDamageCalculator().CalculateReducedDamage(result.OriginalDamage, result.TargetEntry, battleLogic)
		result.DamageDealt = finalDamage
		battleLogic.GetDamageCalculator().ApplyDamage(result.TargetEntry, defendingPartInst, finalDamage, battleLogic)
		result.TargetPartBroken = defendingPartInst.IsBroken
	} else {
		result.ActionIsDefended = false
		intendedTargetPartInstance := PartsComponent.Get(result.TargetEntry).Map[result.TargetPartSlot]
		result.DamageDealt = result.OriginalDamage
		result.ActualHitPartSlot = result.TargetPartSlot

		battleLogic.GetDamageCalculator().ApplyDamage(result.TargetEntry, intendedTargetPartInstance, result.OriginalDamage, battleLogic)
		result.TargetPartBroken = intendedTargetPartInstance.IsBroken
	}
}

func finalizeActionResult(result *ActionResult, battleLogic *BattleLogic) {
	actualHitPartInst := PartsComponent.Get(result.TargetEntry).Map[result.ActualHitPartSlot]
	actualHitPartDef, _ := battleLogic.GetPartInfoProvider().gameDataManager.GetPartDefinition(actualHitPartInst.DefinitionID)

	result.TargetPartType = string(actualHitPartDef.Type)
}

// resolveAttackTarget は攻撃アクションのターゲットを解決します。
func resolveAttackTarget(
	actingEntry *donburi.Entry,
	battleLogic *BattleLogic,
) (targetEntry *donburi.Entry, targetPartSlot PartSlotKey) {
	targetComp := TargetComponent.Get(actingEntry)
	switch targetComp.Policy {
	case PolicyPreselected:
		if targetComp.TargetEntity == 0 { // nil から 0 に変更
			log.Printf("エラー: PolicyPreselected なのにターゲットが設定されていません。")
			return nil, ""
		}
		// donburi.Entity から *donburi.Entry を取得
		targetEntry := battleLogic.world.Entry(targetComp.TargetEntity)
		if targetEntry == nil {
			log.Printf("エラー: ターゲットエンティティID %d がワールドに見つかりません。", targetComp.TargetEntity)
			return nil, ""
		}
		return targetEntry, targetComp.TargetPartSlot
	case PolicyClosestAtExecution:
		closestEnemy := battleLogic.GetTargetSelector().FindClosestEnemy(actingEntry, battleLogic)
		if closestEnemy == nil {
			return nil, "" // ターゲットが見つからない場合は失敗
		}
		targetPart := battleLogic.GetTargetSelector().SelectPartToDamage(closestEnemy, actingEntry, battleLogic)
		if targetPart == nil {
			return nil, "" // ターゲットパーツが見つからない場合は失敗
		}
		slot := battleLogic.GetPartInfoProvider().FindPartSlot(closestEnemy, targetPart)
		if slot == "" {
			return nil, "" // ターゲットスロットが見つからない場合は失敗
		}
		return closestEnemy, slot
	default:
		log.Printf("未対応のTargetingPolicyです: %s", targetComp.Policy)
		return nil, ""
	}
}

// SupportTraitExecutor は TraitSupport の介入アクションを処理します。
type SupportTraitExecutor struct{}

func (h *SupportTraitExecutor) Execute(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	battleLogic *BattleLogic,
	gameConfig *Config,
	actingPartDef *PartDefinition,
) ActionResult {
	settings := SettingsComponent.Get(actingEntry)
	result := ActionResult{
		ActingEntry:    actingEntry,
		ActionDidHit:   true,
		AttackerName:   settings.Name,
		ActionName:     actingPartDef.PartName,
		ActionTrait:    actingPartDef.Trait, // string() を削除
		ActionCategory: actingPartDef.Category,
		WeaponType:     actingPartDef.WeaponType,
	}

	teamBuffsEntry, ok := query.NewQuery(filter.Contains(TeamBuffsComponent)).First(world)
	if !ok {
		log.Println("エラー: TeamBuffsComponent がワールドに見つかりません。")
		result.ActionDidHit = false
		return result
	}
	teamBuffs := TeamBuffsComponent.Get(teamBuffsEntry)

	buffValue := 1.0 + (float64(actingPartDef.Power) / 100.0)
	newBuffSource := &BuffSource{
		SourceEntry: actingEntry,
		SourcePart:  intent.SelectedPartKey,
		Value:       buffValue,
	}

	teamID := settings.Team
	buffType := BuffTypeAccuracy

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

	return result
}

// ObstructTraitExecutor は TraitObstruct の介入アクションを処理します。
type ObstructTraitExecutor struct{}

func (h *ObstructTraitExecutor) Execute(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	battleLogic *BattleLogic,
	gameConfig *Config,
	actingPartDef *PartDefinition,
) ActionResult {
	settings := SettingsComponent.Get(actingEntry)
	result := ActionResult{
		ActingEntry:    actingEntry,
		ActionDidHit:   true,
		AttackerName:   settings.Name,
		ActionName:     actingPartDef.PartName,
		ActionTrait:    actingPartDef.Trait,
		ActionCategory: actingPartDef.Category,
		WeaponType:     actingPartDef.WeaponType,
	}
	targetComp := TargetComponent.Get(actingEntry)
	if targetComp.TargetEntity == 0 { // nil から 0 に変更
		log.Printf("%s は妨害ターゲットが未選択です。", settings.Name)
		result.ActionDidHit = false
		return result
	}
	// donburi.Entity から *donburi.Entry を取得
	targetEntry := world.Entry(targetComp.TargetEntity)
	if targetEntry == nil {
		log.Printf("エラー: ターゲットエンティティID %d がワールドに見つかりません。", targetComp.TargetEntity)
		result.ActionDidHit = false
		return result
	}
	result.TargetEntry = targetEntry
	result.DefenderName = SettingsComponent.Get(targetEntry).Name

	log.Printf("%s が %s に妨害を実行しました（現在効果なし）。", settings.Name, result.DefenderName)
	return result
}
