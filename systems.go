package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// --- メダロット個別の行動ロジック（ECS版） ---

func StartCharge(entry *donburi.Entry, partKey PartSlotKey, target *donburi.Entry, balanceConfig *BalanceConfig) bool {
	parts := PartsComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	targetState := StateComponent.Get(target)
	part := parts.Map[partKey]

	if part == nil || part.IsBroken {
		log.Printf("%s: 選択されたパーツ %s は存在しないか破壊されています。", settings.Name, partKey)
		return false
	}
	if target == nil || targetState.State == StateBroken {
		log.Printf("%s: ターゲットが存在しないか破壊されています。", settings.Name)
		return false
	}

	action := ActionComponent.Get(entry)
	action.SelectedPartKey = partKey
	action.TargetEntity = target
	log.Printf("%sは%sで%sを狙う！", settings.Name, part.PartName, SettingsComponent.Get(target).Name)

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

func ExecuteAction(entry *donburi.Entry, balanceConfig *BalanceConfig) {
	action := ActionComponent.Get(entry)
	settings := SettingsComponent.Get(entry)
	log := LogComponent.Get(entry)

	if action.SelectedPartKey == "" || action.TargetEntity == nil {
		log.LastActionLog = fmt.Sprintf("%sは行動に失敗した。", settings.Name)
		return
	}

	targetEntry := action.TargetEntity
	if StateComponent.Get(targetEntry).State == StateBroken {
		log.LastActionLog = fmt.Sprintf("%sは%sを狙ったが、既に行動不能だった！", settings.Name, SettingsComponent.Get(targetEntry).Name)
		return
	}

	part := PartsComponent.Get(entry).Map[action.SelectedPartKey]
	log.LastActionLog = "実行中..." // Placeholder

	isHit := CalculateHit(entry, targetEntry, part, balanceConfig)
	if isHit {
		damage, isCritical := CalculateDamage(entry, part, balanceConfig)
		targetPart := SelectRandomPartToDamage(targetEntry)
		if targetPart != nil {
			ApplyDamage(targetEntry, targetPart, damage)
			log.LastActionLog = GenerateActionLog(entry, targetEntry, targetPart, damage, isCritical, true)
		} else {
			log.LastActionLog = fmt.Sprintf("%sの攻撃は%sに当たらなかった。", settings.Name, SettingsComponent.Get(targetEntry).Name)
		}
	} else {
		log.LastActionLog = GenerateActionLog(entry, targetEntry, nil, 0, false, false)
	}

	if PartsComponent.Get(entry).Map[PartSlotHead].IsBroken {
		ChangeState(entry, StateBroken)
	}
}

// --- ゲーム全体の進行を管理するシステム関数（ECS版） ---

func SystemUpdateProgress(g *Game) {
	query.NewQuery(filter.And(
		filter.Contains(StateComponent),
		filter.Contains(GaugeComponent),
	)).Each(g.World, func(entry *donburi.Entry) {
		state := StateComponent.Get(entry)
		if state.State != StateCharging && state.State != StateCooldown {
			return // continue
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
				g.actionQueue = append(g.actionQueue, entry)
				log.Printf("%s のチャージが完了。実行キューに追加。", SettingsComponent.Get(entry).Name)
			} else if state.State == StateCooldown {
				ChangeState(entry, StateIdle)
			}
		}
	})
}

func SystemProcessReadyQueue(g *Game) {
	if len(g.actionQueue) == 0 {
		return
	}
	sort.SliceStable(g.actionQueue, func(i, j int) bool {
		propI := GetOverallPropulsion(g.actionQueue[i])
		propJ := GetOverallPropulsion(g.actionQueue[j])
		return propI > propJ
	})

	if len(g.actionQueue) > 0 {
		actingEntry := g.actionQueue[0]
		g.actionQueue = g.actionQueue[1:]

		ExecuteAction(actingEntry, &g.Config.Balance)

		g.enqueueMessage(LogComponent.Get(actingEntry).LastActionLog, func() {
			if StateComponent.Get(actingEntry).State != StateBroken {
				StartCooldown(actingEntry, &g.Config.Balance)
			}
		})
	}
}

func SystemProcessIdleMedarots(g *Game) {
	if g.playerMedarotToAct != nil || g.State != StatePlaying {
		return
	}

	query.NewQuery(filter.And(
		filter.Contains(StateComponent),
		filter.Contains(SettingsComponent),
	)).Each(g.World, func(entry *donburi.Entry) {
		state := StateComponent.Get(entry)
		if state.State == StateIdle {
			if !entry.HasComponent(PlayerControlComponent) {
				aiSelectAction(g, entry)
			}
		}
	})

	query.NewQuery(filter.And(
		filter.Contains(PlayerControlComponent),
		filter.Contains(StateComponent),
	)).Each(g.World, func(entry *donburi.Entry) {
		state := StateComponent.Get(entry)
		if state.State == StateIdle {
			g.playerMedarotToAct = entry
			g.State = StatePlayerActionSelect
			return // 最初の1体を見つけたら抜ける
		}
	})
}

func SystemCheckGameEnd(g *Game) {
	if g.State == StateGameOver {
		return
	}

	team1Leader := FindLeader(g.World, Team1)
	team2Leader := FindLeader(g.World, Team2)

	team1FuncCount := 0
	team2FuncCount := 0
	query.NewQuery(filter.And(
		filter.Contains(SettingsComponent),
		filter.Contains(StateComponent),
	)).Each(g.World, func(entry *donburi.Entry) {
		if StateComponent.Get(entry).State != StateBroken {
			if SettingsComponent.Get(entry).Team == Team1 {
				team1FuncCount++
			} else {
				team2FuncCount++
			}
		}
	})

	if PartsComponent.Get(team1Leader).Map[PartSlotHead].IsBroken || team2FuncCount == 0 {
		g.winner = Team2
		g.State = StateGameOver
		g.enqueueMessage(fmt.Sprintf("%sが機能停止！ チーム2の勝利！", SettingsComponent.Get(team1Leader).Name), nil)
	} else if PartsComponent.Get(team2Leader).Map[PartSlotHead].IsBroken || team1FuncCount == 0 {
		g.winner = Team1
		g.State = StateGameOver
		g.enqueueMessage(fmt.Sprintf("%sが機能停止！ チーム1の勝利！", SettingsComponent.Get(team2Leader).Name), nil)
	}
}

// ChangeState はエンティティの状態を変更し、関連データをリセットする
func ChangeState(entry *donburi.Entry, newState MedarotState) {
	state := StateComponent.Get(entry)
	if state.State == newState {
		return
	}
	log.Printf("%s のステートが %s から %s に変更されました。", SettingsComponent.Get(entry).Name, state.State, newState)
	state.State = newState

	gauge := GaugeComponent.Get(entry)
	action := ActionComponent.Get(entry)

	switch newState {
	case StateIdle:
		gauge.CurrentGauge = 0
		gauge.ProgressCounter = 0
		gauge.TotalDuration = 0
		action.SelectedPartKey = ""
		action.TargetEntity = nil
	case StateReady:
		gauge.CurrentGauge = 100
	case StateBroken:
		gauge.CurrentGauge = 0
	}
}
