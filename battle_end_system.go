package main

import (
	"fmt"
	"medarot-ebiten/core"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// CheckGameEndSystem はゲーム終了条件をチェックします。
// BattleScene への依存をなくし、結果を構造体で返します。
func CheckGameEndSystem(world donburi.World) core.GameEndResult {
	team1Leader := FindLeader(world, core.Team1) // FindLeader は ecs_setup.go にあります
	team2Leader := FindLeader(world, core.Team2) // FindLeader は ecs_setup.go にあります

	team1FuncCount := 0
	team2FuncCount := 0

	query.NewQuery(filter.Contains(SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		if StateComponent.Get(entry).CurrentState != core.StateBroken {
			if SettingsComponent.Get(entry).Team == core.Team1 {
				team1FuncCount++
			} else {
				team2FuncCount++
			}
		}
	})

	var winner core.TeamID
	var gameOverMsg string
	isGameOver := false

	// リーダーが nil の場合、そのチームのリーダーが存在しない（破壊されたとは異なります）。
	// 通常、リーダーは戦闘開始時に設定されるため、nil になるのは異常か、
	// あるいはリーダーが戦闘から除外されるような特殊なケースです。
	// ここでは、リーダーが破壊された場合と、行動可能な機体がいなくなった場合をチェックします。

	// チーム1の敗北条件：
	// 1. チーム1のリーダーが存在しない (nil)
	// 2. チーム1のリーダーの頭部が破壊されている
	// 3. チーム1の行動可能な機体が0 (リーダーが健在でも他の機体が全滅)
	if team1Leader == nil || PartsComponent.Get(team1Leader).Map[core.PartSlotHead].IsBroken || team1FuncCount == 0 {
		winner = core.Team2
		isGameOver = true
		if team1Leader != nil && PartsComponent.Get(team1Leader).Map[core.PartSlotHead].IsBroken {
			gameOverMsg = fmt.Sprintf("%sが機能停止！ チーム2の勝利！", SettingsComponent.Get(team1Leader).Name)
		} else if team1FuncCount == 0 {
			gameOverMsg = "チーム1が全滅！ チーム2の勝利！"
		} else { // team1Leader == nil のケースなど
			gameOverMsg = "チーム1リーダー不在または機能停止！ チーム2の勝利！"
		}
	}

	// チーム2の敗北条件 (チーム1がまだ負けていない場合)：
	// 1. チーム2のリーダーが存在しない (nil)
	// 2. チーム2のリーダーの頭部が破壊されている
	// 3. チーム2の行動可能な機体が0 (リーダーが健在でも他の機体が全滅)
	if !isGameOver { // チーム1がまだ敗北していない場合のみチーム2の敗北をチェック
		if team2Leader == nil || PartsComponent.Get(team2Leader).Map[core.PartSlotHead].IsBroken || team2FuncCount == 0 {
			winner = core.Team1
			isGameOver = true
			if team2Leader != nil && PartsComponent.Get(team2Leader).Map[core.PartSlotHead].IsBroken {
				gameOverMsg = fmt.Sprintf("%sが機能停止！ チーム1の勝利！", SettingsComponent.Get(team2Leader).Name)
			} else if team2FuncCount == 0 {
				gameOverMsg = "チーム2が全滅！ チーム1の勝利！"
			} else { // team2Leader == nil のケースなど
				gameOverMsg = "チーム2リーダー不在または機能停止！ チーム1の勝利！"
			}
		}
	}

	return core.GameEndResult{
		IsGameOver: isGameOver,
		Winner:     winner,
		Message:    gameOverMsg,
	}
}
