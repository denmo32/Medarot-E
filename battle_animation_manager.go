package main

// BattleAnimationManager はゲーム内のアクションアニメーションを管理します。
type BattleAnimationManager struct {
	currentAnimation *ActionAnimationData
	config           *Config
}

// NewBattleAnimationManager は新しいBattleAnimationManagerのインスタンスを作成します。
func NewBattleAnimationManager(config *Config) *BattleAnimationManager {
	return &BattleAnimationManager{
		config: config,
	}
}

// SetAnimation は現在再生するアニメーションを設定します。
func (am *BattleAnimationManager) SetAnimation(anim *ActionAnimationData) {
	am.currentAnimation = anim
}

// IsAnimationFinished は現在のアニメーションが完了したかどうかを返します。
func (am *BattleAnimationManager) IsAnimationFinished(tick int) bool {
	if am.currentAnimation == nil {
		return true
	}
	// ダメージポップアップアニメーションの終了を基準に判断
	const totalAnimationDuration = 120 // UI.DrawAnimationから移動した定数
	return float64(tick-am.currentAnimation.StartTime) >= totalAnimationDuration
}

// ClearAnimation は現在のアニメーションをクリアします。
func (am *BattleAnimationManager) ClearAnimation() {
	am.currentAnimation = nil
}

// GetCurrentAnimationResult は現在のアニメーションの結果を返します。
func (am *BattleAnimationManager) GetCurrentAnimationResult() ActionResult {
	return am.currentAnimation.Result
}
