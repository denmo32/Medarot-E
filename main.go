package main

import (
	"image/color"
	"log"
	"math/rand"
	"os"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	resource "github.com/quasilyte/ebitengine-resource"
)

// グローバルなリソースローダーを宣言
var r *resource.Loader

// main関数がエントリーポイントであることは変わりません
func main() {

	config := LoadConfig()
	wd, err := os.Getwd()
	if err != nil {
		// ここは標準のlogをそのまま使います
		log.Printf("カレントワーキングディレクトリの取得に失敗しました: %v", err)
	} else {
		log.Printf("カレントワーキングディレクトリ: %s", wd)
	}

	// Initialize audio context for the resource loader
	audioContext := audio.NewContext(44100)
	initResources(audioContext, &config.AssetPaths) // initResources は r を初期化する

	fontFace, err := LoadFont(FontMPLUS1pRegular) // resource.FontID へのキャストは不要
	if err != nil {
		log.Fatalf("フォントの読み込みに失敗しました: %v", err)
	}

	gameDataManager, err := NewGameDataManager(fontFace, &config.AssetPaths, r)
	if err != nil {
		log.Fatalf("GameDataManagerの初期化に失敗しました: %v", err)
	}

	formulas, err := LoadFormulas()
	if err != nil {
		log.Fatalf("Failed to load formulas: %v", err)
	}
	gameDataManager.Formulas = formulas

	if err := LoadAllStaticGameData(gameDataManager); err != nil {
		log.Fatalf("静的ゲームデータの読み込みに失敗しました: %v", err)
	}

	medarotLoadouts, err := LoadMedarotLoadouts()
	if err != nil {
		log.Fatalf("メダロットロードアウトの読み込みに失敗しました: %v", err)
	}

	// bamennを使ったシーンマネージャをセットアップします
	// 共有リソースを作成
	// ボタン用のシンプルな画像を作成
	buttonImage := ebiten.NewImage(30, 30)                           // 適当なサイズ
	buttonImage.Fill(color.RGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xFF}) // 暗い灰色

	// シーンマネージャを作成
	manager := NewSceneManager(&SharedResources{
		GameData: &GameData{
			Medarots: medarotLoadouts,
		},
		Config:          config,
		Font:            fontFace,
		GameDataManager: gameDataManager,
		ButtonImage: &widget.ButtonImage{
			Idle:    image.NewNineSliceSimple(buttonImage, 10, 10),
			Hover:   image.NewNineSliceSimple(buttonImage, 10, 10),
			Pressed: image.NewNineSliceSimple(buttonImage, 10, 10),
		},
		Rand:         rand.New(rand.NewSource(config.Game.RandomSeed)), // 追加
		BattleLogger: NewBattleLogger(gameDataManager),                 // 追加
	})

	// Ebitenのゲームを実行します。渡すのはbamennのシーケンスです。
	ebiten.SetWindowSize(config.UI.Screen.Width, config.UI.Screen.Height)
	ebiten.SetWindowTitle("Ebiten Medarot Battle (bamenn)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(manager.sequence); err != nil {
		log.Fatal(err)
	}
}
