package main

import (
	// "fmt"
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ActionHandler はカテゴリ固有のアクション処理全体をカプセル化します。
type TraitActionHandler interface {
	Execute(
		actingEntry *donburi.Entry,
		world donburi.World,
		intent *ActionIntent,
		battleLogic *BattleLogic,
		gameConfig *Config,
		actingPartDef *PartDefinition,
		result *ActionResult,
	)
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
		return result
	}

	// 2. 命中判定
	didHit := performHitCheck(actingEntry, targetEntry, actingPartDef, battleLogic)
	result.ActionDidHit = didHit
	if !didHit {
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

	return result
}

// ShootTraitHandler は TraitShoot のアクションを処理します。
type ShootTraitHandler struct{}

func (h *ShootTraitHandler) Execute(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	battleLogic *BattleLogic,
	gameConfig *Config,
	actingPartDef *PartDefinition,
	result *ActionResult,
) {
	targetComp := TargetComponent.Get(actingEntry)

	if targetComp.TargetEntity == nil || targetComp.TargetPartSlot == "" {
		result.ActionDidHit = false // ターゲット未選択は失敗
		return
	}

	// executeAttackAction は ActionResult を返すので、result ポインタに直接設定
	*result = executeAttackAction(
		actingEntry,
		world,
		intent,
		battleLogic,
		gameConfig,
		targetComp.TargetEntity,
		targetComp.TargetPartSlot,
	)
}

// MeleeTraitHandler は TraitMelee のアクションを処理します。
type MeleeTraitHandler struct{}

func (h *MeleeTraitHandler) Execute(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	battleLogic *BattleLogic,
	gameConfig *Config,
	actingPartDef *PartDefinition,
	result *ActionResult,
) {
	// 1. ターゲット解決
	closestEnemy := battleLogic.TargetSelector.FindClosestEnemy(actingEntry)
	if closestEnemy == nil {
		result.ActionDidHit = false // ターゲット不在は失敗
		return
	}

	targetPart := battleLogic.TargetSelector.SelectRandomPartToDamage(closestEnemy)
	if targetPart == nil {
		result.ActionDidHit = false // 攻撃できる部位がない場合は失敗
		return
	}
	targetPartSlot := battleLogic.PartInfoProvider.FindPartSlot(closestEnemy, targetPart)
	if targetPartSlot == "" {
		result.ActionDidHit = false // パーツスロット特定失敗は失敗
		return
	}

	// executeAttackAction は ActionResult を返すので、result ポインタに直接設定
	*result = executeAttackAction(
		actingEntry,
		world,
		intent,
		battleLogic,
		gameConfig,
		closestEnemy,
		targetPartSlot,
	)
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
	result *ActionResult,
) {
	settings := SettingsComponent.Get(actingEntry)

	// 1. TeamBuffsComponent を取得
	teamBuffsEntry, ok := query.NewQuery(filter.Contains(TeamBuffsComponent)).First(world)
	if !ok {
		log.Println("エラー: TeamBuffsComponent がワールドに見つかりません。")
		result.ActionDidHit = false // 失敗としてマーク
		return
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
	result *ActionResult,
) {
	settings := SettingsComponent.Get(actingEntry)
	targetComp := TargetComponent.Get(actingEntry)
	if targetComp.TargetEntity == nil {
		log.Printf("%s は妨害ターゲットが未選択です。", settings.Name)
		result.ActionDidHit = false
		return
	}
	targetEntry := targetComp.TargetEntity
	result.TargetEntry = targetEntry
	result.DefenderName = SettingsComponent.Get(targetEntry).Name

	// ここに将来的なデバフ処理を実装します。
	// 例: ターゲットの回避デバフコンポーネントを追加
	// donburi.Add(targetEntry, EvasionDebuffComponent, &EvasionDebuff{Multiplier: 0.8})
	log.Printf("%s が %s に妨害を実行しました（現在効果なし）。", settings.Name, result.DefenderName)
}

// traitHandlers は Trait に対応する TraitActionHandler のレジストリです。
var traitHandlers = map[Trait]TraitActionHandler{
	TraitShoot:    &ShootTraitHandler{},
	TraitAim:      &ShootTraitHandler{}, // 狙い撃ちも射撃ハンドラを使用
	TraitMelee:    &MeleeTraitHandler{},
	TraitStrike:   &MeleeTraitHandler{}, // 殴るも格闘ハンドラを使用
	TraitBerserk:  &MeleeTraitHandler{}, // 我武者羅も格闘ハンドラを使用
	TraitSupport:  &SupportTraitExecutor{},
	TraitObstruct: &ObstructTraitExecutor{},
}
