package main

import (
	"context"
	// "fmt"
	"log"
	"math"
	"math/rand"
	"sort"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// BattleLogic は戦闘関連のすべての計算ロジックをカプセル化します。
type BattleLogic struct {
	DamageCalculator *DamageCalculator
	HitCalculator    *HitCalculator
	TargetSelector   *TargetSelector
	PartInfoProvider *PartInfoProvider
}

// NewBattleLogic は BattleLogic とそのすべての依存ヘルパーを初期化します。
func NewBattleLogic(world donburi.World, config *Config) *BattleLogic {
	bl := &BattleLogic{}

	// ヘルパーを初期化
	bl.PartInfoProvider = NewPartInfoProvider(world, config)
	bl.DamageCalculator = NewDamageCalculator(world, config)
	bl.HitCalculator = NewHitCalculator(world, config)
	bl.TargetSelector = NewTargetSelector(world, config)

	// ヘルパー間の依存性を注入
	bl.DamageCalculator.SetPartInfoProvider(bl.PartInfoProvider)
	bl.HitCalculator.SetPartInfoProvider(bl.PartInfoProvider)
	bl.TargetSelector.SetPartInfoProvider(bl.PartInfoProvider)

	return bl
}

// --- DamageCalculator ---

// DamageCalculator はダメージ計算に関連するロジックを担当します。
type DamageCalculator struct {
	world            donburi.World
	config           *Config
	partInfoProvider *PartInfoProvider
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
		log.Print(GlobalGameDataManager.Messages.FormatMessage("log_part_broken_notification", map[string]interface{}{
			"ordered_args": []interface{}{settings.Name, partNameForLog, partInst.DefinitionID},
		}))

		if defFound && partDef.Type == PartTypeHead { // Check Type from PartDefinition
			state := StateComponent.Get(entry)
			if state.FSM.Can("break") {
				err := state.FSM.Event(context.Background(), "break", entry)
				if err != nil {
					log.Printf("Error breaking medarot %s: %v", settings.Name, err)
				}
			}
		}
	}
}

// getParameterValue は指定されたパラメータの値を取得するヘルパー関数です。
func (pip *PartInfoProvider) getParameterValue(entry *donburi.Entry, param PartParameter) float64 {
	legsDef, found := pip.GetLegsPartDefinition(entry)
	if !found {
		return 0
	}
	switch param {
	case Mobility:
		return float64(legsDef.Mobility)
	case Propulsion:
		return float64(legsDef.Propulsion)
	case Stability:
		return float64(legsDef.Stability)
	case Defense:
		return float64(legsDef.Defense)
	default:
		return 0
	}
}

// GetSuccessRate はエンティティの成功度を計算します。
func (pip *PartInfoProvider) GetSuccessRate(entry *donburi.Entry, actingPartDef *PartDefinition) float64 {
	successRate := float64(actingPartDef.Accuracy)

	// 特性によるボーナスを加算
	formula, ok := FormulaManager[actingPartDef.Trait]
	if ok {
		for _, bonus := range formula.SuccessRateBonuses {
			successRate += pip.getParameterValue(entry, bonus.SourceParam) * bonus.Multiplier
		}
	}
	return successRate
}

// GetEvasionRate はエンティティの回避度を計算します。
func (pip *PartInfoProvider) GetEvasionRate(entry *donburi.Entry) float64 {
	evasion := 0.0
	legsDef, found := pip.GetLegsPartDefinition(entry)
	if found {
		evasion = float64(legsDef.Mobility)
	}

	// デバフの影響を適用
	if entry.HasComponent(EvasionDebuffComponent) {
		evasion *= EvasionDebuffComponent.Get(entry).Multiplier
	}
	return evasion
}

// GetDefenseRate はエンティティの防御度を計算します。
func (pip *PartInfoProvider) GetDefenseRate(entry *donburi.Entry) float64 {
	defense := 0.0
	legsDef, found := pip.GetLegsPartDefinition(entry)
	if found {
		defense = float64(legsDef.Defense)
	}

	// デバフの影響を適用
	if entry.HasComponent(DefenseDebuffComponent) {
		defense *= DefenseDebuffComponent.Get(entry).Multiplier
	}
	return defense
}

// CalculateDamage はActionFormulaに基づいてダメージを計算します。
func (dc *DamageCalculator) CalculateDamage(attacker, target *donburi.Entry, actingPartDef *PartDefinition) (int, bool) {
	// 1. 計算式の取得
	formula, ok := FormulaManager[actingPartDef.Trait]
	if !ok {
		log.Printf("警告: 特性 '%s' に対応する計算式が見つかりません。デフォルトを使用します。", actingPartDef.Trait)
		formula = FormulaManager[TraitNormal]
	}

	// 2. 基本パラメータの取得
	successRate := dc.partInfoProvider.GetSuccessRate(attacker, actingPartDef)
	power := float64(actingPartDef.Power)

	// 特性による威力ボーナスを加算
	if formula != nil {
		for _, bonus := range formula.PowerBonuses {
			power += dc.partInfoProvider.getParameterValue(attacker, bonus.SourceParam) * bonus.Multiplier
		}
	}
	evasion := dc.partInfoProvider.GetEvasionRate(target)

	// クリティカル判定
	isCritical := false
	criticalChance := dc.config.Balance.Damage.Critical.BaseChance + (successRate * dc.config.Balance.Damage.Critical.SuccessRateFactor) + formula.CriticalRateBonus

	// クリティカル率の上下限を適用
	if criticalChance < dc.config.Balance.Damage.Critical.MinChance {
		criticalChance = dc.config.Balance.Damage.Critical.MinChance
	}
	if criticalChance > dc.config.Balance.Damage.Critical.MaxChance {
		criticalChance = dc.config.Balance.Damage.Critical.MaxChance
	}

	if rand.Intn(100) < int(criticalChance) {
		isCritical = true
		log.Printf("%s の攻撃がクリティカル！ (確率: %.1f%%)", SettingsComponent.Get(attacker).Name, criticalChance)
		// クリティカル時は回避度を0にする
		evasion = 0
	}

	// 5. 最終ダメージ計算
	damage := (successRate - evasion) / dc.config.Balance.Damage.DamageAdjustmentFactor + power
	// 乱数(±10%)
	randomFactor := 1.0 + (rand.Float64()*0.2 - 0.1)
	damage *= randomFactor

	if damage < 1 {
		damage = 1
	}

	log.Printf("ダメージ計算 (%s): (%.1f - %.1f) / %.1f + %.1f * %.2f = %d (Crit: %t)",
		formula.ID, successRate, evasion, dc.config.Balance.Damage.DamageAdjustmentFactor, power, randomFactor, int(damage), isCritical)

	return int(damage), isCritical
}

// GenerateActionLog は行動の結果ログを生成します。
// targetPartDef はダメージを受けたパーツの定義 (nilの場合あり)
// actingPartDef は攻撃に使用されたパーツの定義
func (dc *DamageCalculator) GenerateActionLog(attacker *donburi.Entry, target *donburi.Entry, actingPartDef *PartDefinition, targetPartDef *PartDefinition, damage int, isCritical bool, didHit bool) string {
	attackerSettings := SettingsComponent.Get(attacker)
	targetSettings := SettingsComponent.Get(target)
	skillName := "(不明なスキル)"
	if actingPartDef != nil {
		skillName = actingPartDef.PartName
	}

	if !didHit {
		return GlobalGameDataManager.Messages.FormatMessage("attack_miss", map[string]interface{}{
			"attacker_name": attackerSettings.Name,
			"skill_name":    skillName,
			"target_name":   targetSettings.Name,
		})
	}

	targetPartNameStr := "(不明部位)"
	if targetPartDef != nil {
		targetPartNameStr = targetPartDef.PartName
	}

	params := map[string]interface{}{
		"attacker_name":    attackerSettings.Name,
		"skill_name":       skillName,
		"target_name":      targetSettings.Name,
		"target_part_name": targetPartNameStr,
		"damage":           damage,
	}

	if isCritical {
		return GlobalGameDataManager.Messages.FormatMessage("critical_hit", params)
	}
	return GlobalGameDataManager.Messages.FormatMessage("attack_hit", params)
}

// CalculateReducedDamage は防御成功時のダメージを計算します。
func (dc *DamageCalculator) CalculateReducedDamage(originalDamage int, defensePartDef *PartDefinition) int {
	// ダメージ軽減ロジック: ダメージ = 元ダメージ - 防御パーツの防御力
	// 将来的に、より複雑な計算式（例：割合軽減）に変更する可能性があります。
	reducedDamage := originalDamage - defensePartDef.Defense
	if reducedDamage < 1 {
		reducedDamage = 1 // 最低でも1ダメージは保証
	}
	log.Printf("防御成功！ ダメージ軽減: %d -> %d (防御パーツ防御力: %d)", originalDamage, reducedDamage, defensePartDef.Defense)
	return reducedDamage
}

// GenerateActionLogDefense は防御時のアクションログを生成します。
// defensePartDef は防御に使用されたパーツの定義
func (dc *DamageCalculator) GenerateActionLogDefense(target *donburi.Entry, defensePartDef *PartDefinition, damageDealt int, originalDamage int, isCritical bool) string {
	targetSettings := SettingsComponent.Get(target)
	defensePartNameStr := "(不明なパーツ)"
	if defensePartDef != nil {
		defensePartNameStr = defensePartDef.PartName
	}

	params := map[string]interface{}{
		"target_name":       targetSettings.Name,
		"defense_part_name": defensePartNameStr,
		"original_damage":   originalDamage,
		"actual_damage":     damageDealt,
	}

	if isCritical {
		return GlobalGameDataManager.Messages.FormatMessage("defense_success_critical", params)
	}
	return GlobalGameDataManager.Messages.FormatMessage("defense_success", params)
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
	// 攻撃側の成功度
	successRate := hc.partInfoProvider.GetSuccessRate(attacker, partDef)

	// 防御側の回避度
	evasion := hc.partInfoProvider.GetEvasionRate(target)

	// 命中確率 = 基準値 + (成功度 - 回避度)
	chance := hc.config.Balance.Hit.BaseChance + (successRate - evasion)

	// 確率の上下限を適用
	if chance < hc.config.Balance.Hit.MinChance {
		chance = hc.config.Balance.Hit.MinChance
	}
	if chance > hc.config.Balance.Hit.MaxChance {
		chance = hc.config.Balance.Hit.MaxChance
	}

	roll := rand.Intn(100)
	log.Print(GlobalGameDataManager.Messages.FormatMessage("log_hit_roll", map[string]interface{}{
		"ordered_args": []interface{}{SettingsComponent.Get(attacker).Name, SettingsComponent.Get(target).Name, chance, successRate, evasion, roll},
	}))
	return float64(roll) < chance
}

// CalculateDefense は防御の成否を判定します。
func (hc *HitCalculator) CalculateDefense(attacker, target *donburi.Entry, actingPartDef *PartDefinition) bool {
	// 攻撃側の成功度
	successRate := hc.partInfoProvider.GetSuccessRate(attacker, actingPartDef)

	// 防御側の防御度
	defenseRate := hc.partInfoProvider.GetDefenseRate(target)

	// 防御成功確率 = 基準値 + (防御度 - 成功度)
	chance := hc.config.Balance.Defense.BaseChance + (defenseRate - successRate)

	// 確率の上下限を適用
	if chance < hc.config.Balance.Defense.MinChance {
		chance = hc.config.Balance.Defense.MinChance
	}
	if chance > hc.config.Balance.Defense.MaxChance {
		chance = hc.config.Balance.Defense.MaxChance
	}

	roll := rand.Intn(100)
	log.Print(GlobalGameDataManager.Messages.FormatMessage("log_defense_roll", map[string]interface{}{
		"ordered_args": []interface{}{SettingsComponent.Get(target).Name, defenseRate, successRate, chance, roll},
	}))
	return float64(roll) < chance
}

// --- TargetSelector ---

// TargetSelector はターゲット選択やパーツ選択に関連するロジックを担当します。
type TargetSelector struct {
	world            donburi.World
	config           *Config
	partInfoProvider *PartInfoProvider
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
	maxArmor := -1 // Initialize with a value lower than any possible armor

	// 腕部と脚部を優先して、最も装甲の高いパーツを探す
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
		case PartTypeRArm, PartTypeLArm, PartTypeLegs:
			if partInst.CurrentArmor > maxArmor {
				maxArmor = partInst.CurrentArmor
				bestPartInstance = partInst
			}
		}
	}

	// 腕部と脚部が全て破壊されている場合、頭部をチェック
	if bestPartInstance == nil {
		if headPart, ok := partsMap[PartSlotHead]; ok && !headPart.IsBroken {
			bestPartInstance = headPart
		}
	}

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
	query.NewQuery(filter.Contains(SettingsComponent)).Each(ts.world, func(entry *donburi.Entry) {
		if StateComponent.Get(entry).FSM.Is(string(StateBroken)) {
			return
		}
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
	PartDef  *PartDefinition // Changed from Part to PartDefinition
	Slot     PartSlotKey
	IsBroken bool // パーツが破壊されているか
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
		partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID)
		if !defFound {
			log.Printf("Warning: Part definition %s not found for available part check.", partInst.DefinitionID)
			continue
		}

		if partDef.Category != CategoryNone && partDef.Category != CategorySupport && partDef.Category != CategoryDefense {
			availableParts = append(availableParts, AvailablePart{PartDef: partDef, Slot: slot, IsBroken: partInst.IsBroken})
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
	return GlobalGameDataManager.GetPartDefinition(legsInstance.DefinitionID)
}

// CalculateIconXPosition はバトルフィールド上のアイコンのX座標を計算します。
// worldWidth はバトルフィールドの表示幅です。
func (pip *PartInfoProvider) CalculateIconXPosition(entry *donburi.Entry, battlefieldWidth float32) float32 {
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
	if state.FSM.Is(string(StateCharging)) {
		xPos = homeX + (execX-homeX)*progress
	} else if state.FSM.Is(string(StateReady)) {
		xPos = execX
	} else if state.FSM.Is(string(StateCooldown)) {
		xPos = execX - (execX-homeX)*progress
	} else if state.FSM.Is(string(StateIdle)) || state.FSM.Is(string(StateBroken)) {
		xPos = homeX
	} else {
		xPos = homeX // 不明な状態の場合はホームポジション
	}
	return xPos
}
