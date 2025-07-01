package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
)

// BattleScene は戦闘シーンのすべてを管理します
type BattleScene struct {
	resources           *SharedResources
	world               donburi.World
	tickCount           int
	debugMode           bool
	state               GameState
	playerTeam          TeamID
	actionQueue         []*donburi.Entry
	ui                  *UI
	message             string
	postMessageCallback func()
	winner              TeamID
	restartRequested    bool
	playerMedarotToAct  *donburi.Entry
	currentTarget       *donburi.Entry
	attackingEntity     *donburi.Entry
	targetedEntity      *donburi.Entry
}

// NewBattleScene は新しい戦闘シーンを初期化します
func NewBattleScene(res *SharedResources) *BattleScene {
	world := donburi.NewWorld()

	bs := &BattleScene{
		resources:          res,
		world:              world,
		tickCount:          0,
		debugMode:          true,
		state:              StatePlaying,
		playerTeam:         Team1,
		actionQueue:        make([]*donburi.Entry, 0),
		playerMedarotToAct: nil,
		attackingEntity:    nil,
		targetedEntity:     nil,
		currentTarget:      nil,
	}

	CreateMedarotEntities(bs.world, bs.resources.GameData, bs.playerTeam)
	bs.ui = NewUI(bs)

	return bs
}

// Update は戦闘シーンのロジックを更新します
func (bs *BattleScene) Update() (SceneType, error) {
	bs.ui.ebitenui.Update()

	switch bs.state {
	case StatePlaying:
		bs.tickCount++
		SystemUpdateProgress(bs)
		SystemProcessReadyQueue(bs)
		SystemProcessIdleMedarots(bs)
		SystemCheckGameEnd(bs)
		updateAllInfoPanels(bs)
		if bs.ui.battlefieldWidget != nil {
			bs.ui.battlefieldWidget.UpdatePositions()
		}
	case StatePlayerActionSelect:
		if bs.ui.battlefieldWidget != nil {
			bs.ui.battlefieldWidget.UpdatePositions()
		}
		if bs.ui.actionModal == nil && bs.playerMedarotToAct != nil {
			bs.ui.ShowActionModal(bs, bs.playerMedarotToAct)
		}
	case StateMessage:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			bs.ui.HideMessageWindow()
			if bs.postMessageCallback != nil {
				bs.postMessageCallback()
				bs.postMessageCallback = nil
			}
			bs.state = StatePlaying
			SystemProcessIdleMedarots(bs)
		}
	case StateGameOver:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			return SceneTypeTitle, nil // ゲームオーバー時にクリックでタイトルに戻る
		}
	}
	return SceneTypeBattle, nil
}

// Draw は戦闘シーンを描画します
func (bs *BattleScene) Draw(screen *ebiten.Image) {
	screen.Fill(bs.resources.Config.UI.Colors.Background)
	bs.ui.ebitenui.Draw(screen)

	bf := bs.ui.battlefieldWidget
	if bf != nil {
		bf.DrawBackground(screen)
		bf.DrawIcons(screen)

		var indicatorTarget *donburi.Entry
		if bs.state == StatePlayerActionSelect && bs.currentTarget != nil {
			indicatorTarget = bs.currentTarget
		} else if bs.state == StateMessage && bs.targetedEntity != nil {
			indicatorTarget = bs.targetedEntity
		}

		if indicatorTarget != nil {
			bf.DrawTargetIndicator(screen, indicatorTarget)
		}
		bf.DrawDebug(screen)
	}
	if bs.debugMode {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\nState: %s",
			ebiten.ActualTPS(), ebiten.ActualFPS(), bs.state))
	}
}

// enqueueMessage はバトルシーン内でのメッセージ表示を扱います
func (bs *BattleScene) enqueueMessage(msg string, callback func()) {
	bs.message = msg
	bs.postMessageCallback = callback
	bs.state = StateMessage
	bs.ui.ShowMessageWindow(bs)
}
