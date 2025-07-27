package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	resource "github.com/quasilyte/ebitengine-resource"
)

// Global resource loader
var r *resource.Loader

func initResources(audioContext *audio.Context, assetPaths *AssetPaths) {
	r = resource.NewLoader(audioContext)

	// In a real application, you would use something like go:embed
	// to bundle your assets. For this example, we'll use os.ReadFile.
	// This function will be used by the loader to get the resource data.
	r.OpenAssetFunc = func(path string) io.ReadCloser {
		// For now, we'll read from the filesystem.
		// This can be replaced with an embedded filesystem later.
		data, err := os.ReadFile(path)
		if err != nil {
			// A simple error handling.
			// A real game would probably have a more robust solution.
			panic(err)
		}
		return io.NopCloser(bytes.NewReader(data))
	}

	// Register raw resources (our CSV files).
	rawResources := map[resource.RawID]resource.RawInfo{
		RawMedalsCSV:    {Path: assetPaths.MedalsCSV},
		RawPartsCSV:     {Path: assetPaths.PartsCSV},
		RawMedarotsCSV:  {Path: assetPaths.MedarotsCSV},
		RawFormulasJSON: {Path: assetPaths.FormulasJSON},
	}
	r.RawRegistry.Assign(rawResources)

	// Register font resources.
	fontResources := map[resource.FontID]resource.FontInfo{
		FontMPLUS1pRegular: {Path: assetPaths.Font, Size: 9}, // フォントサイズ
	}
	r.FontRegistry.Assign(fontResources)

	// Register image resources.
	imageResources := map[resource.ImageID]resource.ImageInfo{
		ImageBattleBackground: {Path: assetPaths.Image},
	}
	r.ImageRegistry.Assign(imageResources)
}

func LoadFont(id resource.FontID) (text.Face, error) {
	f := r.LoadFont(id)
	return text.NewGoXFace(f.Face), nil
}

// LoadFormulas loads action formulas from the JSON resource.
func LoadFormulas() (map[Trait]ActionFormula, error) {
	res := r.LoadRaw(RawFormulasJSON)
	var formulasConfig map[Trait]ActionFormulaConfig
	err := json.Unmarshal(res.Data, &formulasConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal formulas data: %w", err)
	}

	formulas := make(map[Trait]ActionFormula)
	for trait, formulaCfg := range formulasConfig {
		formulas[trait] = ActionFormula{
			ID:                 string(trait),
			SuccessRateBonuses: formulaCfg.SuccessRateBonuses,
			PowerBonuses:       formulaCfg.PowerBonuses,
			CriticalRateBonus:  formulaCfg.CriticalRateBonus,
			UserDebuffs:        formulaCfg.UserDebuffs,
		}
	}
	return formulas, nil
}

// LoadAllStaticGameData re-implements the original function using the resource loader.
func LoadAllStaticGameData(gdm *GameDataManager) error {
	if err := LoadMedals(gdm); err != nil {
		return fmt.Errorf("failed to load medals.csv: %w", err)
	}
	if err := LoadParts(gdm); err != nil {
		return fmt.Errorf("failed to load parts.csv: %w", err)
	}
	return nil
}

// LoadMedals loads medal definitions from the CSV resource.
func LoadMedals(gdm *GameDataManager) error {
	res := r.LoadRaw(RawMedalsCSV)
	reader := csv.NewReader(bytes.NewReader(res.Data))
	_, err := reader.Read() // Skip header
	if err != nil {
		return fmt.Errorf("failed to read header from medals data: %w", err)
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("error reading record from medals data: %v\n", err)
			continue
		}
		if len(record) < 7 {
			fmt.Printf("skipping malformed record in medals data (not enough columns): %v\n", record)
			continue
		}
		medal := Medal{
			ID:          record[0],
			Name:        record[1],
			Personality: record[2],
			SkillLevel:  parseInt(record[6], 1),
		}
		if err := gdm.AddMedalDefinition(&medal); err != nil {
			fmt.Printf("error adding medal definition %s: %v\n", medal.ID, err)
		}
	}
	return nil
}

// LoadParts loads part definitions from the CSV resource.
func LoadParts(gdm *GameDataManager) error {
	res := r.LoadRaw(RawPartsCSV)
	reader := csv.NewReader(bytes.NewReader(res.Data))
	reader.Read() // Skip header

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 15 {
			fmt.Printf("skipping malformed record in parts data: %v (error: %v)\n", record, err)
			continue
		}
		maxArmor := parseInt(record[6], 1)
		partDef := &PartDefinition{
			ID:         record[0],
			PartName:   record[1],
			Type:       PartType(record[2]),
			Category:   PartCategory(record[3]),
			Trait:      Trait(record[4]),
			MaxArmor:   maxArmor,
			Power:      parseInt(record[7], 0),
			Charge:     parseInt(record[8], 1),
			Cooldown:   parseInt(record[9], 1),
			Defense:    parseInt(record[10], 0),
			Accuracy:   parseInt(record[11], 0),
			Mobility:   parseInt(record[12], 0),
			Propulsion: parseInt(record[13], 0),
			Stability:  parseInt(record[14], 0),
			WeaponType: WeaponType(record[5]), // WeaponType型にキャスト
		}
		if err := gdm.AddPartDefinition(partDef); err != nil {
			fmt.Printf("error adding part definition %s: %v\n", partDef.ID, err)
		}
	}
	return nil
}

// LoadMedarotLoadouts loads medarot setup data from the CSV resource.
func LoadMedarotLoadouts() ([]MedarotData, error) {
	res := r.LoadRaw(RawMedarotsCSV)
	reader := csv.NewReader(bytes.NewReader(res.Data))
	_, err := reader.Read() // Skip header
	if err != nil {
		return nil, fmt.Errorf("failed to read header from medarots data: %w", err)
	}

	var medarots []MedarotData
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("error reading record from medarots data: %v\n", err)
			continue
		}
		if len(record) < 10 {
			fmt.Printf("skipping malformed record in medarots data (not enough columns): %v\n", record)
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
