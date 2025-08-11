package ui

import (
	"log"

	"medarot-ebiten/core"
	"medarot-ebiten/event"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/yohamta/donburi"
)

// UIActionModalManager はアクション選択モーダルの表示と状態を管理します。
type UIActionModalManager struct {
	ebitenui             *ebitenui.UI
	actionModal          widget.PreferredSizeLocateableWidget
	isActionModalVisible bool
	actionTargetMap      map[core.PartSlotKey]core.ActionTarget
	eventChannel         chan event.GameEvent
	uiFactory            *UIFactory
	commonUIPanel        *UIPanel
}

// NewUIActionModalManager は新しいUIActionModalManagerのインスタンスを作成します。
func NewUIActionModalManager(ebitenui *ebitenui.UI, eventChannel chan event.GameEvent, uiFactory *UIFactory, commonUIPanel *UIPanel) *UIActionModalManager {
	return &UIActionModalManager{
		ebitenui:             ebitenui,
		isActionModalVisible: false,
		actionTargetMap:      make(map[core.PartSlotKey]core.ActionTarget),
		eventChannel:         eventChannel,
		uiFactory:            uiFactory,
		commonUIPanel:        commonUIPanel,
	}
}

// ShowActionModal はアクション選択モーダルを表示します。
func (m *UIActionModalManager) ShowActionModal(vm core.ActionModalViewModel, world donburi.World, bum *BattleUIManager) {
	m.isActionModalVisible = true

	m.actionTargetMap = make(map[core.PartSlotKey]core.ActionTarget)
	for _, btn := range vm.Buttons {
		m.actionTargetMap[btn.SlotKey] = core.ActionTarget{TargetEntityID: btn.TargetEntityID, Slot: btn.SlotKey}
	}

	modal := createActionModalUI(&vm, m.uiFactory, m.eventChannel, world, bum)
	m.actionModal = modal
	m.commonUIPanel.SetContent(m.actionModal)
	log.Println("アクションモーダルを表示しました。")
}

// HideActionModal はアクション選択モーダルを非表示にします。
func (m *UIActionModalManager) HideActionModal() {
	if !m.isActionModalVisible {
		return
	}
	if m.actionModal != nil {
		m.commonUIPanel.SetContent(nil)
		m.actionModal = nil
	}
	m.isActionModalVisible = false
	m.actionTargetMap = make(map[core.PartSlotKey]core.ActionTarget)
	log.Println("アクションモーダルを非表示にしました。")
}

// IsVisible はアクションモーダルが表示されているかどうかを返します。
func (m *UIActionModalManager) IsVisible() bool {
	return m.isActionModalVisible
}

// GetActionTargetMap は現在のアクションターゲットマップを返します。
func (m *UIActionModalManager) GetActionTargetMap() map[core.PartSlotKey]core.ActionTarget {
	return m.actionTargetMap
}