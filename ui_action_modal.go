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

	// モーダル表示時に各パーツのターゲットを事前計算して保存
	for _, part := range availableParts {
		slotKey := findPartSlot(actingEntry, part)
		if part.Category == CategoryShoot {
			// ★★★ 修正点: プレイヤー専用のターゲット選択関数を呼び出す ★★★
			targetEntity, targetSlot := playerSelectRandomTarget(game, actingEntry)
			game.ui.actionTargetMap[slotKey] = ActionTarget{Target: targetEntity, Slot: targetSlot}
		}
	}

	for _, part := range availableParts {
		capturedPart := part
		slotKey := findPartSlot(actingEntry, capturedPart)

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
			widget.ButtonOpts.CursorEnteredHandler(func(args *widget.ButtonHoverEventArgs) {
				if capturedPart.Category == CategoryShoot {
					if actionTarget, ok := game.ui.actionTargetMap[slotKey]; ok {
						game.currentTarget = actionTarget.Target
					}
				}
			}),
			widget.ButtonOpts.CursorExitedHandler(func(args *widget.ButtonHoverEventArgs) {
				game.currentTarget = nil
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
			game.currentTarget = nil
			game.State = StatePlaying
		}),
	)
	panel.AddChild(cancelButton)

	return overlay
}

func handleActionSelection(game *Game, actingEntry *donburi.Entry, selectedPart *Part) {
	slotKey := findPartSlot(actingEntry, selectedPart)
	var successful bool

	if selectedPart.Category == CategoryShoot {
		actionTarget, ok := game.ui.actionTargetMap[slotKey]
		if !ok || actionTarget.Target == nil || actionTarget.Slot == "" {
			game.enqueueMessage("ターゲットがいません！", func() {
				game.playerMedarotToAct = nil
				game.currentTarget = nil
				game.State = StatePlaying
			})
			game.ui.HideActionModal()
			return
		}
		successful = StartCharge(actingEntry, slotKey, actionTarget.Target, actionTarget.Slot, &game.Config.Balance)

	} else if selectedPart.Category == CategoryMelee {
		successful = StartCharge(actingEntry, slotKey, nil, "", &game.Config.Balance)
	} else {
		log.Printf("未対応のパーツカテゴリです: %s", selectedPart.Category)
		successful = false
	}

	if successful {
		game.ui.HideActionModal()
		game.playerMedarotToAct = nil
		game.currentTarget = nil
		game.State = StatePlaying
		SystemProcessIdleMedarots(game)
	} else {
		log.Printf("エラー: %s の行動選択に失敗しました。",
			SettingsComponent.Get(actingEntry).Name)
		game.ui.HideActionModal()
		game.playerMedarotToAct = nil
		game.currentTarget = nil
		game.State = StatePlaying
	}
}
