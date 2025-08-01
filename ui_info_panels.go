package main

import (
	"fmt"
	"image/color"
	"sort"

	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

type infoPanelUI struct {
	rootPanel *UIPanel // rootContainer を UIPanel に変更
	nameText  *widget.Text
	stateText *widget.Text
	partSlots map[PartSlotKey]*infoPanelPartUI
}

type infoPanelPartUI struct {
	partNameText *widget.Text
	hpText       *widget.Text
	hpBar        *widget.ProgressBar
	displayedHP  float64 // 現在表示されているHP
	targetHP     float64 // 目標とするHP
}

type InfoPanelManager struct {
	panels    map[string]*infoPanelUI
	config    *Config
	uiFactory *UIFactory
}

func NewInfoPanelManager(config *Config, uiFactory *UIFactory) *InfoPanelManager {
	return &InfoPanelManager{
		panels:    make(map[string]*infoPanelUI),
		config:    config,
		uiFactory: uiFactory,
	}
}

// InfoPanelCreationResult は生成された情報パネルとそのチーム情報を持つ構造体です。
type InfoPanelCreationResult struct {
	PanelUI *infoPanelUI
	Team    TeamID
	ID      string
}

func (ipm *InfoPanelManager) UpdatePanels(infoPanelVMs []InfoPanelViewModel, team1Container, team2Container *widget.Container) {
	// 既存のパネルをクリア
	for _, panel := range ipm.panels {
		team1Container.RemoveChild(panel.rootPanel.RootContainer) // RootContainer を使用
		team2Container.RemoveChild(panel.rootPanel.RootContainer) // RootContainer を使用
	}
	ipm.panels = make(map[string]*infoPanelUI) // マップをクリア

	// 新しいViewModelに基づいてパネルを再生成
	// DrawIndexでソート
	sort.Slice(infoPanelVMs, func(i, j int) bool {
		if infoPanelVMs[i].Team != infoPanelVMs[j].Team {
			return infoPanelVMs[i].Team < infoPanelVMs[j].Team
		}
		return infoPanelVMs[i].DrawIndex < infoPanelVMs[j].DrawIndex
	})

	for _, vm := range infoPanelVMs {
		panelUI := createSingleMedarotInfoPanel(ipm.config, ipm.uiFactory, vm)
		ipm.panels[vm.ID] = panelUI
		if vm.Team == Team1 {
			team1Container.AddChild(panelUI.rootPanel.RootContainer) // RootContainer を使用
		} else {
			team2Container.AddChild(panelUI.rootPanel.RootContainer) // RootContainer を使用
		}
	}

	// 各パネルのデータを更新
	for _, vm := range infoPanelVMs {
		if panel, ok := ipm.panels[vm.ID]; ok {
			updateSingleInfoPanel(panel, vm, ipm.config)
		}
	}
}

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
		partTypeStr := "---"
		initialArmor := 0.0

		if ok {
			partTypeStr = string(partVM.PartType)
			initialArmor = float64(partVM.CurrentArmor)
		}

		// 各パーツの行コンテナ
		partRowContainer := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(5),
			)),
		)
		partWidgets = append(partWidgets, partRowContainer)

		// 部位名テキスト
		partTypeText := widget.NewText(
			widget.TextOpts.Text(partTypeStr, uiFactory.Font, c.Colors.White),
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: false,
			})),
		)
		partRowContainer.AddChild(partTypeText)

		// HPバー
		hpBar := widget.NewProgressBar(
			widget.ProgressBarOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Stretch: true, // 横方向にストレッチ
				}),
				widget.WidgetOpts.MinSize(int(c.InfoPanel.PartHPGaugeWidth), int(c.InfoPanel.PartHPGaugeHeight)), // 最小高さを設定
			),
			widget.ProgressBarOpts.Images(
				&widget.ProgressBarImage{
					Idle: eimage.NewNineSliceColor(c.Colors.Gray),
				},
				&widget.ProgressBarImage{
					Idle: eimage.NewNineSliceColor(c.Colors.HP),
				},
			),
			widget.ProgressBarOpts.Values(0, 100, 100),
			widget.ProgressBarOpts.TrackPadding(widget.NewInsetsSimple(1)),
		)
		partRowContainer.AddChild(hpBar)

		// HPテキスト
		hpText := widget.NewText(
			widget.TextOpts.Text("0/0", uiFactory.Font, c.Colors.White),
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: false,
			})),
		)
		partRowContainer.AddChild(hpText)

		partSlots[slotKey] = &infoPanelPartUI{
			partNameText: partTypeText, // partTypeTextを再利用して部位名を表示
			hpText:       hpText,
			hpBar:        hpBar,
			displayedHP:  initialArmor,
			targetHP:     initialArmor,
		}
	}

	// NewPanel を使用して全体のパネルを作成
	panel := NewPanel(&PanelOptions{ // panelContainer を panel に変更
		PanelWidth:      int(c.InfoPanel.BlockWidth),
		Padding:         widget.NewInsetsSimple(5),
		Spacing:         2,
		BackgroundColor: color.NRGBA{50, 50, 70, 200}, // 背景色を設定
		BorderColor:     c.Colors.Gray,                // 枠線の色
		BorderThickness: 5,                            // 枠線の太さ
	}, uiFactory.imageGenerator, uiFactory.Font, append([]widget.PreferredSizeLocateableWidget{headerContainer}, partWidgets...)...)

	return &infoPanelUI{
		rootPanel: panel, // rootContainer を rootPanel に変更
		nameText:  nameText,
		stateText: stateText,
		partSlots: partSlots,
	}
}

// CreateInfoPanels はすべてのメダロットの情報パネルを生成し、そのリストを返します。
// この関数はworldを直接クエリするのではなく、ViewModelFactoryまたはUpdateInfoPanelViewModelSystemが生成した
// InfoPanelViewModelのリストを受け取るように変更されます。

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
			partUI.partNameText.Label = string(partVM.PartType)
		} else {
			partUI.partNameText.Label = string(partVM.PartType)
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
