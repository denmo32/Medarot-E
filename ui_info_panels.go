package main

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

func createSingleMedarotInfoPanel(config *Config, font text.Face, entry *donburi.Entry) *infoPanelUI {
	c := config.UI
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
		widget.TextOpts.Text(settings.Name, font, c.Colors.White),
	)
	headerContainer.AddChild(nameText)

	stateText := widget.NewText(
		widget.TextOpts.Text("待機", font, c.Colors.Yellow),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{
			HorizontalPosition: widget.GridLayoutPositionEnd,
		})),
	)
	headerContainer.AddChild(stateText)

	partSlots := make(map[PartSlotKey]*infoPanelPartUI)
	for _, slotKey := range []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs} {
		partInst, instFound := partsMap[slotKey]
		partName := "---"
		initialArmor := 0.0 // float64に変更

		if instFound && partInst != nil {
			partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID)
			if defFound {
				partName = partDef.PartName
				initialArmor = float64(partInst.CurrentArmor) // float64にキャスト
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
			widget.TextOpts.Text(partName, font, c.Colors.White),
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
			widget.TextOpts.Text("0/0", font, c.Colors.White),
		)
		partContainer.AddChild(hpText)

		partSlots[slotKey] = &infoPanelPartUI{
			partNameText: partNameText,
			hpText:       hpText,
			hpBar:        hpBar,
			displayedHP:  initialArmor, // 初期値として現在のアーマーを設定
			targetHP:     initialArmor, // 初期値として現在のアーマーを設定
		}
	}

	return &infoPanelUI{
		rootContainer: panelContainer,
		nameText:      nameText,
		stateText:     stateText,
		partSlots:     partSlots,
	}
}

func setupInfoPanels(world donburi.World, config *Config, font text.Face, medarotInfoPanels map[string]*infoPanelUI, team1Container, team2Container *widget.Container) {
	var entries []*donburi.Entry
	query.NewQuery(filter.Contains(SettingsComponent)).Each(world, func(entry *donburi.Entry) {
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
		panelUI := createSingleMedarotInfoPanel(config, font, entry)
		medarotInfoPanels[settings.ID] = panelUI
		if settings.Team == Team1 {
			team1Container.AddChild(panelUI.rootContainer)
		} else {
			team2Container.AddChild(panelUI.rootContainer)
		}
	}
}

func updateAllInfoPanels(world donburi.World, config *Config, medarotInfoPanels map[string]*infoPanelUI) {
	query.NewQuery(filter.Contains(SettingsComponent)).Each(world, func(entry *donburi.Entry) {
		settings := SettingsComponent.Get(entry)
		ui, ok := medarotInfoPanels[settings.ID]
		if !ok {
			return
		}
		vm := BuildInfoPanelViewModel(entry)
		updateSingleInfoPanel(ui, vm, config)
	})
}

func updateSingleInfoPanel(ui *infoPanelUI, vm InfoPanelViewModel, config *Config) {
	c := config.UI

	ui.stateText.Label = vm.StateStr

	if vm.IsLeader {
		ui.nameText.Color = c.Colors.Leader
	} else {
		ui.nameText.Color = c.Colors.White
	}

	for slotKey, partUI := range ui.partSlots {
		partVM, ok := vm.Parts[slotKey]
		if !ok {
			partUI.partNameText.Label = "---"
			partUI.hpText.Label = "0/0"
			partUI.hpBar.SetCurrent(0)
			partUI.displayedHP = 0
			partUI.targetHP = 0
			continue
		}

		// 目標HPを設定
		partUI.targetHP = float64(partVM.CurrentArmor)

		// 現在表示されているHPを目標HPに近づける
		if partUI.displayedHP < partUI.targetHP {
			partUI.displayedHP += config.Balance.HPAnimationSpeed
			if partUI.displayedHP > partUI.targetHP {
				partUI.displayedHP = partUI.targetHP
			}
		} else if partUI.displayedHP > partUI.targetHP {
			partUI.displayedHP -= config.Balance.HPAnimationSpeed
			if partUI.displayedHP < partUI.targetHP {
				partUI.displayedHP = partUI.targetHP
			}
		}

		// 表示用のHP値に基づいて計算
		displayedArmor := int(partUI.displayedHP)
		maxArmor := partVM.MaxArmor
		textColor := c.Colors.White
		hpPercentage := 0.0
		if maxArmor > 0 {
			hpPercentage = partUI.displayedHP / float64(maxArmor)
		}

		if partVM.IsBroken {
			textColor = c.Colors.Broken
			partUI.partNameText.Label = partVM.PartName + " (壊)"
		} else {
			partUI.partNameText.Label = partVM.PartName
			if hpPercentage < 0.3 {
				textColor = c.Colors.HPCritical
			}
		}

		partUI.hpText.Label = fmt.Sprintf("%d / %d", displayedArmor, maxArmor)
		partUI.hpText.Color = textColor
		partUI.partNameText.Color = textColor
		partUI.hpBar.SetCurrent(int(hpPercentage * 100))
	}
}
