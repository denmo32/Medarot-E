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
    world donburi.World, // Add world parameter
    config *Config,
    partInfoProvider *PartInfoProvider,
    targetSelector *TargetSelector,
    actionTargetMap map[PartSlotKey]ActionTarget,
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
        widget.TextOpts.Text(fmt.Sprintf("行動選択: %s", settings.Name), font, c.Colors.White), // GlobalGameDataManager.Font → font
    ))
    buttonImage := &widget.ButtonImage{
        Idle:    image.NewNineSliceColor(c.Colors.Gray),
        Hover:   image.NewNineSliceColor(color.RGBA{180, 180, 180, 255}),
        Pressed: image.NewNineSliceColor(color.RGBA{100, 100, 100, 255}),
    }
    if partInfoProvider == nil {
        log.Println("エラー: createActionModalUI - partInfoProvider がnilです。")
        panel.AddChild(widget.NewText(
            widget.TextOpts.Text("エラー:パーツ情報取得不可", font, c.Colors.White), // GlobalGameDataManager.Font → font
        ))
        return overlay
    }
    availableParts := partInfoProvider.GetAvailableAttackParts(actingEntry)
    if len(availableParts) == 0 {
        panel.AddChild(widget.NewText(
            widget.TextOpts.Text("利用可能なパーツがありません。", font, c.Colors.White), // GlobalGameDataManager.Font → font
        ))
    }
    for _, available := range availableParts { // available は AvailablePart { PartDef *PartDefinition, Slot PartSlotKey } 型です
        partDef := available.PartDef
        slotKey := available.Slot
        canSelect := true // このパーツを選択可能か
        switch partDef.Category {
        case CategoryShoot, CategoryMelee:
            var strategy TargetingStrategy
            medal := MedalComponent.Get(actingEntry)
            switch medal.Personality {
            case "アシスト":
                strategy = &AssistStrategy{}
            case "クラッシャー":
                strategy = &CrusherStrategy{}
            case "カウンター":
                strategy = &CounterStrategy{}
            case "チェイス":
                strategy = &ChaseStrategy{}
            case "デュエル":
                strategy = &DuelStrategy{}
            case "フォーカス":
                strategy = &FocusStrategy{}
            case "ガード":
                strategy = &GuardStrategy{}
            case "ハンター":
                strategy = &HunterStrategy{}
            case "インターセプト":
                strategy = &InterceptStrategy{}
            case "ジョーカー":
                strategy = &JokerStrategy{}
            default:
                strategy = &LeaderStrategy{} // デフォルトはリーダー狙い
            }
            targetEntity, targetSlot := strategy.SelectTarget(world, actingEntry, targetSelector, partInfoProvider)
            if targetEntity == nil {
                canSelect = false // ターゲットが見つからない場合は選択不可
                log.Printf("警告: %s の %s (%s) はターゲットが見つからないため選択できません。", settings.Name, partDef.PartName, partDef.Category)
            }
            actionTargetMap[slotKey] = ActionTarget{Target: targetEntity, Slot: targetSlot}
        }

        // ログ出力を削除（未使用変数 actionTarget の削除）
        if !canSelect {
            continue // 選択できないパーツはボタンを作成しない
        }
        // ボタンを作成
        actionButton := widget.NewButton(
            widget.ButtonOpts.Image(buttonImage),
            widget.ButtonOpts.Text(fmt.Sprintf("%s (%s)", partDef.PartName, partDef.Category), font, &widget.ButtonTextColor{ // GlobalGameDataManager.Font → font
                Idle: c.Colors.White,
            }),
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
    cancelButton := widget.NewButton(
        widget.ButtonOpts.Image(buttonImage),
        widget.ButtonOpts.Text("キャンセル", font, &widget.ButtonTextColor{ // GlobalGameDataManager.Font → font
            Idle: c.Colors.White,
        }),
        widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
        widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
            eventChannel <- PlayerActionCancelEvent{ActingEntry: actingEntry}
        }),
    )
    panel.AddChild(cancelButton)
    return overlay
}