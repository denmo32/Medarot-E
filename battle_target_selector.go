package main

import (
	"log"
	"math"
	"sort"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// TargetSelector はターゲット選択やパーツ選択に関連するロジックを担当します。
type TargetSelector struct {
	world            donburi.World
	config           *Config
	partInfoProvider PartInfoProviderInterface
}

// NewTargetSelector は新しい TargetSelector のインスタンスを生成します。
func NewTargetSelector(world donburi.World, config *Config, pip PartInfoProviderInterface) *TargetSelector {
	return &TargetSelector{world: world, config: config, partInfoProvider: pip}
}

// SelectDefensePart は防御に使用するパーツのインスタンスを選択します。
func (ts *TargetSelector) SelectDefensePart(target *donburi.Entry, battleLogic *BattleLogic) *PartInstanceData {
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
		partDef, defFound := ts.partInfoProvider.GetGameDataManager().GetPartDefinition(partInst.DefinitionID)
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

// SelectPartToDamage は、行動者の性格に基づいて攻撃対象のパーツインスタンスを選択します。
func (ts *TargetSelector) SelectPartToDamage(target, actingEntry *donburi.Entry, battleLogic *BattleLogic) *PartInstanceData {
	partsComp := PartsComponent.Get(target)
	if partsComp == nil {
		return nil
	}

	vulnerableInstances := []*PartInstanceData{}
	slots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	for _, s := range slots {
		if partInst, ok := partsComp.Map[s]; ok && partInst != nil && !partInst.IsBroken {
			vulnerableInstances = append(vulnerableInstances, partInst)
		}
	}
	if len(vulnerableInstances) == 0 {
		return nil
	}

	// 行動者の性格を取得
	personality := "ジョーカー" // デフォルト
	if actingEntry.HasComponent(MedalComponent) {
		personality = MedalComponent.Get(actingEntry).Personality
	}

	switch personality {
	case "クラッシャー":
		sort.Slice(vulnerableInstances, func(i, j int) bool {
			return vulnerableInstances[i].CurrentArmor > vulnerableInstances[j].CurrentArmor
		})
		return vulnerableInstances[0]
	case "ハンター":
		sort.Slice(vulnerableInstances, func(i, j int) bool {
			return vulnerableInstances[i].CurrentArmor < vulnerableInstances[j].CurrentArmor
		})
		return vulnerableInstances[0]
	default: // "ジョーカー" やその他の性格
		return vulnerableInstances[battleLogic.rand.Intn(len(vulnerableInstances))]
	}
}

// FindClosestEnemy は指定されたエンティティに最も近い敵エンティティを見つけます。
func (ts *TargetSelector) FindClosestEnemy(actingEntry *donburi.Entry, battleLogic *BattleLogic) *donburi.Entry {
	var closestEnemy *donburi.Entry
	var minDiff float32 = math.MaxFloat32 // float32 を使用するため、MaxFloat32 に変更

	actingX := ts.partInfoProvider.CalculateMedarotXPosition(actingEntry, float32(ts.config.UI.Screen.Width))

	for _, enemy := range ts.GetTargetableEnemies(actingEntry) {
		enemyX := ts.partInfoProvider.CalculateMedarotXPosition(enemy, float32(ts.config.UI.Screen.Width))
		diff := float32(math.Abs(float64(actingX - enemyX))) // float32 の差を計算
		if diff < minDiff {
			minDiff = diff
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
		if StateComponent.Get(entry).CurrentState == StateBroken {
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
