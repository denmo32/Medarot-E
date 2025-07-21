package battle

import (
	"context"
	"log"
	"medarot-ebiten/internal/game"

	// "github.com/looplab/fsm"
	"github.com/yohamta/donburi"
)

// ChargeInitiationSystem はチャージ状態の開始ロジックをカプセル化します。
type ChargeInitiationSystem struct {
	world            donburi.World
	partInfoProvider *PartInfoProvider
}

// NewChargeInitiationSystem は新しいChargeInitiationSystemのインスタンスを生成します。
func NewChargeInitiationSystem(world donburi.World, partInfoProvider *PartInfoProvider) *ChargeInitiationSystem {
	return &ChargeInitiationSystem{
		world:            world,
		partInfoProvider: partInfoProvider,
	}
}

// ProcessChargeRequest はチャージ状態を開始するための主要なロジックを実行します。
func (s *ChargeInitiationSystem) ProcessChargeRequest(
	entry *donburi.Entry,
	partKey game.PartSlotKey,
	targetEntry *donburi.Entry,
	targetPartSlot game.PartSlotKey,
) bool {
	state := game.StateComponent.Get(entry)
	if !state.FSM.Is(string(game.StateIdle)) {
		return false // アイドル状態でない場合は開始できない
	}

	partsComp := game.PartsComponent.Get(entry)
	settings := game.SettingsComponent.Get(entry)
	actingPartInstance := partsComp.Map[partKey]

	if actingPartInstance == nil {
		log.Printf("%s: 選択されたパーツ %s は存在しません。", settings.Name, partKey)
		return false
	}
	actingPartDef, defFound := s.partInfoProvider.gameDataManager.GetPartDefinition(actingPartInstance.DefinitionID)
	if !defFound {
		log.Printf("%s: パーツ定義(%s)が見つかりません。", settings.Name, actingPartInstance.DefinitionID)
		return false
	}

	intent := ActionIntentComponent.Get(entry)
	intent.SelectedPartKey = partKey
	intent.PendingEffects = make([]game.StatusEffect, 0) // 既存の効果をクリア

	target := game.TargetComponent.Get(entry)
	if targetEntry != nil { // targetEntry が nil でない場合のみIDをセット
		target.TargetEntity = targetEntry.Entity()
	} else {
		target.TargetEntity = 0 // nil の場合はゼロ値
	}
	target.TargetPartSlot = targetPartSlot

	// カテゴリに基づいてターゲット決定方針を設定
	switch actingPartDef.Category {
	case game.CategoryRanged, game.CategoryIntervention:
		target.Policy = game.PolicyPreselected
	case game.CategoryMelee:
		target.Policy = game.PolicyClosestAtExecution
	default:
		target.Policy = game.PolicyPreselected // デフォルト
	}

	// 1. 計算式の取得
	formula, ok := FormulaManager[actingPartDef.Trait]
	if !ok {
		log.Printf("警告: 特性 '%s' に対応する計算式が見つかりません。", actingPartDef.Trait)
	} else {
		// 2. 計算式に基づいて自身に適用されるデバフ効果を生成
		for _, debuffInfo := range formula.UserDebuffs {
			log.Printf("%s が %s 特性効果（チャージ時デバフ）を準備。", settings.Name, formula.ID)
			var effect game.StatusEffect
			switch debuffInfo.Type {
			case game.DebuffTypeEvasion:
				effect = &EvasionDebuffEffect{Multiplier: debuffInfo.Multiplier}
			case game.DebuffTypeDefense:
				effect = &DefenseDebuffEffect{Multiplier: debuffInfo.Multiplier}
			default:
				log.Printf("未対応のチャージ時デバフタイプです: %s", debuffInfo.Type)
			}
			if effect != nil {
				intent.PendingEffects = append(intent.PendingEffects, effect)
			}
		}
	}

	if actingPartDef.Category == game.CategoryRanged {
		// targetEntry が有効なエンティティであるか、または破壊されていないかを確認
		if targetEntry == nil || !targetEntry.Valid() || game.StateComponent.Get(targetEntry).FSM.Is(string(game.StateBroken)) {
			log.Printf("%s: [射撃] ターゲットが存在しないか破壊されています。", settings.Name)
			return false
		}
		log.Printf("%sは%sで%sの%sを狙う！", settings.Name, actingPartDef.PartName, game.SettingsComponent.Get(targetEntry).Name, targetPartSlot)
	} else {
		log.Printf("%sは%sで攻撃準備！", settings.Name, actingPartDef.PartName)
	}

	baseSeconds := float64(actingPartDef.Charge)
	// 新しい共通関数を呼び出す
	totalTicks := s.partInfoProvider.CalculateGaugeDuration(baseSeconds, entry)

	gauge := game.GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks

	err := state.FSM.Event(context.Background(), "charge", entry)
	if err != nil {
		log.Printf("Error starting charge for %s: %v", settings.Name, err)
		return false
	}
	return true
}
