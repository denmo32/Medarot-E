package ui

import (
	"fmt"
	"image/color"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ViewModelPartInfoProvider は ViewModelFactory がパーツ情報にアクセスするために必要なインターフェースです。
// このインターフェースは ecs/system.PartInfoProvider によって実装されます。
type ViewModelPartInfoProvider interface {
	GetAvailableAttackParts(entry *donburi.Entry) []core.AvailablePart
	GetNormalizedActionProgress(entry *donburi.Entry) float32
}

// ViewModelFactory はViewModelの生成に特化します。
type ViewModelFactory struct {
	partInfoProvider ViewModelPartInfoProvider
	gameDataManager  *data.GameDataManager
	rand             *rand.Rand
}

// NewViewModelFactory は新しいViewModelFactoryのインスタンスを作成します。
func NewViewModelFactory(partInfoProvider ViewModelPartInfoProvider, gameDataManager *data.GameDataManager, rand *rand.Rand) *ViewModelFactory {
	return &ViewModelFactory{
		partInfoProvider: partInfoProvider,
		gameDataManager:  gameDataManager,
		rand:             rand,
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
func (f *ViewModelFactory) BuildBattlefieldViewModel(world donburi.World) (core.BattlefieldViewModel, error) {
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
		progress := f.partInfoProvider.GetNormalizedActionProgress(entry)

		var iconColor color.Color
		// 色の決定はUIコンポーネントで行うため、ここでは仮の色を設定
		// または、ViewModelに色情報を含めないように変更
		// 今回はViewModelに色情報を含めるため、configから色を取得するロジックを削除
		// configはBuildBattlefieldViewModelの引数から削除されたため、ここではアクセスできない
		// 一旦、デフォルトの色を設定
		if state.CurrentState == core.StateBroken {
			iconColor = color.RGBA{255, 0, 0, 255} // 赤
		} else if settings.Team == core.Team1 {
			iconColor = color.RGBA{0, 0, 255, 255} // 青
		} else {
			iconColor = color.RGBA{0, 255, 0, 255} // 緑
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
			EntryID:            entry.Entity(),
			Team:               settings.Team,
			DrawIndex:          settings.DrawIndex,
			NormalizedProgress: float64(progress),
			Color:              iconColor, // 仮の色
			IsLeader:           settings.IsLeader,
			State:              state.CurrentState,
			GaugeProgress:      gauge.CurrentGauge / 100.0,
			DebugText:          debugText,
		})
	})

	return vm, nil
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
