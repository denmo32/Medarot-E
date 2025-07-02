package main

import (
	"bytes"
	_ "embed"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed MPLUS1p-Regular.ttf
var mplusFontData []byte

// loadFont はEbitenUIが要求する text.Face を返します。
func loadFont() (text.Face, error) {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(mplusFontData))
	if err != nil {
		return nil, err
	}

	face := &text.GoTextFace{
		Source: s,
		Size:   12,
	}
	log.Println("カスタムフォント（text/v2）の読み込みに成功しました。")
	return face, nil
}

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

	// 静的ゲームデータをGameDataManagerに読み込みます。
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
