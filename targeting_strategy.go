package main

import (
	"log"
	"math/rand"
	"sort"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// TargetingStrategy はAIのターゲット選択アルゴリズムをカプセル化するインターフェースです。
type TargetingStrategy interface {
	SelectTarget(
		world donburi.World,
		actingEntry *donburi.Entry,
		targetSelector *TargetSelector,
		partInfoProvider *PartInfoProvider,
	) (*donburi.Entry, PartSlotKey)
}

// --- 内部ヘルパー ---

type targetablePart struct {
	entity   *donburi.Entry
	partInst *PartInstanceData
	partDef  *PartDefinition
	slot     PartSlotKey
}

// getAllTargetableParts はAIがターゲット可能な全パーツのインスタンスと定義のリストを返します。
// この関数は ai.go から移動されました。
func getAllTargetableParts(actingEntry *donburi.Entry, targetSelector *TargetSelector, includeHead bool) []targetablePart {
	var allParts []targetablePart
	if targetSelector == nil {
		log.Println("エラー: getAllTargetableParts - targetSelectorがnilです。")
		return allParts
	}
	candidates := targetSelector.GetTargetableEnemies(actingEntry)

	for _, enemyEntry := range candidates {
		partsComp := PartsComponent.Get(enemyEntry)
		if partsComp == nil {
			continue
		}
		for slotKey, partInst := range partsComp.Map {
			if partInst.IsBroken {
				continue
			}
			partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID)
			if !defFound {
				log.Printf("警告: getAllTargetableParts - PartDefinition %s が見つかりません。", partInst.DefinitionID)
				continue
			}
			if !includeHead && partDef.Type == PartTypeHead {
				continue
			}
			allParts = append(allParts, targetablePart{
				entity:   enemyEntry,
				partInst: partInst,
				partDef:  partDef,
				slot:     slotKey,
			})
		}
	}
	return allParts
}

// --- 戦略の実装 ---

// CrusherStrategy は最も装甲の高いパーツを狙います。
type CrusherStrategy struct{}

func (s *CrusherStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, targetSelector, false) // 脚部以外、頭部以外
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(actingEntry, targetSelector, true) // 脚部以外 (頭部含む)
	}
	if len(targetParts) == 0 {
		return nil, ""
	}

	sort.Slice(targetParts, func(i, j int) bool {
		return targetParts[i].partInst.CurrentArmor > targetParts[j].partInst.CurrentArmor
	})

	selected := targetParts[0]
	return selected.entity, selected.slot
}

// HunterStrategy は最も装甲の低いパーツを狙います。
type HunterStrategy struct{}

func (s *HunterStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, targetSelector, false) // 脚部以外、頭部以外
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(actingEntry, targetSelector, true) // 脚部以外 (頭部含む)
	}
	if len(targetParts) == 0 {
		return nil, ""
	}

	sort.Slice(targetParts, func(i, j int) bool {
		return targetParts[i].partInst.CurrentArmor < targetParts[j].partInst.CurrentArmor
	})

	selected := targetParts[0]
	return selected.entity, selected.slot
}

// JokerStrategy はランダムなパーツを狙います。
type JokerStrategy struct{}

func (s *JokerStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) (*donburi.Entry, PartSlotKey) {
	allEnemyParts := getAllTargetableParts(actingEntry, targetSelector, true) // 脚部以外 (頭部含む)
	if len(allEnemyParts) == 0 {
		return nil, ""
	}

	idx := rand.Intn(len(allEnemyParts))
	return allEnemyParts[idx].entity, allEnemyParts[idx].slot
}

// LeaderStrategy はリーダーのパーツをランダムに狙います。
type LeaderStrategy struct{}

func (s *LeaderStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) (*donburi.Entry, PartSlotKey) {
	if targetSelector == nil || partInfoProvider == nil {
		log.Println("エラー: LeaderStrategy.SelectTarget - targetSelector または partInfoProvider がnilです。")
		return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider) // フォールバック
	}

	opponentTeamID := targetSelector.GetOpponentTeam(actingEntry)
	leader := FindLeader(world, opponentTeamID)

	if leader != nil && StateComponent.Get(leader).Current != StateTypeBroken {
		targetPart := targetSelector.SelectRandomPartToDamage(leader)
		if targetPart != nil {
			slotKey := partInfoProvider.FindPartSlot(leader, targetPart)
			if slotKey != "" {
				return leader, slotKey
			}
		}
	}
	// リーダーを狙えない場合はランダムにフォールバック
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider)
}

// ChaseStrategy は最も推進力の高い脚部パーツを狙います。
type ChaseStrategy struct{}

func (s *ChaseStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, targetSelector, true)
	if len(targetParts) == 0 {
		return nil, ""
	}

	var legParts []targetablePart
	for _, p := range targetParts {
		if p.partDef.Type == PartTypeLegs {
			legParts = append(legParts, p)
		}
	}

	if len(legParts) > 0 {
		sort.Slice(legParts, func(i, j int) bool {
			return legParts[i].partDef.Propulsion > legParts[j].partDef.Propulsion
		})
		// 最も推進力の高い脚部が複数ある場合、その中からランダムに選ぶ
		maxPropulsion := legParts[0].partDef.Propulsion
		var candidates []targetablePart
		for _, p := range legParts {
			if p.partDef.Propulsion == maxPropulsion {
				candidates = append(candidates, p)
			}
		}
		selected := candidates[rand.Intn(len(candidates))]
		return selected.entity, selected.slot
	}

	// 脚部が全破壊の場合、左腕→右腕→頭部の順で狙う
	// このロジックは単純化のため、ランダムな非脚部パーツを狙うことで代替します
	var otherParts []targetablePart
	for _, p := range targetParts {
		if p.partDef.Type != PartTypeLegs {
			otherParts = append(otherParts, p)
		}
	}
	if len(otherParts) > 0 {
		selected := otherParts[rand.Intn(len(otherParts))]
		return selected.entity, selected.slot
	}

	// フォールバック
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider)
}

// DuelStrategy は攻撃系腕パーツ（射撃/格闘）を優先して狙います。
type DuelStrategy struct{}

func (s *DuelStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, targetSelector, true)
	if len(targetParts) == 0 {
		return nil, ""
	}

	var attackArmParts []targetablePart
	for _, p := range targetParts {
		if (p.partDef.Type == PartTypeLArm || p.partDef.Type == PartTypeRArm) &&
			(p.partDef.Category == CategoryShoot || p.partDef.Category == CategoryMelee) {
			attackArmParts = append(attackArmParts, p)
		}
	}

	if len(attackArmParts) > 0 {
		selected := attackArmParts[rand.Intn(len(attackArmParts))]
		return selected.entity, selected.slot
	}

	// 攻撃系腕パーツがない場合、ランダムなパーツを狙う
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider)
}

// InterceptStrategy は非攻撃系パーツ（射撃/格闘以外）を優先して狙います。
type InterceptStrategy struct{}

func (s *InterceptStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, targetSelector, true)
	if len(targetParts) == 0 {
		return nil, ""
	}

	var nonAttackParts []targetablePart
	for _, p := range targetParts {
		if p.partDef.Category != CategoryShoot && p.partDef.Category != CategoryMelee {
			nonAttackParts = append(nonAttackParts, p)
		}
	}

	if len(nonAttackParts) > 0 {
		selected := nonAttackParts[rand.Intn(len(nonAttackParts))]
		return selected.entity, selected.slot
	}

	// 非攻撃系パーツがない場合、ランダムなパーツを狙う
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider)
}

// --- 履歴ベースの戦略 ---

// CounterStrategy は自分を最後に攻撃した敵を狙います。
type CounterStrategy struct{}

func (s *CounterStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) (*donburi.Entry, PartSlotKey) {
	if history := TargetHistoryComponent.Get(actingEntry); history != nil && history.LastAttacker != nil {
		lastAttacker := history.LastAttacker
		// 攻撃者がまだ有効で、破壊されていないことを確認
		if lastAttacker.Valid() && StateComponent.Get(lastAttacker).Current != StateTypeBroken {
			// 攻撃者のランダムなパーツを狙う
			targetPart := targetSelector.SelectRandomPartToDamage(lastAttacker)
			if targetPart != nil {
				slotKey := partInfoProvider.FindPartSlot(lastAttacker, targetPart)
				if slotKey != "" {
					log.Printf("AI戦略 [カウンター]: %s が最後に攻撃してきた %s を狙います。", SettingsComponent.Get(actingEntry).Name, SettingsComponent.Get(lastAttacker).Name)
					return lastAttacker, slotKey
				}
			}
		}
	}
	// 履歴がない、または攻撃者が無効な場合はランダムにフォールバック
	log.Printf("AI戦略 [カウンター]: 履歴がないため、ランダムターゲットにフォールバックします。")
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider)
}

// GuardStrategy は自チームのリーダーを最後に攻撃した敵を狙います。
type GuardStrategy struct{}

func (s *GuardStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) (*donburi.Entry, PartSlotKey) {
	settings := SettingsComponent.Get(actingEntry)
	leader := FindLeader(world, settings.Team)

	if leader != nil && leader != actingEntry { // リーダーがいて、自分自身ではない場合
		if history := TargetHistoryComponent.Get(leader); history != nil && history.LastAttacker != nil {
			lastAttacker := history.LastAttacker
			if lastAttacker.Valid() && StateComponent.Get(lastAttacker).Current != StateTypeBroken {
				targetPart := targetSelector.SelectRandomPartToDamage(lastAttacker)
				if targetPart != nil {
					slotKey := partInfoProvider.FindPartSlot(lastAttacker, targetPart)
					if slotKey != "" {
						log.Printf("AI戦略 [ガード]: %s がリーダーを攻撃した %s を狙います。", SettingsComponent.Get(actingEntry).Name, SettingsComponent.Get(lastAttacker).Name)
						return lastAttacker, slotKey
					}
				}
			}
		}
	}
	// リーダーがいない、自分がリーダー、または履歴がない場合はランダムにフォールバック
	log.Printf("AI戦略 [ガード]: 履歴がないため、ランダムターゲットにフォールバックします。")
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider)
}

// FocusStrategy は自分が最後に攻撃をヒットさせたパーツを再度狙います。
type FocusStrategy struct{}

func (s *FocusStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) (*donburi.Entry, PartSlotKey) {
	if history := LastActionHistoryComponent.Get(actingEntry); history != nil && history.LastHitTarget != nil && history.LastHitPartSlot != "" {
		lastTarget := history.LastHitTarget
		lastSlot := history.LastHitPartSlot

		// ターゲットが有効で、そのパーツがまだ破壊されていないか確認
		if lastTarget.Valid() && StateComponent.Get(lastTarget).Current != StateTypeBroken {
			if parts := PartsComponent.Get(lastTarget); parts != nil {
				if partInst, ok := parts.Map[lastSlot]; ok && !partInst.IsBroken {
					log.Printf("AI戦略 [フォーカス]: %s が前回攻撃した %s の %s を再度狙います。", SettingsComponent.Get(actingEntry).Name, SettingsComponent.Get(lastTarget).Name, lastSlot)
					return lastTarget, lastSlot
				}
			}
		}
	}
	// 履歴がない、またはターゲットが無効な場合はランダムにフォールバック
	log.Printf("AI戦略 [フォーカス]: 履歴がないため、ランダムターゲットにフォールバックします。")
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider)
}

// AssistStrategy は味方が最後に攻撃したパーツを狙います。
type AssistStrategy struct{}

func (s *AssistStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) (*donburi.Entry, PartSlotKey) {
	actingSettings := SettingsComponent.Get(actingEntry)
	var assistTarget *donburi.Entry
	var assistSlot PartSlotKey

	// 自分以外の味方をクエリ
	query.NewQuery(filter.Contains(SettingsComponent)).Each(world, func(teammate *donburi.Entry) {
		if assistTarget != nil { // 既にターゲットを見つけていれば終了
			return
		}
		if teammate == actingEntry { // 自分自身を除外
			return
		}

		teammateSettings := SettingsComponent.Get(teammate)
		if teammateSettings.Team == actingSettings.Team {
			if history := LastActionHistoryComponent.Get(teammate); history != nil && history.LastHitTarget != nil && history.LastHitPartSlot != "" {
				lastTarget := history.LastHitTarget
				lastSlot := history.LastHitPartSlot
				if lastTarget.Valid() && StateComponent.Get(lastTarget).Current != StateTypeBroken {
					if parts := PartsComponent.Get(lastTarget); parts != nil {
						if partInst, ok := parts.Map[lastSlot]; ok && !partInst.IsBroken {
							assistTarget = lastTarget
							assistSlot = lastSlot
						}
					}
				}
			}
		}
	})

	if assistTarget != nil {
		log.Printf("AI戦略 [アシスト]: %s が味方の攻撃に続き %s の %s を狙います。", actingSettings.Name, SettingsComponent.Get(assistTarget).Name, assistSlot)
		return assistTarget, assistSlot
	}

	// 履歴がない場合はランダムにフォールバック
	log.Printf("AI戦略 [アシスト]: 履歴がないため、ランダムターゲットにフォールバックします。")
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, targetSelector, partInfoProvider)
}
