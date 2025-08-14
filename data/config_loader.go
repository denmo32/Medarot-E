package data

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

// LoadConfig は設定ファイルを読み込み、初期化済みのConfig構造体を返します。
// JSONデコードは一回で完了します。色設定は `ParsedColors.UnmarshalJSON` によって
// 自動的に処理されるため、冗長な中間構造体や手動でのコピー処理は不要です。
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
	// `UI.Colors` フィールドは、`ParsedColors` 型に実装されたカスタムの
	// `UnmarshalJSON` メソッドによって自動的にパースされます。
	var cfg Config
	err = json.Unmarshal(jsonFile, &cfg)
	if err != nil {
		log.Fatalf("Error unmarshalling game_settings.json into Config: %v", err)
	}

	// 手順2: JSONファイルには含まれない、コード側で定義する値を設定します。
	cfg.AssetPaths = assetPaths
	// GameConfigのRandomSeedなどは必要に応じてここで設定できます。
	// 例: cfg.Game.RandomSeed = time.Now().UnixNano()

	return cfg
}