package ui

import (
	"fmt"
	"medarot-ebiten/core"
	"medarot-ebiten/event"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/yohamta/donburi"
)

// TargetManager は、UIコンポーネントがターゲットのハイライトを管理するために必要なメソッドを定義します。
// これにより、ActionModalがBattleUIManager全体に依存することを防ぎます。
type TargetManager interface {
	SetCurrentTarget(entityID donburi.Entity)
	ClearCurrentTarget()
}

// ActionModal はプレイヤーの行動選択UIを管理するコンポーネントです。
type ActionModal struct {
	widget       widget.PreferredSizeLocateableWidget
	uiFactory    *UIFactory
	eventChannel chan event.GameEvent
	world        donburi.World
	targetManager TargetManager
	isVisible    bool
}

// NewActionModal は新しいActionModalのインスタンスを作成します。
func NewActionModal(
	uiFactory *UIFactory,
	eventChannel chan event.GameEvent,
	world donburi.World,
	targetManager TargetManager,
) *ActionModal {
	// 初期状態では空のコンテナを持つ
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	return &ActionModal{
		widget:        container,
		uiFactory:     uiFactory,
		eventChannel:  eventChannel,
		world:         world,
		targetManager: targetManager,
		isVisible:     false,
	}
}

// Widget はこのコンポーネントのルートウィジェットを返します。
func (a *ActionModal) Widget() widget.PreferredSizeLocateableWidget {
	return a.widget
}

// IsVisible はモーダルが表示されているかどうかを返します。
func (a *ActionModal) IsVisible() bool {
	return a.isVisible
}

// Show はViewModelに基づいてモーダルの内容を構築し、表示状態にします。
func (a *ActionModal) Show(vm *core.ActionModalViewModel) {
	a.widget = a.createUI(vm)
	a.isVisible = true
}

// Hide はモーダルを非表示にし、内容をクリアします。
func (a *ActionModal) Hide() {
	// 新しい空のコンテナに置き換えることで内容をクリア
	a.widget = widget.NewContainer()
	a.isVisible = false
}

// createUI はViewModelから実際のUIウィジェットを構築します。
func (a *ActionModal) createUI(vm *core.ActionModalViewModel) widget.PreferredSizeLocateableWidget {
	c := a.uiFactory.Config.UI

	// 各パーツカテゴリのボタンを格納するマップ
	partButtons := make(map[core.PartSlotKey][]widget.PreferredSizeLocateableWidget)

	if len(vm.Buttons) == 0 {
		// ボタンがない場合のメッセージ
		noPartsText := widget.NewText(
			widget.TextOpts.Text(a.uiFactory.MessageManager.FormatMessage("ui_no_parts_available", nil), a.uiFactory.Font, c.Colors.White),
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

		// この無名関数内で使用する変数をキャプチャする
		capturedButtonVM := buttonVM
		capturedVM := vm

		actionButton := a.uiFactory.NewCyberpunkButton(
			buttonText,
			buttonTextColor,
			func(args *widget.ButtonClickedEventArgs) {
				actingEntry := a.world.Entry(capturedVM.ActingEntityID)
				if actingEntry == nil {
					return
				}
				var targetEntry *donburi.Entry
				if capturedButtonVM.TargetEntityID != 0 {
					targetEntry = a.world.Entry(capturedButtonVM.TargetEntityID)
				}

				// ゲームイベントを発行
				a.eventChannel <- event.ChargeRequestedGameEvent{
					ActingEntry:     actingEntry,
					SelectedSlotKey: capturedButtonVM.SlotKey,
					TargetEntry:     targetEntry,
					TargetPartSlot:  capturedButtonVM.TargetPartSlot,
				}
				a.eventChannel <- event.PlayerActionProcessedGameEvent{
					ActingEntry: actingEntry,
				}
				a.eventChannel <- event.HideActionModalGameEvent{}
				a.eventChannel <- event.ClearCurrentTargetGameEvent{}
			},
			func(args *widget.ButtonHoverEventArgs) {
				switch capturedButtonVM.PartCategory {
				case core.CategoryRanged:
					if capturedButtonVM.TargetEntityID != 0 {
						a.targetManager.SetCurrentTarget(capturedButtonVM.TargetEntityID)
					}
				case core.CategoryIntervention:
					// 介入の場合はターゲット表示なし
				default:
					// 格闘など、他のカテゴリでターゲット表示が必要な場合はここに追加
				}
			},
			func(args *widget.ButtonHoverEventArgs) {
				a.targetManager.ClearCurrentTarget()
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
		widget.TextOpts.Text(a.uiFactory.MessageManager.FormatMessage("ui_action_select_title", map[string]interface{}{"MedarotName": vm.ActingMedarotName}), a.uiFactory.Font, c.Colors.White),
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
