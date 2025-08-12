package ui

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ViewModelFactory はViewModelの生成に特化します。
type ViewModelFactory struct {
	partInfoProvider ViewModelPartInfoProvider
	gameDataManager  *data.GameDataManager
	rand             *rand.Rand
	uiConfigProvider UIConfigProvider
}

// NewViewModelFactory は新しいViewModelFactoryのインスタンスを作成します。
func NewViewModelFactory(partInfoProvider ViewModelPartInfoProvider, gameDataManager *data.GameDataManager, rand *rand.Rand, uiConfigProvider UIConfigProvider) *ViewModelFactory {
	return &ViewModelFactory{
		partInfoProvider: partInfoProvider,
		gameDataManager:  gameDataManager,
		rand:             rand,
		uiConfigProvider: uiConfigProvider,
	}
}

// BuildInfoPanelViewModel は、指定されたエンティティからInfoPanelViewModelを構築します。
func (f *ViewModelFactory) BuildInfoPanelViewModel(entry *donburi.Entry) (core.InfoPanelViewModel, error) {
	settings := component.SettingsComponent.Get(entry)
	state := component.StateComponent.Get(entry)
	partsComp := component.PartsComponent.Get(entry)

	partViewModels := make(map[core.PartSlotKey]core.PartViewModel)
	if partsComp != nil {
		for slotKey, partInst := range partsComp.Map {
			if partInst == nil {
				continue
			}
			partDef, defFound := f.gameDataManager.GetPartDefinition(partInst.DefinitionID)
			if !defFound {
				partViewModels[slotKey] = core.PartViewModel{PartName: "(不明)"}
				continue
			}

			partViewModels[slotKey] = core.PartViewModel{
				PartName:     partDef.PartName,
				PartType:     partDef.Type,
				CurrentArmor: partInst.CurrentArmor,
				MaxArmor:     partDef.MaxArmor,
				IsBroken:     partInst.IsBroken,
			}
		}
	}

	stateStr := GetStateDisplayName(state.CurrentState)

	return core.InfoPanelViewModel{
		ID:        settings.Name,
		EntityID:  entry.Entity(),
		Name:      settings.Name,
		Team:      settings.Team,
		DrawIndex: settings.DrawIndex,
		StateStr:  stateStr,
		IsLeader:  settings.IsLeader,
		Parts:     partViewModels,
	}, nil
}

// BuildBattlefieldViewModel は、ワールドの状態からBattlefieldViewModelを構築します。
func (f *ViewModelFactory) BuildBattlefieldViewModel(world donburi.World, battlefieldRect image.Rectangle) (core.BattlefieldViewModel, error) {
	vm := core.BattlefieldViewModel{
		Icons: []*core.IconViewModel{},
		DebugMode: func() bool {
			_, ok := query.NewQuery(filter.Contains(component.DebugModeComponent)).First(world)
			return ok
		}(),
	}

	query.NewQuery(filter.Contains(component.SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		settings := component.SettingsComponent.Get(entry)
		state := component.StateComponent.Get(entry)
		gauge := component.GaugeComponent.Get(entry)

		bfWidth := float32(battlefieldRect.Dx())
		bfHeight := float32(battlefieldRect.Dy())
		offsetX := float32(battlefieldRect.Min.X)
		offsetY := float32(battlefieldRect.Min.Y)

		x := f.CalculateMedarotScreenXPosition(entry, f.partInfoProvider, bfWidth)
		y := (bfHeight / float32(core.PlayersPerTeam+1)) * (float32(settings.DrawIndex) + 1)

		x += offsetX
		y += offsetY

		var iconColor color.Color
		if state.CurrentState == core.StateBroken {
			iconColor = f.uiConfigProvider.GetConfig().UI.Colors.Broken
		} else if settings.Team == core.Team1 {
			iconColor = f.uiConfigProvider.GetConfig().UI.Colors.Team1
		} else {
			iconColor = f.uiConfigProvider.GetConfig().UI.Colors.Team2
		}

		var debugText string
		if vm.DebugMode {
			stateStr := GetStateDisplayName(state.CurrentState)
			debugText = fmt.Sprintf(`State: %s
Gauge: %.1f
Prog: %.1f / %.1f`,
				stateStr, gauge.CurrentGauge, gauge.TotalDuration, gauge.ProgressCounter)
		}

		vm.Icons = append(vm.Icons, &core.IconViewModel{
			EntryID:       entry.Entity(),
			X:             x,
			Y:             y,
			Color:         iconColor,
			IsLeader:      settings.IsLeader,
			State:         state.CurrentState,
			GaugeProgress: gauge.CurrentGauge / 100.0,
			DebugText:     debugText,
		})
	})

	return vm, nil
}

// CalculateMedarotScreenXPosition はバトルフィールド上のアイコンのX座標を計算します。
// battlefieldWidth はバトルフィールドの表示幅です。
func (f *ViewModelFactory) CalculateMedarotScreenXPosition(entry *donburi.Entry, partInfoProvider ViewModelPartInfoProvider, battlefieldWidth float32) float32 {
	settings := component.SettingsComponent.Get(entry)
	progress := partInfoProvider.GetNormalizedActionProgress(entry)

	// ホームポジションと実行ラインのX座標を定義します。
	// これらの値は game_settings.json の UI.Battlefield.Team1HomeX, Team2HomeX, Team1ExecutionLineX, Team2ExecutionLineX に対応します。
	homeX, execX := battlefieldWidth*f.uiConfigProvider.GetConfig().UI.Battlefield.Team1HomeX, battlefieldWidth*f.uiConfigProvider.GetConfig().UI.Battlefield.Team1ExecutionLineX
	if settings.Team == core.Team2 {
		homeX, execX = battlefieldWidth*f.uiConfigProvider.GetConfig().UI.Battlefield.Team2HomeX, battlefieldWidth*f.uiConfigProvider.GetConfig().UI.Battlefield.Team2ExecutionLineX
	}

	var xPos float32
	switch component.StateComponent.Get(entry).CurrentState {
	case core.StateCharging:
		xPos = homeX + (execX-homeX)*progress
	case core.StateReady:
		xPos = execX
	case core.StateCooldown:
		xPos = execX + (homeX-execX)*(1.0-progress)
	case core.StateIdle, core.StateBroken:
		xPos = homeX
	default:
		xPos = homeX
	}
	return xPos
}

// BuildActionModalViewModel は、アクション選択モーダルに必要なViewModelを構築します。
func (f *ViewModelFactory) BuildActionModalViewModel(actingEntry *donburi.Entry, actionTargetMap map[core.PartSlotKey]core.ActionTarget) (core.ActionModalViewModel, error) {
	settings := component.SettingsComponent.Get(actingEntry)
	partsComp := component.PartsComponent.Get(actingEntry)

	var buttons []core.ActionModalButtonViewModel
	if partsComp == nil {
		return core.ActionModalViewModel{}, fmt.Errorf("actingEntry に PartsComponent がありません。")
	} else {
		var displayableParts []core.AvailablePart
		for slotKey, partInst := range partsComp.Map {
			partDef, defFound := f.gameDataManager.GetPartDefinition(partInst.DefinitionID)
			if !defFound {
				continue
			}
			if _, ok := actionTargetMap[slotKey]; ok {
				displayableParts = append(displayableParts, core.AvailablePart{PartDef: partDef, Slot: slotKey})
			}
		}

		for _, available := range displayableParts {
			targetInfo := actionTargetMap[available.Slot]
			buttons = append(buttons, core.ActionModalButtonViewModel{
				PartName:          available.PartDef.PartName,
				PartCategory:      available.PartDef.Category,
				SlotKey:           available.Slot,
				TargetEntityID:    targetInfo.TargetEntityID,
				TargetPartSlot:    targetInfo.Slot,
				SelectedPartDefID: available.PartDef.ID,
			})
		}
	}

	return core.ActionModalViewModel{
		ActingMedarotName: settings.Name,
		ActingEntityID:    actingEntry.Entity(),
		Buttons:           buttons,
	}, nil
}

// GetAvailableAttackParts は、指定されたエンティティが利用可能な攻撃パーツのリストを返します。
func (f *ViewModelFactory) GetAvailableAttackParts(entry *donburi.Entry) []core.AvailablePart {
	return f.partInfoProvider.GetAvailableAttackParts(entry)
}

// GetStateDisplayName は StateType に対応する日本語の表示名を返します。
func GetStateDisplayName(state core.StateType) string {
	switch state {
	case core.StateIdle:
		return "待機"
	case core.StateCharging:
		return "チャージ中"
	case core.StateReady:
		return "実行準備"
	case core.StateCooldown:
		return "クールダウン"
	case core.StateBroken:
		return "機能停止"
	default:
		return "不明"
	}
}

// UIConfigProvider はUI設定を提供するインターフェースです。
type UIConfigProvider interface {
	GetConfig() *data.Config
}
