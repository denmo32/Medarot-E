package main

import (
	"github.com/yohamta/donburi"
)

// UITargetIndicatorManager はターゲットインジケーターの表示と状態を管理します。
type UITargetIndicatorManager struct {
	currentTarget *donburi.Entry // 現在ターゲットとして表示されているエンティティ
}

// NewUITargetIndicatorManager は新しいUITargetIndicatorManagerのインスタンスを作成します。
func NewUITargetIndicatorManager() *UITargetIndicatorManager {
	return &UITargetIndicatorManager{}
}

// SetCurrentTarget は現在のターゲットを設定します。
func (m *UITargetIndicatorManager) SetCurrentTarget(entry *donburi.Entry) {
	m.currentTarget = entry
}

// ClearCurrentTarget は現在のターゲットをクリアします。
func (m *UITargetIndicatorManager) ClearCurrentTarget() {
	m.currentTarget = nil
}

// GetCurrentTarget は現在のターゲットエンティティを返します。
func (m *UITargetIndicatorManager) GetCurrentTarget() *donburi.Entry {
	return m.currentTarget
}
