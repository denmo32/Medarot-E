package main

import (
	"log"
	"os"

	"medarot-ebiten/data"
	"medarot-ebiten/scene"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// カレントワーキングディレクトリのログ出力
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("カレントワーキングディレクトリの取得に失敗しました: %v", err)
	} else {
		log.Printf("カレントワーキングディレクトリ: %s", wd)
	}

	// 1. すべての初期データを一括で読み込む
	// この関数は、設定、リソース、静的データを読み込み、
	// 関連するマネージャを初期化して返します。
	// これにより、main関数の関心事は「ゲームの起動と実行」に集中します。
	initialData := data.LoadInitialGameData()
	if initialData == nil {
		log.Fatal("ゲームデータの初期化に失敗しました。")
	}

	// 2. 共有リソースを作成
	// LoadInitialGameDataから返された初期化済みデータを使用して、
	// シーン間で共有されるリソースバンドルを生成します。
	sharedResources := data.NewSharedResources(
		initialData.GameData,
		initialData.Config,
		initialData.NormalFont,
		initialData.ModalButtonFont,
		initialData.MessageWindowFont,
		initialData.GameDataManager,
	)

	// 3. シーンマネージャを作成
	manager := scene.NewSceneManager(sharedResources)

	// 4. Ebitenゲームループを実行
	ebiten.SetWindowSize(initialData.Config.UI.Screen.Width, initialData.Config.UI.Screen.Height)
	ebiten.SetWindowTitle("Ebiten Medarot Battle (bamenn)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(manager.Sequence); err != nil {
		log.Fatal(err)
	}
}