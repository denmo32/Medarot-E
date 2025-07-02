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

	// Load static definitions into GameDataManager
	if err := LoadAllStaticGameData(); err != nil {
		log.Fatalf("Failed to load static game data: %v", err)
	}

	// Load medarot loadouts
	medarotLoadouts, err := LoadMedarotLoadouts("data/medarots.csv")
	if err != nil {
		log.Fatalf("Failed to load medarot loadouts: %v", err)
	}

	// Prepare GameData struct (now only contains Medarots, or could be passed directly)
	gameData := &GameData{
		Medarots: medarotLoadouts,
	}
	// if gameData == nil { // This check might be less relevant if GameData is simplified
	// 	log.Fatal("Game data is nil after loading.")
	// }

	config := LoadConfig()

	// NewGame now expects a *GameData that might only contain Medarots,
	// or its signature could be changed to accept []MedarotData directly.
	// For now, assuming NewGame still takes *GameData.
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
