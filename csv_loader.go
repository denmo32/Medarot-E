package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ... (parseInt, parseBool, LoadMedalsは変更なし) ...
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
	reader.Read() // Skip header

	var medals []Medal
	for {
		// medals.csv の列数が少ないので、lenチェックは不要
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		// medals.csv の列構造に合わせて修正
		medals = append(medals, Medal{
			ID:   record[0],
			Name: record[1],
			// skill_shoot, skill_fightを考慮して、ここでは単純にSkillLevelを固定値にするか、
			// またはCSVに合わせてMedal構造体自体を修正する必要があります。
			// 今回は skill_fight を代表値として使います。
			SkillLevel: parseInt(record[6], 1), // "skill_fight" はインデックス6
		})
	}
	return medals, nil
}

// [FIXED] LoadParts のインデックスをCSVの列に完全に合わせる
func LoadParts(filePath string) (map[string]*Part, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Read() // Skip header

	partsMap := make(map[string]*Part)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 14 { // 列数は14
			continue
		}
		// 正しいインデックスでArmorを読み込む
		armor := parseInt(record[6], 1)
		part := &Part{
			ID:       record[0],
			PartName: record[1],
			Type:     PartType(record[2]),
			Category: PartCategory(record[3]),
			Trait:    Trait(record[4]),
			// weapon_type (record[5]) は現在Part構造体にないので読み飛ばす
			Armor:      armor,
			MaxArmor:   armor,
			Power:      parseInt(record[7], 0),  // power はインデックス7
			Charge:     parseInt(record[8], 1),  // charge はインデックス8
			Cooldown:   parseInt(record[9], 1),  // cooldown はインデックス9
			Defense:    parseInt(record[10], 0), // defense はインデックス10
			Accuracy:   parseInt(record[11], 0), // accuracy はインデックス11
			Mobility:   parseInt(record[12], 0), // mobility はインデックス12
			Propulsion: parseInt(record[13], 0), // propulsion はインデックス13
			IsBroken:   false,
		}
		partsMap[part.ID] = part
	}
	return partsMap, nil
}

// ... (LoadMedarotLoadouts, LoadAllGameDataは変更なし) ...
func LoadMedarotLoadouts(filePath string) ([]MedarotData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Read() // Skip header

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
