package main

import (
	"log"
	"sort"
)

// aiSelectAction はAIメダロットの行動を決定し、チャージを開始させる
func aiSelectAction(game *Game, medarot *Medarot) {
	availableParts := medarot.GetAvailableAttackParts()
	if len(availableParts) == 0 {
		log.Printf("%s: AIは攻撃可能なパーツがないため待機。", medarot.Name)
		return
	}

	targetCandidates := getTargetCandidates(game, medarot)
	if len(targetCandidates) == 0 {
		log.Printf("%s: AIは攻撃対象がいないため待機。", medarot.Name)
		return
	}

	// === シンプルなAI思考ルーチン ===
	// 利用可能な最初のパーツで、相手のリーダーを優先して狙う
	var target *Medarot
	for _, cand := range targetCandidates {
		if cand.IsLeader {
			target = cand
			break
		}
	}
	if target == nil {
		target = targetCandidates[0]
	}

	selectedPart := availableParts[0]

	var slotKey PartSlotKey
	for s, p := range medarot.Parts {
		if p.ID == selectedPart.ID {
			slotKey = s
			break
		}
	}

	// [MODIFIED] メソッド呼び出しから関数呼び出しへ変更
	StartCharge(medarot, slotKey, target, &game.Config.Balance)
}

// getTargetCandidates は指定されたメダロットの攻撃対象候補リストを返す
func getTargetCandidates(game *Game, actingMedarot *Medarot) []*Medarot {
	candidates := []*Medarot{}
	var opponentTeamID TeamID = Team2
	if actingMedarot.Team == Team2 {
		opponentTeamID = Team1
	}

	for _, m := range game.Medarots {
		if m.Team == opponentTeamID && m.State != StateBroken {
			candidates = append(candidates, m)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].DrawIndex < candidates[j].DrawIndex
	})
	return candidates
}