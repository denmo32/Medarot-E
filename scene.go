package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// SceneType はゲーム全体のシーンの種類を定義します
type SceneType int

const (
	SceneTypeTitle     SceneType = iota // タイトルシーン
	SceneTypeBattle                     // バトルシーン
	SceneTypeCustomize                  // カスタマイズシーン
)

// Scene は各シーンが実装すべきメソッドのインターフェースです
type Scene interface {
	Update() (SceneType, error)
	Draw(screen *ebiten.Image)
}

// SharedResources はシーン間で共有されるリソースを保持します
type SharedResources struct {
	GameData *GameData
	Config   Config
	Font     text.Face
}
