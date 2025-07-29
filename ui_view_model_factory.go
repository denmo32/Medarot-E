package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ViewModelFactory は各種ViewModelを生成するためのインターフェースです。
type ViewModelFactory interface {
	BuildInfoPanelViewModel(entry *donburi.Entry, partInfoProvider PartInfoProviderInterface) InfoPanelViewModel
	BuildBattlefieldViewModel(world donburi.World, battleUIState *BattleUIState, partInfoProvider PartInfoProviderInterface, config *Config, battlefieldRect image.Rectangle) BattlefieldViewModel
	BuildActionModalViewModel(actingEntry *donburi.Entry, actionTargetMap map[PartSlotKey]ActionTarget, partInfoProvider PartInfoProviderInterface, gameDataManager *GameDataManager, rand *rand.Rand) ActionModalViewModel
	GetAvailableAttackParts(entry *donburi.Entry) []AvailablePart
	IsActionModalVisible() bool
}

// viewModelFactoryImpl はViewModelFactoryインターフェースの実装です。
type viewModelFactoryImpl struct {
	partInfoProvider PartInfoProviderInterface
	gameDataManager  *GameDataManager
	rand             *rand.Rand
	ui               UIInterface
}

// NewViewModelFactory は新しいViewModelFactoryのインスタンスを作成します。
func NewViewModelFactory(world donburi.World, partInfoProvider PartInfoProviderInterface, gameDataManager *GameDataManager, rand *rand.Rand, ui UIInterface) ViewModelFactory { // world の型を donburi.World に変更
	return &viewModelFactoryImpl{
		partInfoProvider: partInfoProvider,
		gameDataManager:  gameDataManager,
		rand:             rand,
		ui:               ui,
	}
}

// BuildInfoPanelViewModel は、指定されたエンティティからInfoPanelViewModelを構築します。
func (f *viewModelFactoryImpl) BuildInfoPanelViewModel(entry *donburi.Entry, partInfoProvider PartInfoProviderInterface) InfoPanelViewModel {
	settings := SettingsComponent.Get(entry)
	state := StateComponent.Get(entry)
	partsComp := PartsComponent.Get(entry)

	partViewModels := make(map[PartSlotKey]PartViewModel)
	if partsComp != nil {
		for slotKey, partInst := range partsComp.Map {
			if partInst == nil {
				continue
			}
			partDef, defFound := partInfoProvider.GetGameDataManager().GetPartDefinition(partInst.DefinitionID)
			if !defFound {
				partViewModels[slotKey] = PartViewModel{PartName: "(不明)"}
				continue
			}

			partViewModels[slotKey] = PartViewModel{
				PartName:     partDef.PartName,
				CurrentArmor: partInst.CurrentArmor,
				MaxArmor:     partDef.MaxArmor,
				IsBroken:     partInst.IsBroken,
			}
		}
	}

	stateStr := GetStateDisplayName(state.CurrentState)

	return InfoPanelViewModel{
		ID:        settings.ID,
		Name:      settings.Name,
		Team:      settings.Team,
		DrawIndex: settings.DrawIndex,
		StateStr:  stateStr,
		IsLeader:  settings.IsLeader,
		Parts:     partViewModels,
	}
}

// BuildBattlefieldViewModel は、ワールドの状態からBattlefieldViewModelを構築します。
func (f *viewModelFactoryImpl) BuildBattlefieldViewModel(world donburi.World, battleUIState *BattleUIState, partInfoProvider PartInfoProviderInterface, config *Config, battlefieldRect image.Rectangle) BattlefieldViewModel {
	vm := BattlefieldViewModel{
		Icons: []*IconViewModel{},
		DebugMode: func() bool {
			_, ok := query.NewQuery(filter.Contains(DebugModeComponent)).First(world) // f.world の代わりに world を使用
			return ok
		}(),
	}

	query.NewQuery(filter.Contains(SettingsComponent)).Each(world, func(entry *donburi.Entry) { // f.world の代わりに world を使用
		settings := SettingsComponent.Get(entry)
		state := StateComponent.Get(entry)
		gauge := GaugeComponent.Get(entry)

		// バトルフィールドの描画領域を基準にX, Y座標を計算
		bfWidth := float32(battlefieldRect.Dx())
		bfHeight := float32(battlefieldRect.Dy())
		offsetX := float32(battlefieldRect.Min.X)
		offsetY := float32(battlefieldRect.Min.Y)

		x := f.CalculateMedarotScreenXPosition(entry, partInfoProvider, bfWidth)
		y := (bfHeight / float32(PlayersPerTeam+1)) * (float32(settings.DrawIndex) + 1)

		// オフセットを適用
		x += offsetX
		y += offsetY

		var iconColor color.Color
		if state.CurrentState == StateBroken {
			iconColor = config.UI.Colors.Broken
		} else if settings.Team == Team1 {
			iconColor = config.UI.Colors.Team1
		} else {
			iconColor = config.UI.Colors.Team2
		}

		var debugText string
		if vm.DebugMode { // ViewModelのDebugModeを使用
			stateStr := GetStateDisplayName(state.CurrentState)
			debugText = fmt.Sprintf(`State: %s\nGauge: %.1f\nProg: %.1f / %.1f`,
				stateStr, gauge.CurrentGauge, gauge.ProgressCounter, gauge.TotalDuration)
		}

		vm.Icons = append(vm.Icons, &IconViewModel{
			EntryID:       entry.Entity(), // uint32(entry.Id()) から entry.Entity() に変更
			X:             x,
			Y:             y,
			Color:         iconColor,
			IsLeader:      settings.IsLeader,
			State:         state.CurrentState,
			GaugeProgress: gauge.CurrentGauge / 100.0,
			DebugText:     debugText,
		})
	})

	return vm
}

// CalculateMedarotScreenXPosition はバトルフィールド上のアイコンのX座標を計算します。
// battlefieldWidth はバトルフィールドの表示幅です。
func (f *viewModelFactoryImpl) CalculateMedarotScreenXPosition(entry *donburi.Entry, partInfoProvider PartInfoProviderInterface, battlefieldWidth float32) float32 {
	settings := SettingsComponent.Get(entry)
	progress := partInfoProvider.GetNormalizedActionProgress(entry)

	homeX, execX := battlefieldWidth*0.1, battlefieldWidth*0.4
	if settings.Team == Team2 {
		homeX, execX = battlefieldWidth*0.9, battlefieldWidth*0.6
	}

	var xPos float32
	switch StateComponent.Get(entry).CurrentState {
	case StateCharging:
		xPos = homeX + (execX-homeX)*progress
	case StateReady:
		xPos = execX
	case StateCooldown:
		xPos = execX + (homeX - execX) * (1.0 - progress)
	case StateIdle, StateBroken:
		xPos = homeX
	default:
		xPos = homeX // 不明な状態の場合はホームポジション
	}
	return xPos
}

// BuildActionModalViewModel は、アクション選択モーダルに必要なViewModelを構築します。
func (f *viewModelFactoryImpl) BuildActionModalViewModel(actingEntry *donburi.Entry, actionTargetMap map[PartSlotKey]ActionTarget, partInfoProvider PartInfoProviderInterface, gameDataManager *GameDataManager, rand *rand.Rand) ActionModalViewModel {
	settings := SettingsComponent.Get(actingEntry)
	partsComp := PartsComponent.Get(actingEntry)

	var buttons []ActionModalButtonViewModel
	if partsComp == nil {
		// このエラーは通常、呼び出し元でエンティティの有効性を確認すべきですが、念のため
		// log.Println("エラー: BuildActionModalViewModel - actingEntry に PartsComponent がありません。")
	} else {
		var displayableParts []AvailablePart
		for slotKey, partInst := range partsComp.Map {
			partDef, defFound := gameDataManager.GetPartDefinition(partInst.DefinitionID)
			if !defFound {
				continue
			}
			// actionTargetMap に含まれるパーツのみを対象とする（行動可能なパーツ）
			if _, ok := actionTargetMap[slotKey]; ok {
				displayableParts = append(displayableParts, AvailablePart{PartDef: partDef, Slot: slotKey, IsBroken: partInst.IsBroken})
			}
		}

		for _, available := range displayableParts {
			targetInfo := actionTargetMap[available.Slot]
			buttons = append(buttons, ActionModalButtonViewModel{
				PartName:          available.PartDef.PartName,
				PartCategory:      available.PartDef.Category,
				SlotKey:           available.Slot,
				IsBroken:          available.IsBroken,
				TargetEntityID:    targetInfo.TargetEntityID, // TargetEntityID を直接使用
				TargetPartSlot:    targetInfo.Slot,
				SelectedPartDefID: available.PartDef.ID,
			})
		}
	}

	return ActionModalViewModel{
		ActingMedarotName: settings.Name,
		ActingEntityID:    actingEntry.Entity(),
		Buttons:           buttons,
	}
}

// GetAvailableAttackParts は、指定されたエンティティが利用可能な攻撃パーツのリストを返します。
func (f *viewModelFactoryImpl) GetAvailableAttackParts(entry *donburi.Entry) []AvailablePart {
	return f.partInfoProvider.GetAvailableAttackParts(entry)
}

func (f *viewModelFactoryImpl) IsActionModalVisible() bool {
	return f.ui.IsActionModalVisible()
}
