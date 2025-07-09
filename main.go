package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/bamenn"
)

// main関数がエントリーポイントであることは変わりません
func main() {
	rand.Seed(time.Now().UnixNano())

	wd, err := os.Getwd()
	if err != nil {
		// ここは標準のlogをそのまま使います
		log.Printf("カレントワーキングディレクトリの取得に失敗しました: %v", err)
	} else {
		log.Printf("カレントワーキングディレクトリ: %s", wd)
	}

	fontFace, err := loadFont()
	if err != nil {
		log.Fatalf("フォントの読み込みに失敗しました: %v", err)
	}

	GlobalGameDataManager, err = NewGameDataManager("data", fontFace)
	if err != nil {
		log.Fatalf("GameDataManagerの初期化に失敗しました: %v", err)
	}

	if err := LoadAllStaticGameData(); err != nil {
		log.Fatalf("静的ゲームデータの読み込みに失敗しました: %v", err)
	}

	medarotLoadouts, err := LoadMedarotLoadouts("data/medarots.csv")
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
	res := &SharedResources{
		GameData: gameData,
		Config:   config,
		Font:     fontFace,
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

// SceneManagerはbamennのシーケンスと共有リソースを管理します
type SceneManager struct {
	sequence  *bamenn.Sequence
	resources *SharedResources
}

// NewSceneManagerは新しいシーンマネージャを作成し、初期シーンを設定します
func NewSceneManager(res *SharedResources) *SceneManager {
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

func (m *SceneManager) newTitleScene() (Scene, error) {
	return NewTitleScene(m.resources, m), nil
}

func (m *SceneManager) newBattleScene() (Scene, error) {
	return NewBattleScene(m.resources, m), nil
}

func (m *SceneManager) newCustomizeScene() (Scene, error) {
	return NewCustomizeScene(m.resources, m), nil
}

// GoTo... メソッド群は、各シーンから呼び出され、指定されたシーンに遷移させます

func (m *SceneManager) GoToTitleScene() {
	scene, err := m.newTitleScene()
	if err != nil {
		log.Printf("タイトルシーンへの切り替えに失敗しました: %v", err)
		return
	}
	m.sequence.Switch(scene)
}

func (m *SceneManager) GoToBattleScene() {
	scene, err := m.newBattleScene()
	if err != nil {
		log.Printf("バトルシーンへの切り替えに失敗しました: %v", err)
		return
	}
	m.sequence.Switch(scene)
}

func (m *SceneManager) GoToCustomizeScene() {
	scene, err := m.newCustomizeScene()
	if err != nil {
		log.Printf("カスタマイズシーンへの切り替えに失敗しました: %v", err)
		return
	}
	m.sequence.Switch(scene)
}
