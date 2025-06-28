package main

import (
	"log"
	"sort"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// SettingsComponent や PartsComponent などの変数が
// このように定義されていることを想定しています。
// var SettingsComponent = donburi.NewComponentType[Settings]()
// var PartsComponent = donburi.NewComponentType[Parts]()
// var StateComponent = donburi.NewComponentType[State]()

func aiSelectAction(g *Game, entry *donburi.Entry) {
	// 修正: ComponentType.Get(entry) を使用
	settings := SettingsComponent.Get(entry)
	availableParts := GetAvailableAttackParts(entry)
	if len(availableParts) == 0 {
		log.Printf("%s: AIは攻撃可能なパーツがないため待機。", settings.Name)
		return
	}

	targetCandidates := getTargetCandidates(g, entry)
	if len(targetCandidates) == 0 {
		log.Printf("%s: AIは攻撃対象がいないため待機。", settings.Name)
		return
	}

	var target *donburi.Entry
	for _, cand := range targetCandidates {
		// 修正: ComponentType.Get(entry) を使用
		if SettingsComponent.Get(cand).IsLeader {
			target = cand
			break
		}
	}
	if target == nil {
		target = targetCandidates[0]
	}

	selectedPart := availableParts[0]

	var slotKey PartSlotKey
	// 修正: ComponentType.Get(entry) を使用
	partsMap := PartsComponent.Get(entry).Map
	for s, p := range partsMap {
		if p.ID == selectedPart.ID {
			slotKey = s
			break
		}
	}

	StartCharge(entry, slotKey, target, &g.Config.Balance)
}

func getTargetCandidates(g *Game, actingEntry *donburi.Entry) []*donburi.Entry {
	// 修正: ComponentType.Get(entry) を使用
	actingSettings := SettingsComponent.Get(actingEntry)
	var opponentTeamID TeamID = Team2
	if actingSettings.Team == Team2 {
		opponentTeamID = Team1
	}

	candidates := []*donburi.Entry{}
	// 修正: filter.Contains() でコンポーネントをラップ
	query.NewQuery(filter.And(
		filter.Contains(SettingsComponent),
		filter.Contains(StateComponent),
	)).Each(g.World, func(entry *donburi.Entry) {
		// 修正: ComponentType.Get(entry) を使用
		settings := SettingsComponent.Get(entry)
		// 修正: ComponentType.Get(entry) を使用
		state := StateComponent.Get(entry)
		if settings.Team == opponentTeamID && state.State != StateBroken {
			candidates = append(candidates, entry)
		}
	})

	sort.Slice(candidates, func(i, j int) bool {
		// 修正: ComponentType.Get(entry) を使用
		iSettings := SettingsComponent.Get(candidates[i])
		// 修正: ComponentType.Get(entry) を使用
		jSettings := SettingsComponent.Get(candidates[j])
		return iSettings.DrawIndex < jSettings.DrawIndex
	})
	return candidates
}
