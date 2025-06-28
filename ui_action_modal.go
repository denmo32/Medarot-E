package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/yohamta/donburi"
)

func createActionModalUI(game *Game, actingEntry *donburi.Entry) widget.PreferredSizeLocateableWidget {
	c := game.Config.UI
	// 修正: ComponentType.Get を使用
	settings := SettingsComponent.Get(actingEntry)

	overlay := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0, 0, 0, 180})),
	)
	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{20, 20, 30, 255})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(c.ActionModal.ButtonSpacing),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(15)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(int(c.ActionModal.ButtonWidth)+30, 0),
		),
	)
	overlay.AddChild(panel)
	panel.AddChild(widget.NewText(
		widget.TextOpts.Text(fmt.Sprintf("行動選択: %s", settings.Name), game.MplusFont, c.Colors.White),
	))
	buttonImage := &widget.ButtonImage{
		Idle:    image.NewNineSliceColor(c.Colors.Gray),
		Hover:   image.NewNineSliceColor(color.RGBA{180, 180, 180, 255}),
		Pressed: image.NewNineSliceColor(color.RGBA{100, 100, 100, 255}),
	}
	availableParts := GetAvailableAttackParts(actingEntry)
	if len(availableParts) == 0 {
		panel.AddChild(widget.NewText(
			widget.TextOpts.Text("利用可能なパーツがありません。", game.MplusFont, c.Colors.White),
		))
	}
	for _, part := range availableParts {
		capturedPart := part
		actionButton := widget.NewButton(
			widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			})),
			widget.ButtonOpts.Image(buttonImage),
			widget.ButtonOpts.Text(fmt.Sprintf("%s (%s)", capturedPart.PartName, capturedPart.Category), game.MplusFont, &widget.ButtonTextColor{
				Idle: c.Colors.White,
			}),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				handleActionSelection(game, actingEntry, capturedPart)
			}),
		)
		panel.AddChild(actionButton)
	}
	cancelButton := widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Stretch: true,
		})),
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("キャンセル", game.MplusFont, &widget.ButtonTextColor{
			Idle: c.Colors.White,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			game.ui.HideActionModal()
			game.playerMedarotToAct = nil
			game.State = StatePlaying
		}),
	)
	panel.AddChild(cancelButton)
	return overlay
}

func handleActionSelection(game *Game, actingEntry *donburi.Entry, selectedPart *Part) {
	// 修正: ComponentType.Get を使用
	actingSettings := SettingsComponent.Get(actingEntry)
	var opponentTeamID TeamID = Team2
	if actingSettings.Team == Team1 {
		opponentTeamID = Team1
	}
	target := FindLeader(game.World, opponentTeamID)

	candidates := getTargetCandidates(game, actingEntry)
	if len(candidates) > 0 {
		isLeaderFound := false
		for _, cand := range candidates {
			// 修正: ComponentType.Get を使用
			if SettingsComponent.Get(cand).IsLeader {
				target = cand
				isLeaderFound = true
				break
			}
		}
		if !isLeaderFound {
			target = candidates[0]
		}
	} else {
		game.enqueueMessage("ターゲットがいません！", func() {
			game.playerMedarotToAct = nil
			game.State = StatePlaying
		})
		game.ui.HideActionModal()
		return
	}

	var slotKey PartSlotKey
	// 修正: ComponentType.Get を使用
	partsMap := PartsComponent.Get(actingEntry).Map
	for s, p := range partsMap {
		if p.ID == selectedPart.ID {
			slotKey = s
			break
		}
	}

	if StartCharge(actingEntry, slotKey, target, &game.Config.Balance) {
		game.ui.HideActionModal()
		game.playerMedarotToAct = nil
		game.State = StatePlaying
		SystemProcessIdleMedarots(game)
	} else {
		log.Printf("エラー: %s の行動選択に失敗しました。", actingSettings.Name)
		game.ui.HideActionModal()
		game.playerMedarotToAct = nil
		game.State = StatePlaying
	}
}
