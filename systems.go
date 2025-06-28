package main

import (
	"fmt"
	"log"
	"sort"
)

// --- メダロット個別の行動ロジック ---

// StartCharge は行動を選択し、チャージを開始する
func StartCharge(medarot *Medarot, partKey PartSlotKey, target *Medarot, balanceConfig *BalanceConfig) bool {
	part := medarot.GetPart(partKey)
	if part == nil || part.IsBroken {
		log.Printf("%s: 選択されたパーツ %s は存在しないか破壊されています。", medarot.Name, partKey)
		return false
	}
	if target == nil || target.State == StateBroken {
		log.Printf("%s: ターゲットが存在しないか破壊されています。", medarot.Name)
		return false
	}

	medarot.SelectedPartKey = partKey
	medarot.TargetedMedarot = target
	log.Printf("%sは%sで%sを狙う！", medarot.Name, part.PartName, target.Name)

	baseSeconds := float64(part.Charge)
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}
	propulsionFactor := 1.0 + (float64(medarot.GetOverallPropulsion()) * balanceConfig.Time.PropulsionEffectRate)
	totalTicks := (baseSeconds * 60.0) / (balanceConfig.Time.GameSpeedMultiplier * propulsionFactor)

	medarot.TotalDuration = totalTicks
	if medarot.TotalDuration < 1 {
		medarot.TotalDuration = 1
	}

	medarot.ChangeState(StateCharging)
	return true
}

// StartCooldown はクールダウンを開始する
func StartCooldown(medarot *Medarot, balanceConfig *BalanceConfig) {
	part := medarot.GetPart(medarot.SelectedPartKey)
	baseSeconds := 1.0
	if part != nil {
		baseSeconds = float64(part.Cooldown)
	}
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}

	totalTicks := (baseSeconds * 60.0) / balanceConfig.Time.GameSpeedMultiplier

	medarot.TotalDuration = totalTicks
	if medarot.TotalDuration < 1 {
		medarot.TotalDuration = 1
	}

	medarot.ProgressCounter = 0
	medarot.Gauge = 0

	medarot.ChangeState(StateCooldown)
}

// ExecuteAction は選択された行動を実行するロジック
func ExecuteAction(actingMedarot *Medarot, balanceConfig *BalanceConfig) {
	if actingMedarot.SelectedPartKey == "" || actingMedarot.TargetedMedarot == nil {
		actingMedarot.LastActionLog = fmt.Sprintf("%sは行動に失敗した。", actingMedarot.Name)
		return
	}

	part := actingMedarot.GetPart(actingMedarot.SelectedPartKey)
	target := actingMedarot.TargetedMedarot

	if target.State == StateBroken {
		actingMedarot.LastActionLog = fmt.Sprintf("%sは%sを狙ったが、既に行動不能だった！", actingMedarot.Name, target.Name)
		return
	}
	log.Printf("%s が %s を実行！", actingMedarot.Name, part.PartName)

	isHit := CalculateHit(actingMedarot, target, part, balanceConfig)
	if isHit {
		damage, isCritical := CalculateDamage(actingMedarot, part, balanceConfig)
		targetPart := SelectRandomPartToDamage(target)
		if targetPart != nil {
			ApplyDamage(target, targetPart, damage)
			actingMedarot.LastActionLog = GenerateActionLog(actingMedarot, target, targetPart, damage, isCritical, true)
		} else {
			// これは target の全パーツが破壊された場合に発生
			actingMedarot.LastActionLog = fmt.Sprintf("%sの攻撃は%sに当たらなかった。", actingMedarot.Name, target.Name)
		}
	} else {
		actingMedarot.LastActionLog = GenerateActionLog(actingMedarot, target, nil, 0, false, false)
	}

	// 行動の結果自身が機能停止した場合（例：反動ダメージ、カウンターなど将来的な実装）を考慮
	if head := actingMedarot.GetPart(PartSlotHead); head != nil && head.IsBroken {
		actingMedarot.ChangeState(StateBroken)
	}
}


// --- ゲーム全体の進行を管理するシステム関数 ---

// SystemUpdateProgress は全メダロットのゲージ進行を処理する
func SystemUpdateProgress(medarots []*Medarot, game *Game) {
	for _, m := range medarots {
		if m.State != StateCharging && m.State != StateCooldown {
			continue
		}
		m.ProgressCounter++

		if m.TotalDuration > 0 {
			m.Gauge = (m.ProgressCounter / m.TotalDuration) * 100
		} else {
			m.Gauge = 100
		}

		if m.ProgressCounter >= m.TotalDuration {
			if m.State == StateCharging {
				m.ChangeState(StateReady)
				game.actionQueue = append(game.actionQueue, m) // actionQueueはGameが持つ
				log.Printf("%s のチャージが完了。実行キューに追加。", m.Name)
			} else if m.State == StateCooldown {
				m.ChangeState(StateIdle)
			}
		}
	}
}

// SystemProcessReadyQueue は行動準備完了のメダロットを処理する
func SystemProcessReadyQueue(g *Game) {
	if len(g.actionQueue) == 0 {
		return
	}
	// 実行順序を推進でソート
	sort.SliceStable(g.actionQueue, func(i, j int) bool {
		return g.actionQueue[i].GetOverallPropulsion() > g.actionQueue[j].GetOverallPropulsion()
	})

	// キューの先頭から1体だけ処理する
	if len(g.actionQueue) > 0 {
		actingMedarot := g.actionQueue[0]
		g.actionQueue = g.actionQueue[1:] // 処理したメダロットをキューから削除

		ExecuteAction(actingMedarot, &g.Config.Balance)

		g.enqueueMessage(actingMedarot.LastActionLog, func() {
			// 行動後に機能停止していなければクールダウンへ
			if actingMedarot.State != StateBroken {
				StartCooldown(actingMedarot, &g.Config.Balance)
			}
		})
	}
}

// SystemProcessIdleMedarots は待機中のメダロットを処理し、AIまたはプレイヤーの行動選択へ移行させる
func SystemProcessIdleMedarots(g *Game) {
	if g.playerMedarotToAct != nil || g.State != StatePlaying {
		return
	}

	// AIの行動選択
	for _, m := range g.Medarots {
		if m.State == StateIdle && m.Team != g.PlayerTeam {
			aiSelectAction(g, m) // ai.go内の関数を呼び出す
		}
	}

	// プレイヤーの行動選択
	nextPlayerMedarot := g.findNextIdlePlayerMedarot()
	if nextPlayerMedarot != nil {
		g.playerMedarotToAct = nextPlayerMedarot
		g.State = StatePlayerActionSelect
	}
}

// SystemCheckGameEnd はゲームの終了条件をチェックする
func SystemCheckGameEnd(g *Game) {
	if g.State == StateGameOver {
		return
	}
	team1Func := 0
	team2Func := 0
	for _, m := range g.Medarots {
		if m.State != StateBroken {
			if m.Team == Team1 {
				team1Func++
			} else {
				team2Func++
			}
		}
	}
	// チーム1リーダーの頭部が破壊されているか、またはチーム2が全滅した場合
	if g.team1Leader.GetPart(PartSlotHead).IsBroken || team2Func == 0 {
		g.winner = Team2
		g.State = StateGameOver
		g.enqueueMessage(fmt.Sprintf("%sが機能停止！ チーム2の勝利！", g.team1Leader.Name), nil)
	// チーム2リーダーの頭部が破壊されているか、またはチーム1が全滅した場合
	} else if g.team2Leader.GetPart(PartSlotHead).IsBroken || team1Func == 0 {
		g.winner = Team1
		g.State = StateGameOver
		g.enqueueMessage(fmt.Sprintf("%sが機能停止！ チーム1の勝利！", g.team2Leader.Name), nil)
	}
}