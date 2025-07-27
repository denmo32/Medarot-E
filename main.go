package main

import (
	"image/color"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

var globalRand *rand.Rand

// main関数がエントリーポイントであることは変わりません
func main() {
	globalRand = rand.New(rand.NewSource(time.Now().UnixNano()))

	wd, err := os.Getwd()
	if err != nil {
		// ここは標準のlogをそのまま使います
		log.Printf("カレントワーキングディレクトリの取得に失敗しました: %v", err)
	} else {
		log.Printf("カレントワーキングディレクトリ: %s", wd)
	}

	// Initialize audio context for the resource loader
	audioContext := audio.NewContext(44100)
	initResources(audioContext)

	fontFace, err := LoadFont(FontMPLUS1pRegular)
	if err != nil {
		log.Fatalf("フォントの読み込みに失敗しました: %v", err)
	}

	gameDataManager, err := NewGameDataManager(fontFace)
	if err != nil {
		log.Fatalf("GameDataManagerの初期化に失敗しました: %v", err)
	}

	if err := LoadAllStaticGameData(gameDataManager); err != nil {
		log.Fatalf("静的ゲームデータの読み込みに失敗しました: %v", err)
	}

	medarotLoadouts, err := LoadMedarotLoadouts()
	if err != nil {
		log.Fatalf("メダロットロードアウトの読み込みに失敗しました: %v", err)
	}

	gameData := &GameData{
		Medarots: medarotLoadouts,
	}

	config := LoadConfig()
	SetupFormulaManager(&config)

	// bamennを使ったシーンマネージャをセットアップします
	// 共有リソースを作成
	// ボタン用のシンプルな画像を作成
	buttonImage := ebiten.NewImage(30, 30)                           // 適当なサイズ
	buttonImage.Fill(color.RGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xFF}) // 暗い灰色

	res := &SharedResources{
		GameData:        gameData,
		Config:          config,
		Font:            fontFace,
		GameDataManager: gameDataManager,
		ButtonImage: &widget.ButtonImage{
			Idle:    image.NewNineSliceSimple(buttonImage, 10, 10),
			Hover:   image.NewNineSliceSimple(buttonImage, 10, 10),
			Pressed: image.NewNineSliceSimple(buttonImage, 10, 10),
		},
	}

	// シーンマネージャを作成
	manager := NewSceneManager(res)

	// Ebitenのゲームを実行します。渡すのはbamennのシーケンスです。
	ebiten.SetWindowSize(config.UI.Screen.Width, config.UI.Screen.Height)
	ebiten.SetWindowTitle("Ebiten Medarot Battle (bamenn)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(manager.sequence); err != nil {
		log.Fatal(err)
	}
}
