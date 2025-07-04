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

	if bs.battleLogic == nil || bs.battleLogic.PartInfoProvider == nil {
		log.Println("エラー: createActionModalUI - battleLogic または partInfoProvider がnilです。")
		// partInfoProvider がないとパーツリストを取得できないため、モーダルを生成せずに終了するなどのエラーハンドリングが必要です。
		panel.AddChild(widget.NewText(
			widget.TextOpts.Text("エラー:パーツ情報取得不可", bs.resources.Font, c.Colors.White),
		))
		// overlayにpanelを追加しているので、このままでは空のモーダルが表示されます。より適切なハンドリングを検討してください。
		return overlay
	}
	availableParts := bs.battleLogic.PartInfoProvider.GetAvailableAttackParts(actingEntry)
	if len(availableParts) == 0 {
		panel.AddChild(widget.NewText(
			widget.TextOpts.Text("利用可能なパーツがありません。", bs.resources.Font, c.Colors.White),
		))
	}

	for _, available := range availableParts { // available は AvailablePart { PartDef *PartDefinition, Slot PartSlotKey } 型です
		partDef := available.PartDef
		slotKey := available.Slot
		canSelect := true // このパーツを選択可能か

		switch partDef.Category {
		case CategoryShoot:
			targetEntity, targetSlot := playerSelectRandomTargetPart(actingEntry, bs.battleLogic.TargetSelector, bs.battleLogic.PartInfoProvider)
			if targetEntity == nil || targetSlot == "" {
				canSelect = false // ターゲットが見つからない場合は選択不可
				log.Printf("警告: %s の %s (射撃) はターゲットが見つからないため選択できません。", settings.Name, partDef.PartName)
			}
			bs.ui.actionTargetMap[slotKey] = ActionTarget{Target: targetEntity, Slot: targetSlot}
		case CategoryMelee:
			// 格闘の場合はターゲット選択が不要なので、ダミーのターゲットを設定
			bs.ui.actionTargetMap[slotKey] = ActionTarget{Target: nil, Slot: ""}
		}
		log.Printf("createActionModalUI: actionTargetMap[%s] = {Target: %v, Slot: %s}", slotKey, bs.ui.actionTargetMap[slotKey].Target, bs.ui.actionTargetMap[slotKey].Slot)

		if !canSelect {
			continue // 選択できないパーツはボタンを作成しない
		}

		// ボタンを作成
		actionButton := widget.NewButton(
			widget.ButtonOpts.Image(buttonImage),
			widget.ButtonOpts.Text(fmt.Sprintf("%s (%s)", partDef.PartName, partDef.Category), bs.resources.Font, &widget.ButtonTextColor{
				Idle: c.Colors.White,
			}),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				bs.ui.PostEvent(PlayerActionSelectedEvent{
					ActingEntry:     actingEntry,
					SelectedPartDef: partDef,
					SelectedSlotKey: slotKey,
				})
			}),
			widget.ButtonOpts.CursorEnteredHandler(func(args *widget.ButtonHoverEventArgs) {
				if partDef.Category == CategoryShoot {
					if actionTarget, ok := bs.ui.actionTargetMap[slotKey]; ok { // capturedSlotKey を使用
						bs.ui.currentTarget = actionTarget.Target
					}
				}
			}),
			widget.ButtonOpts.CursorExitedHandler(func(args *widget.ButtonHoverEventArgs) {
				bs.ui.currentTarget = nil
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
			bs.ui.PostEvent(PlayerActionCancelEvent{ActingEntry: actingEntry})
		}),
	)
	panel.AddChild(cancelButton)

	return overlay

}


