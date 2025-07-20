package main

import (
	"image"

	"github.com/yohamta/donburi"
)

// ViewModelFactory は各種ViewModelを生成するためのインターフェースです。
type ViewModelFactory interface {
	BuildInfoPanelViewModel(entry *donburi.Entry, battleLogic *BattleLogic) InfoPanelViewModel
	BuildBattlefieldViewModel(battleUIState *BattleUIState, battleLogic *BattleLogic, config *Config, battlefieldRect image.Rectangle) BattlefieldViewModel
	BuildActionModalViewModel(actingEntry *donburi.Entry, actionTargetMap map[PartSlotKey]ActionTarget, battleLogic *BattleLogic) ActionModalViewModel
}

// viewModelFactoryImpl はViewModelFactoryインターフェースの実装です。
type viewModelFactoryImpl struct {
	world *donburi.World
}

// NewViewModelFactory は新しいViewModelFactoryのインスタンスを作成します。
func NewViewModelFactory(world *donburi.World) ViewModelFactory {
	return &viewModelFactoryImpl{
		world: world,
	}
}

// BuildInfoPanelViewModel は、指定されたエンティティからInfoPanelViewModelを構築します。
func (f *viewModelFactoryImpl) BuildInfoPanelViewModel(entry *donburi.Entry, battleLogic *BattleLogic) InfoPanelViewModel {
	return BuildInfoPanelViewModel(entry, battleLogic)
}

// BuildBattlefieldViewModel は、ワールドの状態からBattlefieldViewModelを構築します。
func (f *viewModelFactoryImpl) BuildBattlefieldViewModel(battleUIState *BattleUIState, battleLogic *BattleLogic, config *Config, battlefieldRect image.Rectangle) BattlefieldViewModel {
	return BuildBattlefieldViewModel(battleUIState, battleLogic, config, battlefieldRect)
}

// BuildActionModalViewModel は、アクション選択モーダルに必要なViewModelを構築します。
func (f *viewModelFactoryImpl) BuildActionModalViewModel(actingEntry *donburi.Entry, actionTargetMap map[PartSlotKey]ActionTarget, battleLogic *BattleLogic) ActionModalViewModel {
	return BuildActionModalViewModel(actingEntry, actionTargetMap, battleLogic)
}
