package data

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
)

// JSONColors は game_settings.json の "UI.Colors" セクションを
// デシリアライズするための中間構造体です。16進数文字列として色を読み込みます。
type JSONColors struct {
	White      string `json:"White"`
	Red        string `json:"Red"`
	Blue       string `json:"Blue"`
	Yellow     string `json:"Yellow"`
	Gray       string `json:"Gray"`
	Team1      string `json:"Team1"`
	Team2      string `json:"Team2"`
	Leader     string `json:"Leader"`
	Broken     string `json:"Broken"`
	HP         string `json:"HP"`
	HPCritical string `json:"HPCritical"`
	Background string `json:"Background"`
	Black      string `json:"Black"`
}

// LoadConfig は設定ファイルを読み込み、初期化済みのConfig構造体を返します。
// 中間構造体を廃止し、JSONから直接Config構造体にデシリアライズすることで、
// 冗長な手動コピー処理を排除し、コードを簡潔で保守しやすくしています。
func LoadConfig() Config {
	// アセットパスを一元管理
	assetPaths := AssetPaths{
		GameSettings: "assets/configs/game_settings.json",
		Messages:     "assets/texts/messages.json",
		MedalsCSV:    "assets/databases/medals.csv",
		PartsCSV:     "assets/databases/parts.csv",
		MedarotsCSV:  "assets/databases/medarots.csv",
		FormulasJSON: "assets/configs/formulas.json",
		Font:         "assets/fonts/MPLUS1p-Regular.ttf",
		Image:        "assets/images/Gemini_Generated_Image_hojkprhojkprhojk.png",
	}

	// game_settings.json をファイルから読み込み
	jsonFile, err := ioutil.ReadFile(assetPaths.GameSettings)
	if err != nil {
		log.Fatalf("Error reading game_settings.json: %v", err)
	}

	// 手順1: JSONデータをConfig構造体に直接アンマーシャルします。
	// UI.Colorsは型が異なるため、この時点では設定されません。
	var cfg Config
	err = json.Unmarshal(jsonFile, &cfg)
	if err != nil {
		log.Fatalf("Error unmarshalling game_settings.json into Config: %v", err)
	}

	// 手順2: 色設定(文字列)をデコードするための中間構造体に再度アンマーシャルします。
	var tempColors struct {
		UI struct {
			Colors JSONColors `json:"Colors"`
		} `json:"UI"`
	}
	err = json.Unmarshal(jsonFile, &tempColors)
	if err != nil {
		log.Fatalf("Error unmarshalling game_settings.json for colors: %v", err)
	}

	// 手順3: 文字列からパースした色データを、メインのConfig構造体に設定します。
	cfg.UI.Colors = parseJSONColors(tempColors.UI.Colors)

	// 手順4: JSONファイルには含まれない、コード側で定義する値を設定します。
	cfg.AssetPaths = assetPaths
	// GameConfigのRandomSeedなどは必要に応じてここで設定できます。
	// 例: cfg.Game.RandomSeed = time.Now().UnixNano()

	return cfg
}

// parseJSONColors はJSONColors構造体（16進数文字列）から
// ParsedColors構造体（color.Color）に変換するヘルパー関数です。
func parseJSONColors(jc JSONColors) ParsedColors {
	return ParsedColors{
		White:      parseHexColor(jc.White),
		Red:        parseHexColor(jc.Red),
		Blue:       parseHexColor(jc.Blue),
		Yellow:     parseHexColor(jc.Yellow),
		Gray:       parseHexColor(jc.Gray),
		Team1:      parseHexColor(jc.Team1),
		Team2:      parseHexColor(jc.Team2),
		Leader:     parseHexColor(jc.Leader),
		Broken:     parseHexColor(jc.Broken),
		HP:         parseHexColor(jc.HP),
		HPCritical: parseHexColor(jc.HPCritical),
		Background: parseHexColor(jc.Background),
		Black:      parseHexColor(jc.Black),
	}
}

// parseHexColor は16進数文字列からcolor.Colorをパースします。
func parseHexColor(s string) color.Color {
	var r, g, b uint8
	if len(s) == 6 {
		_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
		if err != nil {
			log.Printf("Failed to parse hex color %s: %v", s, err)
			return color.White // エラー時はデフォルト色として白を返す
		}
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}
}