package main

import (
	"log"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/yohamta/donburi"
)

// UIActionModalManager はアクション選択モーダルの表示と状態を管理します。
type UIActionModalManager struct {
	ebitenui             *ebitenui.UI // UIのルートコンテナにアクセスするため
	actionModal          widget.PreferredSizeLocateableWidget
	isActionModalVisible bool                         // アクションモーダルが表示されているか
	actionTargetMap      map[PartSlotKey]ActionTarget // 選択可能なアクションとターゲットのマップ
	eventChannel         chan UIEvent                 // UIイベント通知用
	uiFactory            *UIFactory                   // 追加
	world                donburi.World                // 追加
}

// NewUIActionModalManager は新しいUIActionModalManagerのインスタンスを作成します。
func NewUIActionModalManager(ebitenui *ebitenui.UI, eventChannel chan UIEvent, uiFactory *UIFactory, world donburi.World) *UIActionModalManager {
	return &UIActionModalManager{
		ebitenui:             ebitenui,
		isActionModalVisible: false,
		actionTargetMap:      make(map[PartSlotKey]ActionTarget),
		eventChannel:         eventChannel,
		uiFactory:            uiFactory,
		world:                world,
	}
}

// ShowActionModal はアクション選択モーダルを表示します。
func (m *UIActionModalManager) ShowActionModal(vm ActionModalViewModel) {
	if m.isActionModalVisible {
		m.HideActionModal()
	}
	m.isActionModalVisible = true

	// actionTargetMap を ViewModel の情報から再構築
	m.actionTargetMap = make(map[PartSlotKey]ActionTarget)
	for _, btn := range vm.Buttons {
		var targetEntry *donburi.Entry
		if btn.TargetEntityID != 0 {
			targetEntry = m.world.Entry(btn.TargetEntityID)
		}
		m.actionTargetMap[btn.SlotKey] = ActionTarget{Target: targetEntry, Slot: btn.SlotKey}
	}

	// ViewModel を直接 createActionModalUI に渡す
	modal := createActionModalUI(&vm, m.uiFactory, m.eventChannel)
	m.actionModal = modal
	m.ebitenui.Container.AddChild(m.actionModal)
	log.Println("アクションモーダルを表示しました。")
}

// HideActionModal はアクション選択モーダルを非表示にします。
func (m *UIActionModalManager) HideActionModal() {
	if !m.isActionModalVisible {
		return
	}
	if m.actionModal != nil {
		if m.ebitenui == nil || m.ebitenui.Container == nil {
			log.Printf("WARNING: HideActionModal: m.ebitenui or m.ebitenui.Container is nil. Cannot remove child.")
			// Reset state to prevent infinite loop if modal is somehow stuck
			m.isActionModalVisible = false
			m.actionModal = nil
			return
		}
		m.ebitenui.Container.RemoveChild(m.actionModal)
		m.actionModal = nil
	}
	m.isActionModalVisible = false
	m.actionTargetMap = make(map[PartSlotKey]ActionTarget) // ターゲットマップをクリア
	log.Println("アクションモーダルを非表示にしました。")
}

// IsVisible はアクションモーダルが表示されているかどうかを返します。
func (m *UIActionModalManager) IsVisible() bool {
	return m.isActionModalVisible
}

// GetActionTargetMap は現在のアクションターゲットマップを返します。
func (m *UIActionModalManager) GetActionTargetMap() map[PartSlotKey]ActionTarget {
	return m.actionTargetMap
}
