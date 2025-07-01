package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

func StartCharge(entry *donburi.Entry, partKey PartSlotKey, target *donburi.Entry, targetPartSlot PartSlotKey, balanceConfig *BalanceConfig) bool {
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
		// ★★★ ここから下の effects の部分を全面的に書き換える ★★★
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
			// Berserkは両方のデバフを付与
			donburi.Add(target, DefenseDebuffComponent, &DefenseDebuff{
				Multiplier: balanceConfig.Effects.Berserk.DefenseRateDebuff,
			})
			donburi.Add(target, EvasionDebuffComponent, &EvasionDebuff{
				Multiplier: balanceConfig.Effects.Berserk.EvasionRateDebuff,
			})
		}
	}

	legs := parts.Map[PartSlotLegs]
	propulsion := 1
	if legs != nil && !legs.IsBroken {
		propulsion = legs.Propulsion
	}

	baseSeconds := float64(part.Charge)
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}

	propulsionFactor := 1.0 + (float64(propulsion) * balanceConfig.Time.PropulsionEffectRate)
	totalTicks := (baseSeconds * 60.0) / (balanceConfig.Time.GameSpeedMultiplier * propulsionFactor)
	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	if gauge.TotalDuration < 1 {
		gauge.TotalDuration = 1
	}
	ChangeState(entry, StateTypeCharging) // ★★★ 修正 ★★★
	return true
}

func StartCooldown(entry *donburi.Entry, balanceConfig *BalanceConfig) {
	part := PartsComponent.Get(entry).Map[ActionComponent.Get(entry).SelectedPartKey]
	if part.Trait != TraitBerserk {
		ResetAllEffects(entry.World)
	}

	parts := PartsComponent.Get(entry)
	action := ActionComponent.Get(entry)
	part = parts.Map[action.SelectedPartKey]

	baseSeconds := 1.0
	if part != nil {
		baseSeconds = float64(part.Cooldown)
	}
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}
	totalTicks := (baseSeconds * 60.0) / balanceConfig.Time.GameSpeedMultiplier

	gauge := GaugeComponent.Get(entry)
	gauge.TotalDuration = totalTicks
	if gauge.TotalDuration < 1 {
		gauge.TotalDuration = 1
	}
	gauge.ProgressCounter = 0
	gauge.CurrentGauge = 0
	ChangeState(entry, StateTypeCooldown) // ★★★ 修正 ★★★
}

// ★★★ ResetAllEffects を全面的に書き換える ★★★
func ResetAllEffects(world donburi.World) {
	// 防御デバフを持つすべてのエンティティを探してコンポーネントを削除
	query.NewQuery(filter.Contains(DefenseDebuffComponent)).Each(world, func(e *donburi.Entry) {
		e.RemoveComponent(DefenseDebuffComponent)
	})
	// 回避デバフを持つすべてのエンティティを探してコンポーネントを削除
	query.NewQuery(filter.Contains(EvasionDebuffComponent)).Each(world, func(e *donburi.Entry) {
		e.RemoveComponent(EvasionDebuffComponent)
	})
}

func ExecuteAction(entry *donburi.Entry, bs *BattleScene) *donburi.Entry {
	action := ActionComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	logComp := LogComponent.Get(entry)
	part := PartsComponent.Get(entry).Map[action.SelectedPartKey]
	config := &bs.resources.Config

	var targetEntry *donburi.Entry
	var intendedTargetPart *Part

	if part.Category == CategoryShoot {
		targetEntry = action.TargetEntity
		if targetEntry == nil || targetEntry.HasComponent(BrokenStateComponent) {
			logComp.LastActionLog = fmt.Sprintf("%sはターゲットを狙ったが、既に行動不能だった！", settings.Name)
			return nil
		}
		intendedTargetPart = PartsComponent.Get(targetEntry).Map[action.TargetPartSlot]
		if intendedTargetPart == nil || intendedTargetPart.IsBroken {
			logComp.LastActionLog = fmt.Sprintf("%sは%sを狙ったが、パーツは既に破壊されていた！", settings.Name, action.TargetPartSlot)
			return nil
		}
	} else if part.Category == CategoryMelee {
		closestEnemy := findClosestEnemy(bs, entry)
		if closestEnemy == nil {
			logComp.LastActionLog = fmt.Sprintf("%sは攻撃しようとしたが、相手がいなかった。", settings.Name)
			return nil
		}
		targetEntry = closestEnemy
		intendedTargetPart = SelectRandomPartToDamage(targetEntry)
		if intendedTargetPart == nil {
			logComp.LastActionLog = fmt.Sprintf("%sは%sを狙ったが、攻撃できる部位がなかった！", settings.Name, SettingsComponent.Get(targetEntry).Name)
			return nil
		}
	} else {
		logComp.LastActionLog = fmt.Sprintf("%sは行動に失敗した。", settings.Name)
		return nil
	}

	balanceConf := &config.Balance

	if !CalculateHit(entry, targetEntry, part, balanceConf) {
		logComp.LastActionLog = GenerateActionLog(entry, targetEntry, nil, 0, false, false)
		if part.Trait == TraitBerserk {
			ResetAllEffects(bs.world)
		}
		return targetEntry
	}

	damage, isCritical := CalculateDamage(entry, part, balanceConf)
	defensePart := SelectDefensePart(targetEntry)

	if defensePart != nil && CalculateDefense(entry, targetEntry, defensePart, balanceConf) {
		finalDamage := damage - defensePart.Defense
		ApplyDamage(targetEntry, defensePart, finalDamage)
		logComp.LastActionLog = GenerateActionLogDefense(entry, targetEntry, defensePart, finalDamage, isCritical)
	} else {
		ApplyDamage(targetEntry, intendedTargetPart, damage)
		logComp.LastActionLog = GenerateActionLog(entry, targetEntry, intendedTargetPart, damage, isCritical, true)
	}

	if part.Trait == TraitBerserk {
		ResetAllEffects(bs.world)
	}

	return targetEntry
}

func GenerateActionLogDefense(attacker, target *donburi.Entry, defensePart *Part, damage int, isCritical bool) string {
	targetSettings := SettingsComponent.Get(target)
	logMsg := fmt.Sprintf("%sは%sで防御し、%dダメージに抑えた！", targetSettings.Name, defensePart.PartName, damage)
	if isCritical {
		logMsg = fmt.Sprintf("%sは%sで防御！クリティカルヒットを%dダメージに抑えた！", targetSettings.Name, defensePart.PartName, damage)
	}
	if defensePart.IsBroken {
		logMsg += " しかし、パーツは破壊された！"
	}
	return logMsg
}

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
			gauge.CurrentGauge = 100
		}
		if gauge.ProgressCounter >= gauge.TotalDuration {
			if entry.HasComponent(ChargingStateComponent) {
				ChangeState(entry, StateTypeReady) // ★★★ 修正 ★★★
				bs.actionQueue = append(bs.actionQueue, entry)
				log.Printf("%s のチャージが完了。実行キューに追加。", SettingsComponent.Get(entry).Name)
			} else if entry.HasComponent(CooldownStateComponent) {
				ChangeState(entry, StateTypeIdle) // ★★★ 修正 ★★★
				part := PartsComponent.Get(entry).Map[ActionComponent.Get(entry).SelectedPartKey]
				if part != nil && part.Trait == TraitBerserk {
					ResetAllEffects(bs.world)
				}
			}
		}
	})
}

func SystemProcessReadyQueue(bs *BattleScene) {
	if len(bs.actionQueue) == 0 {
		return
	}
	sort.SliceStable(bs.actionQueue, func(i, j int) bool {
		propI := GetOverallPropulsion(bs.actionQueue[i])
		propJ := GetOverallPropulsion(bs.actionQueue[j])
		return propI > propJ
	})

	if len(bs.actionQueue) > 0 {
		bs.attackingEntity = nil
		bs.targetedEntity = nil

		actingEntry := bs.actionQueue[0]
		bs.actionQueue = bs.actionQueue[1:]

		finalTarget := ExecuteAction(actingEntry, bs)

		if finalTarget != nil {
			bs.attackingEntity = actingEntry
			bs.targetedEntity = finalTarget
		}

		bs.enqueueMessage(LogComponent.Get(actingEntry).LastActionLog, func() {
			if actingEntry.Valid() && !actingEntry.HasComponent(BrokenStateComponent) {
				StartCooldown(actingEntry, &bs.resources.Config.Balance)
			}
		})
	}
}

func SystemProcessIdleMedarots(bs *BattleScene) {
	if bs.playerMedarotToAct != nil || bs.state != StatePlaying {
		return
	}
	query.NewQuery(filter.And(
		filter.Contains(IdleStateComponent),
		filter.Not(filter.Contains(PlayerControlComponent)),
	)).Each(bs.world, func(entry *donburi.Entry) {
		aiSelectAction(bs, entry)
	})

	query.NewQuery(filter.And(
		filter.Contains(PlayerControlComponent),
		filter.Contains(IdleStateComponent),
	)).Each(bs.world, func(entry *donburi.Entry) {
		bs.playerMedarotToAct = entry
		bs.state = StatePlayerActionSelect
		return
	})
}

func SystemCheckGameEnd(bs *BattleScene) {
	if bs.state == StateGameOver {
		return
	}
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
			gameOverMsg = "チーム1のリーダー不在！ チーム2の勝利！"
		}
		bs.enqueueMessage(gameOverMsg, nil)
	} else if team2Leader == nil || PartsComponent.Get(team2Leader).Map[PartSlotHead].IsBroken || team1FuncCount == 0 {
		bs.winner = Team1
		bs.state = StateGameOver
		if team2Leader != nil {
			gameOverMsg = fmt.Sprintf("%sが機能停止！ チーム1の勝利！", SettingsComponent.Get(team2Leader).Name)
		} else {
			gameOverMsg = "チーム2のリーダー不在！ チーム1の勝利！"
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
