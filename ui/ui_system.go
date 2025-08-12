package ui

import (
	"log"

	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdateInfoPanelViewModelSystem は、すべてのメダロットエンティティからInfoPanelViewModelを構築し、BattleUIStateComponentに格納します。
func UpdateInfoPanelViewModelSystem(battleUIState *BattleUIState, world donburi.World, partInfoProvider ViewModelPartInfoProvider, factory *ViewModelFactory) { // 型名を修正

	query.NewQuery(filter.Contains(component.SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		settings := component.SettingsComponent.Get(entry)
		// state := component.StateComponent.Get(entry) // declared and not used
		// partsData := component.PartsComponent.Get(entry) // declared and not used

		infoPanelVM, err := factory.BuildInfoPanelViewModel(entry)
		if err != nil {
			log.Printf("Error building info panel view model for %s: %v", settings.ID, err)
			return // Or handle the error appropriately
		}
		battleUIState.InfoPanels[settings.ID] = infoPanelVM
	})
}
