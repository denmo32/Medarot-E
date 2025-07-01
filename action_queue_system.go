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
	gameConfig *Config, // Added to pass to executeActionLogic
) ([]ActionResult, error) {
	actionQueueComp := GetActionQueueComponent(world)
	if len(actionQueueComp.Queue) == 0 {
		return nil, nil
	}

	results := []ActionResult{}

	// 行動順序を推進力に基づいてソート
	sort.SliceStable(actionQueueComp.Queue, func(i, j int) bool {
		if partInfoProvider == nil {
			log.Println("Warning: UpdateActionQueueSystem - partInfoProvider is nil, using default propulsion.")
			return false
		}
		propI := partInfoProvider.GetOverallPropulsion(actionQueueComp.Queue[i])
		propJ := partInfoProvider.GetOverallPropulsion(actionQueueComp.Queue[j])
		return propI > propJ
	})

	if len(actionQueueComp.Queue) > 0 {
		actingEntry := actionQueueComp.Queue[0]
		actionQueueComp.Queue = actionQueueComp.Queue[1:] // キューから取り出し

		actionResult := executeActionLogic(actingEntry, world, damageCalculator, hitCalculator, targetSelector, partInfoProvider, gameConfig)
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
	gameConfig *Config, // Added to pass to ApplyActionModifiersSystem
) ActionResult {
	action := ActionComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	actingPart := PartsComponent.Get(entry).Map[action.SelectedPartKey]

	// Apply action modifiers before any calculations
	ApplyActionModifiersSystem(world, entry, gameConfig, partInfoProvider)

	result := ActionResult{
		ActingEntry: entry,
	}

	var targetEntry *donburi.Entry
	// var intendedTargetPart *Part // This will be determined by the handler or from its result

	// --- Target Resolution using ActionHandler ---
	handler := GetActionHandlerForCategory(actingPart.Category)
	if handler == nil {
		result.LogMessage = fmt.Sprintf("%sは行動カテゴリ '%s' の処理に失敗した（対応ハンドラなし）。", settings.Name, actingPart.Category)
		RemoveActionModifiersSystem(entry) // Clean up modifiers if action fails early
		return result
	}

	targetingResult := handler.ResolveTarget(entry, world, action, targetSelector, partInfoProvider)
	if !targetingResult.Success {
		result.LogMessage = targetingResult.LogMessage
		// If ResolveTarget provides a target entity even on failure (e.g. part broken on valid entity), set it.
		result.TargetEntry = targetingResult.TargetEntity
		RemoveActionModifiersSystem(entry) // Clean up modifiers
		return result
	}

	targetEntry = targetingResult.TargetEntity
	// intendedTargetPart is implicitly defined by targetingResult.TargetPartSlot for SHOOT,
	// or resolved to a specific Part for MELEE by the handler.
	// For damage application, we'll need the actual Part struct.
	var intendedTargetPart *Part
	if targetEntry != nil && targetingResult.TargetPartSlot != "" {
		intendedTargetPart = PartsComponent.Get(targetEntry).Map[targetingResult.TargetPartSlot]
		if intendedTargetPart == nil || intendedTargetPart.IsBroken { // Double check after handler
			result.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、パーツは存在しないか既に破壊されていた(Handler後チェック)。", settings.Name, SettingsComponent.Get(targetEntry).Name, targetingResult.TargetPartSlot)
			result.TargetEntry = targetEntry
			RemoveActionModifiersSystem(entry)
			return result
		}
	} else if actingPart.Category != CategoryShoot && targetEntry != nil {
		// For non-shoot actions that might not use TargetPartSlot directly from handler in this specific way,
		// ensure intendedTargetPart is set if applicable (e.g. Melee handler might have picked one).
		// This part might need refinement based on how handlers for other categories (SUPPORT, DEFENSE) work.
		// For Melee, the handler's LogMessage implies a part was chosen. We need to ensure 'intendedTargetPart' is the one.
		// The current MeleeActionHandler.ResolveTarget returns a valid TargetPartSlot for a valid Part.
		// So, the above block should cover it.
	}


	if targetEntry == nil && (actingPart.Category == CategoryShoot || actingPart.Category == CategoryMelee) {
		// If it's an attack category but no target was resolved by the handler (e.g. Melee found no enemies)
		result.LogMessage = targetingResult.LogMessage // Use handler's log
		if result.LogMessage == "" { // Fallback if handler didn't set one
			result.LogMessage = fmt.Sprintf("%sは攻撃対象を見つけられませんでした。", settings.Name)
		}
		RemoveActionModifiersSystem(entry)
		return result
	}
	result.TargetEntry = targetEntry


	// --- 命中判定 ---
	// Note: For actions that don't require a target (e.g. self-buffs), targetEntry might be nil.
	// Hit calculation should only proceed if there's a target and it's an offensive action.
	// This logic might need to be part of the ActionHandler or an earlier check.
	var didHit bool = true // Assume hit for non-targeted actions or if no hit calc is needed
	if targetEntry != nil && (actingPart.Category == CategoryShoot || actingPart.Category == CategoryMelee) {
		didHit = hitCalculator.CalculateHit(entry, targetEntry, actingPart)
	}
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

	// Trait-specific effects post-action
	if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
		log.Printf("%s がBERSERK特性効果（行動後全効果リセット）を発動。", settings.Name)
		ResetAllEffects(world)
	}
	// Add other post-action trait effects here if any

	// Ensure trait tags are removed after action execution, regardless of cooldown.
	// This is a safeguard, primary removal is in StartCooldownSystem.
	if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
		entry.RemoveComponent(ActingWithBerserkTraitTagComponent)
		// log.Printf("%s のBERSERK特性タグをexecuteActionLogicで解除(念のため)。", SettingsComponent.Get(entry).Name)
	}
	if entry.HasComponent(ActingWithAimTraitTagComponent) {
		entry.RemoveComponent(ActingWithAimTraitTagComponent)
		// log.Printf("%s のAIM特性タグをexecuteActionLogi で解除(念のため)。", SettingsComponent.Get(entry).Name)
	}

	RemoveActionModifiersSystem(entry) // Remove modifiers after action is fully resolved

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

	// Remove Trait tags
	if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
		entry.RemoveComponent(ActingWithBerserkTraitTagComponent)
		log.Printf("%s のBERSERK特性タグを解除。", SettingsComponent.Get(entry).Name)
	}
	if entry.HasComponent(ActingWithAimTraitTagComponent) {
		entry.RemoveComponent(ActingWithAimTraitTagComponent)
		log.Printf("%s のAIM特性タグを解除。", SettingsComponent.Get(entry).Name)
	}
	RemoveActionModifiersSystem(entry) // Ensure modifiers are removed when cooldown starts

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

	// Apply debuffs based on traits and category at the start of charge
	if target != nil {
		balanceConfig := &gameConfig.Balance
		// BERSERK trait effect (on charge start)
		if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
			log.Printf("%s がBERSERK特性効果（チャージ時デバフ）を発動。", settings.Name)
			donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{
				Multiplier: balanceConfig.Effects.Berserk.DefenseRateDebuff,
			})
			donburi.Add(target, EvasionDebuffComponent, &EvasionDebuff{
				Multiplier: balanceConfig.Effects.Berserk.EvasionRateDebuff,
			})
		}

		// AIM trait effect (on charge start, for SHOOT category)
		if part.Category == CategoryShoot && entry.HasComponent(ActingWithAimTraitTagComponent) {
			log.Printf("%s がAIM特性効果（チャージ時デバフ）を発動。", settings.Name)
			// Note: If Berserk also adds EvasionDebuff, this might overwrite or be redundant.
			// Assuming donburi.Add overwrites if component already exists.
			donburi.Add(target, EvasionDebuffComponent, &EvasionDebuff{
				Multiplier: balanceConfig.Effects.Aim.EvasionRateDebuff,
			})
		}

		// MELEE category specific debuff (separate from traits)
		if part.Category == CategoryMelee {
			log.Printf("%s がMELEEカテゴリ効果（チャージ時デバフ）を発動。", settings.Name)
			// If Berserk also adds DefenseDebuff, this might overwrite.
			donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{
				Multiplier: balanceConfig.Effects.Melee.DefenseRateDebuff,
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

	// Add Trait tags based on the part used
	switch part.Trait {
	case TraitBerserk:
		donburi.Add(entry, ActingWithBerserkTraitTagComponent, &ActingWithBerserkTraitTag{})
		log.Printf("%s の行動にBERSERK特性タグを付与。", SettingsComponent.Get(entry).Name)
	case TraitAim:
		donburi.Add(entry, ActingWithAimTraitTagComponent, &ActingWithAimTraitTag{})
		log.Printf("%s の行動にAIM特性タグを付与。", SettingsComponent.Get(entry).Name)
	}

	ChangeState(entry, StateTypeCharging) // ChangeState is in entity_utils.go
	return true
}
