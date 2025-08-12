package ui

import (
	"medarot-ebiten/core"
	"medarot-ebiten/event" // GameEventを使用するためインポート
)

// UIMediator はUI関連の操作を仲介する実装です。
type UIMediator struct {
	viewModelFactory *ViewModelFactory
	uiInterface      UIInterface // UIの具体的な操作を行うインターフェース
}

// NewUIMediator は新しいUIMediatorのインスタンスを作成します。
func NewUIMediator(viewModelFactory *ViewModelFactory, uiInterface UIInterface) *UIMediator {
	return &UIMediator{
		viewModelFactory: viewModelFactory,
		uiInterface:      uiInterface,
	}
}

// EnqueueMessage は単一のメッセージをキューに追加します。
func (m *UIMediator) EnqueueMessage(msg string, callback func()) {
	m.uiInterface.GetMessageDisplayManager().EnqueueMessage(msg, callback)
}

// EnqueueMessageQueue は複数のメッセージをキューに追加します。
func (m *UIMediator) EnqueueMessageQueue(messages []string, callback func()) {
	m.uiInterface.GetMessageDisplayManager().EnqueueMessageQueue(messages, callback)
}

// IsMessageFinished はメッセージキューが空で、かつメッセージウィンドウが表示されていない場合にtrueを返します。
func (m *UIMediator) IsMessageFinished() bool {
	return m.uiInterface.GetMessageDisplayManager().IsFinished()
}

// ShowActionModal はアクション選択モーダルを表示します。
func (m *UIMediator) ShowActionModal(vm core.ActionModalViewModel) {
	m.uiInterface.ShowActionModal(vm)
}

// HideActionModal はアクション選択モーダルを非表示にします。
func (m *UIMediator) HideActionModal() {
	m.uiInterface.HideActionModal()
}

// PostUIEvent はUIイベントをBattleSceneのキューに追加します。
func (m *UIMediator) PostUIEvent(e interface{}) { // interface{}で受け取る
	if gameEvent, ok := e.(event.GameEvent); ok {
		m.uiInterface.PostEvent(gameEvent)
	}
}

// ClearAnimation は現在のアニメーションをクリアします。
func (m *UIMediator) ClearAnimation() {
	m.uiInterface.ClearAnimation()
}

// ClearCurrentTarget は現在のターゲットをクリアします。
func (m *UIMediator) ClearCurrentTarget() {
	m.uiInterface.ClearCurrentTarget()
}

// IsActionModalVisible はアクションモーダルが表示されているかどうかを返します。
func (m *UIMediator) IsActionModalVisible() bool {
	return m.uiInterface.IsActionModalVisible()
}

// GetViewModelFactory は内部のViewModelFactoryを返します。
func (m *UIMediator) GetViewModelFactory() *ViewModelFactory {
	return m.viewModelFactory
}
