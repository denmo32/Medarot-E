package main

import (
	"image/color"
	"log"
	"math/rand"
	"os"
	"time"

	"medarot-ebiten/internal/battle"
	"medarot-ebiten/internal/game"
	"medarot-ebiten/internal/scene"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/noppikinatta/bamenn"
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
	game.InitResources(audioContext)

	fontFace, err := game.LoadFont(game.FontMPLUS1pRegular)
	if err != nil {
		log.Fatalf("フォントの読み込みに失敗しました: %v", err)
	}

	gameDataManager, err := game.NewGameDataManager(fontFace)
	if err != nil {
		log.Fatalf("GameDataManagerの初期化に失敗しました: %v", err)
	}

	if err := game.LoadAllStaticGameData(gameDataManager); err != nil {
		log.Fatalf("静的ゲームデータの読み込みに失敗しました: %v", err)
	}

	medarotLoadouts, err := game.LoadMedarotLoadouts()
	if err != nil {
		log.Fatalf("メダロットロードアウトの読み込みに失敗しました: %v", err)
	}

	gameData := &game.GameData{
		Medarots: medarotLoadouts,
	}

	config := battle.LoadConfig()
	formulas, err := battle.LoadFormulas()
	if err != nil {
		log.Fatalf("アクション計算式の読み込みに失敗しました: %v", err)
	}
	config.Balance.Formulas = formulas
	battle.SetupFormulaManager(&config)

	uiConfig := ui.LoadUIConfig()

	// bamennを使ったシーンマネージャをセットアップします
	// 共有リソースを作成
	// ボタン用のシンプルな画像を作成
	buttonImage := ebiten.NewImage(30, 30)                           // 適当なサイズ
	buttonImage.Fill(color.RGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xFF}) // 暗い灰色

	res := &scene.SharedResources{
		GameData:        gameData,
		Config:          config,
		UIConfig:        uiConfig,
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
	ebiten.SetWindowSize(uiConfig.Screen.Width, uiConfig.Screen.Height)
	ebiten.SetWindowTitle("Ebiten Medarot Battle (bamenn)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(manager.sequence); err != nil {
		log.Fatal(err)
	}
}

// SceneManagerはbamennのシーケンスと共有リソースを管理します
type SceneManager struct {
	sequence  *bamenn.Sequence
	resources *scene.SharedResources
}

// NewSceneManagerは新しいシーンマネージャを作成し、初期シーンを設定します
func NewSceneManager(res *scene.SharedResources) *SceneManager {
	m := &SceneManager{
		resources: res,
	}

	// 最初のシーンを生成
	initialScene, err := m.newTitleScene()
	if err != nil {
		log.Fatalf("初期シーンの作成に失敗しました: %v", err)
	}

	// bamennのシーケンスを作成し、最初のシーンを渡します
	seq := bamenn.NewSequence(initialScene)
	m.sequence = seq

	return m
}

// 各シーンを生成するファクトリ関数です
// これにより、循環参照することなく、各シーンからマネージャ経由で他のシーンに遷移できます

func (m *SceneManager) newTitleScene() (scene.Scene, error) {
	return scene.NewTitleScene(m.resources, m), nil
}

func (m *SceneManager) newBattleScene() (scene.Scene, error) {
	return scene.NewBattleScene(m.resources, m), nil
}

func (m *SceneManager) newCustomizeScene() (scene.Scene, error) {
	return scene.NewCustomizeScene(m.resources, m), nil
}

func (m *SceneManager) newTestUIScene() (scene.Scene, error) {
	return scene.NewTestUIScene(m.resources, m), nil
}

// GoTo... メソッド群は、各シーンから呼び出され、指定されたシーンに遷移させます

func (m *SceneManager) GoToTitleScene() {
	s, err := m.newTitleScene()
	if err != nil {
		log.Printf("タイトルシーンへの切り替えに失敗しました: %v", err)
		return
	}
	m.sequence.Switch(s)
}

func (m *SceneManager) GoToBattleScene() {
	s, err := m.newBattleScene()
	if err != nil {
		log.Printf("バトルシーンへの切り替えに失敗しました: %v", err)
		return
	}
	m.sequence.Switch(s)
}

func (m *SceneManager) GoToCustomizeScene() {
	s, err := m.newCustomizeScene()
	if err != nil {
		log.Printf("カスタマイズシーンへの切り替えに失敗しました: %v", err)
		return
	}
	m.sequence.Switch(s)
}

func (m *SceneManager) GoToTestUIScene() {
	s, err := m.newTestUIScene()
	if err != nil {
		log.Printf("テストUIシーンへの切り替えに失敗しました: %v", err)
		return
	}
	m.sequence.Switch(s)
}
