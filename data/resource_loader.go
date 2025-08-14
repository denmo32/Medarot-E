package data

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"medarot-ebiten/core"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	resource "github.com/quasilyte/ebitengine-resource"
)

// グローバル変数 r を削除しました。
// var r *resource.Loader

// NewLoader はリソースローダーを初期化してそのインスタンスを返します。
// 以前の InitResources の役割を引き継ぎますが、グローバル変数に代入する代わりに、
// 生成したローダーを返すことで、依存関係を明確にします。
func NewLoader(audioContext *audio.Context, assetPaths *AssetPaths) *resource.Loader {
	loader := resource.NewLoader(audioContext)

	// In a real application, you would use something like go:embed
	// to bundle your assets. For this example, we'll use os.ReadFile.
	// This function will be used by the loader to get the resource data.
	loader.OpenAssetFunc = func(path string) io.ReadCloser {
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
		RawMessagesJSON: {Path: assetPaths.Messages}, // 追加
	}
	loader.RawRegistry.Assign(rawResources)

	// Register font resources.
	fontResources := map[resource.FontID]resource.FontInfo{
		FontMPLUS1pRegular: {Path: assetPaths.Font, Size: 9}, // フォントサイズ
	}
	loader.FontRegistry.Assign(fontResources)

	// Register image resources.
	imageResources := map[resource.ImageID]resource.ImageInfo{
		ImageBattleBackground: {Path: assetPaths.Image},
	}
	loader.ImageRegistry.Assign(imageResources)

	return loader
}

// LoadFonts は、引数で受け取ったローダーを使用してフォントを読み込みます。
// これにより、この関数がどのリソースローダーに依存しているかが明確になります。
func LoadFonts(loader *resource.Loader, assetPaths *AssetPaths, config *Config) (text.Face, text.Face, text.Face, error) {
	// ベースフォントの読み込み
	loader.FontRegistry.Assign(map[resource.FontID]resource.FontInfo{
		FontMPLUS1pRegular: {Path: assetPaths.Font, Size: 9}, // ベースフォントサイズ
	})
	baseFont := loader.LoadFont(FontMPLUS1pRegular)
	normalFont := text.NewGoXFace(baseFont.Face)

	// モーダルボタン用フォントの読み込み
	loader.FontRegistry.Assign(map[resource.FontID]resource.FontInfo{
		FontModalButton: {Path: assetPaths.Font, Size: int(config.UI.ActionModal.ModalButtonFontSize)},
	})
	modalButtonFont := text.NewGoXFace(loader.LoadFont(FontModalButton).Face)

	// メッセージウィンドウ用フォントの読み込み
	loader.FontRegistry.Assign(map[resource.FontID]resource.FontInfo{
		FontMessageWindow: {Path: assetPaths.Font, Size: int(config.UI.MessageWindow.MessageWindowFontSize)},
	})
	messageWindowFont := text.NewGoXFace(loader.LoadFont(FontMessageWindow).Face)

	return normalFont, modalButtonFont, messageWindowFont, nil
}

// LoadFormulas は、引数で受け取ったローダーを使用して計算式をJSONリソースから読み込みます。
func LoadFormulas(loader *resource.Loader) (map[core.Trait]core.ActionFormula, error) {
	res := loader.LoadRaw(RawFormulasJSON)
	var formulasConfig map[core.Trait]core.ActionFormulaConfig
	err := json.Unmarshal(res.Data, &formulasConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal formulas data: %w", err)
	}

	formulas := make(map[core.Trait]core.ActionFormula)
	for trait, formulaCfg := range formulasConfig {
		formulas[trait] = core.ActionFormula{
			ID:                 string(trait),
			SuccessRateBonuses: formulaCfg.SuccessRateBonuses,
			PowerBonuses:       formulaCfg.PowerBonuses,
			CriticalRateBonus:  formulaCfg.CriticalRateBonus,
			UserDebuffs:        formulaCfg.UserDebuffs,
		}
	}
	return formulas, nil
}

// LoadAllStaticGameData は、引数で受け取ったローダーを使用して全ての静的ゲームデータを読み込みます。
func LoadAllStaticGameData(loader *resource.Loader, gdm *GameDataManager) error {
	if err := LoadMedals(loader, gdm); err != nil {
		return fmt.Errorf("failed to load medals.csv: %w", err)
	}
	if err := LoadParts(loader, gdm); err != nil {
		return fmt.Errorf("failed to load parts.csv: %w", err)
	}
	return nil
}

// LoadMedals は、引数で受け取ったローダーを使用してメダル定義をCSVリソースから読み込みます。
func LoadMedals(loader *resource.Loader, gdm *GameDataManager) error {
	res := loader.LoadRaw(RawMedalsCSV)
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
		medal := core.Medal{
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

// LoadParts は、引数で受け取ったローダーを使用してパーツ定義をCSVリソースから読み込みます。
func LoadParts(loader *resource.Loader, gdm *GameDataManager) error {
	res := loader.LoadRaw(RawPartsCSV)
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
		partDef := &core.PartDefinition{
			ID:         record[0],
			PartName:   record[1],
			Type:       core.PartType(record[2]),
			Category:   core.PartCategory(record[3]),
			Trait:      core.Trait(record[4]),
			MaxArmor:   maxArmor,
			Power:      parseInt(record[7], 0),
			Charge:     parseInt(record[8], 1),
			Cooldown:   parseInt(record[9], 1),
			Defense:    parseInt(record[10], 0),
			Accuracy:   parseInt(record[11], 0),
			Mobility:   parseInt(record[12], 0),
			Propulsion: parseInt(record[13], 0),
			Stability:  parseInt(record[14], 0),
			WeaponType: core.WeaponType(record[5]), // WeaponType型にキャスト
		}
		if err := gdm.AddPartDefinition(partDef); err != nil {
			fmt.Printf("error adding part definition %s: %v\n", partDef.ID, err)
		}
	}
	return nil
}

// LoadMedarotLoadouts は、引数で受け取ったローダーを使用してメダロットの構成データをCSVリソースから読み込みます。
func LoadMedarotLoadouts(loader *resource.Loader) ([]core.MedarotData, error) {
	res := loader.LoadRaw(RawMedarotsCSV)
	reader := csv.NewReader(bytes.NewReader(res.Data))
	_, err := reader.Read() // Skip header
	if err != nil {
		return nil, fmt.Errorf("failed to read header from medarots data: %w", err)
	}

	var medarots []core.MedarotData
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
		medarot := core.MedarotData{
			ID:         record[0],
			Name:       record[1],
			Team:       core.TeamID(parseInt(record[2], 0)),
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

// GetImage は、引数で受け取ったローダーを使用して画像リソースを取得します。
func GetImage(loader *resource.Loader, id resource.ImageID) resource.Image {
	return loader.LoadImage(id)
}
