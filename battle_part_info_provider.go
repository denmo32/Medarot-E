package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// PartInfoProvider はパーツの状態や情報を取得・操作するロジックを担当します。
type PartInfoProvider struct {
	world           donburi.World
	config          *Config
	gameDataManager *GameDataManager
}

// NewPartInfoProvider は新しい PartInfoProvider のインスタンスを生成します。
func NewPartInfoProvider(world donburi.World, config *Config, gdm *GameDataManager) PartInfoProviderInterface {
	return &PartInfoProvider{world: world, config: config, gameDataManager: gdm}
}

// GetGameDataManager は GameDataManager のインスタンスを返します。
func (pip *PartInfoProvider) GetGameDataManager() *GameDataManager {
	return pip.gameDataManager
}

// GetPartParameterValue は指定されたパーツスロットとパラメータの値を取得する汎用ヘルパー関数です。
func (pip *PartInfoProvider) GetPartParameterValue(entry *donburi.Entry, partSlot PartSlotKey, param PartParameter) float64 {
	partsComp := PartsComponent.Get(entry)
	if partsComp == nil {
		return 0
	}
	partInst, ok := partsComp.Map[partSlot]
	if !ok || partInst == nil || partInst.IsBroken {
		return 0
	}
	partDef, found := pip.gameDataManager.GetPartDefinition(partInst.DefinitionID)
	if !found {
		log.Printf("警告: PartDefinition not found for ID %s in slot %s", partInst.DefinitionID, partSlot)
		return 0
	}

	switch param {
	case Power:
		return float64(partDef.Power)
	case Accuracy:
		return float64(partDef.Accuracy)
	case Mobility:
		return float64(partDef.Mobility)
	case Propulsion:
		return float64(partDef.Propulsion)
	case Stability:
		return float64(partDef.Stability)
	case Defense:
		return float64(partDef.Defense)
	default:
		return 0
	}
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
		if !ok || partInst == nil {
			continue
		}
		partDef, defFound := pip.gameDataManager.GetPartDefinition(partInst.DefinitionID)
		if !defFound {
			log.Printf("Warning: Part definition %s not found for available part check.", partInst.DefinitionID)
			continue
		}

		if partDef.Category == CategoryRanged || partDef.Category == CategoryMelee || partDef.Category == CategoryIntervention {
			availableParts = append(availableParts, AvailablePart{PartDef: partDef, Slot: slot, IsBroken: partInst.IsBroken})
		}
	}
	return availableParts
}

// GetOverallPropulsion はエンティティの総推進力を返します。
func (pip *PartInfoProvider) GetOverallPropulsion(entry *donburi.Entry) int {
	return int(pip.GetPartParameterValue(entry, PartSlotLegs, Propulsion))
}

// GetOverallMobility はエンティティの総機動力を返します。
func (pip *PartInfoProvider) GetOverallMobility(entry *donburi.Entry) int {
	return int(pip.GetPartParameterValue(entry, PartSlotLegs, Mobility))
}

// GetLegsPartDefinition はエンティティの脚部パーツの定義を取得します。
func (pip *PartInfoProvider) GetLegsPartDefinition(entry *donburi.Entry) (*PartDefinition, bool) {
	partsComp := PartsComponent.Get(entry)
	if partsComp == nil {
		return nil, false
	}
	legsInstance, ok := partsComp.Map[PartSlotLegs]
	if !ok || legsInstance == nil || legsInstance.IsBroken {
		return nil, false
	}
	return pip.gameDataManager.GetPartDefinition(legsInstance.DefinitionID)
}

// GetSuccessRate はエンティティの成功度を計算します。
func (pip *PartInfoProvider) GetSuccessRate(entry *donburi.Entry, actingPartDef *PartDefinition, selectedPartKey PartSlotKey) float64 {
	successRate := float64(actingPartDef.Accuracy)

	// 特性によるボーナスを加算
	formula, ok := FormulaManager[actingPartDef.Trait]
	if ok {
		for _, bonus := range formula.SuccessRateBonuses {
			// 攻撃パーツのパラメータを参照するように変更
			successRate += pip.GetPartParameterValue(entry, selectedPartKey, bonus.SourceParam) * bonus.Multiplier
		}
	}
	return successRate
}

// GetEvasionRate はエンティティの回避度を計算します。
func (pip *PartInfoProvider) GetEvasionRate(entry *donburi.Entry) float64 {
	evasion := pip.GetPartParameterValue(entry, PartSlotLegs, Mobility)

	// ActiveEffectsComponentから回避デバフの影響を適用
	if entry.HasComponent(ActiveEffectsComponent) {
		activeEffects := ActiveEffectsComponent.Get(entry)
		for _, activeEffect := range activeEffects.Effects {
			if evasionDebuff, ok := activeEffect.EffectData.(*EvasionDebuffEffect); ok {
				evasion *= evasionDebuff.Multiplier
			}
		}
	}
	return evasion
}

// GetDefenseRate はエンティティの防御度を計算します。
func (pip *PartInfoProvider) GetDefenseRate(entry *donburi.Entry) float64 {
	defense := pip.GetPartParameterValue(entry, PartSlotLegs, Defense)

	// ActiveEffectsComponentから防御デバフの影響を適用
	if entry.HasComponent(ActiveEffectsComponent) {
		activeEffects := ActiveEffectsComponent.Get(entry)
		for _, activeEffect := range activeEffects.Effects {
			if defenseDebuff, ok := activeEffect.EffectData.(*DefenseDebuffEffect); ok {
				defense *= defenseDebuff.Multiplier
			}
		}
	}
	return defense
}

// GetTeamAccuracyBuffMultiplier は、指定されたエンティティが所属するチームの
// 命中率バフ（スキャンなど）の中から最も効果の高いものの乗数を返します。
func (pip *PartInfoProvider) GetTeamAccuracyBuffMultiplier(entry *donburi.Entry) float64 {
	teamBuffsEntry, ok := query.NewQuery(filter.Contains(TeamBuffsComponent)).First(pip.world)
	if !ok {
		return 1.0 // バフコンポーネントがなければ効果なし
	}
	teamBuffs := TeamBuffsComponent.Get(teamBuffsEntry)
	settings := SettingsComponent.Get(entry)

	teamID := settings.Team
	buffType := BuffTypeAccuracy

	maxMultiplier := 1.0

	if teamBuffMap, teamOk := teamBuffs.Buffs[teamID]; teamOk {
		if buffSources, buffOk := teamBuffMap[buffType]; buffOk {
			for _, buff := range buffSources {
				if buff.Value > maxMultiplier {
					maxMultiplier = buff.Value
				}
			}
		}
	}

	return maxMultiplier
}

// RemoveBuffsFromSource は、指定されたパーツインスタンスが提供していたバフをすべて削除します。
func (pip *PartInfoProvider) RemoveBuffsFromSource(entry *donburi.Entry, partInst *PartInstanceData) {
	teamBuffsEntry, ok := query.NewQuery(filter.Contains(TeamBuffsComponent)).First(pip.world)
	if !ok {
		return
	}
	teamBuffs := TeamBuffsComponent.Get(teamBuffsEntry)
	partSlot := pip.FindPartSlot(entry, partInst)

	for teamID, buffMap := range teamBuffs.Buffs {
		for buffType, buffSources := range buffMap {
			newBuffSources := make([]*BuffSource, 0, len(buffSources))
			for _, buff := range buffSources {
				// このパーツからのバフでなければ保持する
				if buff.SourceEntry != entry || buff.SourcePart != partSlot {
					newBuffSources = append(newBuffSources, buff)
				}
			}
			teamBuffs.Buffs[teamID][buffType] = newBuffSources
		}
	}
}

// CalculateGaugeDuration は、行動の基本時間と推進力を基に、
// 最終的なゲージの持続時間（tick数）を計算します。
func (pip *PartInfoProvider) CalculateGaugeDuration(baseSeconds float64, entry *donburi.Entry) float64 {
	if baseSeconds <= 0 {
		baseSeconds = 0.1 // 0秒または負の値を避ける
	}

	propulsion := 1
	// entryが有効でPartsComponentを持っているか確認
	if entry != nil && entry.HasComponent(PartsComponent) {
		partsComp := PartsComponent.Get(entry)
		if partsComp != nil {
			legsInstance, ok := partsComp.Map[PartSlotLegs]
			if ok && legsInstance != nil && !legsInstance.IsBroken {
				propulsion = pip.GetOverallPropulsion(entry)
			}
		}
	}

	balanceConfig := &pip.config.Balance
	propulsionFactor := 1.0 + (float64(propulsion) * balanceConfig.Time.PropulsionEffectRate)
	totalTicks := (baseSeconds * 60.0) / (balanceConfig.Time.GameSpeedMultiplier * propulsionFactor)

	if totalTicks < 1 {
		return 1
	}
	return totalTicks
}

// CalculateMedarotXPosition はメダロットのX座標を計算します。
// battlefieldWidth はバトルフィールドの論理的な幅です。
func (pip *PartInfoProvider) CalculateMedarotXPosition(entry *donburi.Entry, battlefieldWidth float32) float32 {
	settings := SettingsComponent.Get(entry)
	gauge := GaugeComponent.Get(entry)
	state := StateComponent.Get(entry)

	progress := float32(0)
	if gauge.TotalDuration > 0 { // TotalDurationが0の場合のゼロ除算を避ける
		progress = float32(gauge.CurrentGauge / 100.0)
	}

	homeX, execX := battlefieldWidth*0.1, battlefieldWidth*0.4
	if settings.Team == Team2 {
		homeX, execX = battlefieldWidth*0.9, battlefieldWidth*0.6
	}

	var xPos float32
	switch state.CurrentState {
	case StateCharging:
		xPos = homeX + (execX-homeX)*progress
	case StateReady:
		xPos = execX
	case StateCooldown:
		xPos = execX - (execX-homeX)*progress
	case StateIdle, StateBroken:
		xPos = homeX
	default:
		xPos = homeX // 不明な状態の場合はホームポジション
	}
	return xPos
}
