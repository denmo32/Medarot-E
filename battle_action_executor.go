package main

import (
	"context"
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
	world             donburi.World
	battleLogic       *BattleLogic
	gameConfig        *Config
	handlers          map[Trait]TraitActionHandler
	weaponHandlers    map[WeaponType]WeaponTypeEffectHandler // WeaponTypeごとのハンドラを追加
	baseAttackHandler *BaseAttackHandler
}

// --- BaseAttackHandler ---

// BaseAttackHandler は、すべての攻撃アクションに共通するロジックをカプセル化します。
type BaseAttackHandler struct{}

// PerformAttack は、ターゲットの解決、命中判定、ダメージ計算、防御処理などの共通攻撃ロジックを実行します。
func (h *BaseAttackHandler) PerformAttack(
	actingEntry *donburi.Entry,
	intent *ActionIntent,
	battleLogic *BattleLogic, // battleLogic を引数に追加
	gameConfig *Config, // gameConfig を引数に追加
	actingPartDef *PartDefinition, // actingPartDef を引数に追加
) ActionResult {
	targetEntry, targetPartSlot := resolveAttackTarget(actingEntry, battleLogic)

	// ターゲットが解決できなかった場合
	if targetEntry == nil {
		return ActionResult{
			ActingEntry:    actingEntry,
			ActionDidHit:   false,
			AttackerName:   SettingsComponent.Get(actingEntry).Name,
			ActionName:     actingPartDef.PartName,
			ActionTrait:    actingPartDef.Trait, // string() を削除
			ActionCategory: actingPartDef.Category,
			WeaponType:     actingPartDef.WeaponType,
		}
	}

	result := ActionResult{
		ActingEntry:    actingEntry,
		TargetEntry:    targetEntry,
		TargetPartSlot: targetPartSlot,
		ActionDidHit:   true, // 初期値
		AttackerName:   SettingsComponent.Get(actingEntry).Name,
		DefenderName:   SettingsComponent.Get(targetEntry).Name,
		ActionName:     actingPartDef.PartName,
		ActionTrait:    actingPartDef.Trait, // string() を削除
		ActionCategory: actingPartDef.Category,
		WeaponType:     actingPartDef.WeaponType,
	}

	if !validateTarget(targetEntry, targetPartSlot) {
		result.ActionDidHit = false
		return result
	}

	didHit := performHitCheck(actingEntry, targetEntry, actingPartDef, battleLogic) // h.battleLogic を battleLogic に変更
	result.ActionDidHit = didHit
	if !didHit {
		return result
	}

	damage, isCritical := battleLogic.DamageCalculator.CalculateDamage(actingEntry, targetEntry, actingPartDef) // h.battleLogic を battleLogic に変更
	result.IsCritical = isCritical
	result.OriginalDamage = damage

	applyDamageAndDefense(&result, actingEntry, actingPartDef, battleLogic) // h.battleLogic を battleLogic に変更

	finalizeActionResult(&result)

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
	return battleLogic.HitCalculator.CalculateHit(actingEntry, targetEntry, actingPartDef)
}

func applyDamageAndDefense(
	result *ActionResult,
	actingEntry *donburi.Entry,
	actingPartDef *PartDefinition,
	battleLogic *BattleLogic,
) {
	defendingPartInst := battleLogic.TargetSelector.SelectDefensePart(result.TargetEntry)

	if defendingPartInst != nil && battleLogic.HitCalculator.CalculateDefense(actingEntry, result.TargetEntry, actingPartDef) {
		result.ActionIsDefended = true
		defendingPartDef, _ := GlobalGameDataManager.GetPartDefinition(defendingPartInst.DefinitionID)
		result.DefendingPartType = string(defendingPartDef.Type)
		result.ActualHitPartSlot = battleLogic.PartInfoProvider.FindPartSlot(result.TargetEntry, defendingPartInst)

		finalDamage := battleLogic.DamageCalculator.CalculateReducedDamage(result.OriginalDamage, result.TargetEntry)
		result.DamageDealt = finalDamage
		battleLogic.DamageCalculator.ApplyDamage(result.TargetEntry, defendingPartInst, finalDamage)
		result.TargetPartBroken = defendingPartInst.IsBroken
	} else {
		result.ActionIsDefended = false
		intendedTargetPartInstance := PartsComponent.Get(result.TargetEntry).Map[result.TargetPartSlot]
		result.DamageDealt = result.OriginalDamage
		result.ActualHitPartSlot = result.TargetPartSlot

		battleLogic.DamageCalculator.ApplyDamage(result.TargetEntry, intendedTargetPartInstance, result.OriginalDamage)
		result.TargetPartBroken = intendedTargetPartInstance.IsBroken
	}
}

func finalizeActionResult(result *ActionResult) {
	actualHitPartInst := PartsComponent.Get(result.TargetEntry).Map[result.ActualHitPartSlot]
	actualHitPartDef, _ := GlobalGameDataManager.GetPartDefinition(actualHitPartInst.DefinitionID)

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
		if targetComp.TargetEntity == nil {
			log.Printf("エラー: PolicyPreselected なのにターゲットが設定されていません。")
			return nil, ""
		}
		return targetComp.TargetEntity, targetComp.TargetPartSlot
	case PolicyClosestAtExecution:
		closestEnemy := battleLogic.TargetSelector.FindClosestEnemy(actingEntry)
		if closestEnemy == nil {
			return nil, "" // ターゲットが見つからない場合は失敗
		}
		targetPart := battleLogic.TargetSelector.SelectPartToDamage(closestEnemy, actingEntry)
		if targetPart == nil {
			return nil, "" // ターゲットパーツが見つからない場合は失敗
		}
		slot := battleLogic.PartInfoProvider.FindPartSlot(closestEnemy, targetPart)
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
	return &ActionExecutor{
		world:             world,
		battleLogic:       battleLogic,
		gameConfig:        gameConfig,
		baseAttackHandler: &BaseAttackHandler{},
		handlers: map[Trait]TraitActionHandler{
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
	actingPartDef, _ := GlobalGameDataManager.GetPartDefinition(actingPartInst.DefinitionID)

	var actionResult ActionResult
	isAttackTrait := actingPartDef.Trait == TraitShoot || actingPartDef.Trait == TraitAim || actingPartDef.Trait == TraitStrike || actingPartDef.Trait == TraitBerserk

	if isAttackTrait {
		// 攻撃系TraitはBaseAttackHandlerで直接処理
		actionResult = e.baseAttackHandler.PerformAttack(actingEntry, intent, e.battleLogic, e.gameConfig, actingPartDef)
	} else {
		// その他のTraitは専用ハンドラで処理
		handler, ok := e.handlers[actingPartDef.Trait]
		if !ok {
			log.Printf("未対応のTraitです: %s", actingPartDef.Trait)
			return ActionResult{
				ActingEntry:  actingEntry,
				ActionDidHit: false,
			}
		}
		actionResult = handler.Execute(actingEntry, e.world, intent, e.battleLogic, e.gameConfig, actingPartDef, nil)
	}

	// WeaponType に基づく追加効果を適用 (Traitの処理から独立)
	if weaponHandler, ok := e.weaponHandlers[actingPartDef.WeaponType]; ok {
		weaponHandler.ApplyEffect(&actionResult, e.world, e.battleLogic, actingPartDef)
	}

	// アクション後の共通処理を実行
	e.processPostActionEffects(&actionResult)

	return actionResult
}

// processPostActionEffects は、アクション実行後の共通処理（パーツ破壊、デバフ解除など）を適用します。
func (e *ActionExecutor) processPostActionEffects(result *ActionResult) {
	if result == nil {
		return
	}

	// 1. パーツ破壊による状態遷移
	if result.TargetEntry != nil && result.TargetPartBroken && result.ActualHitPartSlot == PartSlotHead {
		state := StateComponent.Get(result.TargetEntry)
		if state.FSM.Can("break") {
			err := state.FSM.Event(context.Background(), "break", result.TargetEntry)
			if err != nil {
				log.Printf("Error breaking medarot %s: %v", SettingsComponent.Get(result.TargetEntry).Name, err)
			}
		}
	}

	// 2. 行動後のデバフクリーンアップ
	if result.ActingEntry != nil {
		if result.ActingEntry.HasComponent(EvasionDebuffComponent) {
			result.ActingEntry.RemoveComponent(EvasionDebuffComponent)
		}
		if result.ActingEntry.HasComponent(DefenseDebuffComponent) {
			result.ActingEntry.RemoveComponent(DefenseDebuffComponent)
		}
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
	if targetComp.TargetEntity == nil {
		log.Printf("%s は妨害ターゲットが未選択です。", settings.Name)
		result.ActionDidHit = false
		return result
	}
	targetEntry := targetComp.TargetEntity
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
		// ここに「チャージ停止」のデバフを付与する具体的なロジックを追加
		// 例: ApplyDebuff(result.TargetEntry, DebuffTypeChargeStop, someDuration)
	}
}

// MeltEffectHandler はメルト効果（継続ダメージ）を付与します。
type MeltEffectHandler struct{}

func (h *MeltEffectHandler) ApplyEffect(result *ActionResult, world donburi.World, battleLogic *BattleLogic, actingPartDef *PartDefinition) {
	if result.ActionDidHit && result.TargetEntry != nil {
		log.Printf("%s にメルト効果！継続ダメージを与えます。", result.DefenderName)
		// ここに「継続ダメージ」のデバフを付与する具体的なロジックを追加
		// 例: ApplyDebuff(result.TargetEntry, DebuffTypeDamageOverTime, someDamageAmount, someDuration)
	}
}

// VirusEffectHandler はウイルス効果（ターゲットのランダム化）を付与します。
type VirusEffectHandler struct{}

func (h *VirusEffectHandler) ApplyEffect(result *ActionResult, world donburi.World, battleLogic *BattleLogic, actingPartDef *PartDefinition) {
	if result.ActionDidHit && result.TargetEntry != nil {
		log.Printf("%s にウイルス効果！ターゲットをランダム化します。", result.DefenderName)
		// ここに「ターゲットのランダム化」のデバフを付与する具体的なロジックを追加
		// 例: ApplyDebuff(result.TargetEntry, DebuffTypeTargetRandom, someDuration)
	}
}
