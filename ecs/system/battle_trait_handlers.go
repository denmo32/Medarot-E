package system

import (
	"log"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/ecs/component"

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
	intent *core.ActionIntent,
	damageCalculator *DamageCalculator,
	hitCalculator *HitCalculator,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	actingPartDef *core.PartDefinition,
	// randの型を *core.Rand から正しい *rand.Rand に修正しました。
	rand *rand.Rand,
) component.ActionResult {

	result := initializeAttackResult(actingEntry, actingPartDef)

	targetEntry, targetPartSlot := resolveAttackTarget(actingEntry, world, targetSelector, partInfoProvider, rand)
	if targetEntry == nil {
		return result // ターゲットが見つからない場合は、ActionDidHit: false のまま返す
	}

	result.TargetEntry = targetEntry
	result.TargetPartSlot = targetPartSlot
	result.DefenderName = component.SettingsComponent.Get(targetEntry).Name
	result.ActionDidHit = true // ターゲットが見つかったので、初期値をtrueに設定

	if !validateTarget(targetEntry, targetPartSlot) {
		result.ActionDidHit = false
		return result
	}

	didHit := performHitCheck(actingEntry, targetEntry, actingPartDef, intent.SelectedPartKey, hitCalculator)
	result.ActionDidHit = didHit
	if !didHit {
		return result
	}

	damage, isCritical := damageCalculator.CalculateDamage(actingEntry, targetEntry, actingPartDef, intent.SelectedPartKey)
	result.IsCritical = isCritical
	result.OriginalDamage = damage

	applyDamageAndDefense(&result, actingEntry, actingPartDef, intent.SelectedPartKey, damageCalculator, hitCalculator, targetSelector, partInfoProvider)

	finalizeActionResult(&result, partInfoProvider)

	return result
}

// SupportTraitExecutor は TraitSupport の介入アクションを処理します。
type SupportTraitExecutor struct{}

func (h *SupportTraitExecutor) Execute(
	actingEntry *donburi.Entry,
	world donburi.World,
	intent *core.ActionIntent,
	damageCalculator *DamageCalculator,
	hitCalculator *HitCalculator,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	actingPartDef *core.PartDefinition,
	// randの型を *core.Rand から正しい *rand.Rand に修正しました。
	rand *rand.Rand,
) component.ActionResult {
	settings := component.SettingsComponent.Get(actingEntry)
	result := component.ActionResult{
		ActingEntry:    actingEntry,
		ActionDidHit:   true,
		AttackerName:   settings.Name,
		ActionName:     actingPartDef.PartName,
		ActionTrait:    actingPartDef.Trait, // string() を削除
		ActionCategory: actingPartDef.Category,
		WeaponType:     actingPartDef.WeaponType,
	}

	teamBuffsEntry, ok := query.NewQuery(filter.Contains(component.TeamBuffsComponent)).First(world)
	if !ok {
		log.Println("エラー: TeamBuffsComponent がワールドに見つかりません。")
		result.ActionDidHit = false
		return result
	}
	teamBuffs := component.TeamBuffsComponent.Get(teamBuffsEntry)

	buffValue := 1.0 + (float64(actingPartDef.Power) / 100.0)
	newBuffSource := &component.BuffSource{
		SourceEntry: actingEntry,
		SourcePart:  intent.SelectedPartKey,
		Value:       buffValue,
	}

	teamID := settings.Team
	buffType := core.BuffTypeAccuracy

	if _, exists := teamBuffs.Buffs[teamID]; !exists {
		teamBuffs.Buffs[teamID] = make(map[core.BuffType][]*component.BuffSource)
	}
	if _, exists := teamBuffs.Buffs[teamID][buffType]; !exists {
		teamBuffs.Buffs[teamID][buffType] = make([]*component.BuffSource, 0)
	}

	existingBuffs := teamBuffs.Buffs[teamID][buffType]
	filteredBuffs := make([]*component.BuffSource, 0, len(existingBuffs))
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
	intent *core.ActionIntent,
	damageCalculator *DamageCalculator,
	hitCalculator *HitCalculator,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	actingPartDef *core.PartDefinition,
	// randの型を *core.Rand から正しい *rand.Rand に修正しました。
	rand *rand.Rand,
) component.ActionResult {
	settings := component.SettingsComponent.Get(actingEntry)
	result := component.ActionResult{
		ActingEntry:    actingEntry,
		ActionDidHit:   true,
		AttackerName:   settings.Name,
		ActionName:     actingPartDef.PartName,
		ActionTrait:    actingPartDef.Trait,
		ActionCategory: actingPartDef.Category,
		WeaponType:     actingPartDef.WeaponType,
	}
	targetComp := component.TargetComponent.Get(actingEntry)
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
	result.DefenderName = component.SettingsComponent.Get(targetEntry).Name

	log.Printf("%s が %s に妨害を実行しました（現在効果なし）。", settings.Name, result.DefenderName)
	return result
}