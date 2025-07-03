package main

import (
	"log"
	"sort"

	"github.com/yohamta/donburi"
)

// ActionResult はアクション実行の詳細な結果を保持します。
type ActionResult struct {
	ActingEntry      *donburi.Entry
	TargetEntry      *donburi.Entry
	TargetPartSlot   PartSlotKey // ターゲットのパーツスロット
	LogMessage       string
	ActionDidHit     bool // 命中したかどうか
	IsCritical       bool // クリティカルだったか
	DamageDealt      int  // 実際に与えたダメージ
	TargetPartBroken bool // ターゲットパーツが破壊されたか
	ActionIsDefended bool // 攻撃が防御されたか
}

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
		// ソートのための推進力は脚部パーツ定義から取得する必要があります
		propI := battleLogic.PartInfoProvider.GetOverallPropulsion(actionQueueComp.Queue[i])
		propJ := battleLogic.PartInfoProvider.GetOverallPropulsion(actionQueueComp.Queue[j])
		return propI > propJ
	})

	if len(actionQueueComp.Queue) > 0 {
		actingEntry := actionQueueComp.Queue[0]
		actionQueueComp.Queue = actionQueueComp.Queue[1:]

		actionResult := executeAction(actingEntry, world, battleLogic, gameConfig)
		results = append(results, actionResult)
	}
	return results, nil
}

func executeAction(
	entry *donburi.Entry,
	world donburi.World,
	battleLogic *BattleLogic,
	gameConfig *Config,
) ActionResult {
	ctx := &ActionContext{
		World:            world,
		ActingEntry:      entry,
		ActionResult:     &ActionResult{ActingEntry: entry},
		DamageCalculator: battleLogic.DamageCalculator,
		HitCalculator:    battleLogic.HitCalculator,
		TargetSelector:   battleLogic.TargetSelector,
		PartInfoProvider: battleLogic.PartInfoProvider,
		GameConfig:       gameConfig,
	}

	// 1. アクション解決
	if !ResolveActionSystem(ctx) {
		CleanupActionSystem(ctx)
		return *ctx.ActionResult
	}

	// 2. 命中判定
	if !DetermineHitSystem(ctx) {
		CleanupActionSystem(ctx)
		return *ctx.ActionResult
	}

	// 3. ダメージ適用 (攻撃アクションの場合)
	if ctx.ActingPartDef.Category == CategoryShoot || ctx.ActingPartDef.Category == CategoryMelee {
		ApplyDamageSystem(ctx)
	}

	// 4. アクション結果生成
	GenerateActionResultSystem(ctx)

	// 5. クリーンアップ
	CleanupActionSystem(ctx)

	return *ctx.ActionResult
}

// StartCooldownSystem はクールダウン状態を開始します。
func StartCooldownSystem(entry *donburi.Entry, world donburi.World, gameConfig *Config) {
	actionComp := ActionComponent.Get(entry)
	partsComp := PartsComponent.Get(entry)
	var actingPartDef *PartDefinition

	if actingPartInstance, ok := partsComp.Map[actionComp.SelectedPartKey]; ok {
		if def, defFound := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID); defFound {
			actingPartDef = def
		} else {
			log.Printf("エラー: StartCooldownSystem - ID %s のPartDefinitionが見つかりません。", actingPartInstance.DefinitionID)
		}
	} else {
		log.Printf("エラー: StartCooldownSystem - キー %s の行動パーツインスタンスが見つかりません。", actionComp.SelectedPartKey)
	}

	if actingPartDef != nil && actingPartDef.Trait != TraitBerserk {
		ResetAllEffects(world)
	}

	baseSeconds := 1.0
	if actingPartDef != nil {
		baseSeconds = float64(actingPartDef.Cooldown)
	}
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}
	totalTicks := (baseSeconds * 60.0) / gameConfig.Balance.Time.GameSpeedMultiplier

	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	if gauge.TotalDuration < 1 {
		gauge.TotalDuration = 1
	}
	gauge.ProgressCounter = 0
	gauge.CurrentGauge = 0

	if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
		entry.RemoveComponent(ActingWithBerserkTraitTagComponent)
	}
	if entry.HasComponent(ActingWithAimTraitTagComponent) {
		entry.RemoveComponent(ActingWithAimTraitTagComponent)
	}
	RemoveActionModifiersSystem(entry)

	ChangeState(entry, StateTypeCooldown)
}

// StartCharge はチャージ状態を開始します。
func StartCharge(
	entry *donburi.Entry,
	partKey PartSlotKey,
	target *donburi.Entry,
	targetPartSlot PartSlotKey,
	world donburi.World,
	gameConfig *Config,
	partInfoProvider *PartInfoProvider,
) bool {
	partsComp := PartsComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	actingPartInstance := partsComp.Map[partKey]

	if actingPartInstance == nil || actingPartInstance.IsBroken {
		log.Printf("%s: 選択されたパーツ %s (%s) は存在しないか破壊されています。", settings.Name, partKey, actingPartInstance.DefinitionID)
		return false
	}
	actingPartDef, defFound := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID)
	if !defFound {
		log.Printf("%s: パーツ定義(%s)が見つかりません。", settings.Name, actingPartInstance.DefinitionID)
		return false
	}

	action := ActionComponent.Get(entry)
	action.SelectedPartKey = partKey
	action.TargetEntity = target
	action.TargetPartSlot = targetPartSlot

	switch actingPartDef.Trait {
	case TraitBerserk:
		donburi.Add(entry, ActingWithBerserkTraitTagComponent, &ActingWithBerserkTraitTag{})
		log.Printf("%s の行動にBERSERK特性タグを付与。", settings.Name)
	case TraitAim:
		donburi.Add(entry, ActingWithAimTraitTagComponent, &ActingWithAimTraitTag{})
		log.Printf("%s の行動にAIM特性タグを付与。", settings.Name)
	}

	if actingPartDef.Category == CategoryShoot {
		if target == nil || StateComponent.Get(target).Current == StateTypeBroken {
			log.Printf("%s: [射撃] ターゲットが存在しないか破壊されています。", settings.Name)
			if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
				entry.RemoveComponent(ActingWithBerserkTraitTagComponent)
			}
			if entry.HasComponent(ActingWithAimTraitTagComponent) {
				entry.RemoveComponent(ActingWithAimTraitTagComponent)
			}
			return false
		}
		log.Printf("%sは%sで%sの%sを狙う！", settings.Name, actingPartDef.PartName, SettingsComponent.Get(target).Name, targetPartSlot)
	} else {
		log.Printf("%sは%sで攻撃準備！", settings.Name, actingPartDef.PartName)
	}

	if target != nil {
		balanceConfig := &gameConfig.Balance
		if entry.HasComponent(ActingWithBerserkTraitTagComponent) {
			log.Printf("%s がBERSERK特性効果（チャージ時デバフ）を発動。", settings.Name)
			donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{Multiplier: balanceConfig.Effects.Berserk.DefenseRateDebuff})
			donburi.Add(target, EvasionDebuffComponent, &EvasionDebuff{Multiplier: balanceConfig.Effects.Berserk.EvasionRateDebuff})
		}
		if actingPartDef.Category == CategoryShoot && entry.HasComponent(ActingWithAimTraitTagComponent) {
			log.Printf("%s がAIM特性効果（チャージ時デバフ）を発動。", settings.Name)
			donburi.Add(target, EvasionDebuffComponent, &EvasionDebuff{Multiplier: balanceConfig.Effects.Aim.EvasionRateDebuff})
		}
		if actingPartDef.Category == CategoryMelee {
			log.Printf("%s が格闘カテゴリ効果（チャージ時デバフ）を発動。", settings.Name)
			donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{Multiplier: balanceConfig.Effects.Melee.DefenseRateDebuff})
		}
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

	baseSeconds := float64(actingPartDef.Charge)
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}
	balanceConfig := &gameConfig.Balance
	propulsionFactor := 1.0 + (float64(propulsion) * balanceConfig.Time.PropulsionEffectRate)
	totalTicks := (baseSeconds * 60.0) / (balanceConfig.Time.GameSpeedMultiplier * propulsionFactor)

	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	if gauge.TotalDuration < 1 {
		gauge.TotalDuration = 1
	}
	ChangeState(entry, StateTypeCharging)
	return true
}
