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
		if target == nil || StateComponent.Get(target).State == StateBroken {
			log.Printf("%s: [SHOOT] ターゲットが存在しないか破壊されています。", settings.Name)
			return false
		}
		log.Printf("%sは%sで%sの%sを狙う！", settings.Name, part.PartName, SettingsComponent.Get(target).Name, targetPartSlot)
	} else {
		log.Printf("%sは%sで攻撃準備！", settings.Name, part.PartName)
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
	ChangeState(entry, StateCharging)
	return true
}

func StartCooldown(entry *donburi.Entry, balanceConfig *BalanceConfig) {
	parts := PartsComponent.Get(entry)
	action := ActionComponent.Get(entry)
	part := parts.Map[action.SelectedPartKey]

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
	ChangeState(entry, StateCooldown)
}

func ExecuteAction(entry *donburi.Entry, bs *BattleScene) *donburi.Entry {
	action := ActionComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	logComp := LogComponent.Get(entry)
	part := PartsComponent.Get(entry).Map[action.SelectedPartKey]

	var targetEntry *donburi.Entry
	var targetPart *Part

	if part.Category == CategoryShoot {
		targetEntry = action.TargetEntity
		if targetEntry == nil || StateComponent.Get(targetEntry).State == StateBroken {
			logComp.LastActionLog = fmt.Sprintf("%sはターゲットを狙ったが、既に行動不能だった！", settings.Name)
			return nil
		}
		targetPart = PartsComponent.Get(targetEntry).Map[action.TargetPartSlot]
		if targetPart == nil || targetPart.IsBroken {
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
		targetPart = SelectRandomPartToDamage(targetEntry)
		if targetPart == nil {
			logComp.LastActionLog = fmt.Sprintf("%sは%sを狙ったが、攻撃できる部位がなかった！", settings.Name, SettingsComponent.Get(targetEntry).Name)
			return nil
		}
	} else {
		logComp.LastActionLog = fmt.Sprintf("%sは行動に失敗した。", settings.Name)
		return nil
	}

	logComp.LastActionLog = "実行中..."

	isHit := CalculateHit(entry, targetEntry, part, &bs.resources.Config.Balance)
	if isHit {
		damage, isCritical := CalculateDamage(entry, part, &bs.resources.Config.Balance)
		ApplyDamage(targetEntry, targetPart, damage)
		logComp.LastActionLog = GenerateActionLog(entry, targetEntry, targetPart, damage, isCritical, true)
	} else {
		logComp.LastActionLog = GenerateActionLog(entry, targetEntry, nil, 0, false, false)
	}

	return targetEntry
}

func SystemUpdateProgress(bs *BattleScene) {
	query.NewQuery(filter.And(
		filter.Contains(StateComponent),
		filter.Contains(GaugeComponent),
	)).Each(bs.world, func(entry *donburi.Entry) {
		state := StateComponent.Get(entry)
		if state.State != StateCharging && state.State != StateCooldown {
			return
		}
		gauge := GaugeComponent.Get(entry)
		gauge.ProgressCounter++
		if gauge.TotalDuration > 0 {
			gauge.CurrentGauge = (gauge.ProgressCounter / gauge.TotalDuration) * 100
		} else {
			gauge.CurrentGauge = 100
		}
		if gauge.ProgressCounter >= gauge.TotalDuration {
			if state.State == StateCharging {
				ChangeState(entry, StateReady)
				bs.actionQueue = append(bs.actionQueue, entry)
				log.Printf("%s のチャージが完了。実行キューに追加。", SettingsComponent.Get(entry).Name)
			} else if state.State == StateCooldown {
				ChangeState(entry, StateIdle)
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
			if actingEntry.Valid() && StateComponent.Get(actingEntry).State != StateBroken {
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
		filter.Contains(StateComponent),
		filter.Contains(SettingsComponent),
	)).Each(bs.world, func(entry *donburi.Entry) {
		state := StateComponent.Get(entry)
		if state.State == StateIdle {
			if !entry.HasComponent(PlayerControlComponent) {
				aiSelectAction(bs, entry)
			}
		}
	})
	query.NewQuery(filter.And(
		filter.Contains(PlayerControlComponent),
		filter.Contains(StateComponent),
	)).Each(bs.world, func(entry *donburi.Entry) {
		state := StateComponent.Get(entry)
		if state.State == StateIdle {
			bs.playerMedarotToAct = entry
			bs.state = StatePlayerActionSelect
			return
		}
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
		filter.Contains(StateComponent),
	)).Each(bs.world, func(entry *donburi.Entry) {
		state := StateComponent.Get(entry)
		if state.State != StateBroken {
			if SettingsComponent.Get(entry).Team == Team1 {
				team1FuncCount++
			} else {
				team2FuncCount++
			}
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

func ChangeState(entry *donburi.Entry, newState MedarotState) {
	state := StateComponent.Get(entry)
	if state.State == newState {
		return
	}
	log.Printf("%s のステートが %s から %s に変更されました。",
		SettingsComponent.Get(entry).Name, state.State, newState)
	state.State = newState

	gauge := GaugeComponent.Get(entry)
	action := ActionComponent.Get(entry)

	switch newState {
	case StateIdle:
		gauge.CurrentGauge = 0
		gauge.ProgressCounter = 0
		gauge.TotalDuration = 0
		action.SelectedPartKey = ""
		action.TargetPartSlot = ""
		action.TargetEntity = nil
	case StateReady:
		gauge.CurrentGauge = 100
	case StateBroken:
		gauge.CurrentGauge = 0
	}
}
