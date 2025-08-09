package main

import (
	"fmt"
	"image"
	"image/color"
	"sort"

	"medarot-ebiten/core"
	"medarot-ebiten/ui"

	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/yohamta/donburi"
)

type infoPanelUI struct {
	rootPanel *UIPanel // rootContainer を UIPanel に変更
	nameText  *widget.Text
	stateText *widget.Text
	partSlots map[core.PartSlotKey]*infoPanelPartUI
}

type infoPanelPartUI struct {
	partNameText *widget.Text
	hpText       *widget.Text
	hpBar        *widget.ProgressBar
	displayedHP  float64 // 現在表示されているHP
	targetHP     float64 // 目標とするHP
}

type InfoPanelManager struct {
	panels    map[donburi.Entity]*infoPanelUI // map[string]からmap[donburi.Entity]に変更
	config    *Config
	uiFactory *UIFactory
}

func NewInfoPanelManager(config *Config, uiFactory *UIFactory) *InfoPanelManager {
	return &InfoPanelManager{
		panels:    make(map[donburi.Entity]*infoPanelUI), // map[string]からmap[donburi.Entity]に変更
		config:    config,
		uiFactory: uiFactory,
	}
}

// InfoPanelCreationResult は生成された情報パネルとそのチーム情報を持つ構造体です。
type InfoPanelCreationResult struct {
	PanelUI *infoPanelUI
	Team    core.TeamID
	ID      string
}

func (ipm *InfoPanelManager) UpdatePanels(infoPanelVMs []ui.InfoPanelViewModel, mainUIContainer *widget.Container, battlefieldRect image.Rectangle, iconVMs []*ui.IconViewModel) {
	// 既存のパネルをクリア
	for _, panel := range ipm.panels {
		mainUIContainer.RemoveChild(panel.rootPanel.RootContainer)
	}
	ipm.panels = make(map[donburi.Entity]*infoPanelUI) // マップのキーをdonburi.Entityに変更

	// アイコンのEntryIDをキーとしたY座標のマップを作成
	iconYMap := make(map[donburi.Entity]float32)
	for _, iconVM := range iconVMs {
		iconYMap[iconVM.EntryID] = iconVM.Y
	}

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
		ipm.panels[vm.EntityID] = panelUI // EntityIDをキーとして使用

		// アイコンのY座標を取得し、情報パネルのY座標として使用
		iconY, ok := iconYMap[vm.EntityID] // EntityIDをキーとして使用
		if !ok {
			// アイコンのY座標が見つからない場合は、デフォルトの位置に配置するか、エラー処理を行う
			// ここでは一旦スキップ
			continue
		}

		// バトルフィールドのオフセットを考慮して相対Y座標を計算
		relativeY := iconY - float32(battlefieldRect.Min.Y)

		// バトルフィールドの幅
		bfWidth := float32(battlefieldRect.Dx())
		// バトルフィールドのXオフセット
		bfOffsetX := float32(battlefieldRect.Min.X)

		var panelX int

		// PreferredSizeを使用して、レンダリング前に正しいサイズを取得
		panelWidth, panelHeight := panelUI.rootPanel.RootContainer.PreferredSize()

		if vm.Team == core.Team1 {
			panelX = int(bfOffsetX - float32(panelWidth) - float32(ipm.config.UI.InfoPanel.Padding)) // バトルフィールドの左側に配置
		} else {
			panelX = int(bfOffsetX + bfWidth + float32(ipm.config.UI.InfoPanel.Padding)) // バトルフィールドの右側に配置
		}

		// パネルの中心をアイコンのY座標に合わせる
		panelY := int(relativeY + float32(battlefieldRect.Min.Y) - float32(panelHeight)/2)

		// Rectを直接設定
		panelUI.rootPanel.RootContainer.GetWidget().Rect = image.Rect(panelX, panelY, panelX+panelWidth, panelY+panelHeight)
		mainUIContainer.AddChild(panelUI.rootPanel.RootContainer)
	}

	// 各パネルのデータを更新
	for _, vm := range infoPanelVMs {
		if panel, ok := ipm.panels[vm.EntityID]; ok {
			updateSingleInfoPanel(panel, vm, ipm.config)
		}
	}
}

func createSingleMedarotInfoPanel(config *Config, uiFactory *UIFactory, vm ui.InfoPanelViewModel) *infoPanelUI {
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
	partSlots := make(map[core.PartSlotKey]*infoPanelPartUI)
	for _, slotKey := range []core.PartSlotKey{core.PartSlotHead, core.PartSlotRightArm, core.PartSlotLeftArm, core.PartSlotLegs} {
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

func updateSingleInfoPanel(ui *infoPanelUI, vm ui.InfoPanelViewModel, config *Config) {
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
