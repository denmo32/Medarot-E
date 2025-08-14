package data

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"medarot-ebiten/core"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	resource "github.com/quasilyte/ebitengine-resource"
)

// InitialGameData はゲームの起動時に必要となる、初期化済みの主要なデータを保持します。
type InitialGameData struct {
	Config            Config
	GameDataManager   *GameDataManager
	GameData          *core.GameData
	NormalFont        text.Face
	ModalButtonFont   text.Face
	MessageWindowFont text.Face
	Loader            *resource.Loader
}

// LoadInitialGameData は、すべての設定ファイルと静的データを読み込み、
// ゲームの起動に必要な構造体を初期化して返します。
// 【修正点】`assetPaths`の定義を正しく含め、各関数に`loader`インスタンスを適切に渡すように修正しました。
func LoadInitialGameData() *InitialGameData {
	// 1. アセットパスの定義
	// この定義が前回の回答で欠落していました。
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

	// 2. game_settings.jsonの読み込み
	jsonFile, err := ioutil.ReadFile(assetPaths.GameSettings)
	if err != nil {
		log.Fatalf("game_settings.json の読み込みエラー: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(jsonFile, &cfg); err != nil {
		log.Fatalf("game_settings.json のアンマーシャルエラー: %v", err)
	}
	cfg.AssetPaths = assetPaths

	// 3. リソースローダーの初期化
	audioContext := audio.NewContext(44100)
	loader := NewLoader(audioContext, &assetPaths)

	// 4. フォントの読み込み
	normalFont, modalButtonFont, messageWindowFont, err := LoadFonts(loader, &assetPaths, &cfg)
	if err != nil {
		log.Fatalf("フォントの読み込みに失敗しました: %v", err)
	}

	// 5. MessageManagerの初期化
	messageBytes := loader.LoadRaw(RawMessagesJSON).Data
	messageManager, err := NewMessageManager(messageBytes)
	if err != nil {
		log.Fatalf("MessageManagerの初期化に失敗しました: %v", err)
	}

	// 6. GameDataManagerの初期化
	gameDataManager, err := NewGameDataManager(normalFont, messageManager)
	if err != nil {
		log.Fatalf("GameDataManagerの初期化に失敗しました: %v", err)
	}

	// 7. 各種静的データの読み込みとGameDataManagerへの格納
	formulas, err := LoadFormulas(loader)
	if err != nil {
		log.Fatalf("計算式の読み込みに失敗しました: %v", err)
	}
	gameDataManager.Formulas = formulas

	if err := LoadAllStaticGameData(loader, gameDataManager); err != nil {
		log.Fatalf("静的ゲームデータ（パーツ、メダル）の読み込みに失敗しました: %v", err)
	}

	// 8. メダロットの初期構成データの読み込み
	medarotLoadouts, err := LoadMedarotLoadouts(loader)
	if err != nil {
		log.Fatalf("メダロットロードアウトの読み込みに失敗しました: %v", err)
	}
	gameData := &core.GameData{
		Medarots: medarotLoadouts,
	}

	// 9. すべての初期化済みデータを構造体にまとめて返す
	return &InitialGameData{
		Config:            cfg,
		GameDataManager:   gameDataManager,
		GameData:          gameData,
		NormalFont:        normalFont,
		ModalButtonFont:   modalButtonFont,
		MessageWindowFont: messageWindowFont,
		Loader:            loader,
	}
}