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

// ApplyDamage はパーツにダメージを適用し、メダロットの状態を更新します
func ApplyDamage(entry *donburi.Entry, part *Part, damage int) {
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
			ChangeState(entry, StateBroken)
		}
	}
}

// CalculateHit は新しいルールに基づいて命中判定を行います
func CalculateHit(attacker, target *donburi.Entry, part *Part, config *BalanceConfig) bool {
	// 攻撃側の命中性能を計算
	attackerLegs := PartsComponent.Get(attacker).Map[PartSlotLegs]
	attackerStability := 0
	if attackerLegs != nil {
		attackerStability = attackerLegs.Stability
	}

	baseAccuracy := float64(part.Accuracy) + float64(attackerStability)*config.Factors.AccuracyStabilityFactor

	// アクション特性によるボーナス
	switch part.Category {
	case CategoryMelee:
		baseAccuracy += float64(GetOverallMobility(attacker)) * config.Factors.MeleeAccuracyMobilityFactor
	case CategoryShoot:
		// 「撃つ」は純粋な安定ボーナスのみ
	}

	// 回避側の回避性能を計算
	targetEffects := EffectsComponent.Get(target)
	targetLegs := PartsComponent.Get(target).Map[PartSlotLegs]
	targetStability := 0
	if targetLegs != nil {
		targetStability = targetLegs.Stability
	}

	baseEvasion := float64(GetOverallMobility(target)) + float64(targetStability)*config.Factors.EvasionStabilityFactor
	finalEvasion := baseEvasion * targetEffects.EvasionRateMultiplier // デバフを適用

	// 最終的な命中率を計算 (例: 50 + (攻撃側の命中 - 敵の回避))
	chance := 50 + baseAccuracy - finalEvasion
	if chance < 5 {
		chance = 5
	}
	if chance > 95 {
		chance = 95
	}

	roll := rand.Intn(100)
	log.Printf("命中判定: %s -> %s | 命中率: %.1f, ロール: %d", SettingsComponent.Get(attacker).Name, SettingsComponent.Get(target).Name, chance, roll)
	return roll < int(chance)
}

// CalculateDefense は防御の成否を判定します
func CalculateDefense(attacker, target *donburi.Entry, defensePart *Part, config *BalanceConfig) bool {
	// 防御側の防御性能を計算
	targetEffects := EffectsComponent.Get(target)
	targetLegs := PartsComponent.Get(target).Map[PartSlotLegs]
	targetStability := 0
	if targetLegs != nil {
		targetStability = targetLegs.Stability
	}

	baseDefense := float64(defensePart.Defense) + float64(targetStability)*config.Factors.DefenseStabilityFactor
	finalDefense := baseDefense * targetEffects.DefenseRateMultiplier // デバフを適用

	// 防御成功率を計算 (例: 10 + 防御性能)
	chance := 10 + finalDefense
	if chance < 5 {
		chance = 5
	}
	if chance > 95 {
		chance = 95
	}

	roll := rand.Intn(100)
	log.Printf("防御判定: %s | 防御成功率: %.1f, ロール: %d", SettingsComponent.Get(target).Name, chance, roll)
	return roll < int(chance)
}

// CalculateDamage は新しいルールに基づいてダメージを計算します
func CalculateDamage(attacker *donburi.Entry, part *Part, config *BalanceConfig) (int, bool) {
	// 基本威力を計算
	attackerLegs := PartsComponent.Get(attacker).Map[PartSlotLegs]
	attackerStability := 0
	if attackerLegs != nil {
		attackerStability = attackerLegs.Stability
	}
	baseDamage := float64(part.Power) + float64(attackerStability)*config.Factors.PowerStabilityFactor

	// アクション特性によるボーナス
	if part.Trait == TraitBerserk {
		baseDamage += float64(GetOverallPropulsion(attacker)) * config.Factors.BerserkPowerPropulsionFactor
	}

	// クリティカル判定
	medal := MedalComponent.Get(attacker)
	isCritical := false
	criticalBonus := 0
	switch part.Category {
	case CategoryMelee:
		criticalBonus = config.Effects.Melee.CriticalRateBonus
	case CategoryShoot:
		if part.Trait == TraitAim {
			criticalBonus = config.Effects.Aim.CriticalRateBonus
		}
	}
	criticalChance := medal.SkillLevel*2 + criticalBonus
	if rand.Intn(100) < criticalChance {
		baseDamage *= config.Damage.CriticalMultiplier
		isCritical = true
	}

	finalDamage := baseDamage + float64(medal.SkillLevel*config.Damage.MedalSkillFactor)
	return int(finalDamage), isCritical
}

// SelectDefensePart は防御に使用するパーツを選択します
func SelectDefensePart(target *donburi.Entry) *Part {
	parts := PartsComponent.Get(target).Map
	var bestPart *Part
	var otherParts []*Part

	// 頭部以外の未破壊パーツを探す
	for slot, part := range parts {
		if !part.IsBroken && slot != PartSlotLegs && slot != PartSlotHead {
			otherParts = append(otherParts, part)
		}
	}

	if len(otherParts) > 0 {
		// 装甲が最も高いパーツで防御
		sort.Slice(otherParts, func(i, j int) bool {
			return otherParts[i].Armor > otherParts[j].Armor
		})
		bestPart = otherParts[0]
	} else {
		// 他にパーツがなければ頭部で防御
		if head := parts[PartSlotHead]; head != nil && !head.IsBroken {
			bestPart = head
		}
	}
	return bestPart
}

func SelectRandomPartToDamage(target *donburi.Entry) *Part {
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

func GenerateActionLog(attacker *donburi.Entry, target *donburi.Entry, targetPart *Part, damage int, isCritical bool, didHit bool) string {
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

func findPartSlot(entry *donburi.Entry, part *Part) PartSlotKey {
	partsMap := PartsComponent.Get(entry).Map
	for s, p := range partsMap {
		if p.ID == part.ID {
			return s
		}
	}
	return ""
}

func GetAvailableAttackParts(entry *donburi.Entry) []*Part {
	partsMap := PartsComponent.Get(entry).Map
	var availableParts []*Part
	slotsToConsider := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm}
	for _, slot := range slotsToConsider {
		part := partsMap[slot]
		if part != nil && !part.IsBroken && part.Category != CategoryNone {
			availableParts = append(availableParts, part)
		}
	}
	return availableParts
}

func GetOverallPropulsion(entry *donburi.Entry) int {
	partsMap := PartsComponent.Get(entry).Map
	legs := partsMap[PartSlotLegs]
	if legs == nil || legs.IsBroken {
		return 1
	}
	return legs.Propulsion
}

func GetOverallMobility(entry *donburi.Entry) int {
	partsMap := PartsComponent.Get(entry).Map
	legs := partsMap[PartSlotLegs]
	if legs == nil || legs.IsBroken {
		return 1
	}
	return legs.Mobility
}

func CalculateIconXPosition(entry *donburi.Entry, worldWidth float32) float32 {
	settings := SettingsComponent.Get(entry)
	state := StateComponent.Get(entry)
	gauge := GaugeComponent.Get(entry)

	progress := float32(gauge.CurrentGauge / 100.0)
	homeX, execX := worldWidth*0.1, worldWidth*0.4
	if settings.Team == Team2 {
		homeX, execX = worldWidth*0.9, worldWidth*0.6
	}

	var xPos float32
	switch state.State {
	case StateCharging:
		xPos = homeX + (execX-homeX)*progress
	case StateReady:
		xPos = execX
	case StateCooldown:
		xPos = execX - (execX-homeX)*progress
	case StateIdle, StateBroken:
		xPos = homeX
	default:
		xPos = homeX
	}
	return xPos
}

func findClosestEnemy(bs *BattleScene, actingEntry *donburi.Entry) *donburi.Entry {
	var closestEnemy *donburi.Entry
	minDist := float32(math.MaxFloat32)
	bfWidth := float32(bs.resources.Config.UI.Screen.Width) * 0.5
	actingX := CalculateIconXPosition(actingEntry, bfWidth)

	for _, enemy := range getTargetCandidates(bs, actingEntry) {
		enemyX := CalculateIconXPosition(enemy, bfWidth)
		dist := float32(math.Abs(float64(actingX - enemyX)))
		if dist < minDist {
			minDist = dist
			closestEnemy = enemy
		}
	}
	return closestEnemy
}

func getTargetCandidates(bs *BattleScene, actingEntry *donburi.Entry) []*donburi.Entry {
	actingSettings := SettingsComponent.Get(actingEntry)
	var opponentTeamID TeamID
	if actingSettings.Team == Team1 {
		opponentTeamID = Team2
	} else {
		opponentTeamID = Team1
	}

	candidates := []*donburi.Entry{}
	query.NewQuery(filter.And(
		filter.Contains(SettingsComponent),
		filter.Contains(StateComponent),
	)).Each(bs.world, func(entry *donburi.Entry) {
		settings := SettingsComponent.Get(entry)
		state := StateComponent.Get(entry)
		if settings.Team == opponentTeamID && state.State != StateBroken {
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
