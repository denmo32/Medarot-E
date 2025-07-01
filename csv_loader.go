package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// parseInt, parseBool, LoadMedals, LoadParts, LoadMedarotLoadouts, LoadAllGameData
// ... (既存のLoad系関数は変更なし) ...
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

func LoadMedals(filePath string) ([]Medal, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Read()

	var medals []Medal
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		medals = append(medals, Medal{
			ID:          record[0],
			Name:        record[1],
			Personality: record[2],
			SkillLevel:  parseInt(record[6], 1),
		})
	}
	return medals, nil
}

func LoadParts(filePath string) (map[string]*Part, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Read()

	partsMap := make(map[string]*Part)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 14 {
			continue
		}
		armor := parseInt(record[6], 1)
		part := &Part{
			ID:         record[0],
			PartName:   record[1],
			Type:       PartType(record[2]),
			Category:   PartCategory(record[3]),
			Trait:      Trait(record[4]),
			Armor:      armor,
			MaxArmor:   armor,
			Power:      parseInt(record[7], 0),
			Charge:     parseInt(record[8], 1),
			Cooldown:   parseInt(record[9], 1),
			Defense:    parseInt(record[10], 0),
			Accuracy:   parseInt(record[11], 0),
			Mobility:   parseInt(record[12], 0),
			Propulsion: parseInt(record[13], 0),
			IsBroken:   false,
		}
		partsMap[part.ID] = part
	}
	return partsMap, nil
}

func LoadMedarotLoadouts(filePath string) ([]MedarotData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Read()

	var medarots []MedarotData
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 10 {
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

func LoadAllGameData() (*GameData, error) {
	gameData := &GameData{}
	var err error

	gameData.Medals, err = LoadMedals("data/medals.csv")
	if err != nil {
		return nil, fmt.Errorf("medals.csvの読み込みに失敗: %w", err)
	}

	gameData.AllParts, err = LoadParts("data/parts.csv")
	if err != nil {
		return nil, fmt.Errorf("parts.csvの読み込みに失敗: %w", err)
	}

	gameData.Medarots, err = LoadMedarotLoadouts("data/medarots.csv")
	if err != nil {
		return nil, fmt.Errorf("medarots.csvの読み込みに失敗: %w", err)
	}

	return gameData, nil
}

// ★★★ ここから下を新しく追加 ★★★

// SaveMedarotLoadouts は、現在のメダロットの構成をCSVファイルに保存します。
func SaveMedarotLoadouts(filePath string, medarots []MedarotData) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// ヘッダー行を書き込む
	header := []string{"id", "name", "team", "is_leader", "draw_index", "medal_id", "head_id", "r_arm_id", "l_arm_id", "legs_id"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
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
			return fmt.Errorf("failed to write record for %s: %w", medarot.Name, err)
		}
	}

	return nil
}
