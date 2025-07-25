package main

import (
	"strconv"
	"strings"
)

// parseInt は文字列をintに変換します。変換できない場合はdefaultValueを返します。
func parseInt(s string, defaultValue int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return i
}

// parseBool は文字列をboolに変換します。"true" (大文字小文字を区別しない) の場合のみtrueを返します。
func parseBool(s string) bool {
	return strings.ToLower(strings.TrimSpace(s)) == "true"
}

// GetStateDisplayName は StateType に対応する日本語の表示名を返します。
func GetStateDisplayName(state StateType) string {
	switch state {
	case StateIdle:
		return "待機"
	case StateCharging:
		return "チャージ中"
	case StateReady:
		return "実行準備"
	case StateCooldown:
		return "クールダウン"
	case StateBroken:
		return "機能停止"
	default:
		return "不明"
	}
}
