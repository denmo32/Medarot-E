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

	if bs.partInfoProvider == nil {
		log.Println("Error: createActionModalUI - partInfoProvider is nil")
		// partInfoProvider がないとパーツリストを取得できないため、モーダルを生成せずに終了するなどのエラーハンドリングが必要
		panel.AddChild(widget.NewText(
			widget.TextOpts.Text("エラー:パーツ情報取得不可", bs.resources.Font, c.Colors.White),
		))
		// overlayにpanelを追加しているので、このままでは空のモーダルが表示される。より適切なハンドリングを検討。
		return overlay
	}
	availableParts := bs.partInfoProvider.GetAvailableAttackParts(actingEntry)
	if len(availableParts) == 0 {
		panel.AddChild(widget.NewText(
			widget.TextOpts.Text("利用可能なパーツがありません。", bs.resources.Font, c.Colors.White),
		))
	}

	for _, available := range availableParts { // available is of type AvailablePart { PartDef *PartDefinition, Slot PartSlotKey }
		partDef := available.PartDef // Use PartDef
		slotKey := available.Slot
		if partDef.Category == CategoryShoot {
			targetEntity, targetSlot := playerSelectRandomTarget(bs, actingEntry) // This helper might need update if it uses part info
			bs.ui.actionTargetMap[slotKey] = ActionTarget{Target: targetEntity, Slot: targetSlot}
		}
	}

	for _, available := range availableParts { // available is of type AvailablePart
		capturedPartDef := available.PartDef // Use PartDef, capture it for the handler
		capturedSlotKey := available.Slot    // Capture slot key as well for consistency if needed by handler

		actionButton := widget.NewButton(
			widget.ButtonOpts.Image(buttonImage),
			widget.ButtonOpts.Text(fmt.Sprintf("%s (%s)", capturedPartDef.PartName, capturedPartDef.Category), bs.resources.Font, &widget.ButtonTextColor{
				Idle: c.Colors.White,
			}),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				// Pass PartDefinition and its original slot key to handleActionSelection
				handleActionSelection(bs, actingEntry, capturedPartDef, capturedSlotKey)
			}),
			widget.ButtonOpts.CursorEnteredHandler(func(args *widget.ButtonHoverEventArgs) {
				if capturedPartDef.Category == CategoryShoot {
					if actionTarget, ok := bs.ui.actionTargetMap[capturedSlotKey]; ok { // Use capturedSlotKey
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
			bs.playerActionPendingQueue = make([]*donburi.Entry, 0) // Clear the pending queue
			bs.playerMedarotToAct = nil
			bs.currentTarget = nil
			bs.state = StatePlaying
		}),
	)
	panel.AddChild(cancelButton)

	return overlay

}

func handleActionSelection(bs *BattleScene, actingEntry *donburi.Entry, selectedPartDef *PartDefinition, slotKey PartSlotKey) {
	// partInfoProvider is not directly needed here if slotKey is passed and valid.
	// However, FindPartSlot was used to get slotKey. Now slotKey is passed directly.
	if slotKey == "" { // Should be passed a valid slotKey from createActionModalUI
		log.Printf("Error: handleActionSelection - received empty slotKey for part %s", selectedPartDef.PartName)
		bs.ui.HideActionModal()
		// Reset state as if cancelled
		bs.playerActionPendingQueue = make([]*donburi.Entry, 0)
		bs.playerMedarotToAct = nil
		bs.currentTarget = nil
		bs.state = StatePlaying
		return
	}

	var successful bool

	switch selectedPartDef.Category {
	case CategoryShoot: // Use selectedPartDef
		actionTarget, ok := bs.ui.actionTargetMap[slotKey] // slotKey is now directly available
		if !ok || actionTarget.Target == nil || actionTarget.Slot == "" {
			bs.enqueueMessage("ターゲットがいません！", func() { // This will change state to StateMessage
				bs.playerMedarotToAct = nil
				bs.currentTarget = nil
				bs.state = StatePlaying
			})
			bs.ui.HideActionModal()
			return
		}
		// Pass bs.world, &bs.resources.Config, and bs.partInfoProvider to StartCharge
		successful = StartCharge(actingEntry, slotKey, actionTarget.Target, actionTarget.Slot, bs.world, &bs.resources.Config, bs.partInfoProvider)
	case CategoryMelee: // Use selectedPartDef
		// Pass bs.world, &bs.resources.Config, and bs.partInfoProvider to StartCharge
		successful = StartCharge(actingEntry, slotKey, nil, "", bs.world, &bs.resources.Config, bs.partInfoProvider)
	default:
		log.Printf("未対応のパーツカテゴリです: %s", selectedPartDef.Category) // Use selectedPartDef
		successful = false
	}

	if successful {
		bs.ui.HideActionModal() // Hide current modal first
		bs.currentTarget = nil  // Clear target indicator

		// Dequeue the current medarot
		if len(bs.playerActionPendingQueue) > 0 && bs.playerActionPendingQueue[0] == actingEntry {
			bs.playerActionPendingQueue = bs.playerActionPendingQueue[1:]
		}

		if len(bs.playerActionPendingQueue) > 0 {
			// There are more players waiting, set up for the next one
			bs.playerMedarotToAct = bs.playerActionPendingQueue[0] // Already set by BattleScene's update loop, but good to be explicit
			bs.state = StatePlayerActionSelect                     // Ensure state is correct for modal display next frame
			// UI should re-create the modal for the new playerMedarotToAct in the next Update cycle of BattleScene
		} else {
			// No more players in the queue
			bs.playerMedarotToAct = nil
			bs.state = StatePlaying
		}
	} else {
		// Action was not successful (e.g., part broken, no target)
		log.Printf("エラー: %s の行動選択に失敗しました。", SettingsComponent.Get(actingEntry).Name)
		bs.ui.HideActionModal()
		// If action failed, treat as if this player's turn is done for now regarding the queue.
		// This logic might need refinement: should it try next player or reset queue?
		// For now, similar to successful action, try to proceed with queue.
		if len(bs.playerActionPendingQueue) > 0 && bs.playerActionPendingQueue[0] == actingEntry {
			bs.playerActionPendingQueue = bs.playerActionPendingQueue[1:]
		}
		if len(bs.playerActionPendingQueue) > 0 {
			bs.playerMedarotToAct = bs.playerActionPendingQueue[0]
			bs.state = StatePlayerActionSelect
		} else {
			bs.playerMedarotToAct = nil
			bs.state = StatePlaying
		}
		bs.playerMedarotToAct = nil
		bs.currentTarget = nil
		bs.state = StatePlaying
	}

}
