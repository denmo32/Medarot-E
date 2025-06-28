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
	log.Println("Custom font (text/v2) loaded successfully.")
	return face, nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Failed to get current working directory: %v", err)
	} else {
		log.Printf("Current working directory: %s", wd)
	}

	fontFace, err := loadFont()
	if err != nil {
		log.Fatalf("フォントの読み込みに失敗しました: %v", err)
	}

	gameData, err := LoadAllGameData()
	if err != nil {
		log.Fatalf("Failed to load game data: %v", err)
	}
	if gameData == nil {
		log.Fatal("Game data is nil after loading.")
	}

	config := LoadConfig()

	game := NewGame(gameData, config, fontFace)
	if game == nil {
		log.Fatal("Failed to create new game instance.")
	}

	ebiten.SetWindowSize(config.UI.Screen.Width, config.UI.Screen.Height)
	ebiten.SetWindowTitle("Ebiten Medarot Battle")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
