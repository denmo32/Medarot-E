package main

import (
	"log"
	"sort"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// --- 内部ヘルパー ---

// getAllTargetableParts はAIがターゲット可能な全パーツのインスタンスと定義のリストを返します。
// この関数は ai.go から移動されました。
func getAllTargetableParts(actingEntry *donburi.Entry, battleLogic *BattleLogic, includeHead bool) []targetablePart {
	var allParts []targetablePart
	if battleLogic.GetTargetSelector() == nil { // targetSelector を battleLogic から取得
		log.Println("エラー: getAllTargetableParts - targetSelectorがnilです。")
		return allParts
	}
	candidates := battleLogic.GetTargetSelector().GetTargetableEnemies(actingEntry) // targetSelector を battleLogic から取得

	for _, enemyEntry := range candidates {
		partsComp := PartsComponent.Get(enemyEntry)
		if partsComp == nil {
			continue
		}
		for slotKey, partInst := range partsComp.Map {
			if partInst.IsBroken {
				continue
			}
			partDef, defFound := battleLogic.GetPartInfoProvider().GetGameDataManager().GetPartDefinition(partInst.DefinitionID) // GlobalGameDataManager を battleLogic から取得
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
	battleLogic *BattleLogic, // battleLogic を追加
) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, battleLogic, false)
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(actingEntry, battleLogic, true)
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
	battleLogic *BattleLogic, // battleLogic を追加
) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, battleLogic, false)
	if len(targetParts) == 0 {
		targetParts = getAllTargetableParts(actingEntry, battleLogic, true)
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
	battleLogic *BattleLogic, // battleLogic を追加
) (*donburi.Entry, PartSlotKey) {
	allEnemyParts := getAllTargetableParts(actingEntry, battleLogic, true)
	if len(allEnemyParts) == 0 {
		return nil, ""
	}

	idx := battleLogic.rand.Intn(len(allEnemyParts))
	return allEnemyParts[idx].entity, allEnemyParts[idx].slot
}

// LeaderStrategy はリーダーのパーツをランダムに狙います。
type LeaderStrategy struct{}

func (s *LeaderStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	battleLogic *BattleLogic, // battleLogic を追加
) (*donburi.Entry, PartSlotKey) {
	if battleLogic.GetTargetSelector() == nil || battleLogic.GetPartInfoProvider() == nil { // targetSelector, partInfoProvider を battleLogic から取得
		log.Println("エラー: LeaderStrategy.SelectTarget - targetSelector または partInfoProvider がnilです。")
		return (&JokerStrategy{}).SelectTarget(world, actingEntry, battleLogic) // フォールバック
	}

	opponentTeamID := battleLogic.GetTargetSelector().GetOpponentTeam(actingEntry) // targetSelector を battleLogic から取得
	leader := FindLeader(world, opponentTeamID)

	if leader != nil && StateComponent.Get(leader).CurrentState != StateBroken {
		targetPart := battleLogic.GetTargetSelector().SelectPartToDamage(leader, actingEntry, battleLogic) // battleLogic を追加
		if targetPart != nil {
			slotKey := battleLogic.GetPartInfoProvider().FindPartSlot(leader, targetPart) // partInfoProvider を battleLogic から取得
			if slotKey != "" {
				return leader, slotKey
			}
		}
	}
	// リーダーを狙えない場合はランダムにフォールバック
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, battleLogic) // battleLogic を追加
}

// ChaseStrategy は最も推進力の高い脚部パーツを狙います。
type ChaseStrategy struct{}

func (s *ChaseStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	battleLogic *BattleLogic, // battleLogic を追加
) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, battleLogic, true)
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
		selected := candidates[battleLogic.rand.Intn(len(candidates))]
		return selected.entity, selected.slot
	}

	// 脚部が全破壊の場合、ランダムな非脚部パーツを狙う
	var otherParts []targetablePart
	for _, p := range targetParts {
		if p.partDef.Type != PartTypeLegs {
			otherParts = append(otherParts, p)
		}
	}
	if len(otherParts) > 0 {
		selected := otherParts[battleLogic.rand.Intn(len(otherParts))]
		return selected.entity, selected.slot
	}

	// フォールバック
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, battleLogic) // battleLogic を追加
}

// DuelStrategy は攻撃系腕パーツ（射撃/格闘）を優先して狙います。
type DuelStrategy struct{}

func (s *DuelStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	battleLogic *BattleLogic, // battleLogic を追加
) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, battleLogic, true)
	if len(targetParts) == 0 {
		return nil, ""
	}

	var attackArmParts []targetablePart
	for _, p := range targetParts {
		if (p.partDef.Type == PartTypeLArm || p.partDef.Type == PartTypeRArm) &&
			(p.partDef.Category == CategoryRanged || p.partDef.Category == CategoryMelee) {
			attackArmParts = append(attackArmParts, p)
		}
	}

		if len(attackArmParts) > 0 {
		selected := attackArmParts[battleLogic.rand.Intn(len(attackArmParts))]
		return selected.entity, selected.slot
	}

	// 攻撃系腕パーツがない場合、ランダムなパーツを狙う
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, battleLogic) // battleLogic を追加
}

// InterceptStrategy は非攻撃系パーツ（射撃/格闘以外）を優先して狙います。
type InterceptStrategy struct{}

func (s *InterceptStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	battleLogic *BattleLogic, // battleLogic を追加
) (*donburi.Entry, PartSlotKey) {
	targetParts := getAllTargetableParts(actingEntry, battleLogic, true)
	if len(targetParts) == 0 {
		return nil, ""
	}

	var nonAttackParts []targetablePart
	for _, p := range targetParts {
		if p.partDef.Category != CategoryRanged && p.partDef.Category != CategoryMelee {
			nonAttackParts = append(nonAttackParts, p)
		}
	}

	if len(nonAttackParts) > 0 {
		selected := nonAttackParts[battleLogic.rand.Intn(len(nonAttackParts))]
		return selected.entity, selected.slot
	}

	// 非攻撃系パーツがない場合、ランダムなパーツを狙う
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, battleLogic) // battleLogic を追加
}

// --- 履歴ベースの戦略 ---

// CounterStrategy は自分を最後に攻撃した敵を狙います。
type CounterStrategy struct{}

func (s *CounterStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	battleLogic *BattleLogic, // battleLogic を追加
) (*donburi.Entry, PartSlotKey) {
	if actingEntry.HasComponent(AIComponent) {
		ai := AIComponent.Get(actingEntry)
		if ai.TargetHistory.LastAttacker != nil {
			lastAttacker := ai.TargetHistory.LastAttacker
			// 攻撃者がまだ有効で、破壊されていないことを確認
			if lastAttacker.Valid() && StateComponent.Get(lastAttacker).CurrentState != StateBroken {
				targetPart := battleLogic.GetTargetSelector().SelectPartToDamage(lastAttacker, actingEntry, battleLogic) // battleLogic を追加
				if targetPart != nil {
					slotKey := battleLogic.GetPartInfoProvider().FindPartSlot(lastAttacker, targetPart) // partInfoProvider を battleLogic から取得
					if slotKey != "" {
						log.Printf("AI戦略 [カウンター]: %s が最後に攻撃してきた %s を狙います。", SettingsComponent.Get(actingEntry).Name, SettingsComponent.Get(lastAttacker).Name)
						return lastAttacker, slotKey
					}
				}
			}
		}
	}
	// 履歴がない、または攻撃者が無効な場合はランダムにフォールバック
	log.Printf("AI戦略 [カウンター]: 履歴がないため、ランダムターゲットにフォールバックします。")
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, battleLogic) // battleLogic を追加
}

// GuardStrategy は自チームのリーダーを最後に攻撃した敵を狙います。
type GuardStrategy struct{}

func (s *GuardStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	battleLogic *BattleLogic, // battleLogic を追加
) (*donburi.Entry, PartSlotKey) {
	settings := SettingsComponent.Get(actingEntry)
	leader := FindLeader(world, settings.Team)

	if leader != nil && leader != actingEntry { // リーダーがいて、自分自身ではない場合
		if leader.HasComponent(AIComponent) {
			ai := AIComponent.Get(leader)
			if ai.TargetHistory.LastAttacker != nil {
				lastAttacker := ai.TargetHistory.LastAttacker
				if lastAttacker.Valid() && StateComponent.Get(lastAttacker).CurrentState != StateBroken {
					targetPart := battleLogic.GetTargetSelector().SelectPartToDamage(lastAttacker, actingEntry, battleLogic) // battleLogic を追加
					if targetPart != nil {
						slotKey := battleLogic.GetPartInfoProvider().FindPartSlot(lastAttacker, targetPart) // partInfoProvider を battleLogic から取得
						if slotKey != "" {
							log.Printf("AI戦略 [ガード]: %s がリーダーを攻撃した %s を狙います。", SettingsComponent.Get(actingEntry).Name, SettingsComponent.Get(lastAttacker).Name)
							return lastAttacker, slotKey
						}
					}
				}
			}
		}
	}
	// リーダーがいない、自分がリーダー、または履歴がない場合はランバック
	log.Printf("AI戦略 [ガード]: 履歴がないため、ランダムターゲットにフォールバックします。")
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, battleLogic) // battleLogic を追加
}

// FocusStrategy は自分が最後に攻撃をヒットさせたパーツを再度狙います。
type FocusStrategy struct{}

func (s *FocusStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	battleLogic *BattleLogic, // battleLogic を追加
) (*donburi.Entry, PartSlotKey) {
	if actingEntry.HasComponent(AIComponent) {
		ai := AIComponent.Get(actingEntry)
		if ai.LastActionHistory.LastHitTarget != nil && ai.LastActionHistory.LastHitPartSlot != "" {
			lastTarget := ai.LastActionHistory.LastHitTarget
			lastSlot := ai.LastActionHistory.LastHitPartSlot

			// ターゲットが有効で、そのパーツがまだ破壊されていないか確認
			if lastTarget.Valid() && StateComponent.Get(lastTarget).CurrentState != StateBroken {
				if parts := PartsComponent.Get(lastTarget); parts != nil {
					if partInst, ok := parts.Map[lastSlot]; ok && !partInst.IsBroken {
						log.Printf("AI戦略 [フォーカス]: %s が前回攻撃した %s の %s を再度狙います。", SettingsComponent.Get(actingEntry).Name, SettingsComponent.Get(lastTarget).Name, lastSlot)
						return lastTarget, lastSlot
					}
				}
			}
		}
	}
	// 履歴がない、またはターゲットが無効な場合はランダムにフォールバック
	log.Printf("AI戦略 [フォーカス]: 履歴がないため、ランダムターゲットにフォールバックします。")
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, battleLogic) // battleLogic を追加
}

// AssistStrategy は味方が最後に攻撃したパーツを狙います。
type AssistStrategy struct{}

func (s *AssistStrategy) SelectTarget(
	world donburi.World,
	actingEntry *donburi.Entry,
	battleLogic *BattleLogic, // battleLogic を追加
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
			if teammate.HasComponent(AIComponent) {
				ai := AIComponent.Get(teammate)
				if ai.LastActionHistory.LastHitTarget != nil && ai.LastActionHistory.LastHitPartSlot != "" {
					lastTarget := ai.LastActionHistory.LastHitTarget
					lastSlot := ai.LastActionHistory.LastHitPartSlot
					if lastTarget.Valid() && StateComponent.Get(lastTarget).CurrentState != StateBroken {
						if parts := PartsComponent.Get(lastTarget); parts != nil {
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
		log.Printf("AI戦略 [アシスト]: %s が味方の攻撃に続き %s の %s を狙います。", actingSettings.Name, SettingsComponent.Get(assistTarget).Name, assistSlot)
		return assistTarget, assistSlot
	}

	// 履歴がない場合はランダムにフォールバック
	log.Printf("AI戦略 [アシスト]: 履歴がないため、ランダムターゲットにフォールバックします。")
	return (&JokerStrategy{}).SelectTarget(world, actingEntry, battleLogic) // battleLogic を追加
}
