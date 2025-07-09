package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/yohamta/donburi"
)

func createActionModalUI(
	actingEntry *donburi.Entry,
	config *Config,
	actionTargetMap map[PartSlotKey]ActionTarget, // 修正: 事前計算されたマップを受け取る
	eventChannel chan UIEvent,
	font text.Face,
) widget.PreferredSizeLocateableWidget {
	c := config.UI
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
		widget.TextOpts.Text(fmt.Sprintf("行動選択: %s", settings.Name), font, c.Colors.White),
	))

	buttonImage := &widget.ButtonImage{
		Idle:    image.NewNineSliceColor(c.Colors.Gray),
		Hover:   image.NewNineSliceColor(color.RGBA{180, 180, 180, 255}),
		Pressed: image.NewNineSliceColor(color.RGBA{100, 100, 100, 255}),
	}

	// partInfoProvider は不要になったため、UI構造体から取得する
	// availableParts の取得も ShowActionModal で行われるため、ここでは不要
	// actingEntry から直接パーツ情報を取得するか、ShowActionModal から渡された情報を使用する
	partsComp := PartsComponent.Get(actingEntry)
	if partsComp == nil {
		log.Println("エラー: createActionModalUI - actingEntry に PartsComponent がありません。")
		panel.AddChild(widget.NewText(
			widget.TextOpts.Text("エラー:パーツ情報取得不可", font, c.Colors.White),
		))
		return overlay
	}

	// 利用可能なパーツを直接取得 (ShowActionModalで計算済みのため、ここでは表示用)
	// ここでは、actingEntryのパーツをループして、actionTargetMapに存在するものを表示する
	var displayableParts []AvailablePart
	for slotKey, partInst := range partsComp.Map {
		partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID)
		if !defFound {
			continue
		}
		// actionTargetMap にエントリがある、つまり ShowActionModal で選択可能と判断されたパーツのみ表示
		if _, ok := actionTargetMap[slotKey]; ok {
			displayableParts = append(displayableParts, AvailablePart{PartDef: partDef, Slot: slotKey, IsBroken: partInst.IsBroken})
		}
	}

	if len(displayableParts) == 0 {
		panel.AddChild(widget.NewText(
			widget.TextOpts.Text("利用可能なパーツがありません。", font, c.Colors.White),
		))
	}

	for _, available := range displayableParts {
		partDef := available.PartDef
		slotKey := available.Slot

		buttonTextColor := &widget.ButtonTextColor{Idle: c.Colors.White}
		if available.IsBroken {
			buttonTextColor.Idle = c.Colors.Red // 破壊されている場合は赤色
		}

		actionButton := widget.NewButton(
			widget.ButtonOpts.Image(buttonImage),
			widget.ButtonOpts.Text(fmt.Sprintf("%s (%s)", partDef.PartName, partDef.Category), font, buttonTextColor),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				eventChannel <- PlayerActionSelectedEvent{
					ActingEntry:     actingEntry,
					SelectedPartDef: partDef,
					SelectedSlotKey: slotKey,
				}
			}),
			widget.ButtonOpts.CursorEnteredHandler(func(args *widget.ButtonHoverEventArgs) {
				if partDef.Category == CategoryShoot {
					if targetInfo, ok := actionTargetMap[slotKey]; ok && targetInfo.Target != nil {
						eventChannel <- SetCurrentTargetEvent{Target: targetInfo.Target}
					}
				}
			}),
			widget.ButtonOpts.CursorExitedHandler(func(args *widget.ButtonHoverEventArgs) {
				eventChannel <- ClearCurrentTargetEvent{}
			}),
		)
		panel.AddChild(actionButton)
	}
	return overlay
}
