package data

import (
	"math/rand"

	"medarot-ebiten/core"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

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
}
