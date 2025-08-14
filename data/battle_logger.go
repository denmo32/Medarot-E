package data

import (
	"log"
)

// BattleLogger は戦闘中の詳細な計算過程などをデバッグ目的でコンソールに出力するためのインターフェースです。
// UIに表示されるメッセージの生成は、ui.BattleUIManagerが担当します。
type BattleLogger interface {
	LogHitCheck(attackerName, targetName string, chance, successRate, evasion float64, roll int)
	LogDefenseCheck(targetName, defensePartName string, chance, defenseRate, successRate float64, roll int)
	LogCriticalHit(attackerName string, chance float64)
	LogPartBroken(medarotName, partName, partID string)
}

// BattleLoggerImpl は BattleLogger インターフェースの実装です。
type BattleLoggerImpl struct {
	gameDataManager *GameDataManager
}

// NewBattleLogger は新しい BattleLoggerImpl のインスタンスを生成します。
func NewBattleLogger(gdm *GameDataManager) BattleLogger {
	return &BattleLoggerImpl{gameDataManager: gdm}
}

// LogHitCheck は命中判定のロールと計算過程をログに出力します。
func (l *BattleLoggerImpl) LogHitCheck(attackerName, targetName string, chance, successRate, evasion float64, roll int) {
	log.Print(l.gameDataManager.Messages.FormatMessage("log_hit_roll", map[string]interface{}{
		"ordered_args": []interface{}{attackerName, targetName, chance, successRate, evasion, roll},
	}))
}

// LogDefenseCheck は防御判定のロールと計算過程をログに出力します。
// 防御パーツ名を引数に追加し、ログメッセージを正しくフォーマットできるように修正しました。
func (l *BattleLoggerImpl) LogDefenseCheck(targetName, defensePartName string, chance, defenseRate, successRate float64, roll int) {
	log.Print(l.gameDataManager.Messages.FormatMessage("log_defense_roll", map[string]interface{}{
		"ordered_args": []interface{}{targetName, defensePartName, chance, roll},
	}))
}

// LogCriticalHit はクリティカルヒットの発生と確率をログに出力します。
func (l *BattleLoggerImpl) LogCriticalHit(attackerName string, chance float64) {
	// メッセージテンプレートが `%d` を期待しているため、chanceをintにキャストします。
	log.Print(l.gameDataManager.Messages.FormatMessage("log_critical_hit_details", map[string]interface{}{
		"ordered_args": []interface{}{attackerName, int(chance)},
	}))
}

// LogPartBroken はパーツが破壊されたことをログに出力します。
func (l *BattleLoggerImpl) LogPartBroken(medarotName, partName, partID string) {
	log.Print(l.gameDataManager.Messages.FormatMessage("log_part_broken_notification", map[string]interface{}{
		"ordered_args": []interface{}{medarotName, partName, partID},
	}))
}
