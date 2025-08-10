package main

import (
	"log"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// PartInfoProvider はパーツの状態や情報を取得・操作するロジックを担当します。
type PartInfoProvider struct {
	world           donburi.World
	config          *data.Config
	gameDataManager *data.GameDataManager
}

// NewPartInfoProvider は新しい PartInfoProvider のインスタンスを生成します。
func NewPartInfoProvider(world donburi.World, config *data.Config, gdm *data.GameDataManager) PartInfoProviderInterface {
	return &PartInfoProvider{world: world, config: config, gameDataManager: gdm}
}

// GetGameDataManager は GameDataManager のインスタンスを返します。
func (pip *PartInfoProvider) GetGameDataManager() *data.GameDataManager {
	return pip.gameDataManager
}

// GetPartParameterValue は指定されたパーツスロットとパラメータの値を取得する汎用ヘルパー関数です。
func (pip *PartInfoProvider) GetPartParameterValue(entry *donburi.Entry, partSlot core.PartSlotKey, param core.PartParameter) float64 {
	partsComp := component.PartsComponent.Get(entry)
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
	case core.Power:
		return float64(partDef.Power)
	case core.Accuracy:
		return float64(partDef.Accuracy)
	case core.Mobility:
		return float64(partDef.Mobility)
	case core.Propulsion:
		return float64(partDef.Propulsion)
	case core.Stability:
		return float64(partDef.Stability)
	case core.Defense:
		return float64(partDef.Defense)
	default:
		return 0
	}
}

// FindPartSlot は指定されたパーツインスタンスがどのスロットにあるかを返します。
func (pip *PartInfoProvider) FindPartSlot(entry *donburi.Entry, partToFindInstance *core.PartInstanceData) core.PartSlotKey {
	partsComp := component.PartsComponent.Get(entry)
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
func (pip *PartInfoProvider) GetAvailableAttackParts(entry *donburi.Entry) []core.AvailablePart {
	partsComp := component.PartsComponent.Get(entry)
	if partsComp == nil {
		return nil
	}
	var availableParts []core.AvailablePart
	slotsToConsider := []core.PartSlotKey{core.PartSlotHead, core.PartSlotRightArm, core.PartSlotLeftArm}

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

		if partDef.Category == core.CategoryRanged || partDef.Category == core.CategoryMelee || partDef.Category == core.CategoryIntervention {
			availableParts = append(availableParts, core.AvailablePart{PartDef: partDef, Slot: slot})
		}
	}
	return availableParts
}

// GetOverallPropulsion はエンティティの総推進力を返します。
func (pip *PartInfoProvider) GetOverallPropulsion(entry *donburi.Entry) int {
	return int(pip.GetPartParameterValue(entry, core.PartSlotLegs, core.Propulsion))
}

// GetOverallMobility はエンティティの総機動力を返します。
func (pip *PartInfoProvider) GetOverallMobility(entry *donburi.Entry) int {
	return int(pip.GetPartParameterValue(entry, core.PartSlotLegs, core.Mobility))
}

// GetLegsPartDefinition はエンティティの脚部パーツの定義を取得します。
func (pip *PartInfoProvider) GetLegsPartDefinition(entry *donburi.Entry) (*core.PartDefinition, bool) {
	partsComp := component.PartsComponent.Get(entry)
	if partsComp == nil {
		return nil, false
	}
	legsInstance, ok := partsComp.Map[core.PartSlotLegs]
	if !ok || legsInstance == nil || legsInstance.IsBroken {
		return nil, false
	}
	return pip.gameDataManager.GetPartDefinition(legsInstance.DefinitionID)
}

// GetSuccessRate はエンティティの成功度を計算します。
func (pip *PartInfoProvider) GetSuccessRate(entry *donburi.Entry, actingPartDef *core.PartDefinition, selectedPartKey core.PartSlotKey) float64 {
	successRate := float64(actingPartDef.Accuracy)

	// 特性によるボーナスを加算
	formula, ok := pip.gameDataManager.Formulas[actingPartDef.Trait]
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
	evasion := pip.GetPartParameterValue(entry, core.PartSlotLegs, core.Mobility)

	// ActiveEffectsComponentから回避デバフの影響を適用
	if entry.HasComponent(component.ActiveEffectsComponent) {
		activeEffects := component.ActiveEffectsComponent.Get(entry)
		for _, activeEffect := range activeEffects.Effects {
			if evasionDebuff, ok := activeEffect.EffectData.(*core.EvasionDebuffEffectData); ok {
				evasion *= evasionDebuff.Multiplier
			}
		}
	}
	return evasion
}

// GetDefenseRate はエンティティの防御度を計算します。
func (pip *PartInfoProvider) GetDefenseRate(entry *donburi.Entry) float64 {
	defense := pip.GetPartParameterValue(entry, core.PartSlotLegs, core.Defense)

	// ActiveEffectsComponentから防御デバフの影響を適用
	if entry.HasComponent(component.ActiveEffectsComponent) {
		activeEffects := component.ActiveEffectsComponent.Get(entry)
		for _, activeEffect := range activeEffects.Effects {
			if defenseDebuff, ok := activeEffect.EffectData.(*core.DefenseDebuffEffectData); ok {
				defense *= defenseDebuff.Multiplier
			}
		}
	}
	return defense
}

// GetTeamAccuracyBuffMultiplier は、指定されたエンティティが所属するチームの
// 命中率バフ（スキャンなど）の中から最も効果の高いものの乗数を返します。
func (pip *PartInfoProvider) GetTeamAccuracyBuffMultiplier(entry *donburi.Entry) float64 {
	teamBuffsEntry, ok := query.NewQuery(filter.Contains(component.TeamBuffsComponent)).First(pip.world)
	if !ok {
		return 1.0 // バフコンポーネントがなければ効果なし
	}
	teamBuffs := component.TeamBuffsComponent.Get(teamBuffsEntry)
	settings := component.SettingsComponent.Get(entry)

	teamID := settings.Team
	buffType := core.BuffTypeAccuracy

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
func (pip *PartInfoProvider) RemoveBuffsFromSource(entry *donburi.Entry, partInst *core.PartInstanceData) {
	teamBuffsEntry, ok := query.NewQuery(filter.Contains(component.TeamBuffsComponent)).First(pip.world)
	if !ok {
		return
	}
	teamBuffs := component.TeamBuffsComponent.Get(teamBuffsEntry)
	partSlot := pip.FindPartSlot(entry, partInst)

	for teamID, buffMap := range teamBuffs.Buffs {
		for buffType, buffSources := range buffMap {
			newBuffSources := make([]*component.BuffSource, 0, len(buffSources))
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
	if entry != nil && entry.HasComponent(component.PartsComponent) {
		partsComp := component.PartsComponent.Get(entry)
		if partsComp != nil {
			legsInstance, ok := partsComp.Map[core.PartSlotLegs]
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

// GetNormalizedActionProgress は、メダロットの行動ゲージの進行度を0.0〜1.0の正規化された値で返します。
// これはUIのピクセル座標に依存せず、ゲームロジック内で抽象的な位置を計算するために使用されます。
func (pip *PartInfoProvider) GetNormalizedActionProgress(entry *donburi.Entry) float32 {
	state := component.StateComponent.Get(entry)
	gauge := component.GaugeComponent.Get(entry)

	progress := float32(0)
	if gauge.TotalDuration > 0 { // TotalDurationが0の場合のゼロ除算を避ける
		progress = float32(gauge.ProgressCounter / gauge.TotalDuration)
	}

	switch state.CurrentState {
	case core.StateCharging:
		return progress
	case core.StateReady:
		return 1.0
	case core.StateCooldown:
		return 1.0 - progress
	case core.StateIdle, core.StateBroken:
		return 0.0
	default:
		return 0.0 // 不明な状態の場合はホームポジション
	}
}
