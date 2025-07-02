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
		log.Println("エラー: createActionModalUI - partInfoProvider がnilです。")
		// partInfoProvider がないとパーツリストを取得できないため、モーダルを生成せずに終了するなどのエラーハンドリングが必要です。
		panel.AddChild(widget.NewText(
			widget.TextOpts.Text("エラー:パーツ情報取得不可", bs.resources.Font, c.Colors.White),
		))
		// overlayにpanelを追加しているので、このままでは空のモーダルが表示されます。より適切なハンドリングを検討してください。
		return overlay
	}
	availableParts := bs.partInfoProvider.GetAvailableAttackParts(actingEntry)
	if len(availableParts) == 0 {
		panel.AddChild(widget.NewText(
			widget.TextOpts.Text("利用可能なパーツがありません。", bs.resources.Font, c.Colors.White),
		))
	}

	for _, available := range availableParts { // available は AvailablePart { PartDef *PartDefinition, Slot PartSlotKey } 型です
		partDef := available.PartDef
		slotKey := available.Slot
		if partDef.Category == CategoryShoot {
			targetEntity, targetSlot := playerSelectRandomTarget(bs, actingEntry) // このヘルパーはパーツ情報を使用する場合、更新が必要になることがあります
			bs.ui.actionTargetMap[slotKey] = ActionTarget{Target: targetEntity, Slot: targetSlot}
		}
	}

	for _, available := range availableParts { // available は AvailablePart 型です
		capturedPartDef := available.PartDef // PartDef を使用し、ハンドラ用にキャプチャします
		capturedSlotKey := available.Slot    // 必要に応じて一貫性を保つためにスロットキーもキャプチャします

		actionButton := widget.NewButton(
			widget.ButtonOpts.Image(buttonImage),
			widget.ButtonOpts.Text(fmt.Sprintf("%s (%s)", capturedPartDef.PartName, capturedPartDef.Category), bs.resources.Font, &widget.ButtonTextColor{
				Idle: c.Colors.White,
			}),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				// PartDefinition とその元のスロットキーを handleActionSelection に渡します
				handleActionSelection(bs, actingEntry, capturedPartDef, capturedSlotKey)
			}),
			widget.ButtonOpts.CursorEnteredHandler(func(args *widget.ButtonHoverEventArgs) {
				if capturedPartDef.Category == CategoryShoot {
					if actionTarget, ok := bs.ui.actionTargetMap[capturedSlotKey]; ok { // capturedSlotKey を使用
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
			bs.playerActionPendingQueue = make([]*donburi.Entry, 0) // 保留キューをクリア
			bs.playerMedarotToAct = nil
			bs.currentTarget = nil
			bs.state = StatePlaying
		}),
	)
	panel.AddChild(cancelButton)

	return overlay

}

func handleActionSelection(bs *BattleScene, actingEntry *donburi.Entry, selectedPartDef *PartDefinition, slotKey PartSlotKey) {
	// slotKey が渡され、有効であれば、partInfoProvider はここでは直接必要ありません。
	// ただし、以前は FindPartSlot を使用して slotKey を取得していました。現在は slotKey が直接渡されます。
	if slotKey == "" { // createActionModalUI から有効な slotKey が渡されるはずです
		log.Printf("エラー: handleActionSelection - パーツ %s に対して空のスロットキーを受信しました。", selectedPartDef.PartName)
		bs.ui.HideActionModal()
		// キャンセルされたかのように状態をリセット
		bs.playerActionPendingQueue = make([]*donburi.Entry, 0)
		bs.playerMedarotToAct = nil
		bs.currentTarget = nil
		bs.state = StatePlaying
		return
	}

	var successful bool

	switch selectedPartDef.Category {
	case CategoryShoot:
		actionTarget, ok := bs.ui.actionTargetMap[slotKey] // slotKey は直接利用可能になりました
		if !ok || actionTarget.Target == nil || actionTarget.Slot == "" {
			bs.enqueueMessage("ターゲットがいません！", func() { // これにより状態が StateMessage に変わります
				bs.playerMedarotToAct = nil
				bs.currentTarget = nil
				bs.state = StatePlaying
			})
			bs.ui.HideActionModal()
			return
		}
		// StartCharge に bs.world、&bs.resources.Config、bs.partInfoProvider を渡します
		successful = StartCharge(actingEntry, slotKey, actionTarget.Target, actionTarget.Slot, bs.world, &bs.resources.Config, bs.partInfoProvider)
	case CategoryMelee:
		// StartCharge に bs.world、&bs.resources.Config、bs.partInfoProvider を渡します
		successful = StartCharge(actingEntry, slotKey, nil, "", bs.world, &bs.resources.Config, bs.partInfoProvider)
	default:
		log.Printf("未対応のパーツカテゴリです: %s", selectedPartDef.Category)
		successful = false
	}

	if successful {
		bs.ui.HideActionModal() // まず現在のモーダルを非表示にします
		bs.currentTarget = nil  // ターゲットインジケーターをクリアします

		// 現在のメダロットをデキューします
		if len(bs.playerActionPendingQueue) > 0 && bs.playerActionPendingQueue[0] == actingEntry {
			bs.playerActionPendingQueue = bs.playerActionPendingQueue[1:]
		}

		if len(bs.playerActionPendingQueue) > 0 {
			// 他にも待機中のプレイヤーがいるため、次のプレイヤーの準備をします
			bs.playerMedarotToAct = bs.playerActionPendingQueue[0] // BattleScene の更新ループで既に設定されていますが、明示的に記述
			bs.state = StatePlayerActionSelect                     // 次のフレームでモーダルが表示されるように状態を正しく設定します
			// UI は BattleScene の次の更新サイクルで新しい playerMedarotToAct のモーダルを再作成する必要があります
		} else {
			// キューに他のプレイヤーがいません
			bs.playerMedarotToAct = nil
			bs.state = StatePlaying
		}
	} else {
		// アクションは成功しませんでした（例：パーツ破損、ターゲットなし）
		log.Printf("エラー: %s の行動選択に失敗しました。", SettingsComponent.Get(actingEntry).Name)
		bs.ui.HideActionModal()
		// アクションが失敗した場合、このプレイヤーのキューに関するターンは一旦終了したものとして扱います。
		// このロジックは改善が必要かもしれません：次のプレイヤーを試すべきか、キューをリセットすべきか？
		// 現状では、成功したアクションと同様にキューを進めようとします。
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
