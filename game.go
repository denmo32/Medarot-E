package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/yohamta/donburi"
)

type Game struct {
	World               donburi.World
	GameData            *GameData
	Config              Config
	MplusFont           text.Face
	TickCount           int
	DebugMode           bool
	State               GameState
	PlayerTeam          TeamID
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

func NewGame(gameData *GameData, config Config, font text.Face) *Game {
	world := donburi.NewWorld()

	g := &Game{
		World:              world,
		GameData:           gameData,
		Config:             config,
		MplusFont:          font,
		TickCount:          0,
		DebugMode:          true,
		State:              StatePlaying,
		PlayerTeam:         Team1,
		actionQueue:        make([]*donburi.Entry, 0),
		playerMedarotToAct: nil,
		attackingEntity:    nil,
		targetedEntity:     nil,
		currentTarget:      nil,
	}

	CreateMedarotEntities(g.World, g.GameData, g.PlayerTeam)

	g.ui = NewUI(g)
	log.Println("Game initialized successfully.")
	return g
}

func (g *Game) Update() error {
	g.ui.ebitenui.Update()
	if g.restartRequested {
		g.restartRequested = false
	}

	switch g.State {
	case StatePlaying:
		g.TickCount++
		SystemUpdateProgress(g)
		SystemProcessReadyQueue(g)
		SystemProcessIdleMedarots(g)
		SystemCheckGameEnd(g)

		updateAllInfoPanels(g)
		if g.ui.battlefieldWidget != nil {
			g.ui.battlefieldWidget.UpdatePositions()
		}

	case StatePlayerActionSelect:
		if g.ui.battlefieldWidget != nil {
			g.ui.battlefieldWidget.UpdatePositions()
		}
		if g.ui.actionModal == nil && g.playerMedarotToAct != nil {
			g.ui.ShowActionModal(g, g.playerMedarotToAct)
		}

	case StateMessage:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.ui.HideMessageWindow()
			if g.postMessageCallback != nil {
				g.postMessageCallback()
				g.postMessageCallback = nil
			}
			g.State = StatePlaying
			SystemProcessIdleMedarots(g)
		}

	case StateGameOver:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			// リスタート処理
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(g.Config.UI.Colors.Background)
	g.ui.ebitenui.Draw(screen)
	bf := g.ui.battlefieldWidget
	if bf != nil {
		bf.DrawBackground(screen)
		bf.DrawIcons(screen)

		var indicatorTarget *donburi.Entry
		if g.State == StatePlayerActionSelect && g.currentTarget != nil {
			indicatorTarget = g.currentTarget
		} else if g.State == StateMessage && g.targetedEntity != nil {
			indicatorTarget = g.targetedEntity
		}

		if indicatorTarget != nil {
			bf.DrawTargetIndicator(screen, indicatorTarget)
		}

		bf.DrawDebug(screen)
	}
	if g.DebugMode {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\nState: %s",
			ebiten.ActualTPS(), ebiten.ActualFPS(), g.State))
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.Config.UI.Screen.Width, g.Config.UI.Screen.Height
}

func (g *Game) enqueueMessage(msg string, callback func()) {
	g.message = msg
	g.postMessageCallback = callback
	g.State = StateMessage
	g.ui.ShowMessageWindow(g)
}
