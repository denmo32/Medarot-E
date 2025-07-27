package main

import (
	"log"
	"sort"

	"github.com/yohamta/donburi"
)

// UpdateActionQueueSystem は行動準備完了キューを処理します。
func UpdateActionQueueSystem(
	world donburi.World,
	battleLogic *BattleLogic,
	gameConfig *Config,
) ([]ActionResult, error) {
	actionQueueComp := GetActionQueueComponent(world)
	if len(actionQueueComp.Queue) == 0 {
		return nil, nil
	}
	results := []ActionResult{}

	sort.SliceStable(actionQueueComp.Queue, func(i, j int) bool {
		if battleLogic == nil || battleLogic.GetPartInfoProvider() == nil {
			log.Println("UpdateActionQueueSystem: ソート中にbattleLogicまたはpartInfoProviderがnilです")
			return false
		}
		propI := battleLogic.GetPartInfoProvider().GetOverallPropulsion(actionQueueComp.Queue[i])
		propJ := battleLogic.GetPartInfoProvider().GetOverallPropulsion(actionQueueComp.Queue[j])
		return propI > propJ
	})

	if len(actionQueueComp.Queue) > 0 {
		actingEntry := actionQueueComp.Queue[0]
		actionQueueComp.Queue = actionQueueComp.Queue[1:]

		executor := NewActionExecutor(world, battleLogic, gameConfig)
		actionResult := executor.ExecuteAction(actingEntry)
		results = append(results, actionResult)
	}
	return results, nil
}

// StartCooldownSystem はクールダウン状態を開始します。
func StartCooldownSystem(entry *donburi.Entry, world donburi.World, battleLogic *BattleLogic) {
	intent := ActionIntentComponent.Get(entry)
	partsComp := PartsComponent.Get(entry)
	var actingPartDef *PartDefinition

	if actingPartInstance, ok := partsComp.Map[intent.SelectedPartKey]; ok {
		if def, defFound := battleLogic.GetPartInfoProvider().GetGameDataManager().GetPartDefinition(actingPartInstance.DefinitionID); defFound {
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
	totalTicks := battleLogic.GetPartInfoProvider().CalculateGaugeDuration(baseSeconds, entry)

	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	gauge.ProgressCounter = 0
	gauge.CurrentGauge = 0

	state := StateComponent.Get(entry)
	gauge.ProgressCounter = 0
	state.CurrentState = StateCooldown
}

// StartCharge はチャージ状態を開始します。
func StartCharge(
	entry *donburi.Entry,
	partKey PartSlotKey,
	targetEntry *donburi.Entry,
	targetPartSlot PartSlotKey,
	world donburi.World,
	battleLogic *BattleLogic, // battleLogic を追加
) bool {
	system := NewChargeInitiationSystem(world, battleLogic.GetPartInfoProvider()) // battleLogic から取得
	return system.ProcessChargeRequest(entry, partKey, targetEntry, targetPartSlot)
}
