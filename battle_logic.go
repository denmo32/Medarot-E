package main

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/yohamta/donburi"
)

// 各ComponentTypeが定義されていることを想定しています
// var SettingsComponent = donburi.NewComponentType[Settings]()
// var PartsComponent = donburi.NewComponentType[Parts]()
// var MedalComponent = donburi.NewComponentType[Medal]()
// var ActionComponent = donburi.NewComponentType[Action]()

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

// --- Componentデータからのゲッター関数 ---
// 古いMedarot構造体のゲッターメソッドの代わり

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

func GetOverallPropulsion(entry *donburi.Entry) int {
	// 修正: ComponentType.Get(entry) を使用
	partsMap := PartsComponent.Get(entry).Map
	legs := partsMap[PartSlotLegs]
	if legs == nil || legs.IsBroken {
		return 1
	}
	return legs.Propulsion
}

func GetOverallMobility(entry *donburi.Entry) int {
	// 修正: ComponentType.Get(entry) を使用
	partsMap := PartsComponent.Get(entry).Map
	legs := partsMap[PartSlotLegs]
	if legs == nil || legs.IsBroken {
		return 1
	}
	return legs.Mobility
}
