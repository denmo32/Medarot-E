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

// ApplyDamage はパーツにダメージを適用し、メダロットの状態を更新する
func ApplyDamage(entry *donburi.Entry, part *Part, damage int) {
	part.Armor -= damage
	if part.Armor <= 0 {
		part.Armor = 0
		part.IsBroken = true
		// 修正: ComponentType.Get(entry) を使用
		settings := SettingsComponent.Get(entry)
		log.Printf("%s の %s が破壊された！", settings.Name, part.PartName)
		if part.Type == PartTypeHead {
			ChangeState(entry, StateBroken)
		}
	}
}

// CalculateHit は命中判定を行う
func CalculateHit(attacker *donburi.Entry, target *donburi.Entry, part *Part, balanceConfig *BalanceConfig) bool {
	baseChance := balanceConfig.Hit.BaseChance
	accuracyBonus := part.Accuracy / 2
	evasionPenalty := GetOverallMobility(target) / 2
	chance := baseChance + accuracyBonus - evasionPenalty
	switch part.Trait {
	case TraitAim:
		chance += balanceConfig.Hit.TraitAimBonus
	case TraitStrike:
		chance += balanceConfig.Hit.TraitStrikeBonus
	case TraitBerserk:
		chance += balanceConfig.Hit.TraitBerserkDebuff
	}
	if chance < 10 {
		chance = 10
	} else if chance > 95 {
		chance = 95
	}
	roll := rand.Intn(100)
	// 修正: ComponentType.Get(entry) を使用
	attackerSettings := SettingsComponent.Get(attacker)
	// 修正: ComponentType.Get(entry) を使用
	targetSettings := SettingsComponent.Get(target)
	log.Printf("命中判定: %s -> %s | 命中率: %d, ロール: %d", attackerSettings.Name, targetSettings.Name, chance, roll)
	return roll < chance
}

// CalculateDamage はダメージ計算を行う
func CalculateDamage(attacker *donburi.Entry, part *Part, balanceConfig *BalanceConfig) (damage int, isCritical bool) {
	// 修正: ComponentType.Get(entry) を使用
	medal := MedalComponent.Get(attacker)
	baseDamage := part.Power
	isCritical = false
	criticalChance := medal.SkillLevel * 2
	if rand.Intn(100) < criticalChance {
		baseDamage = int(float64(baseDamage) * balanceConfig.Damage.CriticalMultiplier)
		isCritical = true
	}
	baseDamage += medal.SkillLevel * balanceConfig.Damage.MedalSkillFactor
	return baseDamage, isCritical
}

// SelectRandomPartToDamage はダメージを受けるパーツをランダムに選択する
func SelectRandomPartToDamage(target *donburi.Entry) *Part {
	// 修正: ComponentType.Get(entry) を使用
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

// GenerateActionLog は行動ログの文字列を生成する
func GenerateActionLog(attacker *donburi.Entry, target *donburi.Entry, targetPart *Part, damage int, isCritical bool, didHit bool) string {
	// 修正: ComponentType.Get(entry) を使用
	attackerSettings := SettingsComponent.Get(attacker)
	// 修正: ComponentType.Get(entry) を使用
	targetSettings := SettingsComponent.Get(target)
	if !didHit {
		// 修正: ComponentType.Get(entry) を使用
		actingPartKey := ActionComponent.Get(attacker).SelectedPartKey
		// 修正: ComponentType.Get(entry) を使用
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

// findPartSlot は指定されたパーツがどのスロットにあるかを返す
func findPartSlot(entry *donburi.Entry, part *Part) PartSlotKey {
	partsMap := PartsComponent.Get(entry).Map
	for s, p := range partsMap {
		if p.ID == part.ID {
			return s
		}
	}
	return ""
}

// --- Componentデータからのゲッター関数 ---
// 古いMedarot構造体のゲッターメソッドの代わり

// GetAvailableAttackParts は攻撃可能なパーツ一覧を返す
func GetAvailableAttackParts(entry *donburi.Entry) []*Part {
	// 修正: ComponentType.Get(entry) を使用
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

// GetOverallPropulsion は推進力を取得（足回り）
func GetOverallPropulsion(entry *donburi.Entry) int {
	// 修正: ComponentType.Get(entry) を使用
	partsMap := PartsComponent.Get(entry).Map
	legs := partsMap[PartSlotLegs]
	if legs == nil || legs.IsBroken {
		return 1
	}
	return legs.Propulsion
}

// GetOverallMobility は機動性を取得（足回り）
func GetOverallMobility(entry *donburi.Entry) int {
	// 修正: ComponentType.Get(entry) を使用
	partsMap := PartsComponent.Get(entry).Map
	legs := partsMap[PartSlotLegs]
	if legs == nil || legs.IsBroken {
		return 1
	}
	return legs.Mobility
}

// CalculateIconXPosition はメダロットの描画位置(X座標)をステートから算出
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

// findClosestEnemy は最も近い敵を見つける
func findClosestEnemy(g *Game, actingEntry *donburi.Entry) *donburi.Entry {
	var closestEnemy *donburi.Entry
	minDist := float32(math.MaxFloat32)

	// battlefieldの幅をUI設定から取得
	bfWidth := float32(g.Config.UI.Screen.Width) * 0.5 // 仮。実際のレイアウトに依存

	actingX := CalculateIconXPosition(actingEntry, bfWidth)

	for _, enemy := range getTargetCandidates(g, actingEntry) {
		enemyX := CalculateIconXPosition(enemy, bfWidth)
		dist := float32(math.Abs(float64(actingX - enemyX)))
		if dist < minDist {
			minDist = dist
			closestEnemy = enemy
		}
	}
	return closestEnemy
}

// getTargetCandidates 敵チームの中で行動可能なメダロット一覧を取得
func getTargetCandidates(g *Game, actingEntry *donburi.Entry) []*donburi.Entry {
	actingSettings := SettingsComponent.Get(actingEntry)

	// ★★★ 修正点: ロジックをより明確に ★★★
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
	)).Each(g.World, func(entry *donburi.Entry) {
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
