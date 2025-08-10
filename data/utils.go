package data

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
