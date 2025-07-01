package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Game はシーンマネージャーとして機能し、ゲーム全体の流れを管理します
type Game struct {
	currentScene Scene
	currentType  SceneType
	resources    SharedResources
}

// NewGame はゲーム全体を初期化し、最初のシーン（タイトル）を設定します
func NewGame(gameData *GameData, config Config, font text.Face) *Game {
	g := &Game{
		resources: SharedResources{
			GameData: gameData,
			Config:   config,
			Font:     font,
		},
	}

	g.currentType = SceneTypeTitle
	g.currentScene = NewTitleScene(&g.resources)

	log.Println("Game initialized. Starting with Title Scene.")
	return g
}

// Update は現在のシーンのUpdateを呼び出し、シーン遷移を処理します
func (g *Game) Update() error {
	nextScene, err := g.currentScene.Update()
	if err != nil {
		return err
	}

	if nextScene != g.currentType {
		g.changeScene(nextScene)
	}

	return nil
}

// changeScene は指定された新しいシーンに切り替えます
func (g *Game) changeScene(nextType SceneType) {
	log.Printf("Changing scene from %v to %v", g.currentType, nextType)
	g.currentType = nextType

	switch g.currentType {
	case SceneTypeTitle:
		g.currentScene = NewTitleScene(&g.resources)
	case SceneTypeBattle:
		g.currentScene = NewBattleScene(&g.resources)
	case SceneTypeCustomize:
		// 今回は「未実装」メッセージを表示するシーンに遷移
		g.currentScene = NewPlaceholderScene(&g.resources, "Customize Scene is under construction.")
	}
}

// Draw は現在のシーンのDrawを呼び出します
func (g *Game) Draw(screen *ebiten.Image) {
	if g.currentScene != nil {
		g.currentScene.Draw(screen)
	}
}

// Layout はEbitenのレイアウト計算を行います
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.resources.Config.UI.Screen.Width, g.resources.Config.UI.Screen.Height
}
