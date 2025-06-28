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
	actionQueue         []*donburi.Entry // Medarot* から donburi.Entry* へ変更
	ui                  *UI
	message             string
	postMessageCallback func()
	winner              TeamID
	restartRequested    bool
	playerMedarotToAct  *donburi.Entry // Medarot* から donburi.Entry* へ変更
}

func NewGame(gameData *GameData, config Config, font text.Face) *Game {
	// 修正: donburi.New() を donburi.NewWorld() に変更
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
		// 各Systemを呼び出す
		SystemUpdateProgress(g)
		SystemProcessReadyQueue(g)
		SystemProcessIdleMedarots(g)
		SystemCheckGameEnd(g)

		// UI更新
		updateAllInfoPanels(g)
		if g.ui.battlefieldWidget != nil {
			g.ui.battlefieldWidget.UpdatePositions()
		}

	case StatePlayerActionSelect:
		if g.ui.actionModal == nil && g.playerMedarotToAct != nil {
			g.ui.ShowActionModal(g, g.playerMedarotToAct)
		}

	case StateMessage, StateGameOver:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if g.State == StateMessage {
				g.ui.HideMessageWindow()
				if g.postMessageCallback != nil {
					g.postMessageCallback()
					g.postMessageCallback = nil
				}
				g.State = StatePlaying
				SystemProcessIdleMedarots(g)
			} else if g.State == StateGameOver {
				// リスタート処理など
			}
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
