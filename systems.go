package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// StartCharge はチャージ状態を開始します。
// BattleScene から各種ヘルパーを取得するように変更が必要です。
func StartCharge(entry *donburi.Entry, partKey PartSlotKey, target *donburi.Entry, targetPartSlot PartSlotKey, bs *BattleScene) bool {
	parts := PartsComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	part := parts.Map[partKey]

	if part == nil || part.IsBroken {
		log.Printf("%s: 選択されたパーツ %s は存在しないか破壊されています。", settings.Name, partKey)
		return false
	}

	action := ActionComponent.Get(entry)
	action.SelectedPartKey = partKey
	action.TargetEntity = target
	action.TargetPartSlot = targetPartSlot

	if part.Category == CategoryShoot {
		if target == nil || target.HasComponent(BrokenStateComponent) {
			log.Printf("%s: [SHOOT] ターゲットが存在しないか破壊されています。", settings.Name)
			return false
		}
		log.Printf("%sは%sで%sの%sを狙う！", settings.Name, part.PartName, SettingsComponent.Get(target).Name, targetPartSlot)
	} else {
		log.Printf("%sは%sで攻撃準備！", settings.Name, part.PartName)
	}

	if target != nil {
		balanceConfig := &bs.resources.Config.Balance
		switch part.Category {
		case CategoryMelee:
			donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{
				Multiplier: balanceConfig.Effects.Melee.DefenseRateDebuff,
			})
		case CategoryShoot:
			if part.Trait == TraitAim {
				donburi.Add(target, EvasionDebuffComponent, &EvasionDebuff{
					Multiplier: balanceConfig.Effects.Aim.EvasionRateDebuff,
				})
			}
		}
		if part.Trait == TraitBerserk {
			donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{
				Multiplier: balanceConfig.Effects.Berserk.DefenseRateDebuff,
			})
			donburi.Add(target, EvasionDebuffComponent, &EvasionDebuff{
				Multiplier: balanceConfig.Effects.Berserk.EvasionRateDebuff,
			})
		}
	}

	// Propulsion を PartInfoProvider から取得
	propulsion := 1
	if bs.partInfoProvider != nil { // nilチェックを追加
		legs := parts.Map[PartSlotLegs]
		if legs != nil && !legs.IsBroken { // 脚部パーツの存在と状態をチェック
			propulsion = bs.partInfoProvider.GetOverallPropulsion(entry)
		}
	} else {
		log.Println("Warning: StartCharge - bs.partInfoProvider is nil")
	}


	baseSeconds := float64(part.Charge)
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}

	balanceConfig := &bs.resources.Config.Balance
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

// StartCooldown はクールダウン状態を開始します。
func StartCooldown(entry *donburi.Entry, bs *BattleScene) {
	actionComp := ActionComponent.Get(entry)
	part := PartsComponent.Get(entry).Map[actionComp.SelectedPartKey]

	if part != nil && part.Trait != TraitBerserk { // partのnilチェックを追加
		ResetAllEffects(entry.World)
	}


	baseSeconds := 1.0
	if part != nil {
		baseSeconds = float64(part.Cooldown)
	}
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}
	totalTicks := (baseSeconds * 60.0) / bs.resources.Config.Balance.Time.GameSpeedMultiplier

	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	if gauge.TotalDuration < 1 {
		gauge.TotalDuration = 1
	}
	gauge.ProgressCounter = 0
	gauge.CurrentGauge = 0
	ChangeState(entry, StateTypeCooldown)
}

// ResetAllEffects は全ての効果をリセットします。
func ResetAllEffects(world donburi.World) {
	query.NewQuery(filter.Contains(DefenseDebuffComponent)).Each(world, func(e *donburi.Entry) {
		e.RemoveComponent(DefenseDebuffComponent)
	})
	query.NewQuery(filter.Contains(EvasionDebuffComponent)).Each(world, func(e *donburi.Entry) {
		e.RemoveComponent(EvasionDebuffComponent)
	})
}

// ExecuteAction はエンティティの行動を実行します。
func ExecuteAction(entry *donburi.Entry, bs *BattleScene) *donburi.Entry {
	action := ActionComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	logComp := LogComponent.Get(entry)
	actingPart := PartsComponent.Get(entry).Map[action.SelectedPartKey]

	var targetEntry *donburi.Entry
	var intendedTargetPart *Part // 実際に狙うパーツ

	// --- ターゲット選択 ---
	if actingPart.Category == CategoryShoot {
		targetEntry = action.TargetEntity
		if targetEntry == nil || targetEntry.HasComponent(BrokenStateComponent) {
			logComp.LastActionLog = fmt.Sprintf("%sはターゲットを狙ったが、既に行動不能だった！", settings.Name)
			return nil
		}
		// 射撃攻撃の場合、指定されたスロットのパーツを狙う
		intendedTargetPart = PartsComponent.Get(targetEntry).Map[action.TargetPartSlot]
		if intendedTargetPart == nil || intendedTargetPart.IsBroken {
			logComp.LastActionLog = fmt.Sprintf("%sは%sを狙ったが、パーツは既に破壊されていた！", settings.Name, action.TargetPartSlot)
			return nil
		}
	} else if actingPart.Category == CategoryMelee {
		closestEnemy := bs.targetSelector.FindClosestEnemy(entry)
		if closestEnemy == nil {
			logComp.LastActionLog = fmt.Sprintf("%sは攻撃しようとしたが、相手がいなかった。", settings.Name)
			return nil
		}
		targetEntry = closestEnemy
		// 格闘攻撃の場合、ランダムなパーツを狙う
		intendedTargetPart = bs.targetSelector.SelectRandomPartToDamage(targetEntry)
		if intendedTargetPart == nil { // 攻撃できるパーツがない
			logComp.LastActionLog = fmt.Sprintf("%sは%sを狙ったが、攻撃できる部位がなかった！", settings.Name, SettingsComponent.Get(targetEntry).Name)
			return nil
		}
	} else {
		logComp.LastActionLog = fmt.Sprintf("%sは行動 '%s' に失敗した（未対応カテゴリ）。", settings.Name, actingPart.Category)
		return nil
	}

	// --- 命中判定 ---
	if !bs.hitCalculator.CalculateHit(entry, targetEntry, actingPart) {
		logComp.LastActionLog = bs.damageCalculator.GenerateActionLog(entry, targetEntry, nil, 0, false, false)
		if actingPart.Trait == TraitBerserk {
			ResetAllEffects(bs.world)
		}
		return targetEntry // 攻撃は外れたが、ターゲットは存在した
	}

	// --- ダメージ計算 ---
	damage, isCritical := bs.damageCalculator.CalculateDamage(entry, actingPart)
	originalDamage := damage // 防御前のダメージを記録

	// --- 防御判定とダメージ適用 ---
	defensePart := bs.targetSelector.SelectDefensePart(targetEntry)
	damageDealt := damage

	if defensePart != nil && bs.hitCalculator.CalculateDefense(targetEntry, defensePart) {
		// 防御成功
		finalDamageAfterDefense := damage - defensePart.Defense
		if finalDamageAfterDefense < 0 {
			finalDamageAfterDefense = 0
		}
		damageDealt = finalDamageAfterDefense
		bs.damageCalculator.ApplyDamage(targetEntry, defensePart, finalDamageAfterDefense)
		logComp.LastActionLog = bs.damageCalculator.GenerateActionLogDefense(targetEntry, defensePart, finalDamageAfterDefense, originalDamage, isCritical)
	} else {
		// 防御失敗または防御パーツなし
		bs.damageCalculator.ApplyDamage(targetEntry, intendedTargetPart, damage)
		logComp.LastActionLog = bs.damageCalculator.GenerateActionLog(entry, targetEntry, intendedTargetPart, damage, isCritical, true)
	}


	if actingPart.Trait == TraitBerserk {
		ResetAllEffects(bs.world)
	}

	return targetEntry
}


// SystemUpdateProgress はチャージとクールダウンのゲージ進行を更新します。
func SystemUpdateProgress(bs *BattleScene) {
	query.NewQuery(filter.Or(
		filter.Contains(ChargingStateComponent),
		filter.Contains(CooldownStateComponent),
	)).Each(bs.world, func(entry *donburi.Entry) {
		gauge := GaugeComponent.Get(entry)
		gauge.ProgressCounter++
		if gauge.TotalDuration > 0 {
			gauge.CurrentGauge = (gauge.ProgressCounter / gauge.TotalDuration) * 100
		} else {
			gauge.CurrentGauge = 100 // TotalDurationが0なら即完了
		}

		if gauge.ProgressCounter >= gauge.TotalDuration {
			if entry.HasComponent(ChargingStateComponent) {
				ChangeState(entry, StateTypeReady)
				bs.actionQueue = append(bs.actionQueue, entry)
				log.Printf("%s のチャージが完了。実行キューに追加。", SettingsComponent.Get(entry).Name)
			} else if entry.HasComponent(CooldownStateComponent) {
				ChangeState(entry, StateTypeIdle)
				// バーサーク特性の場合、クールダウン終了時に効果をリセット
				actionComp := ActionComponent.Get(entry)
				if actionComp.SelectedPartKey != "" { // 選択パーツキーが存在するか確認
					part := PartsComponent.Get(entry).Map[actionComp.SelectedPartKey]
					if part != nil && part.Trait == TraitBerserk { // partのnilチェックを追加
						ResetAllEffects(bs.world)
					}
				}
			}
		}
	})
}

// SystemProcessReadyQueue は行動準備完了キューを処理します。
func SystemProcessReadyQueue(bs *BattleScene) {
	if len(bs.actionQueue) == 0 {
		return
	}

	// 行動順序を推進力に基づいてソート
	sort.SliceStable(bs.actionQueue, func(i, j int) bool {
		// nilチェックの追加
		if bs.partInfoProvider == nil {
			log.Println("Warning: SystemProcessReadyQueue - bs.partInfoProvider is nil, using default propulsion.")
			return false // または他のデフォルト比較
		}
		propI := bs.partInfoProvider.GetOverallPropulsion(bs.actionQueue[i])
		propJ := bs.partInfoProvider.GetOverallPropulsion(bs.actionQueue[j])
		return propI > propJ
	})

	if len(bs.actionQueue) > 0 {
		bs.attackingEntity = nil
		bs.targetedEntity = nil

		actingEntry := bs.actionQueue[0]
		bs.actionQueue = bs.actionQueue[1:] // キューから取り出し

		finalTarget := ExecuteAction(actingEntry, bs) // ExecuteAction に bs を渡す

		if finalTarget != nil {
			bs.attackingEntity = actingEntry
			bs.targetedEntity = finalTarget
		}

		// メッセージ表示とコールバックの設定
		bs.enqueueMessage(LogComponent.Get(actingEntry).LastActionLog, func() {
			if actingEntry.Valid() && !actingEntry.HasComponent(BrokenStateComponent) {
				StartCooldown(actingEntry, bs) // StartCooldown に bs を渡す
			}
		})
	}
}

// SystemProcessIdleMedarots は待機状態のメダロットの行動選択を処理します。
func SystemProcessIdleMedarots(bs *BattleScene) {
	if bs.playerMedarotToAct != nil || bs.state != StatePlaying {
		return // プレイヤー行動選択中、またはゲームプレイ中以外は処理しない
	}

	// AI制御のメダロットの行動選択
	query.NewQuery(filter.And(
		filter.Contains(IdleStateComponent),
		filter.Not(filter.Contains(PlayerControlComponent)), // プレイヤー制御でない
	)).Each(bs.world, func(entry *donburi.Entry) {
		// aiSelectAction は bs を引数に取るように変更される想定
		aiSelectAction(bs, entry) // ai.go の aiSelectAction を呼び出す
	})

	// プレイヤー制御のメダロットを行動選択状態へ
	query.NewQuery(filter.And(
		filter.Contains(PlayerControlComponent),
		filter.Contains(IdleStateComponent),
	)).Each(bs.world, func(entry *donburi.Entry) {
		bs.playerMedarotToAct = entry
		bs.state = StatePlayerActionSelect
		// 複数のプレイヤー機体が同時にIdleになる場合、Eachは複数回呼ばれるが、
		// playerMedarotToAct が設定された時点でループを抜けるべき。
		// ただし、donburi の Each は途中で抜けられないため、このままだと最後にIdleになった機体が選択される。
		// 設計上、一度にプレイヤーが操作するのは1体なので、現状は問題ない想定。
		return // このreturnはEachのコールバックから抜けるだけで、クエリのループは続く
	})
}

// SystemCheckGameEnd はゲーム終了条件をチェックします。
func SystemCheckGameEnd(bs *BattleScene) {
	if bs.state == StateGameOver {
		return
	}

	// FindLeader はグローバル関数 (ecs_setup.go)
	team1Leader := FindLeader(bs.world, Team1)
	team2Leader := FindLeader(bs.world, Team2)

	team1FuncCount := 0
	team2FuncCount := 0

	query.NewQuery(filter.And(
		filter.Contains(SettingsComponent),
		filter.Not(filter.Contains(BrokenStateComponent)),
	)).Each(bs.world, func(entry *donburi.Entry) {
		if SettingsComponent.Get(entry).Team == Team1 {
			team1FuncCount++
		} else {
			team2FuncCount++
		}
	})

	var gameOverMsg string
	if team1Leader == nil || PartsComponent.Get(team1Leader).Map[PartSlotHead].IsBroken || team2FuncCount == 0 {
		bs.winner = Team2
		bs.state = StateGameOver
		if team1Leader != nil {
			gameOverMsg = fmt.Sprintf("%sが機能停止！ チーム2の勝利！", SettingsComponent.Get(team1Leader).Name)
		} else {
			gameOverMsg = "チーム1のリーダー不在か全滅！ チーム2の勝利！"
		}
		bs.enqueueMessage(gameOverMsg, nil)
	} else if team2Leader == nil || PartsComponent.Get(team2Leader).Map[PartSlotHead].IsBroken || team1FuncCount == 0 {
		bs.winner = Team1
		bs.state = StateGameOver
		if team2Leader != nil {
			gameOverMsg = fmt.Sprintf("%sが機能停止！ チーム1の勝利！", SettingsComponent.Get(team2Leader).Name)
		} else {
			gameOverMsg = "チーム2のリーダー不在か全滅！ チーム1の勝利！"
		}
		bs.enqueueMessage(gameOverMsg, nil)
	}
}

func ChangeState(entry *donburi.Entry, newStateType StateType) {
	// 既存の状態コンポーネントをすべて削除
	if entry.HasComponent(IdleStateComponent) {
		entry.RemoveComponent(IdleStateComponent)
	}
	if entry.HasComponent(ChargingStateComponent) {
		entry.RemoveComponent(ChargingStateComponent)
	}
	if entry.HasComponent(ReadyStateComponent) {
		entry.RemoveComponent(ReadyStateComponent)
	}
	if entry.HasComponent(CooldownStateComponent) {
		entry.RemoveComponent(CooldownStateComponent)
	}
	if entry.HasComponent(BrokenStateComponent) {
		entry.RemoveComponent(BrokenStateComponent)
	}

	log.Printf("%s のステートが変更されました。", SettingsComponent.Get(entry).Name)

	gauge := GaugeComponent.Get(entry)
	action := ActionComponent.Get(entry)

	// 新しい状態に応じた初期化処理とコンポーネントの追加
	switch newStateType {
	case StateTypeIdle:
		donburi.Add(entry, IdleStateComponent, &IdleState{})
		gauge.CurrentGauge = 0
		gauge.ProgressCounter = 0
		gauge.TotalDuration = 0
		action.SelectedPartKey = ""
		action.TargetPartSlot = ""
		action.TargetEntity = nil
	case StateTypeCharging:
		donburi.Add(entry, ChargingStateComponent, &ChargingState{})
	case StateTypeReady:
		donburi.Add(entry, ReadyStateComponent, &ReadyState{})
		gauge.CurrentGauge = 100
	case StateTypeCooldown:
		donburi.Add(entry, CooldownStateComponent, &CooldownState{})
	case StateTypeBroken:
		donburi.Add(entry, BrokenStateComponent, &BrokenState{})
		gauge.CurrentGauge = 0
	}
}
