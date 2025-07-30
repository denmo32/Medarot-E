package main

import (
	"fmt"

	"github.com/ebitenui/ebitenui/widget"
)

func createActionModalUI(
	vm *ActionModalViewModel,
	uiFactory *UIFactory, // UIFactoryを追加
	eventChannel chan UIEvent,
) widget.PreferredSizeLocateableWidget {
	c := uiFactory.Config.UI

	// オーバーレイ用のコンテナ
	overlay := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		// 背景色を削除し、完全に透明にする
		// widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0, 0, 0, 180})),
	)

	// ボタンウィジェットのスライスを作成
	buttons := []widget.PreferredSizeLocateableWidget{}

	if len(vm.Buttons) == 0 {
		buttons = append(buttons, widget.NewText(
			widget.TextOpts.Text(uiFactory.MessageManager.FormatMessage("ui_no_parts_available", nil), uiFactory.Font, c.Colors.White),
		))
	} else {
		for _, buttonVM := range vm.Buttons {
			buttonText := fmt.Sprintf("%s (%s)", buttonVM.PartName, buttonVM.PartCategory)
			buttonTextColor := &widget.ButtonTextColor{Idle: c.Colors.White}
			// IsBroken は PartInstanceData から取得するため、ここでは不要
			// if buttonVM.IsBroken {
			// 	buttonTextColor.Idle = c.Colors.Red
			// }

			actionButton := uiFactory.NewCyberpunkButton(
				buttonText,
				buttonTextColor,
				func(args *widget.ButtonClickedEventArgs) {
					// IsBroken は PartInstanceData から取得するため、ここでは不要
					// if !buttonVM.IsBroken {
						// ターゲットインジケーターをクリア
						eventChannel <- ClearCurrentTargetUIEvent{}

						// 選択されたパーツとターゲット情報をGameEventとして発行
						eventChannel <- ActionConfirmedUIEvent{
							ActingEntityID:    vm.ActingEntityID,
							SelectedPartDefID: buttonVM.SelectedPartDefID,
							SelectedSlotKey:   buttonVM.SlotKey,
							TargetEntityID:    buttonVM.TargetEntityID,
							TargetPartSlot:    buttonVM.TargetPartSlot,
						}
					// }
				},
				func(args *widget.ButtonHoverEventArgs) {
					switch buttonVM.PartCategory {
					case CategoryRanged:
						if buttonVM.TargetEntityID != 0 {
							eventChannel <- TargetSelectedUIEvent{
								ActingEntityID: vm.ActingEntityID,
								SelectedSlotKey: buttonVM.SlotKey,
								TargetEntityID: buttonVM.TargetEntityID,
								TargetPartSlot: "", // ターゲットパーツスロットはここでは不明
							}
						}
					case CategoryIntervention:
						// 介入の場合はターゲット表示なし
					default:
						// 格闘など、他のカテゴリでターゲット表示が必要な場合はここに追加
					}
				},
				func(args *widget.ButtonHoverEventArgs) {
					eventChannel <- ClearCurrentTargetUIEvent{}
				},
			)
			buttons = append(buttons, actionButton)
		}
	}

	// NewPanel を使用してモーダルを作成
	panel := NewPanel(&PanelOptions{
		Title:           uiFactory.MessageManager.FormatMessage("ui_action_select_title", map[string]interface{}{"MedarotName": vm.ActingMedarotName}),
		Padding:         widget.NewInsetsSimple(15),
		Spacing:         c.ActionModal.ButtonSpacing,
		PanelWidth:      int(c.ActionModal.ButtonWidth) + 30,
		TitleFont:       uiFactory.Font,
		BackgroundColor: c.Colors.Background, // 背景色を設定
		BorderColor:     c.Colors.Gray,       // 枠線の色
		BorderThickness: 5,                   // 枠線の太さ
	}, uiFactory.imageGenerator, uiFactory.Font, buttons...) // uiFactory.imageGeneratorとuiFactory.Fontを渡す

	// パネルをオーバーレイの中央に配置
	panel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	overlay.AddChild(panel)

	return overlay
}
