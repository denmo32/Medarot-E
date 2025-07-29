package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// ChargeInitiationSystem はチャージ状態の開始ロジックをカプセル化します。
type ChargeInitiationSystem struct {
	world            donburi.World
	partInfoProvider PartInfoProviderInterface
	gameDataManager  *GameDataManager
}

// NewChargeInitiationSystem は新しいChargeInitiationSystemのインスタンスを生成します。
func NewChargeInitiationSystem(world donburi.World, partInfoProvider PartInfoProviderInterface, gdm *GameDataManager) *ChargeInitiationSystem {
	return &ChargeInitiationSystem{
		world:            world,
		partInfoProvider: partInfoProvider,
		gameDataManager:  gdm,
	}
}

// ProcessChargeRequest はチャージ状態を開始するための主要なロジックを実行します。
func (s *ChargeInitiationSystem) ProcessChargeRequest(
	entry *donburi.Entry,
	partKey PartSlotKey,
	targetEntry *donburi.Entry,
	targetPartSlot PartSlotKey,
) bool {
	state := StateComponent.Get(entry)
	if state.CurrentState != StateIdle {
		return false // アイドル状態でない場合は開始できない
	}

	partsComp := PartsComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	actingPartInstance := partsComp.Map[partKey]

	if actingPartInstance == nil {
		log.Printf("%s: 選択されたパーツ %s は存在しません。", settings.Name, partKey)
		return false
	}
	actingPartDef, defFound := s.partInfoProvider.GetGameDataManager().GetPartDefinition(actingPartInstance.DefinitionID)
	if !defFound {
		log.Printf("%s: パーツ定義(%s)が見つかりません。", settings.Name, actingPartInstance.DefinitionID)
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
	case CategoryRanged, CategoryIntervention:
		target.Policy = PolicyPreselected
	case CategoryMelee:
		target.Policy = PolicyClosestAtExecution
	default:
		target.Policy = PolicyPreselected // デフォルト
	}

	// 1. 計算式の取得
	formula, ok := s.gameDataManager.Formulas[actingPartDef.Trait]
	if !ok {
		log.Printf("警告: 特性 '%s' に対応する計算式が見つかりません。", actingPartDef.Trait)
	} else {
		// 2. 計算式に基づいて自身に適用されるデバフ効果を生成
		for _, debuffInfo := range formula.UserDebuffs {
			log.Printf("%s が %s 特性効果（チャージ時デバフ）を準備。", settings.Name, formula.ID)
			var effectData interface{}
			switch debuffInfo.Type {
			case DebuffTypeEvasion:
				effectData = &EvasionDebuffEffectData{Multiplier: debuffInfo.Multiplier}
			case DebuffTypeDefense:
				effectData = &DefenseDebuffEffectData{Multiplier: debuffInfo.Multiplier}
			default:
				log.Printf("未対応のチャージ時デバフタイプです: %s", debuffInfo.Type)
			}
			if effectData != nil {
				intent.PendingEffects = append(intent.PendingEffects, effectData)
			}
		}
	}

	if actingPartDef.Category == CategoryRanged {
		// targetEntry が有効なエンティティであるか、または破壊されていないかを確認
		if targetEntry == nil || !targetEntry.Valid() || StateComponent.Get(targetEntry).CurrentState == StateBroken {
			log.Printf("%s: [射撃] ターゲットが存在しないか破壊されています。", settings.Name)
			return false
		}
		log.Printf("%sは%sで%sの%sを狙う！", settings.Name, actingPartDef.PartName, SettingsComponent.Get(targetEntry).Name, targetPartSlot)
	} else {
		log.Printf("%sは%sで攻撃準備！", settings.Name, actingPartDef.PartName)
	}

	baseSeconds := float64(actingPartDef.Charge)
	// 新しい共通関数を呼び出す
	totalTicks := s.partInfoProvider.CalculateGaugeDuration(baseSeconds, entry)

	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	gauge.ProgressCounter = 0

	state.CurrentState = StateCharging
	return true
}
