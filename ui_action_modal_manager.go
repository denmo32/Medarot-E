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
	playerMedarotToAct   *donburi.Entry               // 現在アクション選択中のプレイヤーメダロット
	isActionModalVisible bool                         // アクションモーダルが表示されているか
	actionTargetMap      map[PartSlotKey]ActionTarget // 選択可能なアクションとターゲットのマップ
	eventChannel         chan UIEvent                 // UIイベント通知用
	config               *Config
}

// NewUIActionModalManager は新しいUIActionModalManagerのインスタンスを作成します。
func NewUIActionModalManager(ebitenui *ebitenui.UI, eventChannel chan UIEvent, config *Config) *UIActionModalManager {
	return &UIActionModalManager{
		ebitenui:             ebitenui,
		isActionModalVisible: false,
		actionTargetMap:      make(map[PartSlotKey]ActionTarget),
		eventChannel:         eventChannel,
		config:               config,
	}
}

// ShowActionModal はアクション選択モーダルを表示します。
func (m *UIActionModalManager) ShowActionModal(actingEntry *donburi.Entry, actionTargetMap map[PartSlotKey]ActionTarget) {
	if m.isActionModalVisible {
		m.HideActionModal()
	}
	m.playerMedarotToAct = actingEntry
	m.isActionModalVisible = true
	m.actionTargetMap = actionTargetMap // Set the pre-calculated map

	// ViewModelを構築
	settings := SettingsComponent.Get(actingEntry)
	partsComp := PartsComponent.Get(actingEntry)

	var buttons []ActionModalButtonViewModel
	if partsComp == nil {
		log.Println("エラー: ShowActionModal - actingEntry に PartsComponent がありません。")
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

		for _, available := range displayableParts {
			targetInfo := actionTargetMap[available.Slot]
			buttons = append(buttons, ActionModalButtonViewModel{
				PartName:        available.PartDef.PartName,
				PartCategory:    available.PartDef.Category,
				SlotKey:         available.Slot,
				IsBroken:        available.IsBroken,
				TargetEntry:     targetInfo.Target,
				SelectedPartDef: available.PartDef,
			})
		}
	}

	vm := &ActionModalViewModel{
		ActingMedarotName: settings.Name,
		ActingEntry:       actingEntry,
		Buttons:           buttons,
	}

	modal := createActionModalUI(vm, m.config, m.eventChannel, GlobalGameDataManager.Font)
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
	m.playerMedarotToAct = nil
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
