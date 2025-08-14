package data

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"medarot-ebiten/core"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// InitialGameData はゲームの起動時に必要となる、初期化済みの主要なデータを保持します。
// これにより、main関数でのデータ読み込みと初期化処理が簡潔になります。
type InitialGameData struct {
	Config            Config
	GameDataManager   *GameDataManager
	GameData          *core.GameData
	NormalFont        text.Face
	ModalButtonFont   text.Face
	MessageWindowFont text.Face
}

// LoadInitialGameData は、すべての設定ファイルと静的データを読み込み、
// ゲームの起動に必要な構造体を初期化して返します。
// データ読み込みのロジックをこの関数に集約することで、main関数の責務を軽減し、
// 依存関係を明確にします。
func LoadInitialGameData() *InitialGameData {
	// 1. アセットパスの定義
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
	// `Config` 構造体は `game_settings.json` の内容を直接マッピングします。
	jsonFile, err := ioutil.ReadFile(assetPaths.GameSettings)
	if err != nil {
		log.Fatalf("game_settings.json の読み込みエラー: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(jsonFile, &cfg); err != nil {
		log.Fatalf("game_settings.json のアンマーシャルエラー: %v", err)
	}
	cfg.AssetPaths = assetPaths
	// GameConfigのRandomSeedなどは必要に応じてここで設定できます。
	// 例: cfg.Game.RandomSeed = time.Now().UnixNano()

	// 3. リソースローダーの初期化
	// このローダー `r` は、このパッケージ内の他のロード関数から参照されます。
	audioContext := audio.NewContext(44100)
	InitResources(audioContext, &cfg.AssetPaths)

	// 4. フォントの読み込み
	normalFont, modalButtonFont, messageWindowFont, err := LoadFonts(&assetPaths, &cfg)
	if err != nil {
		log.Fatalf("フォントの読み込みに失敗しました: %v", err)
	}

	// 5. MessageManagerの初期化
	// まずリソースローダーを使ってメッセージファイルのバイトデータを読み込みます。
	messageBytes := r.LoadRaw(RawMessagesJSON).Data
	// 読み込んだバイトデータを`NewMessageManager`に渡します。
	// これにより、MessageManagerはファイルI/Oから独立します。
	messageManager, err := NewMessageManager(messageBytes)
	if err != nil {
		log.Fatalf("MessageManagerの初期化に失敗しました: %v", err)
	}

	// 6. GameDataManagerの初期化
	// 初期化済みのMessageManagerを渡すことで、GameDataManagerもファイルパスに依存しなくなります。
	gameDataManager, err := NewGameDataManager(normalFont, messageManager)
	if err != nil {
		log.Fatalf("GameDataManagerの初期化に失敗しました: %v", err)
	}

	// 7. 各種静的データの読み込みとGameDataManagerへの格納
	// 計算式
	formulas, err := LoadFormulas()
	if err != nil {
		log.Fatalf("計算式の読み込みに失敗しました: %v", err)
	}
	gameDataManager.Formulas = formulas

	// パーツとメダル
	if err := LoadAllStaticGameData(gameDataManager); err != nil {
		log.Fatalf("静的ゲームデータ（パーツ、メダル）の読み込みに失敗しました: %v", err)
	}

	// 8. メダロットの初期構成データの読み込み
	medarotLoadouts, err := LoadMedarotLoadouts()
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
	}
}
