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

// executeAttackAction は射撃と格闘の共通攻撃ロジックをカプセル化します。
func executeAttackAction(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	battleLogic *BattleLogic,
	gameConfig *Config,
	targetEntry *donburi.Entry,
	targetPartSlot PartSlotKey,
	logPrefix string, // ログメッセージのプレフィックス
) ActionResult {
	result := ActionResult{ActingEntry: actingEntry}
	settings := SettingsComponent.Get(actingEntry)
	partsComp := PartsComponent.Get(actingEntry)
	actingPartInstance := partsComp.Map[intent.SelectedPartKey]
	actingPartDef, _ := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID)

	result.TargetEntry = targetEntry
	result.TargetPartSlot = targetPartSlot

	// 命中判定やダメージ計算の前に、基本的な情報を結果に設定
	result.AttackerName = settings.Name
	result.DefenderName = SettingsComponent.Get(targetEntry).Name
	result.ActionName = string(actingPartDef.Trait)
	result.WeaponType = actingPartDef.WeaponType

	if StateComponent.Get(result.TargetEntry).FSM.Is(string(StateBroken)) {
		// ログメッセージは event/animation 層で生成するため、ここでは何もしない
		cleanupActionDebuffs(actingEntry)
		return result
	}
	targetParts := PartsComponent.Get(result.TargetEntry)
	if targetParts.Map[result.TargetPartSlot] == nil || targetParts.Map[result.TargetPartSlot].IsBroken {
		// ログメッセージは event/animation 層で生成するため、ここでは何もしない
		cleanupActionDebuffs(actingEntry)
		return result
	}

	// 2. 命中判定
	didHit := battleLogic.HitCalculator.CalculateHit(actingEntry, result.TargetEntry, actingPartDef)
	result.ActionDidHit = didHit
	if !didHit {
		// ログメッセージは event/animation 層で生成するため、ここでは何もしない
		cleanupActionDebuffs(actingEntry)
		return result
	}

	// 3. 初期ダメージ計算
	damage, isCritical := battleLogic.DamageCalculator.CalculateDamage(actingEntry, result.TargetEntry, actingPartDef)
	result.IsCritical = isCritical
	result.OriginalDamage = damage

	// 4. 防御判定と最終ダメージ適用
	// 4a. 防御パーツの選択
	defendingPartInst := battleLogic.TargetSelector.SelectDefensePart(result.TargetEntry)

	// 4b. 防御成功判定
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

	// 5. アクション結果生成 (ログ用)
	// 実際にダメージを受けたパーツの情報を取得
	actualHitPartInst := PartsComponent.Get(result.TargetEntry).Map[result.ActualHitPartSlot]
	actualHitPartDef, _ := GlobalGameDataManager.GetPartDefinition(actualHitPartInst.DefinitionID)
	result.AttackerName = settings.Name
	result.DefenderName = SettingsComponent.Get(result.TargetEntry).Name
	result.ActionName = string(actingPartDef.Trait)
	result.WeaponType = actingPartDef.WeaponType
	result.TargetPartType = string(actualHitPartDef.Type) // 実際にヒットしたパーツのタイプ

	// ログメッセージはscene_battle.goで生成するため、ここでは生成しない
	// if result.TargetPartBroken { ... }

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
	settings := SettingsComponent.Get(actingEntry)
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
		settings.Name+"は",
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
	settings := SettingsComponent.Get(actingEntry)
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
		settings.Name+"は"+SettingsComponent.Get(closestEnemy).Name+"の"+string(targetPartSlot)+"に格闘攻撃！",
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

	// 1. TeamBuffsComponent を取得
	teamBuffsEntry, ok := query.NewQuery(filter.Contains(TeamBuffsComponent)).First(world)
	if !ok {
		log.Println("エラー: TeamBuffsComponent がワールドに見つかりません。")
		return ActionResult{
			ActingEntry: actingEntry,
			// LogMessage:  settings.Name + "は支援行動に失敗した(バフ管理エラー)。",
		}
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
	buffType := BuffTypeAccuracy

	// チームのバフマップがなければ初期化
	if _, exists := teamBuffs.Buffs[teamID]; !exists {
		teamBuffs.Buffs[teamID] = make(map[BuffType][]*BuffSource)
	}
	// バフタイプのスライスがなければ初期化
	if _, exists := teamBuffs.Buffs[teamID][buffType]; !exists {
		teamBuffs.Buffs[teamID][buffType] = make([]*BuffSource, 0)
	}

	// 同じソースからの既存のバフがあれば削除 (念のため)
	existingBuffs := teamBuffs.Buffs[teamID][buffType]
	filteredBuffs := make([]*BuffSource, 0, len(existingBuffs))
	for _, buff := range existingBuffs {
		if buff.SourceEntry != actingEntry || buff.SourcePart != intent.SelectedPartKey {
			filteredBuffs = append(filteredBuffs, buff)
		}
	}

	// 新しいバフを追加
	teamBuffs.Buffs[teamID][buffType] = append(filteredBuffs, newBuffSource)

	log.Printf("チーム%dに命中バフを追加: %s (%.2f倍)", teamID, settings.Name, buffValue)

	// 4. アクション結果を生成
	result := ActionResult{
		ActingEntry:  actingEntry,
		ActionDidHit: true, // 支援行動は必ず「成功」とする
		AttackerName: settings.Name,
		ActionName:   string(actingPartDef.Trait),
		WeaponType:   actingPartDef.WeaponType,
	}

	// LogMessage は scene_battle.go で生成するため、ここでは設定しない
	// result.LogMessage = GlobalGameDataManager.Messages.FormatMessage("support_action_generic", params)

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
	case CategoryShoot:
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
