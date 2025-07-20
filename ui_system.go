package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdateInfoPanelViewModelSystem は、すべてのメダロットエンティティからInfoPanelViewModelを構築し、BattleUIStateComponentに格納します。
func UpdateInfoPanelViewModelSystem(world donburi.World, battleLogic *BattleLogic) {
	battleUIStateEntry, ok := query.NewQuery(filter.Contains(BattleUIStateComponent)).First(world)
	if !ok {
		log.Println("エラー: BattleUIStateComponent がワールドに見つかりません。")
		return
	}
	battleUIState := BattleUIStateComponent.Get(battleUIStateEntry)

	query.NewQuery(filter.Contains(SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		settings := SettingsComponent.Get(entry)
		battleUIState.InfoPanels[settings.ID] = BuildInfoPanelViewModel(entry, battleLogic)
	})
}
