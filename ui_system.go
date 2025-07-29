package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdateInfoPanelViewModelSystem は、すべてのメダロットエンティティからInfoPanelViewModelを構築し、BattleUIStateComponentに格納します。
func UpdateInfoPanelViewModelSystem(battleUIState *BattleUIState, world donburi.World, partInfoProvider PartInfoProviderInterface, factory ViewModelFactory) {

	query.NewQuery(filter.Contains(SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		settings := SettingsComponent.Get(entry)
		battleUIState.InfoPanels[settings.ID] = factory.BuildInfoPanelViewModel(entry, partInfoProvider)
	})
}
