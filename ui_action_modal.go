package main

import (
	"fmt"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func createActionModalUI(
	vm *ActionModalViewModel,
	config *Config,
	eventChannel chan UIEvent,
	font text.Face,
) widget.PreferredSizeLocateableWidget {
	c := config.UI

	// オーバーレイ用のコンテナ
	overlay := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		// 背景色を削除し、完全に透明にする
		// widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0, 0, 0, 180})),
	)

	// ボタンウィジェットのスライスを作成
	buttons := []widget.PreferredSizeLocateableWidget{}
	buttonImage := createCyberpunkButtonImageSet(5)

	if len(vm.Buttons) == 0 {
		buttons = append(buttons, widget.NewText(
			widget.TextOpts.Text("利用可能なパーツがありません。", font, c.Colors.White),
		))
	} else {
		for _, buttonVM := range vm.Buttons {
			buttonTextColor := &widget.ButtonTextColor{Idle: c.Colors.White}
			if buttonVM.IsBroken {
				buttonTextColor.Idle = c.Colors.Red
			}

			actionButton := widget.NewButton(
				widget.ButtonOpts.Image(buttonImage),
				widget.ButtonOpts.Text(fmt.Sprintf("%s (%s)", buttonVM.PartName, buttonVM.PartCategory), font, buttonTextColor),
				widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
				widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
					if !buttonVM.IsBroken {
						eventChannel <- ClearCurrentTargetEvent{}
						eventChannel <- PlayerActionSelectedEvent{
							ActingEntry:     vm.ActingEntry,
							SelectedPartDef: buttonVM.SelectedPartDef,
							SelectedSlotKey: buttonVM.SlotKey,
						}
					}
				}),
				widget.ButtonOpts.CursorEnteredHandler(func(args *widget.ButtonHoverEventArgs) {
					if buttonVM.PartCategory == CategoryRanged {
						if buttonVM.TargetEntry != nil {
							eventChannel <- SetCurrentTargetEvent{Target: buttonVM.TargetEntry}
						}
					} else if buttonVM.PartCategory == CategoryIntervention {
						// 介入の場合はターゲット表示なし
					} else {
						// 格闘など、他のカテゴリでターゲット表示が必要な場合はここに追加
					}
				}),
				widget.ButtonOpts.CursorExitedHandler(func(args *widget.ButtonHoverEventArgs) {
					eventChannel <- ClearCurrentTargetEvent{}
				}),
			)
			buttons = append(buttons, actionButton)
		}
	}

	// NewPanel を使用してモーダルを作成
	panel := NewPanel(&PanelOptions{
		Title:           fmt.Sprintf("行動選択: %s", vm.ActingMedarotName),
		Padding:         widget.NewInsetsSimple(15),
		Spacing:         c.ActionModal.ButtonSpacing,
		PanelWidth:      int(c.ActionModal.ButtonWidth) + 30,
		TitleFont:       font,
		BackgroundColor: c.Colors.Background, // 背景色を設定
		BorderColor:     c.Colors.Gray,       // 枠線の色
		BorderThickness: 5,                   // 枠線の太さ
	}, buttons...)

	// パネルをオーバーレイの中央に配置
	panel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	overlay.AddChild(panel)

	return overlay
}
