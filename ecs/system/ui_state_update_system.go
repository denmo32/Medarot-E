package system

import (
	"log"

	"medarot-ebiten/ui"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// UpdateBattleUIStateSystem は BattleUIStateComponent を更新し、UIに反映します。
func UpdateBattleUIStateSystem(
	world donburi.World,
	battleLogic BattleLogic,
	viewModelFactory ui.ViewModelFactoryInterface,
	battleUIManager ui.UIInterface,
) {
	battleUIStateEntry, ok := query.NewQuery(filter.Contains(ui.BattleUIStateComponent)).First(world)
	if !ok {
		log.Println("BattleUIStateComponent not found in world.")
		return
	}
	battleUIState := ui.BattleUIStateComponent.Get(battleUIStateEntry)

	// ViewModelを更新
	ui.UpdateInfoPanelViewModelSystem(battleUIState, world, battleLogic.GetPartInfoProvider(), viewModelFactory.(*ui.ViewModelFactoryImpl))
	battlefieldViewModel, err := viewModelFactory.BuildBattlefieldViewModel(world, battleUIManager.GetBattlefieldWidgetRect())
	if err != nil {
		log.Printf("Error building battlefield view model: %v", err)
	}
	battleUIState.BattlefieldViewModel = battlefieldViewModel

	// 更新されたViewModelをUIウィジェットに設定
	battleUIManager.SetBattleUIState(battleUIState)
}
