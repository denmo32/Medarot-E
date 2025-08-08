package main

import (
	"log"

	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ui"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
)

// UIActionModalManager はアクション選択モーダルの表示と状態を管理します。
type UIActionModalManager struct {
	ebitenui             *ebitenui.UI // UIのルートコンテナにアクセスするため
	actionModal          widget.PreferredSizeLocateableWidget
	isActionModalVisible bool                                             // アクションモーダルが表示されているか
	actionTargetMap      map[component.PartSlotKey]component.ActionTarget // 選択可能なアクションとターゲットのマップ
	eventChannel         chan UIEvent                                     // UIイベント通知用
	uiFactory            *UIFactory                                       // 追加
	commonUIPanel        *UIPanel                                         // 共通のUIPanelを保持
}

// NewUIActionModalManager は新しいUIActionModalManagerのインスタンスを作成します。
func NewUIActionModalManager(ebitenui *ebitenui.UI, eventChannel chan UIEvent, uiFactory *UIFactory, commonUIPanel *UIPanel) *UIActionModalManager {
	return &UIActionModalManager{
		ebitenui:             ebitenui,
		isActionModalVisible: false,
		actionTargetMap:      make(map[component.PartSlotKey]component.ActionTarget),
		eventChannel:         eventChannel,
		uiFactory:            uiFactory,
		commonUIPanel:        commonUIPanel, // 共通UIPanelを設定
	}
}

// ShowActionModal はアクション選択モーダルを表示します。
func (m *UIActionModalManager) ShowActionModal(vm ui.ActionModalViewModel) {
	m.isActionModalVisible = true

	// actionTargetMap を ViewModel の情報から再構築
	m.actionTargetMap = make(map[component.PartSlotKey]component.ActionTarget)
	for _, btn := range vm.Buttons {
		m.actionTargetMap[btn.SlotKey] = component.ActionTarget{TargetEntityID: btn.TargetEntityID, Slot: btn.SlotKey}
	}

	// ViewModel を直接 createActionModalUI に渡す
	modal := createActionModalUI(&vm, m.uiFactory, m.eventChannel)
	m.actionModal = modal
	m.commonUIPanel.SetContent(m.actionModal) // commonUIPanel の SetContent を呼び出す
	log.Println("アクションモーダルを表示しました。")
}

// HideActionModal はアクション選択モーダルを非表示にします。
func (m *UIActionModalManager) HideActionModal() {
	if !m.isActionModalVisible {
		return
	}
	if m.actionModal != nil {
		m.commonUIPanel.SetContent(nil) // commonUIPanel の SetContent でコンテンツをクリア
		m.actionModal = nil
	}
	m.isActionModalVisible = false
	m.actionTargetMap = make(map[component.PartSlotKey]component.ActionTarget) // ターゲットマップをクリア
	log.Println("アクションモーダルを非表示にしました。")
}

// IsVisible はアクションモーダルが表示されているかどうかを返します。
func (m *UIActionModalManager) IsVisible() bool {
	return m.isActionModalVisible
}

// GetActionTargetMap は現在のアクションターゲットマップを返します。
func (m *UIActionModalManager) GetActionTargetMap() map[component.PartSlotKey]component.ActionTarget {
	return m.actionTargetMap
}
