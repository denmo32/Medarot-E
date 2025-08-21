package system

import (
	"log"
	"math/rand"
	"sort"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ecs/entity"
	"medarot-ebiten/donburi"
)

// UpdateActionQueueSystem は行動準備完了キューを処理します。
func UpdateActionQueueSystem(
	world donburi.World,
	damageCalculator *DamageCalculator,
	hitCalculator *HitCalculator,
	targetSelector *TargetSelector,
	partInfoProvider PartInfoProviderInterface,
	gameConfig *data.Config,
	statusEffectSystem *StatusEffectSystem,
	postActionEffectSystem *PostActionEffectSystem,
	rand *rand.Rand,
) ([]component.ActionResult, error) {
	actionQueueComp := entity.GetActionQueueComponent(world)
	if len(actionQueueComp.Queue) == 0 {
		return nil, nil
	}
	results := []component.ActionResult{}

	sort.SliceStable(actionQueueComp.Queue, func(i, j int) bool {
		if partInfoProvider == nil {
			log.Println("UpdateActionQueueSystem: ソート中にpartInfoProviderがnilです")
			return false
		}
		propI := partInfoProvider.GetOverallPropulsion(actionQueueComp.Queue[i])
		propJ := partInfoProvider.GetOverallPropulsion(actionQueueComp.Queue[j])
		return propI > propJ
	})

	if len(actionQueueComp.Queue) > 0 {
		actingEntry := actionQueueComp.Queue[0]
		actionQueueComp.Queue = actionQueueComp.Queue[1:]

		executor := NewActionExecutor(world, damageCalculator, hitCalculator, targetSelector, partInfoProvider, gameConfig, statusEffectSystem, postActionEffectSystem, rand)
		actionResult := executor.ExecuteAction(actingEntry)
		results = append(results, actionResult)
	}
	return results, nil
}

// StartCooldownSystem はクールダウン状態を開始します。
func StartCooldownSystem(entry *donburi.Entry, world donburi.World, partInfoProvider PartInfoProviderInterface) {
	intent := component.ActionIntentComponent.Get(entry)
	partsComp := component.PartsComponent.Get(entry)
	var actingPartDef *core.PartDefinition

	if actingPartInstance, ok := partsComp.Map[intent.SelectedPartKey]; ok {
		if def, defFound := partInfoProvider.GetGameDataManager().GetPartDefinition(actingPartInstance.DefinitionID); defFound {
			actingPartDef = def
		} else {
			log.Printf("エラー: StartCooldownSystem - ID %s のPartDefinitionが見つかりません。", actingPartInstance.DefinitionID)
		}
	} else {
		log.Printf("エラー: StartCooldownSystem - キー %s の行動パーツインスタンスが見つかりません。", intent.SelectedPartKey)
	}

	baseSeconds := 1.0
	if actingPartDef != nil {
		baseSeconds = float64(actingPartDef.Cooldown)
	}

	// 新しい共通関数を呼び出す
	totalTicks := partInfoProvider.CalculateGaugeDuration(baseSeconds, entry)

	gauge := component.GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	gauge.ProgressCounter = 0
	gauge.CurrentGauge = 0

	state := component.StateComponent.Get(entry)
	gauge.ProgressCounter = 0
	state.CurrentState = core.StateCooldown
}
