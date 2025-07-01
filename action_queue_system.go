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
	// ActionSuccess bool // ActionDidHit で代替可能か検討
}

// UpdateActionQueueSystem は行動準備完了キューを処理します。
// このシステムは BattleScene に直接依存しません。
// 戻り値として、行動結果のリストとエラーを返します。
func UpdateActionQueueSystem(
	world donburi.World,
	partInfoProvider *PartInfoProvider,
	damageCalculator *DamageCalculator, // ExecuteActionから移動したため追加
	hitCalculator *HitCalculator, // ExecuteActionから移動したため追加
	targetSelector *TargetSelector, // ExecuteActionから移動したため追加
) ([]ActionResult, error) {
	actionQueue := GetActionQueue(world)
	if len(actionQueue.Queue) == 0 {
		return nil, nil
	}

	results := []ActionResult{}

	// 行動順序を推進力に基づいてソート
	sort.SliceStable(actionQueue.Queue, func(i, j int) bool {
		if partInfoProvider == nil {
			log.Println("Warning: UpdateActionQueueSystem - partInfoProvider is nil, using default propulsion.")
			return false
		}
		propI := partInfoProvider.GetOverallPropulsion(actionQueue.Queue[i])
		propJ := partInfoProvider.GetOverallPropulsion(actionQueue.Queue[j])
		return propI > propJ
	})

	if len(actionQueue.Queue) > 0 {
		actingEntry := actionQueue.Queue[0]
		actionQueue.Queue = actionQueue.Queue[1:] // キューから取り出し

		actionResult := executeActionLogic(actingEntry, world, damageCalculator, hitCalculator, targetSelector, partInfoProvider)
		results = append(results, actionResult)

		// StartCooldown の呼び出しは BattleScene 側で行うか、ここで完了イベントを生成する
		// BattleScene (呼び出し側) で actionResult を見て判断する
	}
	return results, nil
}

// executeActionLogic は元々 systems.go の ExecuteAction にあったロジックです。
// BattleScene への依存をなくし、必要な情報を引数で受け取り、詳細な ActionResult を返します。
func executeActionLogic(
	entry *donburi.Entry,
	world donburi.World,
	damageCalculator *DamageCalculator,
	hitCalculator *HitCalculator,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) ActionResult {
	action := ActionComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	actingPart := PartsComponent.Get(entry).Map[action.SelectedPartKey]

	result := ActionResult{
		ActingEntry: entry,
	}

	var targetEntry *donburi.Entry
	var intendedTargetPart *Part

	// --- ターゲット選択 ---
	if actingPart.Category == CategoryShoot {
		targetEntry = action.TargetEntity
		if targetEntry == nil || targetEntry.HasComponent(BrokenStateComponent) {
			result.LogMessage = fmt.Sprintf("%sはターゲットを狙ったが、既に行動不能だった！", settings.Name)
			return result
		}
		intendedTargetPart = PartsComponent.Get(targetEntry).Map[action.TargetPartSlot]
		if intendedTargetPart == nil || intendedTargetPart.IsBroken {
			result.LogMessage = fmt.Sprintf("%sは%sを狙ったが、パーツは既に破壊されていた！", settings.Name, action.TargetPartSlot)
			result.TargetEntry = targetEntry // ターゲット自体は存在した
			return result
		}
	} else if actingPart.Category == CategoryMelee {
		closestEnemy := targetSelector.FindClosestEnemy(entry)
		if closestEnemy == nil {
			result.LogMessage = fmt.Sprintf("%sは攻撃しようとしたが、相手がいなかった。", settings.Name)
			return result
		}
		targetEntry = closestEnemy
		intendedTargetPart = targetSelector.SelectRandomPartToDamage(targetEntry)
		if intendedTargetPart == nil {
			result.LogMessage = fmt.Sprintf("%sは%sを狙ったが、攻撃できる部位がなかった！", settings.Name, SettingsComponent.Get(targetEntry).Name)
			result.TargetEntry = targetEntry // ターゲット自体は存在した
			return result
		}
	} else {
		result.LogMessage = fmt.Sprintf("%sは行動 '%s' に失敗した（未対応カテゴリ）。", settings.Name, actingPart.Category)
		return result
	}
	result.TargetEntry = targetEntry

	// --- 命中判定 ---
	didHit := hitCalculator.CalculateHit(entry, targetEntry, actingPart)
	result.ActionDidHit = didHit

	if !didHit {
		result.LogMessage = damageCalculator.GenerateActionLog(entry, targetEntry, nil, 0, false, false)
		if actingPart.Trait == TraitBerserk {
			ResetAllEffects(world)
		}
		return result
	}

	// --- ダメージ計算 ---
	damage, isCritical := damageCalculator.CalculateDamage(entry, actingPart)
	result.IsCritical = isCritical
	originalDamage := damage

	// --- 防御判定とダメージ適用 ---
	var finalDamageDealt int
	var actualTargetPart *Part // 実際にダメージを受けたパーツ

	defensePart := targetSelector.SelectDefensePart(targetEntry)
	// isDefended := false // result.ActionIsDefended を直接使用
	result.ActionIsDefended = false
	if defensePart != nil && defensePart != intendedTargetPart && hitCalculator.CalculateDefense(targetEntry, defensePart) {
		// 防御成功 (狙ったパーツと防御パーツが異なる場合)
		// もし狙ったパーツが防御可能な腕や頭で、それが選択された場合、それは「防御」ではなく通常の被弾として扱う
		// ここでは、防御行動として別のパーツが使われた場合を想定
		result.ActionIsDefended = true
		actualTargetPart = defensePart
		finalDamageAfterDefense := damage - defensePart.Defense
		if finalDamageAfterDefense < 0 {
			finalDamageAfterDefense = 0
		}
		finalDamageDealt = finalDamageAfterDefense
		damageCalculator.ApplyDamage(targetEntry, defensePart, finalDamageDealt)
		result.LogMessage = damageCalculator.GenerateActionLogDefense(targetEntry, defensePart, finalDamageDealt, originalDamage, isCritical)
	} else {
		// 防御失敗、防御パーツなし、または狙ったパーツ自身で受ける場合
		actualTargetPart = intendedTargetPart
		finalDamageDealt = damage
		damageCalculator.ApplyDamage(targetEntry, intendedTargetPart, finalDamageDealt)
		result.LogMessage = damageCalculator.GenerateActionLog(entry, targetEntry, intendedTargetPart, finalDamageDealt, isCritical, true)
	}
	result.DamageDealt = finalDamageDealt
	if actualTargetPart != nil { // actualTargetPart が nil になるケースは現状ないはずだが念のため
		result.TargetPartBroken = actualTargetPart.IsBroken
	}


	if actingPart.Trait == TraitBerserk {
		ResetAllEffects(world)
	}

	return result
}

// StartCooldownSystem はクールダウン状態を開始します。
// この関数は BattleScene に直接依存しません。
// systems.go から移動し、シグネチャを変更。
func StartCooldownSystem(entry *donburi.Entry, world donburi.World, gameConfig *Config) {
	actionComp := ActionComponent.Get(entry)
	partEntity := PartsComponent.Get(entry)
	var part *Part
	if partEntity != nil && partEntity.Map != nil && actionComp.SelectedPartKey != "" {
		part = partEntity.Map[actionComp.SelectedPartKey]
	}

	if part != nil && part.Trait != TraitBerserk {
		ResetAllEffects(world) // ResetAllEffects は world を引数に取る
	}

	baseSeconds := 1.0
	if part != nil {
		baseSeconds = float64(part.Cooldown)
	}
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}
	// gameConfig を使用
	totalTicks := (baseSeconds * 60.0) / gameConfig.Balance.Time.GameSpeedMultiplier

	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	if gauge.TotalDuration < 1 {
		gauge.TotalDuration = 1
	}
	gauge.ProgressCounter = 0
	gauge.CurrentGauge = 0
	ChangeState(entry, StateTypeCooldown) // ChangeState is in entity_utils.go
}

// StartCharge はチャージ状態を開始します。
// BattleScene への依存をなくし、必要な情報を引数で受け取ります。
// 元は systems.go にありましたが、行動の開始処理としてこちらに移動しました。
func StartCharge(
	entry *donburi.Entry,
	partKey PartSlotKey,
	target *donburi.Entry,
	targetPartSlot PartSlotKey,
	world donburi.World,
	gameConfig *Config,
	partInfoProvider *PartInfoProvider,
) bool {
	parts := PartsComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	part := parts.Map[partKey]

	if part == nil || part.IsBroken {
		log.Printf("%s: 選択されたパーツ %s は存在しないか破壊されています。", settings.Name, partKey)
		return false
	}

	action := ActionComponent.Get(entry)
	action.SelectedPartKey = partKey
	action.TargetEntity = target
	action.TargetPartSlot = targetPartSlot

	if part.Category == CategoryShoot {
		if target == nil || target.HasComponent(BrokenStateComponent) {
			log.Printf("%s: [SHOOT] ターゲットが存在しないか破壊されています。", settings.Name)
			return false
		}
		log.Printf("%sは%sで%sの%sを狙う！", settings.Name, part.PartName, SettingsComponent.Get(target).Name, targetPartSlot)
	} else {
		log.Printf("%sは%sで攻撃準備！", settings.Name, part.PartName)
	}

	if target != nil {
		balanceConfig := &gameConfig.Balance
		switch part.Category {
		case CategoryMelee:
			donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{
				Multiplier: balanceConfig.Effects.Melee.DefenseRateDebuff,
			})
		case CategoryShoot:
			if part.Trait == TraitAim {
				donburi.Add(target, EvasionDebuffComponent, &EvasionDebuff{
					Multiplier: balanceConfig.Effects.Aim.EvasionRateDebuff,
				})
			}
		}
		if part.Trait == TraitBerserk {
			donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{
				Multiplier: balanceConfig.Effects.Berserk.DefenseRateDebuff,
			})
			donburi.Add(target, EvasionDebuffComponent, &EvasionDebuff{
				Multiplier: balanceConfig.Effects.Berserk.EvasionRateDebuff,
			})
		}
	}

	propulsion := 1
	if partInfoProvider != nil {
		legs := parts.Map[PartSlotLegs]
		if legs != nil && !legs.IsBroken {
			propulsion = partInfoProvider.GetOverallPropulsion(entry)
		}
	} else {
		log.Println("Warning: StartCharge - partInfoProvider is nil")
	}

	baseSeconds := float64(part.Charge)
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
	ChangeState(entry, StateTypeCharging) // ChangeState is in entity_utils.go
	return true
}
