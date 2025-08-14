package main

import (
	"log"

	"medarot-ebiten/data"
	"medarot-ebiten/scene"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// ... (ログ出力部分は変更なし) ...

	// 1. すべての初期データを一括で読み込む
	initialData := data.LoadInitialGameData()
	if initialData == nil {
		log.Fatal("ゲームデータの初期化に失敗しました。")
	}

	// 2. 共有リソースを作成
	// 【変更点】`initialData`からローダーを取り出し、`NewSharedResources`に渡します。
	sharedResources := data.NewSharedResources(
		initialData.GameData,
		initialData.Config,
		initialData.NormalFont,
		initialData.ModalButtonFont,
		initialData.MessageWindowFont,
		initialData.GameDataManager,
		initialData.Loader, // 追加
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
