package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// --- DamageCalculator ---

// DamageCalculator はダメージ計算に関連するロジックを担当します。
type DamageCalculator struct {
	world            donburi.World
	config           *Config
	partInfoProvider *PartInfoProvider // 後で初期化
}

// NewDamageCalculator は新しい DamageCalculator のインスタンスを生成します。
func NewDamageCalculator(world donburi.World, config *Config) *DamageCalculator {
	return &DamageCalculator{world: world, config: config}
}

// SetPartInfoProvider は PartInfoProvider の依存性を設定します。
func (dc *DamageCalculator) SetPartInfoProvider(pip *PartInfoProvider) {
	dc.partInfoProvider = pip
}

// ApplyDamage はパーツインスタンスにダメージを適用し、メダロットの状態を更新します。
func (dc *DamageCalculator) ApplyDamage(entry *donburi.Entry, partInst *PartInstanceData, damage int) {
	if damage < 0 {
		damage = 0
	}
	partInst.CurrentArmor -= damage
	if partInst.CurrentArmor <= 0 {
		partInst.CurrentArmor = 0
		partInst.IsBroken = true
		settings := SettingsComponent.Get(entry)
		// Get PartDefinition for logging PartName
		partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID)
		partNameForLog := "(不明パーツ)"
		if defFound {
			partNameForLog = partDef.PartName
		}
		log.Printf("%s の %s (%s) が破壊された！", settings.Name, partNameForLog, partInst.DefinitionID)

		if defFound && partDef.Type == PartTypeHead { // Check Type from PartDefinition
			ChangeState(entry, StateTypeBroken)
		}
	}
}

// CalculateDamage は ActionModifierComponent を考慮してダメージを計算します。
func (dc *DamageCalculator) CalculateDamage(attacker *donburi.Entry, partDef *PartDefinition) (int, bool) {
	// attackerLegs instance for stability
	var attackerLegsInstance *PartInstanceData
	if partsComp := PartsComponent.Get(attacker); partsComp != nil {
		attackerLegsInstance = partsComp.Map[PartSlotLegs]
	}

	attackerStability := 0
	if attackerLegsInstance != nil && !attackerLegsInstance.IsBroken { // Check instance for broken state
		// Stability comes from leg's definition
		if legsDef, found := GlobalGameDataManager.GetPartDefinition(attackerLegsInstance.DefinitionID); found {
			attackerStability = legsDef.Stability
		}
	}

	// Get modifiers if available
	var powerAdditiveBonus int
	var powerMultiplierBonus float64 = 1.0
	var criticalRateBonus int
	var customCriticalMultiplier float64 = 0 // 0 means use default

	if attacker.HasComponent(ActionModifierComponent) {
		modifiers := ActionModifierComponent.Get(attacker)
		powerAdditiveBonus = modifiers.PowerAdditiveBonus
		powerMultiplierBonus = modifiers.PowerMultiplierBonus
		criticalRateBonus = modifiers.CriticalRateBonus
		customCriticalMultiplier = modifiers.CriticalMultiplier
		// Note: DamageAdditiveBonus and DamageMultiplierBonus are applied at the very end if used.
	}

	// Base power calculation
	basePower := float64(partDef.Power) // Use partDef for Power
	// Apply additive power bonuses (from traits like Berserk via ActionModifierComponent, and stability)
	modifiedPower := basePower + float64(powerAdditiveBonus) + float64(attackerStability)*dc.config.Balance.Factors.PowerStabilityFactor
	// Apply multiplicative power bonuses
	modifiedPower *= powerMultiplierBonus

	// Critical Hit Calculation
	medal := MedalComponent.Get(attacker)
	isCritical := false
	// Base critical chance from medal skill level
	// Original: criticalChance := medal.SkillLevel*2 + criticalBonus
	// criticalBonus from AIM trait is now in modifiers.CriticalRateBonus
	// The original switch for part.Category (Melee crit bonus) is not yet in modifiers.
	// For now, let's assume all crit bonuses are aggregated into modifiers.CriticalRateBonus by ApplyActionModifiersSystem
	// or this part needs ApplyActionModifiersSystem to be aware of part.Category for AIM.
	// The ApplyActionModifiersSystem currently adds AIM crit bonus if ActingWithAimTraitTag is present.
	// Let's assume medal.SkillLevel*2 is a base, and modifiers.CriticalRateBonus contains trait/other bonuses.
	criticalChance := medal.SkillLevel*2 + criticalRateBonus
	if criticalChance < 0 {
		criticalChance = 0
	} // Ensure non-negative chance

	if rand.Intn(100) < criticalChance {
		critMultiplierToUse := dc.config.Balance.Damage.CriticalMultiplier
		if customCriticalMultiplier > 0 { // If a custom multiplier is set by a trait/effect
			critMultiplierToUse = customCriticalMultiplier
		}
		modifiedPower *= critMultiplierToUse
		isCritical = true
		log.Printf("%sの攻撃がクリティカルヒット！(Chance: %d%%, Multiplier: %.2f)", SettingsComponent.Get(attacker).Name, criticalChance, critMultiplierToUse)
	}

	// Final damage calculation (includes medal skill factor as an additive bonus at the end)
	// Original: finalDamage := baseDamage + float64(medal.SkillLevel*dc.config.Balance.Damage.MedalSkillFactor)
	// This MedalSkillFactor should ideally also be part of ActionModifierComponent.DamageAdditiveBonus or PowerAdditiveBonus.
	// For now, keeping it separate as per original logic, applied after critical.
	finalDamage := modifiedPower + float64(medal.SkillLevel*dc.config.Balance.Damage.MedalSkillFactor)

	// Apply final overall damage multipliers if any (e.g. from global buffs/debuffs)
	if attacker.HasComponent(ActionModifierComponent) {
		modifiers := ActionModifierComponent.Get(attacker)
		finalDamage *= modifiers.DamageMultiplierBonus
		finalDamage += float64(modifiers.DamageAdditiveBonus)
	}

	if finalDamage < 0 {
		finalDamage = 0
	}

	return int(finalDamage), isCritical
}

// GenerateActionLog は行動の結果ログを生成します。
// targetPartDef はダメージを受けたパーツの定義 (nilの場合あり)
func (dc *DamageCalculator) GenerateActionLog(attacker *donburi.Entry, target *donburi.Entry, targetPartDef *PartDefinition, damage int, isCritical bool, didHit bool) string {
	attackerSettings := SettingsComponent.Get(attacker)
	targetSettings := SettingsComponent.Get(target)

	if !didHit {
		actionComp := ActionComponent.Get(attacker)
		actingPartInst := PartsComponent.Get(attacker).Map[actionComp.SelectedPartKey]
		var actingPartName string
		if actingPartDef, defFound := GlobalGameDataManager.GetPartDefinition(actingPartInst.DefinitionID); defFound {
			actingPartName = actingPartDef.PartName
		} else {
			actingPartName = "(不明なパーツ)"
		}
		return fmt.Sprintf("%sの%s攻撃は%sに外れた！", attackerSettings.Name, actingPartName, targetSettings.Name)
	}

	targetPartNameStr := "(不明部位)"
	// targetPartWasBrokenStr := "" // Removed: Unused variable
	if targetPartDef != nil {
		targetPartNameStr = targetPartDef.PartName
		// To check if the part *became* broken, we'd need the PartInstanceData here,
		// or this function needs to be called after ApplyDamage and passed the PartInstanceData.
		// For now, let's assume if a PartDefinition is passed, we are logging about that part.
		// The TargetPartBroken in ActionResult is a better source for "パーツを破壊した！"
		// This log might need to be simplified or rely on ActionResult's data.
		// Let's assume for now this function is for generating the core message, and "パーツを破壊した！" is appended based on ActionResult.
	}

	logMsg := fmt.Sprintf("%sの%sに%dダメージ！", targetSettings.Name, targetPartNameStr, damage)
	if isCritical {
		logMsg = fmt.Sprintf("%sの%sにクリティカル！ %dダメージ！", targetSettings.Name, targetPartNameStr, damage)
	}
	// "パーツを破壊した！" は ActionResult.TargetPartBroken を見て呼び出し側で追加する方が正確
	return logMsg
}

// GenerateActionLogDefense は防御時のアクションログを生成します。
// defensePartDef は防御に使用されたパーツの定義
func (dc *DamageCalculator) GenerateActionLogDefense(target *donburi.Entry, defensePartDef *PartDefinition, damageDealt int, originalDamage int, isCritical bool) string {
	targetSettings := SettingsComponent.Get(target)
	defensePartName := "(不明なパーツ)"
	if defensePartDef != nil {
		defensePartName = defensePartDef.PartName
	}

	logMsg := fmt.Sprintf("%sは%sで防御し、ダメージを%dから%dに抑えた！", targetSettings.Name, defensePartName, originalDamage, damageDealt)
	if isCritical {
		logMsg = fmt.Sprintf("%sは%sで防御！クリティカルヒットのダメージを%dから%dに抑えた！", targetSettings.Name, defensePartName, originalDamage, damageDealt)
	}
	// "しかし、パーツは破壊された！" は同様に呼び出し側で判断・追加
	return logMsg
}

// --- HitCalculator ---

// HitCalculator は命中・回避・防御判定に関連するロジックを担当します。
type HitCalculator struct {
	world            donburi.World
	config           *Config
	partInfoProvider *PartInfoProvider // 後で初期化
}

// NewHitCalculator は新しい HitCalculator のインスタンスを生成します。
func NewHitCalculator(world donburi.World, config *Config) *HitCalculator {
	return &HitCalculator{world: world, config: config}
}

// SetPartInfoProvider は PartInfoProvider の依存性を設定します。
func (hc *HitCalculator) SetPartInfoProvider(pip *PartInfoProvider) {
	hc.partInfoProvider = pip
}

// CalculateHit は新しいルールに基づいて命中判定を行います。
func (hc *HitCalculator) CalculateHit(attacker, target *donburi.Entry, partDef *PartDefinition) bool {
	// Attacker stability
	var attackerLegsInstance *PartInstanceData
	if partsComp := PartsComponent.Get(attacker); partsComp != nil {
		attackerLegsInstance = partsComp.Map[PartSlotLegs]
	}
	attackerStability := 0
	if attackerLegsInstance != nil && !attackerLegsInstance.IsBroken {
		if legsDef, found := GlobalGameDataManager.GetPartDefinition(attackerLegsInstance.DefinitionID); found {
			attackerStability = legsDef.Stability
		}
	}

	// Get modifiers if available
	var accuracyAdditiveBonus int = 0
	if attacker.HasComponent(ActionModifierComponent) {
		modifiers := ActionModifierComponent.Get(attacker)
		accuracyAdditiveBonus = modifiers.AccuracyAdditiveBonus
	}

	baseAccuracy := float64(partDef.Accuracy) + float64(attackerStability)*hc.config.Balance.Factors.AccuracyStabilityFactor + float64(accuracyAdditiveBonus)

	// Original category-based accuracy bonus
	if hc.partInfoProvider == nil {
		log.Println("Error: HitCalculator.partInfoProvider is not initialized for category bonus")
	} else {
		switch partDef.Category { // Use partDef for Category
		case CategoryMelee:
			baseAccuracy += float64(hc.partInfoProvider.GetOverallMobility(attacker)) * hc.config.Balance.Factors.MeleeAccuracyMobilityFactor
		case CategoryShoot:
			// No specific bonus
		}
	}

	// Target stability for evasion
	var targetLegsInstance *PartInstanceData
	if partsComp := PartsComponent.Get(target); partsComp != nil {
		targetLegsInstance = partsComp.Map[PartSlotLegs]
	}
	targetStability := 0
	if targetLegsInstance != nil && !targetLegsInstance.IsBroken {
		if legsDef, found := GlobalGameDataManager.GetPartDefinition(targetLegsInstance.DefinitionID); found {
			targetStability = legsDef.Stability
		}
	}

	// Target evasion calculation (mobility also needs to come from PartDefinition of target's legs)
	// This part of the logic for GetOverallMobility needs to be checked/updated if it relied on *Part
	baseEvasion := 0.0
	if hc.partInfoProvider == nil {
		log.Println("Error: HitCalculator.partInfoProvider is not initialized for target evasion")
	} else {
		// GetOverallMobility should use PartDefinition for calculation
		baseEvasion = float64(hc.partInfoProvider.GetOverallMobility(target)) + float64(targetStability)*hc.config.Balance.Factors.EvasionStabilityFactor
	}

	finalEvasion := baseEvasion
	if target.HasComponent(EvasionDebuffComponent) {
		debuff := EvasionDebuffComponent.Get(target)
		finalEvasion *= debuff.Multiplier
	}

	chance := 50 + baseAccuracy - finalEvasion
	if chance < 5 {
		chance = 5
	}
	if chance > 95 {
		chance = 95
	}

	roll := rand.Intn(100)
	log.Printf("命中判定: %s -> %s | 命中率: %.1f (%f vs %f), ロール: %d", SettingsComponent.Get(attacker).Name, SettingsComponent.Get(target).Name, chance, baseAccuracy, finalEvasion, roll)
	return roll < int(chance)
}

// CalculateDefense は防御の成否を判定します。
// defensePartDef は防御に使用されるパーツの定義です。
func (hc *HitCalculator) CalculateDefense(target *donburi.Entry, defensePartDef *PartDefinition) bool {
	// Target stability
	var targetLegsInstance *PartInstanceData
	if partsComp := PartsComponent.Get(target); partsComp != nil {
		targetLegsInstance = partsComp.Map[PartSlotLegs]
	}
	targetStability := 0
	if targetLegsInstance != nil && !targetLegsInstance.IsBroken {
		if legsDef, found := GlobalGameDataManager.GetPartDefinition(targetLegsInstance.DefinitionID); found {
			targetStability = legsDef.Stability
		}
	}

	baseDefense := float64(defensePartDef.Defense) + float64(targetStability)*hc.config.Balance.Factors.DefenseStabilityFactor
	finalDefense := baseDefense
	if target.HasComponent(DefenseDebuffComponent) {
		debuff := DefenseDebuffComponent.Get(target)
		finalDefense *= debuff.Multiplier
	}

	chance := 10 + finalDefense
	if chance < 5 {
		chance = 5
	}
	if chance > 95 {
		chance = 95
	}

	roll := rand.Intn(100)
	log.Printf("防御判定: %s (%s) | 防御成功率: %.1f, ロール: %d", SettingsComponent.Get(target).Name, defensePartDef.PartName, chance, roll)
	return roll < int(chance)
}

// --- TargetSelector ---

// TargetSelector はターゲット選択やパーツ選択に関連するロジックを担当します。
type TargetSelector struct {
	world            donburi.World
	config           *Config
	partInfoProvider *PartInfoProvider // 後で初期化
}

// NewTargetSelector は新しい TargetSelector のインスタンスを生成します。
func NewTargetSelector(world donburi.World, config *Config) *TargetSelector {
	return &TargetSelector{world: world, config: config}
}

// SetPartInfoProvider は PartInfoProvider の依存性を設定します。
func (ts *TargetSelector) SetPartInfoProvider(pip *PartInfoProvider) {
	ts.partInfoProvider = pip
}

// SelectDefensePart は防御に使用するパーツのインスタンスを選択します。
func (ts *TargetSelector) SelectDefensePart(target *donburi.Entry) *PartInstanceData {
	partsComp := PartsComponent.Get(target)
	if partsComp == nil {
		return nil
	}
	partsMap := partsComp.Map // map[PartSlotKey]*PartInstanceData

	var bestPartInstance *PartInstanceData
	var armParts []*PartInstanceData
	var headPart *PartInstanceData

	for _, partInst := range partsMap {
		if partInst.IsBroken {
			continue
		}
		partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID)
		if !defFound {
			log.Printf("SelectDefensePart: PartDefinition not found for ID %s", partInst.DefinitionID)
			continue
		}

		switch partDef.Type {
		case PartTypeRArm, PartTypeLArm:
			armParts = append(armParts, partInst)
		case PartTypeHead:
			headPart = partInst // Assuming only one head part
		}
	}

	if len(armParts) > 0 {
		sort.Slice(armParts, func(i, j int) bool {
			return armParts[i].CurrentArmor > armParts[j].CurrentArmor
		})
		bestPartInstance = armParts[0]
	} else if headPart != nil && !headPart.IsBroken { // Ensure head part itself isn't broken (already checked above but good for clarity)
		bestPartInstance = headPart
	}
	// If no suitable arm or head part is found, bestPartInstance will remain nil.
	return bestPartInstance
}

// SelectRandomPartToDamage は攻撃対象のパーツインスタンスをランダムに選択します。
func (ts *TargetSelector) SelectRandomPartToDamage(target *donburi.Entry) *PartInstanceData {
	partsComp := PartsComponent.Get(target)
	if partsComp == nil {
		return nil
	}
	partsMap := partsComp.Map // map[PartSlotKey]*PartInstanceData

	vulnerableInstances := []*PartInstanceData{}
	slots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	for _, s := range slots {
		if partInst, ok := partsMap[s]; ok && partInst != nil && !partInst.IsBroken {
			vulnerableInstances = append(vulnerableInstances, partInst)
		}
	}
	if len(vulnerableInstances) == 0 {
		return nil
	}
	return vulnerableInstances[rand.Intn(len(vulnerableInstances))]
}

// FindClosestEnemy は指定されたエンティティに最も近い敵エンティティを見つけます。
func (ts *TargetSelector) FindClosestEnemy(actingEntry *donburi.Entry) *donburi.Entry {
	var closestEnemy *donburi.Entry
	minDist := float32(math.MaxFloat32)
	bfWidth := float32(ts.config.UI.Screen.Width) * 0.5 // BattleFieldの幅

	if ts.partInfoProvider == nil {
		log.Println("Error: TargetSelector.partInfoProvider is not initialized")
		return nil
	}
	actingX := ts.partInfoProvider.CalculateIconXPosition(actingEntry, bfWidth)

	for _, enemy := range ts.GetTargetableEnemies(actingEntry) {
		enemyX := ts.partInfoProvider.CalculateIconXPosition(enemy, bfWidth)
		dist := float32(math.Abs(float64(actingX - enemyX)))
		if dist < minDist {
			minDist = dist
			closestEnemy = enemy
		}
	}
	return closestEnemy
}

// GetTargetableEnemies は指定されたエンティティが攻撃可能な敵のリストを返します。
// 破壊されていない敵チームのエンティティを返します。
func (ts *TargetSelector) GetTargetableEnemies(actingEntry *donburi.Entry) []*donburi.Entry {
	opponentTeamID := ts.GetOpponentTeam(actingEntry)
	candidates := []*donburi.Entry{}
	query := query.NewQuery(filter.And(
		filter.Contains(SettingsComponent),
		filter.Not(filter.Contains(BrokenStateComponent)),
	))

	query.Each(ts.world, func(entry *donburi.Entry) {
		settings := SettingsComponent.Get(entry)
		if settings.Team == opponentTeamID {
			candidates = append(candidates, entry)
		}
	})

	sort.Slice(candidates, func(i, j int) bool {
		iSettings := SettingsComponent.Get(candidates[i])
		jSettings := SettingsComponent.Get(candidates[j])
		return iSettings.DrawIndex < jSettings.DrawIndex
	})
	return candidates
}

// GetOpponentTeam は指定されたエンティティの敵チームIDを返します。
func (ts *TargetSelector) GetOpponentTeam(actingEntry *donburi.Entry) TeamID {
	if SettingsComponent.Get(actingEntry).Team == Team1 {
		return Team2
	}
	return Team1
}

// --- PartInfoProvider ---

// PartInfoProvider はパーツの状態や情報を取得・操作するロジックを担当します。
type PartInfoProvider struct {
	world  donburi.World
	config *Config
}

// NewPartInfoProvider は新しい PartInfoProvider のインスタンスを生成します。
func NewPartInfoProvider(world donburi.World, config *Config) *PartInfoProvider {
	return &PartInfoProvider{world: world, config: config}
}

// FindPartSlot は指定されたパーツインスタンスがどのスロットにあるかを返します。
func (pip *PartInfoProvider) FindPartSlot(entry *donburi.Entry, partToFindInstance *PartInstanceData) PartSlotKey {
	partsComp := PartsComponent.Get(entry)
	if partsComp == nil || partToFindInstance == nil {
		return ""
	}
	for slotKey, partInst := range partsComp.Map {
		// Compare by DefinitionID, assuming part instances are unique by their definition within a Medarot,
		// or rely on pointer equality if partToFindInstance is guaranteed to be from this entry's map.
		// Using DefinitionID is safer if partToFindInstance might be a copy or from elsewhere.
		// However, if partToFindInstance is directly from this entry's map, pointer equality is fine.
		// For now, let's assume we are trying to find the slot of an instance we already have a pointer to from this map.
		if partInst == partToFindInstance { // Pointer comparison
			return slotKey
		}
		// If we need to find based on ID (e.g. from a PartDefinition):
		// if partInst.DefinitionID == partToFindInstance.DefinitionID { return slotKey }
	}
	return ""
}

// AvailablePart now holds PartDefinition for AI/UI to see base stats.
type AvailablePart struct {
	PartDef *PartDefinition // Changed from Part to PartDefinition
	Slot    PartSlotKey
}

// GetAvailableAttackParts は攻撃に使用可能なパーツの定義リストを返します。
func (pip *PartInfoProvider) GetAvailableAttackParts(entry *donburi.Entry) []AvailablePart {
	partsComp := PartsComponent.Get(entry)
	if partsComp == nil {
		return nil
	}
	var availableParts []AvailablePart
	slotsToConsider := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm}

	for _, slot := range slotsToConsider {
		partInst, ok := partsComp.Map[slot]
		if !ok || partInst == nil || partInst.IsBroken {
			continue
		}
		partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID)
		if !defFound {
			log.Printf("Warning: Part definition %s not found for available part check.", partInst.DefinitionID)
			continue
		}

		if partDef.Category != CategoryNone && partDef.Category != CategorySupport && partDef.Category != CategoryDefense {
			availableParts = append(availableParts, AvailablePart{PartDef: partDef, Slot: slot})
		}
	}
	return availableParts
}

// GetOverallPropulsion はエンティティの総推進力を返します。
func (pip *PartInfoProvider) GetOverallPropulsion(entry *donburi.Entry) int {
	partsComp := PartsComponent.Get(entry)
	if partsComp == nil {
		return 1
	}
	legsInstance, ok := partsComp.Map[PartSlotLegs]
	if !ok || legsInstance == nil || legsInstance.IsBroken {
		return 1 // 脚部がない、または破壊されている場合はデフォルト値
	}
	legsDef, defFound := GlobalGameDataManager.GetPartDefinition(legsInstance.DefinitionID)
	if !defFound {
		log.Printf("Warning: Legs part definition %s not found for propulsion.", legsInstance.DefinitionID)
		return 1
	}
	return legsDef.Propulsion
}

// GetOverallMobility はエンティティの総機動力を返します。
func (pip *PartInfoProvider) GetOverallMobility(entry *donburi.Entry) int {
	partsComp := PartsComponent.Get(entry)
	if partsComp == nil {
		return 1
	}
	legsInstance, ok := partsComp.Map[PartSlotLegs]
	if !ok || legsInstance == nil || legsInstance.IsBroken {
		return 1 // 脚部がない、または破壊されている場合はデフォルト値
	}
	legsDef, defFound := GlobalGameDataManager.GetPartDefinition(legsInstance.DefinitionID)
	if !defFound {
		log.Printf("Warning: Legs part definition %s not found for mobility.", legsInstance.DefinitionID)
		return 1
	}
	return legsDef.Mobility
}

// CalculateIconXPosition はバトルフィールド上のアイコンのX座標を計算します。
// worldWidth はバトルフィールドの表示幅です。
func (pip *PartInfoProvider) CalculateIconXPosition(entry *donburi.Entry, worldWidth float32) float32 {
	settings := SettingsComponent.Get(entry)
	gauge := GaugeComponent.Get(entry)

	progress := float32(0)
	if gauge.TotalDuration > 0 { // TotalDurationが0の場合のゼロ除算を避ける
		progress = float32(gauge.CurrentGauge / 100.0)
	}

	homeX, execX := worldWidth*0.1, worldWidth*0.4
	if settings.Team == Team2 {
		homeX, execX = worldWidth*0.9, worldWidth*0.6
	}

	var xPos float32
	if entry.HasComponent(ChargingStateComponent) {
		xPos = homeX + (execX-homeX)*progress
	} else if entry.HasComponent(ReadyStateComponent) {
		xPos = execX
	} else if entry.HasComponent(CooldownStateComponent) {
		// クールダウンは execX から homeX に戻る動き
		xPos = execX - (execX-homeX)*progress
	} else if entry.HasComponent(IdleStateComponent) || entry.HasComponent(BrokenStateComponent) {
		xPos = homeX
	} else {
		xPos = homeX // 不明な状態の場合はホームポジション
	}
	return xPos
}
