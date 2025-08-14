package system

import (
	"log"
	"math/rand"
	"sort"

	"medarot-ebiten/core"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ecs/entity"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// getAllTargetablePartsは、指定された攻撃者がターゲットにできる全てのパーツを取得します。
// BattleLogicへの依存をなくし、必要なTargetSelectorとPartInfoProviderを直接受け取ります。
func getAllTargetableParts(actingEntry *donburi.Entry, targetSelector *TargetSelector, partInfoProvider PartInfoProviderInterface, includeHead bool) []component.TargetablePart {
	var allParts []component.TargetablePart

	// ターゲット候補となる敵エンティティのリストを取得
	candidates := targetSelector.GetTargetableEnemies(actingEntry)
	gameDataManager := partInfoProvider.GetGameDataManager()

	for _, enemyEntry := range candidates {
		partsComp := component.PartsComponent.Get(enemyEntry)
		if partsComp == nil {
			continue
		}
		for slotKey, partInst := range partsComp.Map {
			if partInst.IsBroken {
				continue
			}
			// 頭部パーツを除外するオプション
			if !includeHead && slotKey == core.PartSlotHead {
				continue
			}
			partDef, defFound := gameDataManager.GetPartDefinition(partInst.DefinitionID)
			if !defFound {
				log.Printf("警告: getAllTargetableParts - PartDefinition %s が見つかりません。", partInst.DefinitionID)
				continue
			}

			allParts = append(allParts, component.TargetablePart{
				Entity:   enemyEntry,
				PartInst: partInst,
				PartDef:  partDef,
				Slot:     slotKey,
			})
		}
	}
	return allParts
}

// TargetSortFunc は、ターゲット候補のリストをソートするための関数の型を定義します。
// これにより、各戦略はソートロジックのみを提供すればよくなります。
type TargetSortFunc func(parts []component.TargetablePart)

// --- 戦略の実装 ---

// selectTargetWithSort は、提供されたソート関数を使用してターゲットを選択する共通ロジックです。
func selectTargetWithSort(actingEntry *donburi.Entry, targetSelector *TargetSelector, partInfoProvider PartInfoProviderInterface, sortFunc TargetSortFunc) (*donburi.Entry, core.PartSlotKey) {
	// まずは頭部以外のパーツをターゲット候補とする
	targetParts := getAllTargetableParts(actingEntry, targetSelector, partInfoProvider, false)
	// 候補がなければ頭部も含めて再検索
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(actingEntry, targetSelector, partInfoProvider, true)
	}
	if len(targetParts) == 0 {
		return nil, ""
	}

	sortFunc(targetParts)

	selected := targetParts[0]
	return selected.Entity, selected.Slot
}

// CrusherStrategy は最も装甲の高いパーツを狙います。
type CrusherStrategy struct{}

func (s *CrusherStrategy) SelectTarget(world donburi.World, actingEntry *donburi.Entry, targetSelector *TargetSelector, partInfoProvider PartInfoProviderInterface, rand *rand.Rand) (*donburi.Entry, core.PartSlotKey) {
	return selectTargetWithSort(actingEntry, targetSelector, partInfoProvider, s.GetSortFunction())
}

func (s *CrusherStrategy) GetSortFunction() TargetSortFunc {
	return func(parts []component.TargetablePart) {
		sort.Slice(parts, func(i, j int) bool {
			return parts[i].PartInst.CurrentArmor > parts[j].PartInst.CurrentArmor
		})
	}
}

// HunterStrategy は最も装甲の低いパーツを狙います。
type HunterStrategy struct{}

func (s *HunterStrategy) SelectTarget(world donburi.World, actingEntry *donburi.Entry, targetSelector *TargetSelector, partInfoProvider PartInfoProviderInterface, rand *rand.Rand) (*donburi.Entry, core.PartSlotKey) {
	return selectTargetWithSort(actingEntry, targetSelector, partInfoProvider, s.GetSortFunction())
}

func (s *HunterStrategy) GetSortFunction() TargetSortFunc {
	return func(parts []component.TargetablePart) {
		sort.Slice(parts, func(i, j int) bool {
			return parts[i].PartInst.CurrentArmor < parts[j].PartInst.CurrentArmor
		})
	}
}

// JokerStrategy はランダムなパーツを狙います。
type JokerStrategy struct{}

func (s *JokerStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	rand *rand.Rand,
) (*donburi.Entry, core.PartSlotKey) {
	allEnemyParts := getAllTargetableParts(actingEntry, targetSelector, partInfoProvider, true)
	if len(allEnemyParts) == 0 {
		return nil, ""
	}

	idx := rand.Intn(len(allEnemyParts))
	return allEnemyParts[idx].Entity, allEnemyParts[idx].Slot
}

// LeaderStrategy はリーダーのパーツをランダムに狙います。
type LeaderStrategy struct{}

func (s *LeaderStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	rand *rand.Rand,
) (*donburi.Entry, core.PartSlotKey) {
	opponentTeamID := targetSelector.GetOpponentTeam(actingEntry)
	leader := entity.FindLeader(world, opponentTeamID)

	if leader != nil && component.StateComponent.Get(leader).CurrentState != core.StateBroken {
		targetPart := targetSelector.SelectPartToDamage(leader, actingEntry, rand)
		if targetPart != nil {
			slotKey := partInfoProvider.FindPartSlot(leader, targetPart)
			if slotKey != "" {
				return leader, slotKey
			}
		}
	}
	// リーダーを狙えない場合はランダムにフォールバック
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider, rand)
}

// ChaseStrategy は最も推進力の高い脚部パーツを狙います。
type ChaseStrategy struct{}

func (s *ChaseStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	rand *rand.Rand,
) (*donburi.Entry, core.PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, targetSelector, partInfoProvider, true)
	if len(targetParts) == 0 {
		return nil, ""
	}

	var legParts []component.TargetablePart
	for _, p := range targetParts {
		if p.PartDef.Type == core.PartTypeLegs {
			legParts = append(legParts, p)
		}
	}

	if len(legParts) > 0 {
		sort.Slice(legParts, func(i, j int) bool {
			return legParts[i].PartDef.Propulsion > legParts[j].PartDef.Propulsion
		})
		// 最も推進力の高い脚部が複数ある場合、その中からランダムに選ぶ
		maxPropulsion := legParts[0].PartDef.Propulsion
		var candidates []component.TargetablePart
		for _, p := range legParts {
			if p.PartDef.Propulsion == maxPropulsion {
				candidates = append(candidates, p)
			}
		}
		selected := candidates[rand.Intn(len(candidates))]
		return selected.Entity, selected.Slot
	}

	// 脚部が全破壊の場合、ランダムな非脚部パーツを狙う
	var otherParts []component.TargetablePart
	for _, p := range targetParts {
		if p.PartDef.Type != core.PartTypeLegs {
			otherParts = append(otherParts, p)
		}
	}
	if len(otherParts) > 0 {
		selected := otherParts[rand.Intn(len(otherParts))]
		return selected.Entity, selected.Slot
	}

	// フォールバック
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider, rand)
}

// DuelStrategy は攻撃系腕パーツ（射撃/格闘）を優先して狙います。
type DuelStrategy struct{}

func (s *DuelStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	rand *rand.Rand,
) (*donburi.Entry, core.PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, targetSelector, partInfoProvider, true)
	if len(targetParts) == 0 {
		return nil, ""
	}

	var attackArmParts []component.TargetablePart
	for _, p := range targetParts {
		if (p.PartDef.Type == core.PartTypeLArm || p.PartDef.Type == core.PartTypeRArm) &&
			(p.PartDef.Category == core.CategoryRanged || p.PartDef.Category == core.CategoryMelee) {
			attackArmParts = append(attackArmParts, p)
		}
	}

	if len(attackArmParts) > 0 {
		selected := attackArmParts[rand.Intn(len(attackArmParts))]
		return selected.Entity, selected.Slot
	}

	// 攻撃系腕パーツがない場合、ランダムなパーツを狙う
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider, rand)
}

// InterceptStrategy は非攻撃系パーツ（射撃/格闘以外）を優先して狙います。
type InterceptStrategy struct{}

func (s *InterceptStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	rand *rand.Rand,
) (*donburi.Entry, core.PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, targetSelector, partInfoProvider, true)
	if len(targetParts) == 0 {
		return nil, ""
	}

	var nonAttackParts []component.TargetablePart
	for _, p := range targetParts {
		if p.PartDef.Category != core.CategoryRanged && p.PartDef.Category != core.CategoryMelee {
			nonAttackParts = append(nonAttackParts, p)
		}
	}

	if len(nonAttackParts) > 0 {
		selected := nonAttackParts[rand.Intn(len(nonAttackParts))]
		return selected.Entity, selected.Slot
	}

	// 非攻撃系パーツがない場合、ランダムなパーツを狙う
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider, rand)
}

// --- 履歴ベースの戦略 ---

// CounterStrategy は自分を最後に攻撃した敵を狙います。
type CounterStrategy struct{}

func (s *CounterStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	rand *rand.Rand,
) (*donburi.Entry, core.PartSlotKey) {
	if actingEntry.HasComponent(component.AIComponent) {
		ai := component.AIComponent.Get(actingEntry)
		if ai.TargetHistory.LastAttacker != nil {
			lastAttacker := ai.TargetHistory.LastAttacker
			// 攻撃者がまだ有効で、破壊されていないことを確認
			if lastAttacker.Valid() && component.StateComponent.Get(lastAttacker).CurrentState != core.StateBroken {
				targetPart := targetSelector.SelectPartToDamage(lastAttacker, actingEntry, rand)
				if targetPart != nil {
					slotKey := partInfoProvider.FindPartSlot(lastAttacker, targetPart)
					if slotKey != "" {
						log.Printf("AI戦略 [カウンター]: %s が最後に攻撃してきた %s を狙います。", component.SettingsComponent.Get(actingEntry).Name, component.SettingsComponent.Get(lastAttacker).Name)
						return lastAttacker, slotKey
					}
				}
			}
		}
	}
	// 履歴がない、または攻撃者が無効な場合はランダムにフォールバック
	log.Printf("AI戦略 [カウンター]: 履歴がないため、ランダムターゲットにフォールバックします。")
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider, rand)
}

// GuardStrategy は自チームのリーダーを最後に攻撃した敵を狙います。
type GuardStrategy struct{}

func (s *GuardStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	rand *rand.Rand,
) (*donburi.Entry, core.PartSlotKey) {
	settings := component.SettingsComponent.Get(actingEntry)
	leader := entity.FindLeader(world, settings.Team)

	if leader != nil && leader != actingEntry {
		if leader.HasComponent(component.AIComponent) {
			ai := component.AIComponent.Get(leader)
			if ai.TargetHistory.LastAttacker != nil {
				lastAttacker := ai.TargetHistory.LastAttacker
				if lastAttacker.Valid() && component.StateComponent.Get(lastAttacker).CurrentState != core.StateBroken {
					targetPart := targetSelector.SelectPartToDamage(lastAttacker, actingEntry, rand)
					if targetPart != nil {
						slotKey := partInfoProvider.FindPartSlot(lastAttacker, targetPart)
						if slotKey != "" {
							log.Printf("AI戦略 [ガード]: %s がリーダーを攻撃した %s を狙います。", component.SettingsComponent.Get(actingEntry).Name, component.SettingsComponent.Get(lastAttacker).Name)
							return lastAttacker, slotKey
						}
					}
				}
			}
		}
	}
	// リーダーがいない、自分がリーダー、または履歴がない場合はランバック
	log.Printf("AI戦略 [ガード]: 履歴がないため、ランダムターゲットにフォールバックします。")
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider, rand)
}

// FocusStrategy は自分が最後に攻撃をヒットさせたパーツを再度狙います。
type FocusStrategy struct{}

func (s *FocusStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	rand *rand.Rand,
) (*donburi.Entry, core.PartSlotKey) {
	if actingEntry.HasComponent(component.AIComponent) {
		ai := component.AIComponent.Get(actingEntry)
		if ai.LastActionHistory.LastHitTarget != nil && ai.LastActionHistory.LastHitPartSlot != "" {
			lastTarget := ai.LastActionHistory.LastHitTarget
			lastSlot := ai.LastActionHistory.LastHitPartSlot

			// ターゲットが有効で、そのパーツがまだ破壊されていないか確認
			if lastTarget.Valid() && component.StateComponent.Get(lastTarget).CurrentState != core.StateBroken {
				if parts := component.PartsComponent.Get(lastTarget); parts != nil {
					if partInst, ok := parts.Map[lastSlot]; ok && !partInst.IsBroken {
						log.Printf("AI戦略 [フォーカス]: %s が前回攻撃した %s の %s を再度狙います。", component.SettingsComponent.Get(actingEntry).Name, component.SettingsComponent.Get(lastTarget).Name, lastSlot)
						return lastTarget, lastSlot
					}
				}
			}
		}
	}
	// 履歴がない、またはターゲットが無効な場合はランダムにフォールバック
	log.Printf("AI戦略 [フォーカス]: 履歴がないため、ランダムターゲットにフォールバックします。")
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider, rand)
}

// AssistStrategy は味方が最後に攻撃したパーツを狙います。
type AssistStrategy struct{}

func (s *AssistStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	rand *rand.Rand,
) (*donburi.Entry, core.PartSlotKey) {
	actingSettings := component.SettingsComponent.Get(actingEntry)
	var assistTarget *donburi.Entry
	var assistSlot core.PartSlotKey

	// 自分以外の味方をクエリ
	query.NewQuery(filter.Contains(component.SettingsComponent)).Each(world, func(teammate *donburi.Entry) {
		if assistTarget != nil { // 既にターゲットを見つけていれば終了
			return
		}
		if teammate == actingEntry { // 自分自身を除外
			return
		}

		teammateSettings := component.SettingsComponent.Get(teammate)
		if teammateSettings.Team == actingSettings.Team {
			if teammate.HasComponent(component.AIComponent) {
				ai := component.AIComponent.Get(teammate)
				if ai.LastActionHistory.LastHitTarget != nil && ai.LastActionHistory.LastHitPartSlot != "" {
					lastTarget := ai.LastActionHistory.LastHitTarget
					lastSlot := ai.LastActionHistory.LastHitPartSlot
					if lastTarget.Valid() && component.StateComponent.Get(lastTarget).CurrentState != core.StateBroken {
						if parts := component.PartsComponent.Get(lastTarget); parts != nil {
							if partInst, ok := parts.Map[lastSlot]; ok && !partInst.IsBroken {
								assistTarget = lastTarget
								assistSlot = lastSlot
							}
						}
					}
				}
			}
		}
	})

	if assistTarget != nil {
		log.Printf("AI戦略 [アシスト]: %s が味方の攻撃に続き %s の %s を狙います。", actingSettings.Name, component.SettingsComponent.Get(assistTarget).Name, assistSlot)
		return assistTarget, assistSlot
	}

	// 履歴がない場合はランダムにフォールバック
	log.Printf("AI戦略 [アシスト]: 履歴がないため、ランダムターゲットにフォールバックします。")
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider, rand)
}