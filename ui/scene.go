package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Sceneは、bamennで管理される全てのシーンが満たすべきインターフェースです。
// ebiten.Gameを埋め込むことで、Update/Draw/Layoutメソッドを持つことが保証されます。
type Scene interface {
	ebiten.Game
}
