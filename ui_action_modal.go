package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/yohamta/donburi"
)

func createActionModalUI(bs *BattleScene, actingEntry *donburi.Entry) widget.PreferredSizeLocateableWidget {
	c := bs.resources.Config.UI
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
		widget.TextOpts.Text(fmt.Sprintf("行動選択: %s", settings.Name), bs.resources.Font, c.Colors.White),
	))

	buttonImage := &widget.ButtonImage{
		Idle:    image.NewNineSliceColor(c.Colors.Gray),
		Hover:   image.NewNineSliceColor(color.RGBA{180, 180, 180, 255}),
		Pressed: image.NewNineSliceColor(color.RGBA{100, 100, 100, 255}),
	}

	availableParts := GetAvailableAttackParts(actingEntry)
	if len(availableParts) == 0 {
		panel.AddChild(widget.NewText(
			widget.TextOpts.Text("利用可能なパーツがありません。", bs.resources.Font, c.Colors.White),
		))
	}

	for _, part := range availableParts {
		slotKey := findPartSlot(actingEntry, part)
		if part.Category == CategoryShoot {
			targetEntity, targetSlot := playerSelectRandomTarget(bs, actingEntry)
			bs.ui.actionTargetMap[slotKey] = ActionTarget{Target: targetEntity, Slot: targetSlot}
		}
	}

	for _, part := range availableParts {
		capturedPart := part
		slotKey := findPartSlot(actingEntry, capturedPart)

		actionButton := widget.NewButton(
			widget.ButtonOpts.Image(buttonImage),
			widget.ButtonOpts.Text(fmt.Sprintf("%s (%s)", capturedPart.PartName, capturedPart.Category), bs.resources.Font, &widget.ButtonTextColor{
				Idle: c.Colors.White,
			}),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				handleActionSelection(bs, actingEntry, capturedPart)
			}),
			widget.ButtonOpts.CursorEnteredHandler(func(args *widget.ButtonHoverEventArgs) {
				if capturedPart.Category == CategoryShoot {
					if actionTarget, ok := bs.ui.actionTargetMap[slotKey]; ok {
						bs.currentTarget = actionTarget.Target
					}
				}
			}),
			widget.ButtonOpts.CursorExitedHandler(func(args *widget.ButtonHoverEventArgs) {
				bs.currentTarget = nil
			}),
		)
		panel.AddChild(actionButton)
	}

	cancelButton := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("キャンセル", bs.resources.Font, &widget.ButtonTextColor{
			Idle: c.Colors.White,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			bs.ui.HideActionModal()
			bs.playerMedarotToAct = nil
			bs.currentTarget = nil
			bs.state = StatePlaying
		}),
	)
	panel.AddChild(cancelButton)

	return overlay

}

func handleActionSelection(bs *BattleScene, actingEntry *donburi.Entry, selectedPart *Part) {
	slotKey := findPartSlot(actingEntry, selectedPart)
	var successful bool

	if selectedPart.Category == CategoryShoot {
		actionTarget, ok := bs.ui.actionTargetMap[slotKey]
		if !ok || actionTarget.Target == nil || actionTarget.Slot == "" {
			bs.enqueueMessage("ターゲットがいません！", func() {
				bs.playerMedarotToAct = nil
				bs.currentTarget = nil
				bs.state = StatePlaying
			})
			bs.ui.HideActionModal()
			return
		}
		successful = StartCharge(actingEntry, slotKey, actionTarget.Target, actionTarget.Slot, &bs.resources.Config.Balance)
	} else if selectedPart.Category == CategoryMelee {
		successful = StartCharge(actingEntry, slotKey, nil, "", &bs.resources.Config.Balance)
	} else {
		log.Printf("未対応のパーツカテゴリです: %s", selectedPart.Category)
		successful = false
	}

	if successful {
		bs.ui.HideActionModal()
		bs.playerMedarotToAct = nil
		bs.currentTarget = nil
		bs.state = StatePlaying
		SystemProcessIdleMedarots(bs)
	} else {
		log.Printf("エラー: %s の行動選択に失敗しました。", SettingsComponent.Get(actingEntry).Name)
		bs.ui.HideActionModal()
		bs.playerMedarotToAct = nil
		bs.currentTarget = nil
		bs.state = StatePlaying
	}

}
