package main

import (
	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
)

// ChargeInitiationSystem はチャージ状態の開始ロジックをカプセル化します。
type ChargeInitiationSystem struct {
	world            donburi.World
	partInfoProvider PartInfoProviderInterface
	gameDataManager  *GameDataManager
}

// NewChargeInitiationSystem は新しいChargeInitiationSystemのインスタンスを生成します。
func NewChargeInitiationSystem(world donburi.World, partInfoProvider PartInfoProviderInterface) *ChargeInitiationSystem {
	return &ChargeInitiationSystem{
		world:            world,
		partInfoProvider: partInfoProvider,
		gameDataManager:  partInfoProvider.GetGameDataManager(),
	}
}

// StartCharge はチャージ状態を開始するための主要なロジックを実行します。
func (s *ChargeInitiationSystem) StartCharge(
	entry *donburi.Entry,
	partKey component.PartSlotKey,
	targetEntry *donburi.Entry,
	targetPartSlot component.PartSlotKey,
) bool {
	state := StateComponent.Get(entry)
	if state.CurrentState != component.StateIdle {
		return false // アイドル状態でない場合は開始できない
	}

	partsComp := PartsComponent.Get(entry)
	actingPartInstance := partsComp.Map[partKey]

	if actingPartInstance == nil {
		return false
	}
	actingPartDef, defFound := s.partInfoProvider.GetGameDataManager().GetPartDefinition(actingPartInstance.DefinitionID)
	if !defFound {
		return false
	}

	intent := ActionIntentComponent.Get(entry)
	intent.SelectedPartKey = partKey
	intent.PendingEffects = make([]interface{}, 0) // 既存の効果をクリア

	target := TargetComponent.Get(entry)
	if targetEntry != nil { // targetEntry が nil でない場合のみIDをセット
		target.TargetEntity = targetEntry.Entity()
	} else {
		target.TargetEntity = 0 // nil の場合はゼロ値
	}
	target.TargetPartSlot = targetPartSlot

	// カテゴリに基づいてターゲット決定方針を設定
	switch actingPartDef.Category {
	case component.CategoryRanged, component.CategoryIntervention:
		target.Policy = component.PolicyPreselected
	case component.CategoryMelee:
		target.Policy = component.PolicyClosestAtExecution
	default:
		target.Policy = component.PolicyPreselected // デフォルト
	}

	// 1. 計算式の取得
	formula, ok := s.gameDataManager.Formulas[actingPartDef.Trait]
	if !ok {
		// 警告ログは削除
	} else {
		// 2. 計算式に基づいて自身に適用されるデバフ効果を生成
		for _, debuffInfo := range formula.UserDebuffs {
			// ログは削除
			var effectData interface{}
			switch debuffInfo.Type {
			case component.DebuffTypeEvasion:
				effectData = &component.EvasionDebuffEffectData{Multiplier: debuffInfo.Multiplier}
			case component.DebuffTypeDefense:
				effectData = &component.DefenseDebuffEffectData{Multiplier: debuffInfo.Multiplier}
			default:
				// ログは削除
			}
			if effectData != nil {
				intent.PendingEffects = append(intent.PendingEffects, effectData)
			}
		}
	}

	if actingPartDef.Category == component.CategoryRanged {
		// targetEntry が有効なエンティティであるか、または破壊されていないかを確認
		if targetEntry == nil || !targetEntry.Valid() || StateComponent.Get(targetEntry).CurrentState == component.StateBroken {
			return false
		}
		// ログは削除
	} else {
		// ログは削除
	}

	baseSeconds := float64(actingPartDef.Charge)
	// 新しい共通関数を呼び出す
	totalTicks := s.partInfoProvider.CalculateGaugeDuration(baseSeconds, entry)

	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	gauge.ProgressCounter = 0

	state.CurrentState = component.StateCharging
	return true
}
