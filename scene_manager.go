package main

import (
	"log"

	"github.com/noppikinatta/bamenn"
)

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

func (m *SceneManager) newTestUIScene() (Scene, error) {
	return NewTestUIScene(m.resources, m), nil
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

func (m *SceneManager) GoToTestUIScene() {
	scene, err := m.newTestUIScene()
	if err != nil {
		log.Printf("テストUIシーンへの切り替えに失敗しました: %v", err)
		return
	}
	m.sequence.Switch(scene)
}
