package main

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

func createSingleMedarotInfoPanel(bs *BattleScene, entry *donburi.Entry) *infoPanelUI {
	c := bs.resources.Config.UI
	settings := SettingsComponent.Get(entry)
	partsComp := PartsComponent.Get(entry) // This is *PartsComponentData
	partsMap := partsComp.Map              // map[PartSlotKey]*PartInstanceData

	panelContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{50, 50, 70, 200})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(5)),
			widget.RowLayoutOpts.Spacing(2),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(int(c.InfoPanel.BlockWidth), 0)),
	)

	headerContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{true, false}, []bool{true}),
		)),
	)
	panelContainer.AddChild(headerContainer)

	nameText := widget.NewText(
		widget.TextOpts.Text(settings.Name, bs.resources.Font, c.Colors.White),
	)
	headerContainer.AddChild(nameText)

	stateText := widget.NewText(
		widget.TextOpts.Text("待機", bs.resources.Font, c.Colors.Yellow),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{
			HorizontalPosition: widget.GridLayoutPositionEnd,
		})),
	)
	headerContainer.AddChild(stateText)

	partSlots := make(map[PartSlotKey]*infoPanelPartUI)
	for _, slotKey := range []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs} {
		partInst, instFound := partsMap[slotKey]
		partName := "---"
		if instFound && partInst != nil {
			if partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID); defFound {
				partName = partDef.PartName
			} else {
				partName = "(定義なし)"
			}
		}

		partContainer := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(2),
			)),
		)
		panelContainer.AddChild(partContainer)

		partNameText := widget.NewText(
			widget.TextOpts.Text(partName, bs.resources.Font, c.Colors.White),
		)
		partContainer.AddChild(partNameText)

		hpBar := widget.NewProgressBar(
			widget.ProgressBarOpts.WidgetOpts(widget.WidgetOpts.MinSize(int(c.InfoPanel.PartHPGaugeWidth), int(c.InfoPanel.PartHPGaugeHeight))),
			widget.ProgressBarOpts.Images(
				&widget.ProgressBarImage{
					Idle: image.NewNineSliceColor(c.Colors.Gray),
				},
				&widget.ProgressBarImage{
					Idle: image.NewNineSliceColor(c.Colors.HP),
				},
			),
			widget.ProgressBarOpts.Values(0, 100, 100),
			widget.ProgressBarOpts.TrackPadding(widget.NewInsetsSimple(1)),
		)
		partContainer.AddChild(hpBar)

		hpText := widget.NewText(
			widget.TextOpts.Text("0/0", bs.resources.Font, c.Colors.White),
		)
		partContainer.AddChild(hpText)

		partSlots[slotKey] = &infoPanelPartUI{
			partNameText: partNameText,
			hpText:       hpText,
			hpBar:        hpBar,
		}
	}

	return &infoPanelUI{
		rootContainer: panelContainer,
		nameText:      nameText,
		stateText:     stateText,
		partSlots:     partSlots,
	}
}

func setupInfoPanels(bs *BattleScene, team1Container, team2Container *widget.Container) {
	var entries []*donburi.Entry
	query.NewQuery(filter.Contains(SettingsComponent)).Each(bs.world, func(entry *donburi.Entry) {
		entries = append(entries, entry)
	})

	sort.Slice(entries, func(i, j int) bool {
		iSettings := SettingsComponent.Get(entries[i])
		jSettings := SettingsComponent.Get(entries[j])
		if iSettings.Team != jSettings.Team {
			return iSettings.Team < jSettings.Team
		}
		return iSettings.DrawIndex < jSettings.DrawIndex
	})

	for _, entry := range entries {
		settings := SettingsComponent.Get(entry)
		panelUI := createSingleMedarotInfoPanel(bs, entry)
		bs.ui.medarotInfoPanels[settings.ID] = panelUI
		if settings.Team == Team1 {
			team1Container.AddChild(panelUI.rootContainer)
		} else {
			team2Container.AddChild(panelUI.rootContainer)
		}
	}
}

func updateAllInfoPanels(bs *BattleScene) {
	query.NewQuery(filter.Contains(SettingsComponent)).Each(bs.world, func(entry *donburi.Entry) {
		settings := SettingsComponent.Get(entry)
		ui, ok := bs.ui.medarotInfoPanels[settings.ID]
		if !ok {
			return
		}
		updateSingleInfoPanel(entry, ui, &bs.resources.Config)
	})
}

func updateSingleInfoPanel(entry *donburi.Entry, ui *infoPanelUI, config *Config) {
	c := config.UI
	// settings := SettingsComponent.Get(entry) // Already available via entry in updateAllInfoPanels context
	partsComp := PartsComponent.Get(entry)
	if partsComp == nil {
		return // No parts component, nothing to update for parts
	}
	partsMap := partsComp.Map // map[PartSlotKey]*PartInstanceData

	var stateStr string
	if entry.HasComponent(IdleStateComponent) {
		stateStr = "待機"
	} else if entry.HasComponent(ChargingStateComponent) {
		stateStr = "チャージ中"
	} else if entry.HasComponent(ReadyStateComponent) {
		stateStr = "実行準備"
	} else if entry.HasComponent(CooldownStateComponent) {
		stateStr = "クールダウン"
	} else if entry.HasComponent(BrokenStateComponent) {
		stateStr = "機能停止"
	}
	ui.stateText.Label = stateStr

	// Leader color update was here, ensure settings is available or passed if needed for this.
	// Assuming settings is still accessible via entry if this function is called from updateAllInfoPanels.
	settings := SettingsComponent.Get(entry) // Re-get settings if not passed or available in wider scope
	if settings.IsLeader {
		ui.nameText.Color = c.Colors.Leader
	} else {
		ui.nameText.Color = c.Colors.White
	}

	for slotKey, partUI := range ui.partSlots {
		partInst, instFound := partsMap[slotKey]
		if !instFound || partInst == nil {
			// Optionally clear or hide this part's UI elements if it's missing
			partUI.partNameText.Label = "---"
			partUI.hpText.Label = "0/0"
			partUI.hpBar.SetCurrent(0)
			continue
		}

		partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID)
		if !defFound {
			partUI.partNameText.Label = "(定義なし)"
			partUI.hpText.Label = fmt.Sprintf("%d / ?", partInst.CurrentArmor)
			partUI.hpBar.SetCurrent(0) // Or some indication of error/unknown max
			continue
		}

		currentArmor := partInst.CurrentArmor
		maxArmor := partDef.MaxArmor // MaxArmor from PartDefinition
		textColor := c.Colors.White
		hpPercentage := 0.0
		if maxArmor > 0 {
			hpPercentage = float64(currentArmor) / float64(maxArmor)
		}

		if partInst.IsBroken { // IsBroken from PartInstanceData
			textColor = c.Colors.Broken
			partUI.partNameText.Label = partDef.PartName + " (壊)" // Indicate broken part name
		} else {
			partUI.partNameText.Label = partDef.PartName
			if hpPercentage < 0.3 {
				textColor = c.Colors.HPCritical
			}
		}

		partUI.hpText.Label = fmt.Sprintf("%d / %d", currentArmor, maxArmor)
		partUI.hpText.Color = textColor
		partUI.partNameText.Color = textColor
		partUI.hpBar.SetCurrent(int(hpPercentage * 100))
	}
}
