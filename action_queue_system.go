package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/yohamta/donburi"
)

// ActionResult holds the detailed result of an action execution.
type ActionResult struct {
	ActingEntry      *donburi.Entry
	TargetEntry      *donburi.Entry
	LogMessage       string
	ActionDidHit     bool // 命中したかどうか
	IsCritical       bool // クリティカルだったか
	DamageDealt      int  // 実際に与えたダメージ
	TargetPartBroken bool // ターゲットパーツが破壊されたか
	ActionIsDefended bool // 攻撃が防御されたか
}

// UpdateActionQueueSystem は行動準備完了キューを処理します。
func UpdateActionQueueSystem(
	world donburi.World,
	partInfoProvider *PartInfoProvider,
	damageCalculator *DamageCalculator,
	hitCalculator *HitCalculator,
	targetSelector *TargetSelector,
	gameConfig *Config,
) ([]ActionResult, error) {
	actionQueueComp := GetActionQueueComponent(world)
	if len(actionQueueComp.Queue) == 0 {
		return nil, nil
	}

	results := []ActionResult{}

	sort.SliceStable(actionQueueComp.Queue, func(i, j int) bool {
		if partInfoProvider == nil { // Should not happen in normal flow
			log.Println("UpdateActionQueueSystem: partInfoProvider is nil during sort")
			return false
		}
		// Propulsion for sorting should come from leg part definition
		propI := partInfoProvider.GetOverallPropulsion(actionQueueComp.Queue[i])
		propJ := partInfoProvider.GetOverallPropulsion(actionQueueComp.Queue[j])
		return propI > propJ
	})

	if len(actionQueueComp.Queue) > 0 {
		actingEntry := actionQueueComp.Queue[0]
		actionQueueComp.Queue = actionQueueComp.Queue[1:]

		actionResult := executeActionLogic(actingEntry, world, damageCalculator, hitCalculator, targetSelector, partInfoProvider, gameConfig)
		results = append(results, actionResult)
	}
	return results, nil
}

func executeActionLogic(
	entry *donburi.Entry,
	world donburi.World,
	damageCalculator *DamageCalculator,
	hitCalculator *HitCalculator,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
	gameConfig *Config,
) ActionResult {
	action := ActionComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	partsComp := PartsComponent.Get(entry)
	actingPartInstance := partsComp.Map[action.SelectedPartKey]

	result := ActionResult{ActingEntry: entry}

	if actingPartInstance == nil {
		log.Printf("Error: executeActionLogic - actingPartInstance is nil for %s, part key %s", settings.Name, action.SelectedPartKey)
		result.LogMessage = fmt.Sprintf("%sは行動パーツの取得に失敗しました。", settings.Name)
		return result
	}
	actingPartDef, defFound := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID)
	if !defFound {
		log.Printf("Error: executeActionLogic - PartDefinition not found for ID %s (entity: %s)", actingPartInstance.DefinitionID, settings.Name)
		result.LogMessage = fmt.Sprintf("%sはパーツ定義(%s)の取得に失敗しました。", settings.Name, actingPartInstance.DefinitionID)
		return result
	}

	ApplyActionModifiersSystem(world, entry, gameConfig, partInfoProvider)

	var targetEntry *donburi.Entry
	var intendedTargetPartInstance *PartInstanceData
	var intendedTargetPartDef *PartDefinition

	handler := GetActionHandlerForCategory(actingPartDef.Category)
	if handler == nil {
		result.LogMessage = fmt.Sprintf("%sのパーツカテゴリ '%s' に対応する行動ハンドラがありません。", settings.Name, actingPartDef.Category)
		RemoveActionModifiersSystem(entry)
		return result
	}

	targetingResult := handler.ResolveTarget(entry, world, action, targetSelector, partInfoProvider)
	if !targetingResult.Success {
		result.LogMessage = targetingResult.LogMessage
		result.TargetEntry = targetingResult.TargetEntity // Handler might set target even on failure
		RemoveActionModifiersSystem(entry)
		return result
	}
	targetEntry = targetingResult.TargetEntity
	result.TargetEntry = targetEntry

	if targetEntry != nil && targetingResult.TargetPartSlot != "" {
		targetPartsComp := PartsComponent.Get(targetEntry)
		if targetPartsComp != nil {
			intendedTargetPartInstance = targetPartsComp.Map[targetingResult.TargetPartSlot]
			if intendedTargetPartInstance != nil {
				var tDefFound bool
				intendedTargetPartDef, tDefFound = GlobalGameDataManager.GetPartDefinition(intendedTargetPartInstance.DefinitionID)
				if !tDefFound {
					result.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、ターゲットパーツ定義(%s)が見つかりませんでした。", settings.Name, SettingsComponent.Get(targetEntry).Name, targetingResult.TargetPartSlot, intendedTargetPartInstance.DefinitionID)
					RemoveActionModifiersSystem(entry)
					return result
				}
				if intendedTargetPartInstance.IsBroken {
					result.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、パーツは既に破壊されていました。", settings.Name, SettingsComponent.Get(targetEntry).Name, targetingResult.TargetPartSlot)
					RemoveActionModifiersSystem(entry)
					return result
				}
			} else {
				result.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、ターゲットパーツインスタンスが見つかりませんでした。", settings.Name, SettingsComponent.Get(targetEntry).Name, targetingResult.TargetPartSlot)
				RemoveActionModifiersSystem(entry)
				return result
			}
		} else { // Target has no PartsComponent
			result.LogMessage = fmt.Sprintf("%sは%sを狙ったが、ターゲットにパーツコンポーネントがありません。", settings.Name, SettingsComponent.Get(targetEntry).Name)
			RemoveActionModifiersSystem(entry)
			return result
		}
	}

	// If it's an offensive action but no valid target part instance was resolved (e.g. target has no parts, or melee couldn't pick one)
    if (actingPartDef.Category == CategoryShoot || actingPartDef.Category == CategoryMelee) && targetEntry != nil && intendedTargetPartInstance == nil {
        result.LogMessage = fmt.Sprintf("%s は %s を攻撃しようとしましたが、有効な対象部位がありませんでした。", settings.Name, SettingsComponent.Get(targetEntry).Name)
        RemoveActionModifiersSystem(entry)
        return result
    }


	var didHit bool = true // Assume hit for non-targeted/non-offensive actions
	if targetEntry != nil && (actingPartDef.Category == CategoryShoot || actingPartDef.Category == CategoryMelee) {
		didHit = hitCalculator.CalculateHit(entry, targetEntry, actingPartDef)
	}
	result.ActionDidHit = didHit

	if !didHit {
		result.LogMessage = damageCalculator.GenerateActionLog(entry, targetEntry, nil, 0, false, false)
		if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
			ResetAllEffects(world)
		}
		RemoveActionModifiersSystem(entry)
		return result
	}

	// Proceed only if it's an offensive action that hit, or a non-offensive action
	if actingPartDef.Category == CategoryShoot || actingPartDef.Category == CategoryMelee {
		if targetEntry == nil || intendedTargetPartInstance == nil {
			// This should ideally be caught by earlier checks if it's an offensive action
			result.LogMessage = fmt.Sprintf("%s の攻撃は対象または対象パーツが不明です (内部エラー)。", settings.Name)
			RemoveActionModifiersSystem(entry)
			return result
		}

		damage, isCritical := damageCalculator.CalculateDamage(entry, actingPartDef)
		result.IsCritical = isCritical
		// originalDamage := damage // For defense log, if re-enabled

		// --- Simplified Damage Application (Defense Logic Bypassed for now) ---
		result.ActionIsDefended = false
		damageCalculator.ApplyDamage(targetEntry, intendedTargetPartInstance, damage)
		result.DamageDealt = damage // Since no defense, damage dealt is full pre-defense damage
		result.TargetPartBroken = intendedTargetPartInstance.IsBroken
		result.LogMessage = damageCalculator.GenerateActionLog(entry, targetEntry, intendedTargetPartDef, result.DamageDealt, isCritical, true)
		if result.TargetPartBroken {
			result.LogMessage += " パーツを破壊した！"
		}

	} else { // Non-offensive actions (e.g., SUPPORT, DEFENSE if they had handlers)
		if result.LogMessage == "" { // If handler didn't set a log (e.g. for future SUPPORT/DEFENSE actions)
			result.LogMessage = fmt.Sprintf("%s は %s を実行した。", settings.Name, actingPartDef.PartName)
		}
	}

	if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
		log.Printf("%s がBERSERK特性効果（行動後全効果リセット）を発動。", settings.Name)
		ResetAllEffects(world)
	}

	if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
		entry.RemoveComponent(ActingWithBerserkTraitTagComponent)
	}
	if entry.HasComponent(ActingWithAimTraitTagComponent) {
		entry.RemoveComponent(ActingWithAimTraitTagComponent)
	}
	RemoveActionModifiersSystem(entry)
	return result
}

// StartCooldownSystem はクールダウン状態を開始します。
func StartCooldownSystem(entry *donburi.Entry, world donburi.World, gameConfig *Config) {
	actionComp := ActionComponent.Get(entry)
	partsComp := PartsComponent.Get(entry)
	var actingPartDef *PartDefinition

	if actingPartInstance, ok := partsComp.Map[actionComp.SelectedPartKey]; ok {
		if def, defFound := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID); defFound {
			actingPartDef = def
		} else {
			log.Printf("Error: StartCooldownSystem - PartDefinition not found for ID %s", actingPartInstance.DefinitionID)
		}
	} else {
		log.Printf("Error: StartCooldownSystem - actingPartInstance not found for key %s", actionComp.SelectedPartKey)
	}

	// Reset effects if not Berserk (based on part definition's trait)
	if actingPartDef != nil && actingPartDef.Trait != TraitBerserk {
		ResetAllEffects(world)
	}

	baseSeconds := 1.0 // Default cooldown
	if actingPartDef != nil {
		baseSeconds = float64(actingPartDef.Cooldown)
	}
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}
	totalTicks := (baseSeconds * 60.0) / gameConfig.Balance.Time.GameSpeedMultiplier

	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	if gauge.TotalDuration < 1 {
		gauge.TotalDuration = 1
	}
	gauge.ProgressCounter = 0
	gauge.CurrentGauge = 0

	// Trait tags and ActionModifierComponent should have been removed by executeActionLogic's end.
	// Adding a safeguard removal here as well.
	if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
		entry.RemoveComponent(ActingWithBerserkTraitTagComponent)
	}
	if entry.HasComponent(ActingWithAimTraitTagComponent) {
		entry.RemoveComponent(ActingWithAimTraitTagComponent)
	}
	RemoveActionModifiersSystem(entry)

	ChangeState(entry, StateTypeCooldown)
}

// StartCharge はチャージ状態を開始します。
func StartCharge(
	entry *donburi.Entry,
	partKey PartSlotKey,
	target *donburi.Entry,
	targetPartSlot PartSlotKey,
	world donburi.World,
	gameConfig *Config,
	partInfoProvider *PartInfoProvider,
) bool {
	partsComp := PartsComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	actingPartInstance := partsComp.Map[partKey]

	if actingPartInstance == nil || actingPartInstance.IsBroken {
		log.Printf("%s: 選択されたパーツ %s (%s) は存在しないか破壊されています。", settings.Name, partKey, actingPartInstance.DefinitionID)
		return false
	}
	actingPartDef, defFound := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID)
	if !defFound {
		log.Printf("%s: パーツ定義(%s)が見つかりません。", settings.Name, actingPartInstance.DefinitionID)
		return false
	}

	action := ActionComponent.Get(entry)
	action.SelectedPartKey = partKey
	action.TargetEntity = target
	action.TargetPartSlot = targetPartSlot

	// Add Trait tags first
	switch actingPartDef.Trait {
	case TraitBerserk:
		donburi.Add(entry, ActingWithBerserkTraitTagComponent, &ActingWithBerserkTraitTag{})
		log.Printf("%s の行動にBERSERK特性タグを付与。", settings.Name)
	case TraitAim:
		donburi.Add(entry, ActingWithAimTraitTagComponent, &ActingWithAimTraitTag{})
		log.Printf("%s の行動にAIM特性タグを付与。", settings.Name)
	}

	if actingPartDef.Category == CategoryShoot {
		if target == nil || target.HasComponent(BrokenStateComponent) {
			log.Printf("%s: [SHOOT] ターゲットが存在しないか破壊されています。", settings.Name)
			if entry.HasComponent(ActingWithBerserkTraitTagComponent) { entry.RemoveComponent(ActingWithBerserkTraitTagComponent) }
			if entry.HasComponent(ActingWithAimTraitTagComponent) { entry.RemoveComponent(ActingWithAimTraitTagComponent) }
			return false
		}
		log.Printf("%sは%sで%sの%sを狙う！", settings.Name, actingPartDef.PartName, SettingsComponent.Get(target).Name, targetPartSlot)
	} else {
		log.Printf("%sは%sで攻撃準備！", settings.Name, actingPartDef.PartName)
	}

	if target != nil {
		balanceConfig := &gameConfig.Balance
		if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
			log.Printf("%s がBERSERK特性効果（チャージ時デバフ）を発動。", settings.Name)
			donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{Multiplier: balanceConfig.Effects.Berserk.DefenseRateDebuff})
			donburi.Add(target, EvasionDebuffComponent, &EvasionDebuff{Multiplier: balanceConfig.Effects.Berserk.EvasionRateDebuff})
		}
		if actingPartDef.Category == CategoryShoot && entry.HasComponent(ActingWithAimTraitTagComponent) {
			log.Printf("%s がAIM特性効果（チャージ時デバフ）を発動。", settings.Name)
			donburi.Add(target, EvasionDebuffComponent, &EvasionDebuff{Multiplier: balanceConfig.Effects.Aim.EvasionRateDebuff})
		}
		if actingPartDef.Category == CategoryMelee {
			log.Printf("%s がMELEEカテゴリ効果（チャージ時デバフ）を発動。", settings.Name)
			donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{Multiplier: balanceConfig.Effects.Melee.DefenseRateDebuff})
		}
	}

	propulsion := 1
	if partInfoProvider != nil {
		legsInstance := partsComp.Map[PartSlotLegs]
		if legsInstance != nil && !legsInstance.IsBroken {
			propulsion = partInfoProvider.GetOverallPropulsion(entry) // This already uses definition via GameDataManager
		}
	} else {
		log.Println("Warning: StartCharge - partInfoProvider is nil")
	}

	baseSeconds := float64(actingPartDef.Charge)
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}
	balanceConfig := &gameConfig.Balance
	propulsionFactor := 1.0 + (float64(propulsion) * balanceConfig.Time.PropulsionEffectRate)
	totalTicks := (baseSeconds * 60.0) / (balanceConfig.Time.GameSpeedMultiplier * propulsionFactor)

	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	if gauge.TotalDuration < 1 {
		gauge.TotalDuration = 1
	}
	ChangeState(entry, StateTypeCharging)
	return true
}
