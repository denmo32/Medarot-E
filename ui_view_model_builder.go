package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// BuildInfoPanelViewModel は、指定されたエンティティからInfoPanelViewModelを構築します。
func BuildInfoPanelViewModel(entry *donburi.Entry, battleLogic *BattleLogic) InfoPanelViewModel {
	settings := SettingsComponent.Get(entry)
	state := StateComponent.Get(entry)
	partsComp := PartsComponent.Get(entry)

	partViewModels := make(map[PartSlotKey]PartViewModel)
	if partsComp != nil {
		for slotKey, partInst := range partsComp.Map {
			if partInst == nil {
				continue
			}
			partDef, defFound := battleLogic.GetPartInfoProvider().gameDataManager.GetPartDefinition(partInst.DefinitionID)
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

	stateStr := GetStateDisplayName(StateType(state.FSM.Current()))

	return InfoPanelViewModel{
		MedarotName: settings.Name,
		StateStr:    stateStr,
		IsLeader:    settings.IsLeader,
		Parts:       partViewModels,
	}
}

// BuildBattlefieldViewModel は、ワールドの状態からBattlefieldViewModelを構築します。
func BuildBattlefieldViewModel(battleUIState *BattleUIState, battleLogic *BattleLogic, config *Config, battlefieldRect image.Rectangle) BattlefieldViewModel {
	vm := BattlefieldViewModel{
		Icons: []*IconViewModel{},
		DebugMode: func() bool {
			_, ok := query.NewQuery(filter.Contains(DebugModeComponent)).First(battleLogic.world)
			return ok
		}(),
	}

	query.NewQuery(filter.Contains(SettingsComponent)).Each(battleLogic.world, func(entry *donburi.Entry) {
		settings := SettingsComponent.Get(entry)
		state := StateComponent.Get(entry)
		gauge := GaugeComponent.Get(entry)

		// バトルフィールドの描画領域を基準にX, Y座標を計算
		bfWidth := float32(battlefieldRect.Dx())
		bfHeight := float32(battlefieldRect.Dy())
		offsetX := float32(battlefieldRect.Min.X)
		offsetY := float32(battlefieldRect.Min.Y)

		x := CalculateIconXPosition(entry, battleLogic.GetPartInfoProvider(), bfWidth)
		y := (bfHeight / float32(PlayersPerTeam+1)) * (float32(settings.DrawIndex) + 1)

		// オフセットを適用
		x += offsetX
		y += offsetY

		var iconColor color.Color
		if state.FSM.Is(string(StateBroken)) {
			iconColor = config.UI.Colors.Broken
		} else if settings.Team == Team1 {
			iconColor = config.UI.Colors.Team1
		} else {
			iconColor = config.UI.Colors.Team2
		}

		var debugText string
		if vm.DebugMode { // ViewModelのDebugModeを使用
			stateStr := GetStateDisplayName(StateType(state.FSM.Current()))
			debugText = fmt.Sprintf(`State: %s
Gauge: %.1f
Prog: %.1f / %.1f`,
				stateStr, gauge.CurrentGauge, gauge.ProgressCounter, gauge.TotalDuration)
		}

		vm.Icons = append(vm.Icons, &IconViewModel{
			EntryID:       uint32(entry.Id()),
			X:             x,
			Y:             y,
			Color:         iconColor,
			IsLeader:      settings.IsLeader,
			State:         StateType(state.FSM.Current()),
			GaugeProgress: gauge.CurrentGauge / 100.0,
			DebugText:     debugText,
		})
	})

	return vm
}

// CalculateIconXPosition はバトルフィールド上のアイコンのX座標を計算します。
// worldWidth はバトルフィールドの表示幅です。
func CalculateIconXPosition(entry *donburi.Entry, partInfoProvider *PartInfoProvider, battlefieldWidth float32) float32 {
	return partInfoProvider.CalculateMedarotXPosition(entry, battlefieldWidth)
}

// BuildActionModalViewModel は、アクション選択モーダルに必要なViewModelを構築します。
func BuildActionModalViewModel(actingEntry *donburi.Entry, actionTargetMap map[PartSlotKey]ActionTarget, battleLogic *BattleLogic) ActionModalViewModel {
	settings := SettingsComponent.Get(actingEntry)
	partsComp := PartsComponent.Get(actingEntry)

	var buttons []ActionModalButtonViewModel
	if partsComp == nil {
		// このエラーは通常、呼び出し元でエンティティの有効性を確認すべきですが、念のため
		// log.Println("エラー: BuildActionModalViewModel - actingEntry に PartsComponent がありません。")
	} else {
		var displayableParts []AvailablePart
		for slotKey, partInst := range partsComp.Map {
			partDef, defFound := battleLogic.GetPartInfoProvider().gameDataManager.GetPartDefinition(partInst.DefinitionID)
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
				PartName:        available.PartDef.PartName,
				PartCategory:    available.PartDef.Category,
				SlotKey:         available.Slot,
				IsBroken:        available.IsBroken,
				TargetEntry:     targetInfo.Target,
				SelectedPartDef: available.PartDef,
			})
		}
	}

	return ActionModalViewModel{
		ActingMedarotName: settings.Name,
		ActingEntry:       actingEntry,
		Buttons:           buttons,
	}
}
