package main

import (
	"fmt"
	"medarot-ebiten/core"
	"medarot-ebiten/ui"

	"github.com/ebitenui/ebitenui/widget"
)

func createActionModalUI(
	vm *ui.ActionModalViewModel,
	uiFactory *UIFactory,
	eventChannel chan UIEvent,
) widget.PreferredSizeLocateableWidget {
	c := uiFactory.Config.UI

	// 各パーツカテゴリのボタンを格納するマップ
	partButtons := make(map[core.PartSlotKey][]widget.PreferredSizeLocateableWidget)

	if len(vm.Buttons) == 0 {
		// ボタンがない場合のメッセージ
		noPartsText := widget.NewText(
			widget.TextOpts.Text(uiFactory.MessageManager.FormatMessage("ui_no_parts_available", nil), uiFactory.Font, c.Colors.White),
		)
		// 中央に配置するためのコンテナ
		centeredTextContainer := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		)
		centeredTextContainer.AddChild(noPartsText)
		return centeredTextContainer
	}

	for _, buttonVM := range vm.Buttons {
		buttonText := fmt.Sprintf("%s (%s)", buttonVM.PartName, buttonVM.PartCategory)
		buttonTextColor := &widget.ButtonTextColor{
			Idle:  c.Colors.White,
			Hover: c.Colors.Black,
		}

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
				case core.CategoryRanged:
					if buttonVM.TargetEntityID != 0 {
						eventChannel <- TargetSelectedUIEvent{
							ActingEntityID:  vm.ActingEntityID,
							SelectedSlotKey: buttonVM.SlotKey,
							TargetEntityID:  buttonVM.TargetEntityID,
							TargetPartSlot:  "",
						}
					}
				case core.CategoryIntervention:
					// 介入の場合はターゲット表示なし
				default:
					// 格闘など、他のカテゴリでターゲット表示が必要な場合はここに追加
				}
			},
			func(args *widget.ButtonHoverEventArgs) {
				eventChannel <- ClearCurrentTargetUIEvent{}
			},
		)
		partButtons[buttonVM.SlotKey] = append(partButtons[buttonVM.SlotKey], actionButton)
	}

	// 最外側のコンテナ（中央配置用）
	outerContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// メインのコンテンツコンテナ
	contentContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(c.ActionModal.ButtonSpacing),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(15)),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)

	// タイトルセクション
	title := widget.NewText(
		widget.TextOpts.Text(uiFactory.MessageManager.FormatMessage("ui_action_select_title", map[string]interface{}{"MedarotName": vm.ActingMedarotName}), uiFactory.Font, c.Colors.White),
	)
	contentContainer.AddChild(title)

	// パーツ配置用のメインコンテナ（3行1列のグリッド）
	partsContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, false}),
			widget.GridLayoutOpts.Spacing(0, c.ActionModal.ButtonSpacing*2),
		)),
	)

	// 1行目: 頭部パーツ（中央配置）
	headRowContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	headPartsContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(c.ActionModal.ButtonSpacing),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)

	for _, btn := range partButtons[core.PartSlotHead] {
		headPartsContainer.AddChild(btn)
	}
	headRowContainer.AddChild(headPartsContainer)
	partsContainer.AddChild(headRowContainer)

	// 2行目: 腕パーツ（左右分割）
	armRowContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{true, true}, []bool{true}),
			widget.GridLayoutOpts.Spacing(c.ActionModal.ButtonSpacing*4, 0), // 左右の間隔を広く
		)),
	)

	// 右腕パーツ（1列目、右寄せ）
	rightArmContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	rightArmPartsContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(c.ActionModal.ButtonSpacing),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionEnd,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)
	for _, btn := range partButtons[core.PartSlotRightArm] {
		rightArmPartsContainer.AddChild(btn)
	}
	rightArmContainer.AddChild(rightArmPartsContainer)
	armRowContainer.AddChild(rightArmContainer)

	// 左腕パーツ（2列目、左寄せ）
	leftArmContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	leftArmPartsContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(c.ActionModal.ButtonSpacing),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)
	for _, btn := range partButtons[core.PartSlotLeftArm] {
		leftArmPartsContainer.AddChild(btn)
	}
	leftArmContainer.AddChild(leftArmPartsContainer)
	armRowContainer.AddChild(leftArmContainer)

	partsContainer.AddChild(armRowContainer)
	contentContainer.AddChild(partsContainer)

	// 最外側のコンテナにメインコンテンツを追加
	outerContainer.AddChild(contentContainer)

	return outerContainer
}
