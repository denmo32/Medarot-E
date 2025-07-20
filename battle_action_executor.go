package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// TraitActionHandler はカテゴリ固有のアクション処理全体をカプセル化します。
// ActionResultを返し、副作用をなくします。
type TraitActionHandler interface {
	Execute(
		actingEntry *donburi.Entry,
		world donburi.World,
		intent *ActionIntent,
		battleLogic *BattleLogic,
		gameConfig *Config,
		actingPartDef *PartDefinition,
		initialResult *ActionResult,
	) ActionResult
}

// WeaponTypeEffectHandler は weapon_type 固有の追加効果を処理します。
// ActionResult を受け取り、デバフ付与などの副作用を適用します。
type WeaponTypeEffectHandler interface {
	ApplyEffect(result *ActionResult, world donburi.World, battleLogic *BattleLogic, actingPartDef *PartDefinition)
}

// ActionExecutor はアクションの実行に関するロジックをカプセル化します。
type ActionExecutor struct {
	world                  donburi.World
	battleLogic            *BattleLogic
	gameConfig             *Config
	statusEffectSystem     *StatusEffectSystem
	postActionEffectSystem *PostActionEffectSystem // 新しく追加したシステム
	handlers               map[Trait]TraitActionHandler
	weaponHandlers         map[WeaponType]WeaponTypeEffectHandler // WeaponTypeごとのハンドラを追加

}

// --- BaseAttackHandler ---

// BaseAttackHandler は、すべての攻撃アクションに共通するロジックをカプセル化します。
type BaseAttackHandler struct{}

// Execute は TraitActionHandler インターフェースを実装します。
func (h *BaseAttackHandler) Execute(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *ActionIntent,
	battleLogic *BattleLogic, // battleLogic を再度受け取るように変更
	gameConfig *Config,
	actingPartDef *PartDefinition,
	initialResult *ActionResult,
) ActionResult {
	_ = battleLogic // リンターの未使用パラメータ警告を抑制
	// PerformAttack は、ターゲットの解決、命中判定、ダメージ計算、防御処理などの共通攻撃ロジックを実行します。
	// Execute メソッドから呼び出されるため、引数を調整します。
	return h.performAttackLogic(actingEntry, battleLogic, actingPartDef)
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

	didHit := performHitCheck(actingEntry, targetEntry, actingPartDef, battleLogic)
	result.ActionDidHit = didHit
	if !didHit {
		return result
	}

	damage, isCritical := battleLogic.GetDamageCalculator().CalculateDamage(actingEntry, targetEntry, actingPartDef, battleLogic)
	result.IsCritical = isCritical
	result.OriginalDamage = damage

	applyDamageAndDefense(&result, actingEntry, actingPartDef, battleLogic)

	finalizeActionResult(&result, battleLogic)

	return result
}

// --- attack action helpers ---

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

func performHitCheck(actingEntry, targetEntry *donburi.Entry, actingPartDef *PartDefinition, battleLogic *BattleLogic) bool {
	return battleLogic.GetHitCalculator().CalculateHit(actingEntry, targetEntry, actingPartDef, battleLogic)
}

func applyDamageAndDefense(
	result *ActionResult,
	actingEntry *donburi.Entry,
	actingPartDef *PartDefinition,
	battleLogic *BattleLogic,
) {
	defendingPartInst := battleLogic.GetTargetSelector().SelectDefensePart(result.TargetEntry, battleLogic)

	if defendingPartInst != nil && battleLogic.GetHitCalculator().CalculateDefense(actingEntry, result.TargetEntry, actingPartDef, battleLogic) {
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

// --- 具体的なハンドラ ---

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

// NewActionExecutor は新しいActionExecutorのインスタンスを生成します。
func NewActionExecutor(world donburi.World, battleLogic *BattleLogic, gameConfig *Config) *ActionExecutor {
	statusEffectSystem := NewStatusEffectSystem(world)                             // Create once
	postActionEffectSystem := NewPostActionEffectSystem(world, statusEffectSystem) // Use the created instance

	return &ActionExecutor{
		world:                  world,
		battleLogic:            battleLogic,
		gameConfig:             gameConfig,
		statusEffectSystem:     statusEffectSystem,     // Assign the created instance
		postActionEffectSystem: postActionEffectSystem, // Assign the new system

		handlers: map[Trait]TraitActionHandler{
			TraitShoot:    &BaseAttackHandler{},
			TraitAim:      &BaseAttackHandler{},
			TraitStrike:   &BaseAttackHandler{},
			TraitBerserk:  &BaseAttackHandler{},
			TraitSupport:  &SupportTraitExecutor{},
			TraitObstruct: &ObstructTraitExecutor{},
		},
		weaponHandlers: map[WeaponType]WeaponTypeEffectHandler{
			// 将来の拡張に備え、ここにハンドラを登録していく
			// 例: WeaponTypeThunder: &ThunderEffectHandler{},
			// 例: WeaponTypeMelt:    &MeltEffectHandler{},
		},
	}
}

// ExecuteAction は単一のアクションを実行し、その結果を返します。
func (e *ActionExecutor) ExecuteAction(actingEntry *donburi.Entry) ActionResult {
	intent := ActionIntentComponent.Get(actingEntry)
	partsComp := PartsComponent.Get(actingEntry)
	actingPartInst := partsComp.Map[intent.SelectedPartKey]
	actingPartDef, _ := e.battleLogic.GetPartInfoProvider().gameDataManager.GetPartDefinition(actingPartInst.DefinitionID)

	handler, ok := e.handlers[actingPartDef.Trait]
	if !ok {
		log.Printf("未対応のTraitです: %s", actingPartDef.Trait)
		return ActionResult{
			ActingEntry:  actingEntry,
			ActionDidHit: false,
		}
	}

	actionResult := handler.Execute(actingEntry, e.world, intent, e.battleLogic, e.gameConfig, actingPartDef, nil)

	// チャージ時に生成された保留中の効果をActionResultにコピー
	if len(intent.PendingEffects) > 0 {
		actionResult.AppliedEffects = append(actionResult.AppliedEffects, intent.PendingEffects...)
		// 保留中の効果をクリア
		intent.PendingEffects = nil
	}

	// WeaponType に基づく追加効果を適用 (Traitの処理から独立)
	if weaponHandler, ok := e.weaponHandlers[actingPartDef.WeaponType]; ok {
		weaponHandler.ApplyEffect(&actionResult, e.world, e.battleLogic, actingPartDef)
	}

	// アクション後の共通処理を実行
	e.postActionEffectSystem.Process(&actionResult)

	return actionResult
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
	initialResult *ActionResult, // 新しい引数
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
	initialResult *ActionResult,
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

// --- WeaponTypeEffectHandlers ---
// 以下は構想案であり、名称や効果は変更の可能性があります。
// ThunderEffectHandler はサンダー効果（チャージ停止）を付与します。
type ThunderEffectHandler struct{}

func (h *ThunderEffectHandler) ApplyEffect(result *ActionResult, world donburi.World, battleLogic *BattleLogic, actingPartDef *PartDefinition) {
	if result.ActionDidHit && result.TargetEntry != nil {
		log.Printf("%s にサンダー効果！チャージを停止させます。", result.DefenderName)
		// ActionResult.AppliedEffectsにChargeStopEffectを追加
		result.AppliedEffects = append(result.AppliedEffects, &ChargeStopEffect{DurationTurns: 1}) // 例として1ターン
	}
}

// MeltEffectHandler はメルト効果（継続ダメージ）を付与します。
type MeltEffectHandler struct{}

func (h *MeltEffectHandler) ApplyEffect(result *ActionResult, world donburi.World, battleLogic *BattleLogic, actingPartDef *PartDefinition) {
	if result.ActionDidHit && result.TargetEntry != nil {
		log.Printf("%s にメルト効果！継続ダメージを与えます。", result.DefenderName)
		// ActionResult.AppliedEffectsにDamageOverTimeEffectを追加
		result.AppliedEffects = append(result.AppliedEffects, &DamageOverTimeEffect{DamagePerTurn: 10, DurationTurns: 2}) // 例としてダメージ10、2ターン
	}
}

// VirusEffectHandler はウイルス効果（ターゲットのランダム化）を付与します。
type VirusEffectHandler struct{}

func (h *VirusEffectHandler) ApplyEffect(result *ActionResult, world donburi.World, battleLogic *BattleLogic, actingPartDef *PartDefinition) {
	if result.ActionDidHit && result.TargetEntry != nil {
		log.Printf("%s にウイルス効果！ターゲットをランダム化します。", result.DefenderName)
		// ActionResult.AppliedEffectsにTargetRandomEffectを追加
		result.AppliedEffects = append(result.AppliedEffects, &TargetRandomEffect{DurationTurns: 1}) // 例として1ターン
	}
}
