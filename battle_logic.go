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

// AvailablePart は利用可能なパーツとそのスロットキーを保持します。
type AvailablePart struct {
	Part *Part
	Slot PartSlotKey
}

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

// ApplyDamage はパーツにダメージを適用し、メダロットの状態を更新します。
func (dc *DamageCalculator) ApplyDamage(entry *donburi.Entry, part *Part, damage int) {
	if damage < 0 {
		damage = 0
	}
	part.Armor -= damage
	if part.Armor <= 0 {
		part.Armor = 0
		part.IsBroken = true
		settings := SettingsComponent.Get(entry)
		log.Printf("%s の %s が破壊された！", settings.Name, part.PartName)
		if part.Type == PartTypeHead {
			ChangeState(entry, StateTypeBroken) // systems.go にある ChangeState を呼び出す
		}
	}
}

// CalculateDamage は ActionModifierComponent を考慮してダメージを計算します。
func (dc *DamageCalculator) CalculateDamage(attacker *donburi.Entry, part *Part) (int, bool) {
	attackerLegs := PartsComponent.Get(attacker).Map[PartSlotLegs]
	attackerStability := 0
	if attackerLegs != nil {
		attackerStability = attackerLegs.Stability
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
	basePower := float64(part.Power)
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
	if criticalChance < 0 { criticalChance = 0 } // Ensure non-negative chance

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
func (dc *DamageCalculator) GenerateActionLog(attacker *donburi.Entry, target *donburi.Entry, targetPart *Part, damage int, isCritical bool, didHit bool) string {
	attackerSettings := SettingsComponent.Get(attacker)
	targetSettings := SettingsComponent.Get(target)
	if !didHit {
		actingPartKey := ActionComponent.Get(attacker).SelectedPartKey
		actingPart := PartsComponent.Get(attacker).Map[actingPartKey]
		return fmt.Sprintf("%sの%s攻撃は%sに外れた！", attackerSettings.Name, actingPart.PartName, targetSettings.Name)
	}
	logMsg := fmt.Sprintf("%sの%sに%dダメージ！", targetSettings.Name, targetPart.PartName, damage)
	if isCritical {
		logMsg = fmt.Sprintf("%sの%sにクリティカル！ %dダメージ！", targetSettings.Name, targetPart.PartName, damage)
	}
	if targetPart.IsBroken {
		logMsg += " パーツを破壊した！"
	}
	return logMsg
}

// GenerateActionLogDefense は防御時のアクションログを生成します。
func (dc *DamageCalculator) GenerateActionLogDefense(target *donburi.Entry, defensePart *Part, damageDealt int, originalDamage int, isCritical bool) string {
	targetSettings := SettingsComponent.Get(target)
	logMsg := fmt.Sprintf("%sは%sで防御し、ダメージを%dから%dに抑えた！", targetSettings.Name, defensePart.PartName, originalDamage, damageDealt)
	if isCritical {
		logMsg = fmt.Sprintf("%sは%sで防御！クリティカルヒットのダメージを%dから%dに抑えた！", targetSettings.Name, defensePart.PartName, originalDamage, damageDealt)
	}
	if defensePart.IsBroken {
		logMsg += " しかし、パーツは破壊された！"
	}
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
func (hc *HitCalculator) CalculateHit(attacker, target *donburi.Entry, part *Part) bool {
	attackerLegs := PartsComponent.Get(attacker).Map[PartSlotLegs]
	attackerStability := 0
	if attackerLegs != nil {
		attackerStability = attackerLegs.Stability
	}

	// Get modifiers if available
	var accuracyAdditiveBonus int = 0
	// var accuracyMultiplier float64 = 1.0 // If added later to ActionModifierComponentData
	if attacker.HasComponent(ActionModifierComponent) {
		modifiers := ActionModifierComponent.Get(attacker)
		accuracyAdditiveBonus = modifiers.AccuracyAdditiveBonus
		// accuracyMultiplier = modifiers.AccuracyMultiplier
	}

	baseAccuracy := float64(part.Accuracy) + float64(attackerStability)*hc.config.Balance.Factors.AccuracyStabilityFactor + float64(accuracyAdditiveBonus)
	// baseAccuracy *= accuracyMultiplier // Apply multiplicative bonus if exists

	// Original category-based accuracy bonus (e.g., Melee from mobility)
	// This should ideally be moved into ApplyActionModifiersSystem if it's a fixed bonus for a category/trait.
	if hc.partInfoProvider == nil {
		log.Println("Error: HitCalculator.partInfoProvider is not initialized for category bonus")
	} else {
		switch part.Category {
		case CategoryMelee:
			baseAccuracy += float64(hc.partInfoProvider.GetOverallMobility(attacker)) * hc.config.Balance.Factors.MeleeAccuracyMobilityFactor
		case CategoryShoot:
			// No specific bonus here in original logic beyond stability/base accuracy
		}
	}

	targetLegs := PartsComponent.Get(target).Map[PartSlotLegs]
	targetStability := 0
	if targetLegs != nil {
		targetStability = targetLegs.Stability
	}

	baseEvasion := 0.0
	if hc.partInfoProvider == nil {
		log.Println("Error: HitCalculator.partInfoProvider is not initialized")
	} else {
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
func (hc *HitCalculator) CalculateDefense(target *donburi.Entry, defensePart *Part) bool {
	targetLegs := PartsComponent.Get(target).Map[PartSlotLegs]
	targetStability := 0
	if targetLegs != nil {
		targetStability = targetLegs.Stability
	}
	baseDefense := float64(defensePart.Defense) + float64(targetStability)*hc.config.Balance.Factors.DefenseStabilityFactor
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
	log.Printf("防御判定: %s (%s) | 防御成功率: %.1f, ロール: %d", SettingsComponent.Get(target).Name, defensePart.PartName, chance, roll)
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

// SelectDefensePart は防御に使用するパーツを選択します。
func (ts *TargetSelector) SelectDefensePart(target *donburi.Entry) *Part {
	parts := PartsComponent.Get(target).Map
	var bestPart *Part
	var otherParts []*Part

	for slot, part := range parts {
		if !part.IsBroken && slot != PartSlotLegs && slot != PartSlotHead {
			otherParts = append(otherParts, part)
		}
	}

	if len(otherParts) > 0 {
		sort.Slice(otherParts, func(i, j int) bool {
			return otherParts[i].Armor > otherParts[j].Armor
		})
		bestPart = otherParts[0]
	} else {
		if head := parts[PartSlotHead]; head != nil && !head.IsBroken {
			bestPart = head
		}
	}
	return bestPart
}

// SelectRandomPartToDamage は攻撃対象のパーツをランダムに選択します。
func (ts *TargetSelector) SelectRandomPartToDamage(target *donburi.Entry) *Part {
	parts := PartsComponent.Get(target).Map
	vulnerable := []*Part{}
	slots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	for _, s := range slots {
		if part := parts[s]; part != nil && !part.IsBroken {
			vulnerable = append(vulnerable, part)
		}
	}
	if len(vulnerable) == 0 {
		return nil
	}
	return vulnerable[rand.Intn(len(vulnerable))]
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

// FindPartSlot は指定されたパーツがどのスロットにあるかを返します。
func (pip *PartInfoProvider) FindPartSlot(entry *donburi.Entry, partToFind *Part) PartSlotKey {
	partsMap := PartsComponent.Get(entry).Map
	for s, p := range partsMap {
		if p.ID == partToFind.ID {
			return s
		}
	}
	return ""
}

// GetAvailableAttackParts は攻撃に使用可能なパーツのリストを返します。
func (pip *PartInfoProvider) GetAvailableAttackParts(entry *donburi.Entry) []AvailablePart {
	partsMap := PartsComponent.Get(entry).Map
	var availableParts []AvailablePart
	slotsToConsider := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm}
	for _, slot := range slotsToConsider {
		part := partsMap[slot]
		if part != nil && !part.IsBroken && part.Category != CategoryNone && part.Category != CategorySupport && part.Category != CategoryDefense { // 防御と支援以外
			availableParts = append(availableParts, AvailablePart{Part: part, Slot: slot})
		}
	}
	return availableParts
}

// GetOverallPropulsion はエンティティの総推進力を返します。
func (pip *PartInfoProvider) GetOverallPropulsion(entry *donburi.Entry) int {
	partsMap := PartsComponent.Get(entry).Map
	legs := partsMap[PartSlotLegs]
	if legs == nil || legs.IsBroken {
		return 1 // 脚部がない、または破壊されている場合はデフォルト値
	}
	return legs.Propulsion
}

// GetOverallMobility はエンティティの総機動力を返します。
func (pip *PartInfoProvider) GetOverallMobility(entry *donburi.Entry) int {
	partsMap := PartsComponent.Get(entry).Map
	legs := partsMap[PartSlotLegs]
	if legs == nil || legs.IsBroken {
		return 1 // 脚部がない、または破壊されている場合はデフォルト値
	}
	return legs.Mobility
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
