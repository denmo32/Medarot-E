package main

import (
	"fmt"
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

func createSingleMedarotInfoPanel(game *Game, medarot *Medarot) *infoPanelUI {
	c := game.Config.UI

	panelContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{50, 50, 70, 200})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(5)),
			widget.RowLayoutOpts.Spacing(2),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(int(c.InfoPanel.BlockWidth), 0)),
	)

	// 名前と状態
	headerContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{true, false}, []bool{true}),
		)),
	)
	panelContainer.AddChild(headerContainer)

	nameText := widget.NewText(
		widget.TextOpts.Text(medarot.Name, game.MplusFont, c.Colors.White),
	)
	headerContainer.AddChild(nameText)

	stateText := widget.NewText(
		widget.TextOpts.Text(string(medarot.State), game.MplusFont, c.Colors.Yellow),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{
			HorizontalPosition: widget.GridLayoutPositionEnd,
		})),
	)
	headerContainer.AddChild(stateText)

	// パーツ情報
	partSlots := make(map[PartSlotKey]*infoPanelPartUI)
	for _, slotKey := range []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs} {
		part := medarot.GetPart(slotKey)
		partName := "---"
		if part != nil {
			partName = part.PartName
		}

		partContainer := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(2),
			)),
		)
		panelContainer.AddChild(partContainer)

		partNameText := widget.NewText(
			widget.TextOpts.Text(partName, game.MplusFont, c.Colors.White),
		)
		partContainer.AddChild(partNameText)

		// [FIXED] ProgressBarの画像設定方法を修正しました
		hpBar := widget.NewProgressBar(
			widget.ProgressBarOpts.WidgetOpts(widget.WidgetOpts.MinSize(int(c.InfoPanel.PartHPGaugeWidth), int(c.InfoPanel.PartHPGaugeHeight))),
			widget.ProgressBarOpts.Images(
				// Fill (progress) image
				&widget.ProgressBarImage{
					Idle: image.NewNineSliceColor(c.Colors.HP),
				},
				// Track (background) image
				&widget.ProgressBarImage{
					Idle: image.NewNineSliceColor(c.Colors.Broken),
				},
			),
			widget.ProgressBarOpts.Values(0, 100, 0),
			widget.ProgressBarOpts.TrackPadding(widget.NewInsetsSimple(1)),
		)
		partContainer.AddChild(hpBar)

		hpText := widget.NewText(
			widget.TextOpts.Text("0/0", game.MplusFont, c.Colors.White),
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

func updateAllInfoPanels(game *Game) {
	for _, medarot := range game.Medarots {
		// グローバル変数 `medarotInfoPanelUIs` の代わりに `game.ui.medarotInfoPanels` を参照する
		ui, ok := game.ui.medarotInfoPanels[medarot.ID]
		if !ok {
			continue
		}
		updateSingleInfoPanel(medarot, ui, &game.Config)
	}
}

func updateSingleInfoPanel(medarot *Medarot, ui *infoPanelUI, config *Config) {
	c := config.UI
	ui.stateText.Label = string(medarot.State)

	if medarot.IsLeader {
		ui.nameText.Color = c.Colors.Leader
	} else {
		ui.nameText.Color = c.Colors.White
	}

	// [FIXED] 未使用変数エラーを解消するため、partUI変数を使用するようにしました
	for slotKey, partUI := range ui.partSlots {
		part := medarot.GetPart(slotKey)
		if part == nil {
			continue
		}

		currentArmor := part.Armor
		maxArmor := part.MaxArmor
		textColor := c.Colors.White
		hpPercentage := 0.0
		if maxArmor > 0 {
			hpPercentage = float64(currentArmor) / float64(maxArmor)
		}

		if part.IsBroken {
			textColor = c.Colors.Broken
		} else if hpPercentage < 0.3 {
			textColor = c.Colors.HPCritical
		}

		partUI.hpText.Label = fmt.Sprintf("%d / %d", currentArmor, maxArmor)
		partUI.hpText.Color = textColor
		partUI.partNameText.Color = textColor
		partUI.hpBar.SetCurrent(int(hpPercentage * 100))
	}
}
