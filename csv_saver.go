package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

// SaveMedarotLoadouts は、現在のメダロットの構成をCSVファイルに保存します。
func SaveMedarotLoadouts(filePath string, medarots []MedarotData) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("ファイルの作成に失敗しました: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// ヘッダー行を書き込む
	header := []string{"id", "name", "team", "is_leader", "draw_index", "medal_id", "head_id", "r_arm_id", "l_arm_id", "legs_id"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("ヘッダーの書き込みに失敗しました: %w", err)
	}

	// 各メダロットのデータを書き込む
	for _, medarot := range medarots {
		record := []string{
			medarot.ID,
			medarot.Name,
			strconv.Itoa(int(medarot.Team)),
			strconv.FormatBool(medarot.IsLeader),
			strconv.Itoa(medarot.DrawIndex),
			medarot.MedalID,
			medarot.HeadID,
			medarot.RightArmID,
			medarot.LeftArmID,
			medarot.LegsID,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("%s のレコード書き込みに失敗しました: %w", medarot.Name, err)
		}
	}
	return nil
}
