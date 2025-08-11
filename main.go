package main

import (
	"log"
	"os"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/scene"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

// main関数がエントリーポイントであることは変わりません
func main() {

	config := data.LoadConfig()
	wd, err := os.Getwd()
	if err != nil {
		// ここは標準のlogをそのまま使います
		log.Printf("カレントワーキングディレクトリの取得に失敗しました: %v", err)
	} else {
		log.Printf("カレントワーキングディレクトリ: %s", wd)
	}

	// Initialize audio context for the resource loader
	audioContext := audio.NewContext(44100)
	data.InitResources(audioContext, &config.AssetPaths) // data.InitResources は r を初期化する

	normalFont, modalButtonFont, messageWindowFont, err := data.LoadFonts(&config.AssetPaths, &config)
	if err != nil {
		log.Fatalf("フォントの読み込みに失敗しました: %v", err)
	}

	gameDataManager, err := data.NewGameDataManager(normalFont, &config.AssetPaths)
	if err != nil {
		log.Fatalf("GameDataManagerの初期化に失敗しました: %v", err)
	}

	formulas, err := data.LoadFormulas()
	if err != nil {
		log.Fatalf("Failed to load formulas: %v", err)
	}
	gameDataManager.Formulas = formulas

	if err := data.LoadAllStaticGameData(gameDataManager); err != nil {
		log.Fatalf("静的ゲームデータの読み込みに失敗しました: %v", err)
	}

	medarotLoadouts, err := data.LoadMedarotLoadouts()
	if err != nil {
		log.Fatalf("メダロットロードアウトの読み込みに失敗しました: %v", err)
	}

	// 共有リソースを作成
	sharedResources := data.NewSharedResources(
		&core.GameData{Medarots: medarotLoadouts},
		config,
		normalFont,
		modalButtonFont,
		messageWindowFont,
		gameDataManager,
	)

	// シーンマネージャを作成
	manager := scene.NewSceneManager(sharedResources)

	// Ebitenのゲームを実行します。渡すのはbamennのシーケンスです。
	ebiten.SetWindowSize(config.UI.Screen.Width, config.UI.Screen.Height)
	ebiten.SetWindowTitle("Ebiten Medarot Battle (bamenn)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(manager.Sequence); err != nil {
		log.Fatal(err)
	}
}
