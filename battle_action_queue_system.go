package main

import (
	"context"
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
		if battleLogic == nil || battleLogic.PartInfoProvider == nil {
			log.Println("UpdateActionQueueSystem: ソート中にbattleLogicまたはpartInfoProviderがnilです")
			return false
		}
		propI := battleLogic.PartInfoProvider.GetOverallPropulsion(actionQueueComp.Queue[i])
		propJ := battleLogic.PartInfoProvider.GetOverallPropulsion(actionQueueComp.Queue[j])
		return propI > propJ
	})

	if len(actionQueueComp.Queue) > 0 {
		actingEntry := actionQueueComp.Queue[0]
		actionQueueComp.Queue = actionQueueComp.Queue[1:]

		intent := ActionIntentComponent.Get(actingEntry)
		partsComp := PartsComponent.Get(actingEntry)
		actingPartInst := partsComp.Map[intent.SelectedPartKey]
		actingPartDef, _ := GlobalGameDataManager.GetPartDefinition(actingPartInst.DefinitionID)

		handler := GetActionHandlerForCategory(actingPartDef.Category)

		if handler != nil {
			actionResult := handler.Execute(actingEntry, world, intent, battleLogic, gameConfig)
			battleLogic.ApplyDefenseAndDamage(&actionResult) // 新しいメソッドを呼び出す
			results = append(results, actionResult)
		} else {
			// Handle error: no handler found
			log.Printf("No action handler found for category: %s", actingPartDef.Category)
		}
	}
	return results, nil
}

// StartCooldownSystem はクールダウン状態を開始します。
func StartCooldownSystem(entry *donburi.Entry, world donburi.World, gameConfig *Config, partInfoProvider *PartInfoProvider) {
	intent := ActionIntentComponent.Get(entry)
	partsComp := PartsComponent.Get(entry)
	var actingPartDef *PartDefinition

	if actingPartInstance, ok := partsComp.Map[intent.SelectedPartKey]; ok {
		if def, defFound := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID); defFound {
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
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}

	propulsion := 1
	if partInfoProvider != nil {
		legsInstance := partsComp.Map[PartSlotLegs]
		if legsInstance != nil && !legsInstance.IsBroken {
			propulsion = partInfoProvider.GetOverallPropulsion(entry)
		}
	} else {
		log.Println("警告: StartCooldownSystem - partInfoProviderがnilです。")
	}

	balanceConfig := &gameConfig.Balance
	propulsionFactor := 1.0 + (float64(propulsion) * balanceConfig.Time.PropulsionEffectRate)
	totalTicks := (baseSeconds * 60.0) / (balanceConfig.Time.GameSpeedMultiplier * propulsionFactor)

	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	if gauge.TotalDuration < 1 {
		gauge.TotalDuration = 1
	}
	gauge.ProgressCounter = 0
	gauge.CurrentGauge = 0

	state := StateComponent.Get(entry)
	err := state.FSM.Event(context.Background(), "cooldown", entry)
	if err != nil {
		log.Printf("Error starting cooldown for %s: %v", SettingsComponent.Get(entry).Name, err)
	}
}

// StartCharge はチャージ状態を開始します。
func StartCharge(
	entry *donburi.Entry,
	partKey PartSlotKey,
	targetEntry *donburi.Entry,
	targetPartSlot PartSlotKey,
	world donburi.World,
	gameConfig *Config,
	partInfoProvider *PartInfoProvider,
) bool {
	state := StateComponent.Get(entry)
	if !state.FSM.Is(string(StateIdle)) {
		return false // アイドル状態でない場合は開始できない
	}

	partsComp := PartsComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	actingPartInstance := partsComp.Map[partKey]

	if actingPartInstance == nil {
		log.Printf("%s: 選択されたパーツ %s は存在しません。", settings.Name, partKey)
		return false
	}
	actingPartDef, defFound := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID)
	if !defFound {
		log.Printf("%s: パーツ定義(%s)が見つかりません。", settings.Name, actingPartInstance.DefinitionID)
		return false
	}

	intent := ActionIntentComponent.Get(entry)
	intent.SelectedPartKey = partKey

	target := TargetComponent.Get(entry)
	target.TargetEntity = targetEntry
	target.TargetPartSlot = targetPartSlot

	// 1. 計算式の取得
	formula, ok := FormulaManager[actingPartDef.Trait]
	if !ok {
		log.Printf("警告: 特性 '%s' に対応する計算式が見つかりません。", actingPartDef.Trait)
	} else {
		// 2. 計算式に基づいて自身にデバフを適用
		for _, debuff := range formula.UserDebuffs {
			log.Printf("%s が %s 特性効果（チャージ時デバフ）を発動。", settings.Name, formula.ID)
			switch debuff.Type {
			case DebuffTypeEvasion:
				donburi.Add(entry, EvasionDebuffComponent, &EvasionDebuff{Multiplier: debuff.Multiplier})
			case DebuffTypeDefense:
				donburi.Add(entry, DefenseDebuffComponent, &DefenseDebuff{Multiplier: debuff.Multiplier})
			}
		}
	}

	if actingPartDef.Category == CategoryShoot {
		if targetEntry == nil || StateComponent.Get(targetEntry).FSM.Is(string(StateBroken)) {
			log.Printf("%s: [射撃] ターゲットが存在しないか破壊されています。", settings.Name)
			return false
		}
		log.Printf("%sは%sで%sの%sを狙う！", settings.Name, actingPartDef.PartName, SettingsComponent.Get(targetEntry).Name, targetPartSlot)
	} else {
		log.Printf("%sは%sで攻撃準備！", settings.Name, actingPartDef.PartName)
	}

	propulsion := 1
	if partInfoProvider != nil {
		legsInstance := partsComp.Map[PartSlotLegs]
		if legsInstance != nil && !legsInstance.IsBroken {
			propulsion = partInfoProvider.GetOverallPropulsion(entry)
		}
	} else {
		log.Println("警告: StartCharge - partInfoProviderがnilです。")
	}

	balanceConfig := &gameConfig.Balance
	baseSeconds := float64(actingPartDef.Charge)
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}
	// balanceConfig := &gameConfig.Balance // ここが重複していた
	propulsionFactor := 1.0 + (float64(propulsion) * balanceConfig.Time.PropulsionEffectRate)
	totalTicks := (baseSeconds * 60.0) / (balanceConfig.Time.GameSpeedMultiplier * propulsionFactor)

	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	if gauge.TotalDuration < 1 {
		gauge.TotalDuration = 1
	}
	err := state.FSM.Event(context.Background(), "charge", entry)
	if err != nil {
		log.Printf("Error starting charge for %s: %v", settings.Name, err)
		return false
	}
	return true
}
