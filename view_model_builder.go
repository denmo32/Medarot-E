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
func BuildInfoPanelViewModel(entry *donburi.Entry) InfoPanelViewModel {
	settings := SettingsComponent.Get(entry)
	state := StateComponent.Get(entry)
	partsComp := PartsComponent.Get(entry)

	partViewModels := make(map[PartSlotKey]PartViewModel)
	if partsComp != nil {
		for slotKey, partInst := range partsComp.Map {
			if partInst == nil {
				continue
			}
			partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID)
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

	var stateStr string
	switch state.Current {
	case StateTypeIdle:
		stateStr = "待機"
	case StateTypeCharging:
		stateStr = "チャージ中"
	case StateTypeReady:
		stateStr = "実行準備"
	case StateTypeCooldown:
		stateStr = "クールダウン"
	case StateTypeBroken:
		stateStr = "機能停止"
	}

	return InfoPanelViewModel{
		MedarotName: settings.Name,
		StateStr:    stateStr,
		IsLeader:    settings.IsLeader,
		Parts:       partViewModels,
	}
}

// BuildBattlefieldViewModel は、ワールドの状態からBattlefieldViewModelを構築します。
func BuildBattlefieldViewModel(world donburi.World, partInfoProvider *PartInfoProvider, config *Config, debugMode bool, battlefieldRect image.Rectangle) BattlefieldViewModel {
	vm := BattlefieldViewModel{
		Icons: []*IconViewModel{},
	}

	query.NewQuery(filter.Contains(SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		settings := SettingsComponent.Get(entry)
		state := StateComponent.Get(entry)
		gauge := GaugeComponent.Get(entry)

		// バトルフィールドの描画領域を基準にX, Y座標を計算
		bfWidth := float32(battlefieldRect.Dx())
		bfHeight := float32(battlefieldRect.Dy())
		offsetX := float32(battlefieldRect.Min.X)
		offsetY := float32(battlefieldRect.Min.Y)

		x := partInfoProvider.CalculateIconXPosition(entry, bfWidth)
		y := (bfHeight / float32(PlayersPerTeam+1)) * (float32(settings.DrawIndex) + 1)

		// オフセットを適用
		x += offsetX
		y += offsetY

		var iconColor color.Color
		if state.Current == StateTypeBroken {
			iconColor = config.UI.Colors.Broken
		} else if settings.Team == Team1 {
			iconColor = config.UI.Colors.Team1
		} else {
			iconColor = config.UI.Colors.Team2
		}

		var debugText string
		if debugMode {
			stateStr := ""
			switch state.Current {
			case StateTypeIdle:
				stateStr = "待機"
			case StateTypeCharging:
				stateStr = "チャージ中"
			case StateTypeReady:
				stateStr = "実行準備"
			case StateTypeCooldown:
				stateStr = "クールダウン"
			case StateTypeBroken:
				stateStr = "機能停止"
			}
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
			State:         state.Current,
			GaugeProgress: gauge.CurrentGauge / 100.0,
			DebugText:     debugText,
		})
	})

	return vm
}
