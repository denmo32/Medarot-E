package system

import (
	"log"

	"medarot-ebiten/core"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ui"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdateBattleUIStateSystem は BattleUIStateComponent を更新し、UIに反映します。
func UpdateBattleUIStateSystem(
	world donburi.World,
	battleLogic BattleLogic,
	uiMediator core.UIMediator, // UIMediatorを受け取る
	battleUIManager ui.UIInterface,
) {
	battleUIStateEntry, ok := query.NewQuery(filter.Contains(ui.BattleUIStateComponent)).First(world)
	if !ok {
		log.Println("BattleUIStateComponent not found in world.")
		return
	}
	battleUIState := ui.BattleUIStateComponent.Get(battleUIStateEntry)

	// ViewModelを更新
	// UIMediatorを通じてViewModelFactoryにアクセスし、ViewModelを構築
	infoPanelVMs := make(map[string]core.InfoPanelViewModel)
	query.NewQuery(filter.Contains(component.SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		settings := component.SettingsComponent.Get(entry)
		infoPanelVM, err := uiMediator.(*ui.UIMediator).GetViewModelFactory().BuildInfoPanelViewModel(entry) // UIMediatorからViewModelFactoryを取得
		if err != nil {
			log.Printf("Error building info panel view model for %s: %v", settings.ID, err)
			return
		}
		infoPanelVMs[settings.ID] = infoPanelVM
	})
	battleUIState.InfoPanels = infoPanelVMs

	battlefieldViewModel, err := uiMediator.(*ui.UIMediator).GetViewModelFactory().BuildBattlefieldViewModel(world, battleUIManager.GetBattlefieldWidgetRect()) // UIMediatorからViewModelFactoryを取得
	if err != nil {
		log.Printf("Error building battlefield view model: %v", err)
	}
	battleUIState.BattlefieldViewModel = battlefieldViewModel

	// 更新されたViewModelをUIウィジェットに設定
	battleUIManager.SetBattleUIState(battleUIState)
}
