package main

import (
	"github.com/yohamta/donburi"
)

// UITargetIndicatorManager はターゲットインジケーターの表示と状態を管理します。
type UITargetIndicatorManager struct {
	currentTarget donburi.Entity // 現在ターゲットとして表示されているエンティティのID
}

// NewUITargetIndicatorManager は新しいUITargetIndicatorManagerのインスタンスを作成します。
func NewUITargetIndicatorManager() *UITargetIndicatorManager {
	return &UITargetIndicatorManager{}
}

// SetCurrentTarget は現在のターゲットを設定します。
func (m *UITargetIndicatorManager) SetCurrentTarget(entityID donburi.Entity) {
	m.currentTarget = entityID
}

// ClearCurrentTarget は現在のターゲットをクリアします。
func (m *UITargetIndicatorManager) ClearCurrentTarget() {
	m.currentTarget = 0 // donburi.Entity のゼロ値
}

// GetCurrentTarget は現在のターゲットエンティティのIDを返します。
func (m *UITargetIndicatorManager) GetCurrentTarget() donburi.Entity {
	return m.currentTarget
}
