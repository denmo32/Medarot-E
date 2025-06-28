package main

import (
	"fmt"
	"log"
	"math/rand"
)

// ApplyDamage はパーツにダメージを適用し、メダロットの状態を更新する
func ApplyDamage(medarot *Medarot, part *Part, damage int) {
	part.Armor -= damage
	if part.Armor <= 0 {
		part.Armor = 0
		part.IsBroken = true
		log.Printf("%s の %s が破壊された！", medarot.Name, part.PartName)

		// もし破壊されたのが頭パーツなら、即座に機能停止状態にする
		if part.Type == PartTypeHead {
			medarot.ChangeState(StateBroken)
		}
	}
}

// CalculateHit は命中判定を行う
func CalculateHit(attacker *Medarot, target *Medarot, part *Part, balanceConfig *BalanceConfig) bool {
	baseChance := balanceConfig.Hit.BaseChance
	accuracyBonus := part.Accuracy / 2
	evasionPenalty := target.GetOverallMobility() / 2
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
	log.Printf("命中判定: %s -> %s | 命中率: %d, ロール: %d", attacker.Name, target.Name, chance, roll)
	return roll < chance
}

// CalculateDamage はダメージ計算を行う
func CalculateDamage(attacker *Medarot, part *Part, balanceConfig *BalanceConfig) (damage int, isCritical bool) {
	baseDamage := part.Power
	isCritical = false

	criticalChance := attacker.Medal.SkillLevel * 2
	if rand.Intn(100) < criticalChance {
		baseDamage = int(float64(baseDamage) * balanceConfig.Damage.CriticalMultiplier)
		isCritical = true
	}

	baseDamage += attacker.Medal.SkillLevel * balanceConfig.Damage.MedalSkillFactor
	return baseDamage, isCritical
}

// SelectRandomPartToDamage はダメージを受けるパーツをランダムに選択する
func SelectRandomPartToDamage(target *Medarot) *Part {
	vulnerable := []*Part{}
	slots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	for _, s := range slots {
		if part := target.GetPart(s); part != nil && !part.IsBroken {
			vulnerable = append(vulnerable, part)
		}
	}
	if len(vulnerable) == 0 {
		return nil
	}
	return vulnerable[rand.Intn(len(vulnerable))]
}

// GenerateActionLog は行動ログの文字列を生成する
func GenerateActionLog(attacker *Medarot, target *Medarot, targetPart *Part, damage int, isCritical bool, didHit bool) string {
	if !didHit {
		actingPart := attacker.GetPart(attacker.SelectedPartKey)
		return fmt.Sprintf("%sの%s攻撃は%sに外れた！", attacker.Name, actingPart.PartName, target.Name)
	}

	logMsg := fmt.Sprintf("%sの%sに%dダメージ！", target.Name, targetPart.PartName, damage)
	if isCritical {
		logMsg = fmt.Sprintf("%sの%sにクリティカル！ %dダメージ！", target.Name, targetPart.PartName, damage)
	}
	if targetPart.IsBroken {
		logMsg += " パーツを破壊した！"
	}
	return logMsg
}