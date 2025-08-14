package data

import (
	"image/color"
	"math/rand"

	"medarot-ebiten/core"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	resource "github.com/quasilyte/ebitengine-resource"
)

// SharedResources はゲーム全体で共有されるリソースを保持します。
// 【変更点】リソースローダーへの参照を追加しました。
type SharedResources struct {
	GameData          *core.GameData
	Config            Config
	Font              text.Face
	ModalButtonFont   text.Face
	MessageWindowFont text.Face
	GameDataManager   *GameDataManager
	ButtonImage       *widget.ButtonImage
	Rand              *rand.Rand
	BattleLogger      BattleLogger
	Loader            *resource.Loader // 追加
}

// NewSharedResources はSharedResourcesを初期化して返します。
// 【変更点】引数にリソースローダーを追加しました。
func NewSharedResources(
	gameData *core.GameData,
	config Config,
	normalFont text.Face,
	modalButtonFont text.Face,
	messageWindowFont text.Face,
	gameDataManager *GameDataManager,
	loader *resource.Loader, // 追加
) *SharedResources {
	// ボタン用のシンプルな画像を作成
	buttonImage := ebiten.NewImage(core.ButtonImageWidth, core.ButtonImageHeight)
	buttonImage.Fill(color.RGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xFF}) // 暗い灰色

	return &SharedResources{
		GameData:          gameData,
		Config:            config,
		Font:              normalFont,
		ModalButtonFont:   modalButtonFont,
		MessageWindowFont: messageWindowFont,
		GameDataManager:   gameDataManager,
		ButtonImage: &widget.ButtonImage{
			Idle:    image.NewNineSliceSimple(buttonImage, core.ButtonImageBorder, core.ButtonImageBorder),
			Hover:   image.NewNineSliceSimple(buttonImage, core.ButtonImageBorder, core.ButtonImageBorder),
			Pressed: image.NewNineSliceSimple(buttonImage, core.ButtonImageBorder, core.ButtonImageBorder),
		},
		Rand:         rand.New(rand.NewSource(config.Game.RandomSeed)),
		BattleLogger: NewBattleLogger(gameDataManager),
		Loader:       loader, // 追加
	}
}