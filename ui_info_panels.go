package main

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

func createSingleMedarotInfoPanel(config *Config, uiFactory *UIFactory, vm InfoPanelViewModel) *infoPanelUI {
	c := config.UI

	// ヘッダー部分を作成
	headerContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{true, false}, []bool{true}),
		)),
	)
	nameText := widget.NewText(
		widget.TextOpts.Text(vm.Name, uiFactory.Font, c.Colors.White),
	)
	headerContainer.AddChild(nameText)
	stateText := widget.NewText(
		widget.TextOpts.Text(vm.StateStr, uiFactory.Font, c.Colors.Yellow),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{
			HorizontalPosition: widget.GridLayoutPositionEnd,
		})),
	)
	headerContainer.AddChild(stateText)

	// パーツ部分のウィジェットを作成
	partWidgets := []widget.PreferredSizeLocateableWidget{}
	partSlots := make(map[PartSlotKey]*infoPanelPartUI)
	for _, slotKey := range []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs} {
		partVM, ok := vm.Parts[slotKey]
		partName := "---"
		initialArmor := 0.0

		if ok {
			partName = partVM.PartName
			initialArmor = float64(partVM.CurrentArmor)
		}

		partContainer := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(2),
			)),
		)
		partWidgets = append(partWidgets, partContainer)

		partNameText := widget.NewText(
			widget.TextOpts.Text(partName, uiFactory.Font, c.Colors.White),
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
			widget.TextOpts.Text("0/0", uiFactory.Font, c.Colors.White),
		)
		partContainer.AddChild(hpText)

		partSlots[slotKey] = &infoPanelPartUI{
			partNameText: partNameText,
			hpText:       hpText,
			hpBar:        hpBar,
			displayedHP:  initialArmor,
			targetHP:     initialArmor,
		}
	}

	// NewPanel を使用して全体のパネルを作成
	panelContainer := NewPanel(&PanelOptions{
		PanelWidth:      int(c.InfoPanel.BlockWidth),
		Padding:         widget.NewInsetsSimple(5),
		Spacing:         2,
		BackgroundColor: color.NRGBA{50, 50, 70, 200}, // 背景色を設定
		BorderColor:     c.Colors.Gray,                // 枠線の色
		BorderThickness: 5,                            // 枠線の太さ
	}, uiFactory.imageGenerator, uiFactory.Font, append([]widget.PreferredSizeLocateableWidget{headerContainer}, partWidgets...)...)

	return &infoPanelUI{
		rootContainer: panelContainer,
		nameText:      nameText,
		stateText:     stateText,
		partSlots:     partSlots,
	}
}



// CreateInfoPanels はすべてのメダロットの情報パネルを生成し、そのリストを返します。
// この関数はworldを直接クエリするのではなく、ViewModelFactoryまたはUpdateInfoPanelViewModelSystemが生成した
// InfoPanelViewModelのリストを受け取るように変更されます。
func CreateInfoPanels(config *Config, uiFactory *UIFactory, infoPanelVMs []InfoPanelViewModel) []InfoPanelCreationResult {
	var results []InfoPanelCreationResult
	// DrawIndexでソート
	sort.Slice(infoPanelVMs, func(i, j int) bool {
		if infoPanelVMs[i].Team != infoPanelVMs[j].Team {
			return infoPanelVMs[i].Team < infoPanelVMs[j].Team
		}
		return infoPanelVMs[i].DrawIndex < infoPanelVMs[j].DrawIndex
	})

	for _, vm := range infoPanelVMs {
		panelUI := createSingleMedarotInfoPanel(config, uiFactory, vm)
		results = append(results, InfoPanelCreationResult{
			PanelUI: panelUI,
			Team:    vm.Team,
			ID:      vm.ID,
		})
	}
	return results
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
