package battle

import (
	"log"
	"math"
	"medarot-ebiten/internal/game"
	"sort"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// TargetSelector はターゲット選択やパーツ選択に関連するロジックを担当します。
type TargetSelector struct {
	world  donburi.World
	config *game.Config
	// partInfoProvider *PartInfoProvider // 削除
}

// NewTargetSelector は新しい TargetSelector のインスタンスを生成します。
func NewTargetSelector(world donburi.World, config *game.Config) *TargetSelector {
	return &TargetSelector{world: world, config: config}
}



// SelectDefensePart は防御に使用するパーツのインスタンスを選択します。
func (ts *TargetSelector) SelectDefensePart(target *donburi.Entry, battleLogic *BattleLogic) *game.PartInstanceData {
	partsComp := game.PartsComponent.Get(target)
	if partsComp == nil {
		return nil
	}
	partsMap := partsComp.Map // map[PartSlotKey]*PartInstanceData

	var bestPartInstance *game.PartInstanceData
	maxArmor := -1 // Initialize with a value lower than any possible armor

	// 腕部と脚部を優先して、最も装甲の高いパーツを探す
	for _, partInst := range partsMap {
		if partInst.IsBroken {
			continue
		}
		partDef, defFound := battleLogic.GetPartInfoProvider().gameDataManager.GetPartDefinition(partInst.DefinitionID)
		if !defFound {
			log.Printf("SelectDefensePart: PartDefinition not found for ID %s", partInst.DefinitionID)
			continue
		}

		switch partDef.Type {
		case game.PartTypeRArm, game.PartTypeLArm, game.PartTypeLegs:
			if partInst.CurrentArmor > maxArmor {
				maxArmor = partInst.CurrentArmor
				bestPartInstance = partInst
			}
		}
	}

	// 腕部と脚部が全て破壊されている場合、頭部をチェック
	if bestPartInstance == nil {
		if headPart, ok := partsMap[game.PartSlotHead]; ok && !headPart.IsBroken {
			bestPartInstance = headPart
		}
	}

	return bestPartInstance
}

// SelectPartToDamage は、行動者の性格に基づいて攻撃対象のパーツインスタンスを選択します。
func (ts *TargetSelector) SelectPartToDamage(target, actingEntry *donburi.Entry, battleLogic *BattleLogic) *game.PartInstanceData {
	partsComp := game.PartsComponent.Get(target)
	if partsComp == nil {
		return nil
	}

	vulnerableInstances := []*game.PartInstanceData{}
	slots := []game.PartSlotKey{game.PartSlotHead, game.PartSlotRightArm, game.PartSlotLeftArm, game.PartSlotLegs}
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
	if actingEntry.HasComponent(game.MedalComponent) {
		personality = game.MedalComponent.Get(actingEntry).Personality
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
		return vulnerableInstances[globalRand.Intn(len(vulnerableInstances))]
	}
}

// FindClosestEnemy は指定されたエンティティに最も近い敵エンティティを見つけます。
func (ts *TargetSelector) FindClosestEnemy(actingEntry *donburi.Entry, battleLogic *BattleLogic) *donburi.Entry {
	var closestEnemy *donburi.Entry
	var minDiff float32 = math.MaxFloat32 // float32 を使用するため、MaxFloat32 に変更

	actingX := battleLogic.GetPartInfoProvider().CalculateMedarotXPosition(actingEntry, float32(ts.config.UI.Screen.Width))

	for _, enemy := range ts.GetTargetableEnemies(actingEntry) {
		enemyX := battleLogic.GetPartInfoProvider().CalculateMedarotXPosition(enemy, float32(ts.config.UI.Screen.Width))
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
	query.NewQuery(filter.Contains(game.SettingsComponent)).Each(ts.world, func(entry *donburi.Entry) {
		if game.StateComponent.Get(entry).FSM.Is(string(game.StateBroken)) {
			return
		}
		settings := game.SettingsComponent.Get(entry)
		if settings.Team == opponentTeamID {
			candidates = append(candidates, entry)
		}
	})

	sort.Slice(candidates, func(i, j int) bool {
		iSettings := game.SettingsComponent.Get(candidates[i])
		jSettings := game.SettingsComponent.Get(candidates[j])
		return iSettings.DrawIndex < jSettings.DrawIndex
	})
	return candidates
}

// GetOpponentTeam は指定されたエンティティの敵チームIDを返します。
func (ts *TargetSelector) GetOpponentTeam(actingEntry *donburi.Entry) game.TeamID {
	if game.SettingsComponent.Get(actingEntry).Team == game.Team1 {
		return game.Team2
	}
	return game.Team1
}
