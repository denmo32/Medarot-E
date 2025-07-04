package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	// "github.com/hajimehoshi/ebiten/v2/text/v2"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	wd, err := os.Getwd()
	if err != nil {
		log.Printf("カレントワーキングディレクトリの取得に失敗しました: %v", err)
	} else {
		log.Printf("カレントワーキングディレクトリ: %s", wd)
	}

	fontFace, err := loadFont()
	if err != nil {
		log.Fatalf("フォントの読み込みに失敗しました: %v", err)
	}

	// GameDataManagerを初期化します。
	// これにはメッセージ定義の読み込みも含まれます。
	GlobalGameDataManager, err = NewGameDataManager("data", fontFace)
	if err != nil {
		log.Fatalf("GameDataManagerの初期化に失敗しました: %v", err)
	}

	// 静的ゲームデータ（パーツ、メダル）をGameDataManagerに読み込みます。
	if err := LoadAllStaticGameData(); err != nil {
		log.Fatalf("静的ゲームデータの読み込みに失敗しました: %v", err)
	}

	// メダロットのロードアウトを読み込みます。
	medarotLoadouts, err := LoadMedarotLoadouts("data/medarots.csv")
	if err != nil {
		log.Fatalf("メダロットロードアウトの読み込みに失敗しました: %v", err)
	}

	// GameData構造体を準備します。（現在はMedarotsのみを格納）
	gameData := &GameData{
		Medarots: medarotLoadouts,
	}

	config := LoadConfig()

	// NewGameは*GameData（現在はMedarotsのみを格納）を期待します。
	// 将来的には[]MedarotDataを直接受け取るようにシグネチャが変更される可能性があります。
	game := NewGame(gameData, config, fontFace)
	if game == nil {
		log.Fatal("新しいゲームインスタンスの作成に失敗しました。")
	}

	ebiten.SetWindowSize(config.UI.Screen.Width, config.UI.Screen.Height)
	ebiten.SetWindowTitle("Ebiten Medarot Battle")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
