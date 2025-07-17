package main

import (
	"fmt"
	"log"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/yohamta/donburi"
)

func createActionModalUI(
	actingEntry *donburi.Entry,
	config *Config,
	actionTargetMap map[PartSlotKey]ActionTarget,
	eventChannel chan UIEvent,
	font text.Face,
) widget.PreferredSizeLocateableWidget {
	c := config.UI
	settings := SettingsComponent.Get(actingEntry)

	// オーバーレイ用のコンテナ
	overlay := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		// 背景色を削除し、完全に透明にする
		// widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0, 0, 0, 180})),
	)

	// ボタンウィジェットのスライスを作成
	buttons := []widget.PreferredSizeLocateableWidget{}
	buttonImage := createCyberpunkButtonImageSet(5)

	partsComp := PartsComponent.Get(actingEntry)
	if partsComp == nil {
		log.Println("エラー: createActionModalUI - actingEntry に PartsComponent がありません。")
		buttons = append(buttons, widget.NewText(
			widget.TextOpts.Text("エラー:パーツ情報取得不可", font, c.Colors.White),
		))
	} else {
		var displayableParts []AvailablePart
		for slotKey, partInst := range partsComp.Map {
			partDef, defFound := GlobalGameDataManager.GetPartDefinition(partInst.DefinitionID)
			if !defFound {
				continue
			}
			if _, ok := actionTargetMap[slotKey]; ok {
				displayableParts = append(displayableParts, AvailablePart{PartDef: partDef, Slot: slotKey, IsBroken: partInst.IsBroken})
			}
		}

		if len(displayableParts) == 0 {
			buttons = append(buttons, widget.NewText(
				widget.TextOpts.Text("利用可能なパーツがありません。", font, c.Colors.White),
			))
		} else {
			for _, available := range displayableParts {
				partDef := available.PartDef
				slotKey := available.Slot

				buttonTextColor := &widget.ButtonTextColor{Idle: c.Colors.White}
				if available.IsBroken {
					buttonTextColor.Idle = c.Colors.Red
				}

				actionButton := widget.NewButton(
					widget.ButtonOpts.Image(buttonImage),
					widget.ButtonOpts.Text(fmt.Sprintf("%s (%s)", partDef.PartName, partDef.Category), font, buttonTextColor),
					widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
					widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
						if !available.IsBroken {
							eventChannel <- PlayerActionSelectedEvent{
								ActingEntry:     actingEntry,
								SelectedPartDef: partDef,
								SelectedSlotKey: slotKey,
							}
						}
					}),
					widget.ButtonOpts.CursorEnteredHandler(func(args *widget.ButtonHoverEventArgs) {
						if partDef.Category == CategoryRanged {
							if targetInfo, ok := actionTargetMap[slotKey]; ok && targetInfo.Target != nil {
								eventChannel <- SetCurrentTargetEvent{Target: targetInfo.Target}
							}
						} else if partDef.Category == CategoryIntervention {
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
	}

	// NewPanel を使用してモーダルを作成
	panel := NewPanel(&PanelOptions{
		Title:           fmt.Sprintf("行動選択: %s", settings.Name),
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
