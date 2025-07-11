package main

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yohamta/donburi"
)

func (u *UI) IsActionModalVisible() bool {
	return u.isActionModalVisible
}

func (u *UI) SetBattlefieldViewModel(vm BattlefieldViewModel) {
	u.battlefieldWidget.SetViewModel(vm)
}

type UI struct {
	ebitenui          *ebitenui.UI
	actionModal       widget.PreferredSizeLocateableWidget
	messageWindow     widget.PreferredSizeLocateableWidget
	battlefieldWidget *BattlefieldWidget
	medarotInfoPanels map[string]*infoPanelUI
	actionTargetMap   map[PartSlotKey]ActionTarget
	// UIの状態
	playerMedarotToAct   *donburi.Entry // 現在アクション選択中のプレイヤーメダロット
	currentTarget        *donburi.Entry // 現在ターゲットとして表示されているエンティティ
	isActionModalVisible bool           // アクションモーダルが表示されているか
	// イベント通知用チャネル
	eventChannel chan UIEvent
	// 依存性
	world      donburi.World
	config     *Config
	whitePixel *ebiten.Image
}

// PostEvent はUIイベントをBattleSceneのキューに追加します。
func (u *UI) PostEvent(event UIEvent) {
	u.eventChannel <- event
}

// NewUI は新しいUIインスタンスを作成します。
func NewUI(world donburi.World, config *Config, eventChannel chan UIEvent) *UI {
	whiteImg := ebiten.NewImage(1, 1)
	whiteImg.Fill(color.White)
	ui := &UI{
		medarotInfoPanels:    make(map[string]*infoPanelUI),
		actionTargetMap:      make(map[PartSlotKey]ActionTarget),
		isActionModalVisible: false,
		eventChannel:         eventChannel,
		world:                world,
		config:               config,
		whitePixel:           whiteImg,
	}
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewStackedLayout()),
	)
	mainUIContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{false, true, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(config.UI.InfoPanel.Padding, 0),
		)),
	)
	rootContainer.AddChild(mainUIContainer)
	team1PanelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(config.UI.InfoPanel.Padding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(config.UI.InfoPanel.BlockWidth), 0),
		),
	)
	mainUIContainer.AddChild(team1PanelContainer)
	ui.battlefieldWidget = NewBattlefieldWidget(config)
	ui.battlefieldWidget.Container.GetWidget().LayoutData = widget.GridLayoutData{}
	mainUIContainer.AddChild(ui.battlefieldWidget.Container)
	team2PanelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(config.UI.InfoPanel.Padding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(config.UI.InfoPanel.BlockWidth), 0),
		),
	)
	mainUIContainer.AddChild(team2PanelContainer)
	setupInfoPanels(world, config, GlobalGameDataManager.Font, ui.medarotInfoPanels, team1PanelContainer, team2PanelContainer)
	ui.ebitenui = &ebitenui.UI{
		Container: rootContainer,
	}
	return ui
}

// ShowActionModal はアクション選択モーダルを表示します。
func (u *UI) ShowActionModal(actingEntry *donburi.Entry, actionTargetMap map[PartSlotKey]ActionTarget) {
	if u.isActionModalVisible {
		u.HideActionModal()
	}
	u.playerMedarotToAct = actingEntry
	u.isActionModalVisible = true
	u.actionTargetMap = actionTargetMap // Set the pre-calculated map

	modal := createActionModalUI(actingEntry, u.config, u.actionTargetMap, u.eventChannel, GlobalGameDataManager.Font)
	u.actionModal = modal
	u.ebitenui.Container.AddChild(u.actionModal)
	log.Println("アクションモーダルを表示しました。")
}

// HideActionModal はアクション選択モーダルを非表示にします。
func (u *UI) HideActionModal() {
	if !u.isActionModalVisible {
		return
	}
	if u.actionModal != nil {
		u.ebitenui.Container.RemoveChild(u.actionModal)
		u.actionModal = nil
	}
	u.playerMedarotToAct = nil
	u.currentTarget = nil
	u.isActionModalVisible = false
	u.actionTargetMap = make(map[PartSlotKey]ActionTarget) // ターゲットマップをクリア
	log.Println("アクションモーダルを非表示にしました。")
}

// ShowMessageWindow はメッセージウィンドウを表示します。
func (u *UI) ShowMessageWindow(message string) {
	if u.messageWindow != nil {
		u.HideMessageWindow()
	}
	win := createMessageWindow(message, u.config, GlobalGameDataManager.Font)
	u.messageWindow = win
	u.ebitenui.Container.AddChild(u.messageWindow)
}

// HideMessageWindow はメッセージウィンドウを非表示にします。
func (u *UI) HideMessageWindow() {
	if u.messageWindow != nil {
		u.ebitenui.Container.RemoveChild(u.messageWindow)
		u.messageWindow = nil
	}
}

func (u *UI) UpdateInfoPanels(world donburi.World, config *Config) {
	updateAllInfoPanels(world, config, u.medarotInfoPanels)
}

func (u *UI) GetActionTargetMap() map[PartSlotKey]ActionTarget {
	return u.actionTargetMap
}

func (u *UI) SetCurrentTarget(entry *donburi.Entry) {
	u.currentTarget = entry
}

func (u *UI) ClearCurrentTarget() {
	u.currentTarget = nil
}

func (u *UI) Update() {
	u.ebitenui.Update()
}

func (u *UI) Draw(screen *ebiten.Image, tick int) {
	// ターゲットインジケーターの描画に必要な IconViewModel を取得
	var indicatorTargetVM *IconViewModel
	if u.currentTarget != nil && u.battlefieldWidget.viewModel != nil {
		for _, iconVM := range u.battlefieldWidget.viewModel.Icons {
			if iconVM.EntryID == uint32(u.currentTarget.Id()) {
				indicatorTargetVM = iconVM
				break
			}
		}
	}

	// BattlefieldWidget の Draw メソッドを先に呼び出す
	u.battlefieldWidget.Draw(screen, indicatorTargetVM, tick)

	// その後でebitenuiを描画する
	u.ebitenui.Draw(screen)
}

func (u *UI) DrawBackground(screen *ebiten.Image) {
	u.battlefieldWidget.DrawBackground(screen)
}

func (u *UI) DrawAnimation(screen *ebiten.Image, anim *ActionAnimationData, tick int) {
	if anim == nil {
		return
	}

	progress := float64(tick - anim.StartTime)
	bfVM := u.battlefieldWidget.viewModel
	if bfVM == nil {
		return
	}

	var attackerVM, targetVM *IconViewModel
	for _, icon := range bfVM.Icons {
		if icon.EntryID == uint32(anim.Result.ActingEntry.Id()) {
			attackerVM = icon
		}
		if anim.Result.TargetEntry != nil && icon.EntryID == uint32(anim.Result.TargetEntry.Id()) {
			targetVM = icon
		}
	}

	// 攻撃者とターゲットが両方見つかった場合のみアニメーションを実行
	if attackerVM != nil && targetVM != nil {
		// アニメーションのタイミング設定
		const firstPingDuration = 30.0
		const secondPingDuration = 30.0
		const delayBetweenPings = 0.0 // 連続して再生

		// 1回目のピング（攻撃者） - 拡大
		if progress >= 0 && progress < firstPingDuration {
			pingProgress := progress / firstPingDuration
			drawPingAnimation(screen, attackerVM.X, attackerVM.Y, pingProgress, true)
		}

		// 2回目のピング（ターゲット） - 縮小
		secondPingStart := firstPingDuration + delayBetweenPings
		if progress >= secondPingStart && progress < secondPingStart+secondPingDuration {
			pingProgress := (progress - secondPingStart) / secondPingDuration
			drawPingAnimation(screen, targetVM.X, targetVM.Y, pingProgress, false)
		}

		// ダメージポップアップのアニメーション
		const popupDelay = 60 // 両方のピングが終わった後に開始
		const popupDuration = 60
		if progress >= popupDelay && progress < popupDelay+popupDuration {
			popupProgress := (progress - popupDelay) / popupDuration
			x := targetVM.X
			y := targetVM.Y - 20 - (20 * float32(popupProgress))
			alpha := 1.0
			if popupProgress > 0.7 {
				alpha = (1.0 - popupProgress) / 0.3
			}

			drawOpts := &text.DrawOptions{}

			// Set position and scale
			drawOpts.GeoM.Scale(1.5, 1.5)
			drawOpts.GeoM.Translate(float64(x), float64(y))

			// Set layout options to center the text
			drawOpts.LayoutOptions = text.LayoutOptions{
				PrimaryAlign:   text.AlignCenter,
				SecondaryAlign: text.AlignCenter,
			}

			// Set color and alpha
			r, g, b, a := u.config.UI.Colors.Red.RGBA()
			cr := float32(r) / 0xffff
			cg := float32(g) / 0xffff
			cb := float32(b) / 0xffff
			ca := float32(a) / 0xffff

			drawOpts.DrawImageOptions.ColorScale.Scale(cr, cg, cb, ca)
			drawOpts.DrawImageOptions.ColorScale.ScaleAlpha(float32(alpha))

			text.Draw(screen, fmt.Sprintf("-%d", anim.Result.OriginalDamage), GlobalGameDataManager.Font, drawOpts)
		}
	}
}

func (u *UI) GetBattlefieldWidgetRect() image.Rectangle {
	return u.battlefieldWidget.Container.GetWidget().Rect
}

// drawPingAnimation は、指定された中心にレーダーのようなピングアニメーションを描画します。
// progress は 0.0 から 1.0 の値で、アニメーションの進行状況を示します。
// expandがtrueの場合は拡大、falseの場合は縮小アニメーションになります。
func drawPingAnimation(screen *ebiten.Image, centerX, centerY float32, progress float64, expand bool) {
	if progress < 0 || progress > 1 {
		return
	}

	// アニメーションのパラメータ
	maxRadius := float32(40.0)
	pingColor := color.RGBA{R: 0, G: 255, B: 255, A: 255} // ネオン風の水色

	// 進行状況に基づいて半径とアルファ値を計算
	var radius float32
	if expand {
		radius = maxRadius * float32(progress) // 拡大
	} else {
		radius = maxRadius * (1.0 - float32(progress)) // 縮小
	}
	alpha := 1.0 - progress // 徐々にフェードアウト

	// アルファ値を適用した色を作成
	r, g, b, _ := pingColor.RGBA()
	finalColor := color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(255 * alpha),
	}

	// 二重丸(◎)を描画
	vector.StrokeCircle(screen, centerX, centerY, radius, 2, finalColor, true)
	vector.StrokeCircle(screen, centerX, centerY, radius*0.4, 1.5, finalColor, true)
}