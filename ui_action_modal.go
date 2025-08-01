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

			actionButton := uiFactory.NewCyberpunkButton(
				buttonText,
				buttonTextColor,
				func(args *widget.ButtonClickedEventArgs) {
					eventChannel <- ClearCurrentTargetUIEvent{}

					eventChannel <- ActionConfirmedUIEvent{
						ActingEntityID:    vm.ActingEntityID,
						SelectedPartDefID: buttonVM.SelectedPartDefID,
						SelectedSlotKey:   buttonVM.SlotKey,
						TargetEntityID:    buttonVM.TargetEntityID,
						TargetPartSlot:    buttonVM.TargetPartSlot,
					}
				},
				func(args *widget.ButtonHoverEventArgs) {
					switch buttonVM.PartCategory {
					case CategoryRanged:
						if buttonVM.TargetEntityID != 0 {
							eventChannel <- TargetSelectedUIEvent{
								ActingEntityID:  vm.ActingEntityID,
								SelectedSlotKey: buttonVM.SlotKey,
								TargetEntityID:  buttonVM.TargetEntityID,
								TargetPartSlot:  "",
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

	// コンテンツを格納するコンテナを作成
	contentContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(15)), // パディングはここで設定
			widget.RowLayoutOpts.Spacing(c.ActionModal.ButtonSpacing),
		)),
	)

	// タイトルを追加
	title := widget.NewText(
		widget.TextOpts.Text(uiFactory.MessageManager.FormatMessage("ui_action_select_title", map[string]interface{}{"MedarotName": vm.ActingMedarotName}), uiFactory.Font, c.Colors.White),
	)
	contentContainer.AddChild(title)

	// ボタンを追加
	for _, btn := range buttons {
		contentContainer.AddChild(btn)
	}

	return contentContainer
}
