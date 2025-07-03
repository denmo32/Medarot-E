package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/yohamta/donburi"
)

// ActionResult はアクション実行の詳細な結果を保持します。
type ActionResult struct {
	ActingEntry      *donburi.Entry
	TargetEntry      *donburi.Entry
	TargetPartSlot   PartSlotKey // ターゲットのパーツスロット
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
	battleLogic *BattleLogic,
	gameConfig *Config,
) ([]ActionResult, error) {
	actionQueueComp := GetActionQueueComponent(world)
	if len(actionQueueComp.Queue) == 0 {
		return nil, nil
	}
	results := []ActionResult{}

	sort.SliceStable(actionQueueComp.Queue, func(i, j int) bool {
		if battleLogic == nil || battleLogic.PartInfoProvider == nil {
			log.Println("UpdateActionQueueSystem: ソート中にbattleLogicまたはpartInfoProviderがnilです")
			return false
		}
		// ソートのための推進力は脚部パーツ定義から取得する必要があります
		propI := battleLogic.PartInfoProvider.GetOverallPropulsion(actionQueueComp.Queue[i])
		propJ := battleLogic.PartInfoProvider.GetOverallPropulsion(actionQueueComp.Queue[j])
		return propI > propJ
	})

	if len(actionQueueComp.Queue) > 0 {
		actingEntry := actionQueueComp.Queue[0]
		actionQueueComp.Queue = actionQueueComp.Queue[1:]

		actionResult := executeActionLogic(actingEntry, world, battleLogic.DamageCalculator, battleLogic.HitCalculator, battleLogic.TargetSelector, battleLogic.PartInfoProvider, gameConfig)
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
		log.Printf("エラー: executeActionLogic - %s の行動パーツインスタンスがnilです。パーツキー: %s", settings.Name, action.SelectedPartKey)
		result.LogMessage = fmt.Sprintf("%sは行動パーツの取得に失敗しました。", settings.Name)
		return result
	}
	actingPartDef, defFound := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID)
	if !defFound {
		log.Printf("エラー: executeActionLogic - ID %s (エンティティ: %s) のPartDefinitionが見つかりません。", actingPartInstance.DefinitionID, settings.Name)
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

	if !handler.ResolveTarget(entry, world, action, targetSelector, partInfoProvider, &result) {
		// 失敗した場合、LogMessageとTargetEntryはハンドラによってresultに設定されています
		RemoveActionModifiersSystem(entry)
		return result
	}
	targetEntry = result.TargetEntry // ローカル変数に設定

	if targetEntry != nil && result.TargetPartSlot != "" {
		targetPartsComp := PartsComponent.Get(targetEntry)
		if targetPartsComp != nil {
			intendedTargetPartInstance = targetPartsComp.Map[result.TargetPartSlot]
			if intendedTargetPartInstance != nil {
				var tDefFound bool
				intendedTargetPartDef, tDefFound = GlobalGameDataManager.GetPartDefinition(intendedTargetPartInstance.DefinitionID)
				if !tDefFound { // このチェックはハンドラ側でも行われるべきかもしれませんが、二重チェックとして残します
					result.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、ターゲットパーツ定義(%s)が見つかりませんでした。", settings.Name, SettingsComponent.Get(targetEntry).Name, result.TargetPartSlot, intendedTargetPartInstance.DefinitionID)
					RemoveActionModifiersSystem(entry)
					return result
				}
				if intendedTargetPartInstance.IsBroken { // 同上
					result.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、パーツは既に破壊されていました。", settings.Name, SettingsComponent.Get(targetEntry).Name, result.TargetPartSlot)
					RemoveActionModifiersSystem(entry)
					return result
				}
			} else {
				result.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、ターゲットパーツインスタンスが見つかりませんでした。", settings.Name, SettingsComponent.Get(targetEntry).Name, result.TargetPartSlot)
				RemoveActionModifiersSystem(entry)
				return result
			}
		} else {
			result.LogMessage = fmt.Sprintf("%sは%sを狙ったが、ターゲットにパーツコンポーネントがありません。", settings.Name, SettingsComponent.Get(targetEntry).Name)
			RemoveActionModifiersSystem(entry)
			return result
		}
	}

	if (actingPartDef.Category == CategoryShoot || actingPartDef.Category == CategoryMelee) && targetEntry != nil && intendedTargetPartInstance == nil {
		result.LogMessage = fmt.Sprintf("%s は %s を攻撃しようとしましたが、有効な対象部位がありませんでした。", settings.Name, SettingsComponent.Get(targetEntry).Name)
		RemoveActionModifiersSystem(entry)
		return result
	}

	var didHit bool = true
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

	if actingPartDef.Category == CategoryShoot || actingPartDef.Category == CategoryMelee {
		if targetEntry == nil || intendedTargetPartInstance == nil {
			result.LogMessage = fmt.Sprintf("%s の攻撃は対象または対象パーツが不明です (内部エラー)。", settings.Name)
			RemoveActionModifiersSystem(entry)
			return result
		}

		damage, isCritical := damageCalculator.CalculateDamage(entry, actingPartDef)
		result.IsCritical = isCritical
		originalDamage := damage

		var finalDamageDealt int
		var actualHitPartInstance *PartInstanceData = intendedTargetPartInstance
		var actualHitPartSlot PartSlotKey = result.TargetPartSlot
		var actualHitPartDef *PartDefinition = intendedTargetPartDef

		result.ActionIsDefended = false
		defensePartInstance := targetSelector.SelectDefensePart(targetEntry)

		if defensePartInstance != nil && defensePartInstance != intendedTargetPartInstance {
			defensePartDef, defFound := GlobalGameDataManager.GetPartDefinition(defensePartInstance.DefinitionID)
			if defFound && hitCalculator.CalculateDefense(targetEntry, defensePartDef) {
				result.ActionIsDefended = true
				actualHitPartInstance = defensePartInstance
				actualHitPartDef = defensePartDef
				actualHitPartSlot = partInfoProvider.FindPartSlot(targetEntry, actualHitPartInstance)

				finalDamageAfterDefense := originalDamage - defensePartDef.Defense
				if finalDamageAfterDefense < 0 {
					finalDamageAfterDefense = 0
				}
				finalDamageDealt = finalDamageAfterDefense
				damageCalculator.ApplyDamage(targetEntry, actualHitPartInstance, finalDamageDealt)
				result.LogMessage = damageCalculator.GenerateActionLogDefense(targetEntry, actualHitPartDef, finalDamageDealt, originalDamage, isCritical)
			}
		}

		if !result.ActionIsDefended {
			actualHitPartInstance = intendedTargetPartInstance
			actualHitPartDef = intendedTargetPartDef
			actualHitPartSlot = result.TargetPartSlot
			damageCalculator.ApplyDamage(targetEntry, actualHitPartInstance, originalDamage)
			finalDamageDealt = originalDamage
			result.LogMessage = damageCalculator.GenerateActionLog(entry, targetEntry, actualHitPartDef, finalDamageDealt, isCritical, true)
		}
		result.DamageDealt = finalDamageDealt
		result.TargetPartBroken = actualHitPartInstance.IsBroken
		if result.TargetPartBroken {
			if result.ActionIsDefended {
				if !strings.Contains(result.LogMessage, "しかし、パーツは破壊された！") {
					result.LogMessage += " しかし、パーツは破壊された！"
				}
			} else {
				if !strings.Contains(result.LogMessage, "パーツを破壊した！") {
					result.LogMessage += " パーツを破壊した！"
				}
			}
		}

		// --- 履歴コンポーネントの更新 ---
		if targetEntry.HasComponent(TargetHistoryComponent) {
			targetHistory := TargetHistoryComponent.Get(targetEntry)
			targetHistory.LastAttacker = entry
			log.Printf("履歴更新: %s の LastAttacker を %s に設定", SettingsComponent.Get(targetEntry).Name, settings.Name)
		}
		if entry.HasComponent(LastActionHistoryComponent) {
			lastActionHistory := LastActionHistoryComponent.Get(entry)
			lastActionHistory.LastHitTarget = targetEntry
			lastActionHistory.LastHitPartSlot = actualHitPartSlot
			log.Printf("履歴更新: %s の LastHit を %s の %s に設定", settings.Name, SettingsComponent.Get(targetEntry).Name, actualHitPartSlot)
		}
	} else {
		if result.LogMessage == "" {
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
			log.Printf("エラー: StartCooldownSystem - ID %s のPartDefinitionが見つかりません。", actingPartInstance.DefinitionID)
		}
	} else {
		log.Printf("エラー: StartCooldownSystem - キー %s の行動パーツインスタンスが見つかりません。", actionComp.SelectedPartKey)
	}

	if actingPartDef != nil && actingPartDef.Trait != TraitBerserk {
		ResetAllEffects(world)
	}

	baseSeconds := 1.0
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

	switch actingPartDef.Trait {
	case TraitBerserk:
		donburi.Add(entry, ActingWithBerserkTraitTagComponent, &ActingWithBerserkTraitTag{})
		log.Printf("%s の行動にBERSERK特性タグを付与。", settings.Name)
	case TraitAim:
		donburi.Add(entry, ActingWithAimTraitTagComponent, &ActingWithAimTraitTag{})
		log.Printf("%s の行動にAIM特性タグを付与。", settings.Name)
	}

	if actingPartDef.Category == CategoryShoot {
		if target == nil || StateComponent.Get(target).Current == StateTypeBroken {
			log.Printf("%s: [射撃] ターゲットが存在しないか破壊されています。", settings.Name)
			if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
				entry.RemoveComponent(ActingWithBerserkTraitTagComponent)
			}
			if entry.HasComponent(ActingWithAimTraitTagComponent) {
				entry.RemoveComponent(ActingWithAimTraitTagComponent)
			}
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
			log.Printf("%s が格闘カテゴリ効果（チャージ時デバフ）を発動。", settings.Name)
			donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{Multiplier: balanceConfig.Effects.Melee.DefenseRateDebuff})
		}
	}

	propulsion := 1
	if partInfoProvider != nil {
		legsInstance := partsComp.Map[PartSlotLegs]
		if legsInstance != nil && !legsInstance.IsBroken {
			propulsion = partInfoProvider.GetOverallPropulsion(entry)
		}
	} else {
		log.Println("警告: StartCharge - partInfoProviderがnilです。")
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
