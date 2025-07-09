package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// SharedResources はシーン間で共有されるリソースを保持します
// (旧game.goから移動)
type SharedResources struct {
	GameData *GameData
	Config   Config
	Font     text.Face
}

// Sceneは、bamennで管理される全てのシーンが満たすべきインターフェースです。
// ebiten.Gameを埋め込むことで、Update/Draw/Layoutメソッドを持つことが保証されます。
type Scene interface {
	ebiten.Game
}