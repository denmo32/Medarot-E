package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

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

func parseBool(s string) bool {
	return strings.ToLower(strings.TrimSpace(s)) == "true"
}

// LoadMedals はCSVからメダル定義を読み込み、GameDataManagerに格納します。
func LoadMedals(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	_, err = reader.Read() // ヘッダーをスキップ
	if err != nil {
		return fmt.Errorf("%s からヘッダーの読み込みに失敗しました: %w", filePath, err)
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("%s からレコードの読み込み中にエラーが発生しました: %v\n", filePath, err)
			continue
		}
		if len(record) < 7 { // メダルの列数が十分か基本的なチェック
			fmt.Printf("%s の不正な形式のレコードをスキップします (列数が不足しています): %v\n", filePath, record)
			continue
		}
		medal := Medal{
			ID:          record[0],
			Name:        record[1],
			Personality: record[2],
			// Medaforce (record[3]), Attribute (record[4]), skill_shoot (record[5]) が存在する可能性があると仮定
			SkillLevel: parseInt(record[6], 1), // 元のparseIntに従い、skill_fightがインデックス6にあると仮定
		}
		if err := GlobalGameDataManager.AddMedalDefinition(&medal); err != nil {
			fmt.Printf("メダル定義 %s の追加中にエラーが発生しました: %v\n", medal.ID, err)
		}
	}
	return nil
}

// LoadParts はCSVからパーツ定義を読み込み、GameDataManagerに格納します。
func LoadParts(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Read() // ヘッダーをスキップ

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 15 { // レコードに十分な列があることを確認
			fmt.Printf("%s の不正な形式のレコードをスキップします: %v (エラー: %v)\n", filePath, record, err)
			continue
		}
		maxArmor := parseInt(record[6], 1) // MaxArmorはCSVから取得
		partDef := &PartDefinition{
			ID:         record[0],
			PartName:   record[1],
			Type:       PartType(record[2]),
			Category:   PartCategory(record[3]),
			Trait:      Trait(record[4]),
			MaxArmor:   maxArmor, // 定義からMaxArmorを使用
			Power:      parseInt(record[7], 0),
			Charge:     parseInt(record[8], 1),
			Cooldown:   parseInt(record[9], 1),
			Defense:    parseInt(record[10], 0),
			Accuracy:   parseInt(record[11], 0),
			Mobility:   parseInt(record[12], 0),
			Propulsion: parseInt(record[13], 0),
			Stability:  parseInt(record[14], 0),
			WeaponType: record[5],
		}
		if err := GlobalGameDataManager.AddPartDefinition(partDef); err != nil {
			fmt.Printf("パーツ定義 %s の追加中にエラーが発生しました: %v\n", partDef.ID, err)
		}
	}
	return nil // データはGlobalGameDataManagerにロードされます
}

// LoadMedarotLoadouts はメダロットのセットアップデータのみをロードします。
// パーツとメダルの定義は、別途GameDataManagerにロードする必要があります。
func LoadMedarotLoadouts(filePath string) ([]MedarotData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	_, err = reader.Read() // ヘッダーをスキップ
	if err != nil {
		return nil, fmt.Errorf("%s からヘッダーの読み込みに失敗しました: %w", filePath, err)
	}

	var medarots []MedarotData
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("%s からレコードの読み込み中にエラーが発生しました: %v\n", filePath, err)
			continue
		}
		if len(record) < 10 {
			fmt.Printf("%s の不正な形式のレコードをスキップします (列数が不足しています): %v\n", filePath, record)
			continue
		}
		medarot := MedarotData{
			ID:         record[0],
			Name:       record[1],
			Team:       TeamID(parseInt(record[2], 0)),
			IsLeader:   parseBool(record[3]),
			DrawIndex:  parseInt(record[4], 0),
			MedalID:    record[5],
			HeadID:     record[6],
			RightArmID: record[7],
			LeftArmID:  record[8],
			LegsID:     record[9],
		}
		medarots = append(medarots, medarot)
	}
	return medarots, nil
}

// LoadAllStaticGameData はすべての静的定義（パーツ、メダル）をGameDataManagerにロードします。
// ゲーム開始時に一度呼び出す必要があります。
// MedarotLoadoutsは通常、この関数の呼び出し元、またはそれらを必要とするゲームシーンによってロードされます。
func LoadAllStaticGameData() error {
	if err := LoadMedals("data/medals.csv"); err != nil {
		return fmt.Errorf("medals.csvの読み込みに失敗: %w", err)
	}
	if err := LoadParts("data/parts.csv"); err != nil {
		return fmt.Errorf("parts.csvの読み込みに失敗: %w", err)
	}
	return nil
}

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
