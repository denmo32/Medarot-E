package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Game struct {
	GameData              *GameData
	Config                Config
	MplusFont             text.Face
	TickCount             int
	DebugMode             bool
	State                 GameState
	PlayerTeam            TeamID
	actionQueue           []*Medarot
	sortedMedarotsForDraw []*Medarot
	Medarots              []*Medarot
	team1Leader           *Medarot
	team2Leader           *Medarot
	ui                    *UI
	message               string
	postMessageCallback   func()
	winner                TeamID
	restartRequested      bool
	playerMedarotToAct    *Medarot
}

func NewGame(gameData *GameData, config Config, font text.Face) *Game {
	g := &Game{
		GameData:              gameData,
		Config:                config,
		MplusFont:             font,
		TickCount:             0,
		DebugMode:             true,
		State:                 StatePlaying,
		PlayerTeam:            Team1,
		actionQueue:           make([]*Medarot, 0),
		sortedMedarotsForDraw: make([]*Medarot, 0),
		playerMedarotToAct:    nil,
	}
	g.Medarots = InitializeAllMedarots(g.GameData)
	if len(g.Medarots) == 0 {
		log.Fatal("No medarots were initialized.")
	}
	g.initializeMedarotLists()
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
		// ロジックをSystem関数に委譲
		SystemUpdateProgress(g.Medarots, g)
		SystemProcessReadyQueue(g)
		SystemProcessIdleMedarots(g)
		SystemCheckGameEnd(g)

		// UI更新
		updateAllInfoPanels(g)
		if g.ui.battlefieldWidget != nil {
			g.ui.battlefieldWidget.UpdatePositions()
		}

	case StatePlayerActionSelect:
		// プレイヤーの行動選択モーダル表示
		if g.ui.actionModal == nil && g.playerMedarotToAct != nil {
			g.ui.ShowActionModal(g, g.playerMedarotToAct)
		}

	case StateMessage, StateGameOver:
		// メッセージウィンドウまたはゲームオーバー画面での入力待ち
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if g.State == StateMessage {
				g.ui.HideMessageWindow()
				if g.postMessageCallback != nil {
					g.postMessageCallback()
					g.postMessageCallback = nil
				}
				g.State = StatePlaying
				// メッセージ表示後に待機中のメダロットがいれば即座に処理
				SystemProcessIdleMedarots(g)
			} else if g.State == StateGameOver {
				// (将来的にリスタート処理などを追加)
			}
		}
	}
	return nil
}

// getTargetCandidates は ai.go からも参照されるため、ここに残す
func (g *Game) getTargetCandidates(actingMedarot *Medarot) []*Medarot {
	return getTargetCandidates(g, actingMedarot)
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

// --- 以下、変更なしのヘルパー関数 ---

func (g *Game) initializeMedarotLists() {
	g.sortedMedarotsForDraw = make([]*Medarot, len(g.Medarots))
	copy(g.sortedMedarotsForDraw, g.Medarots)
	sort.Slice(g.sortedMedarotsForDraw, func(i, j int) bool {
		if g.sortedMedarotsForDraw[i].Team != g.sortedMedarotsForDraw[j].Team {
			return g.sortedMedarotsForDraw[i].Team < g.sortedMedarotsForDraw[j].Team
		}
		return g.sortedMedarotsForDraw[i].DrawIndex < g.sortedMedarotsForDraw[j].DrawIndex
	})
	for _, m := range g.Medarots {
		if m.IsLeader {
			if m.Team == Team1 {
				g.team1Leader = m
			} else {
				g.team2Leader = m
			}
		}
	}
}

func (g *Game) findNextIdlePlayerMedarot() *Medarot {
	for _, m := range g.Medarots {
		if m.Team == g.PlayerTeam && m.State == StateIdle {
			return m
		}
	}
	return nil
}

func (g *Game) enqueueMessage(msg string, callback func()) {
	g.message = msg
	g.postMessageCallback = callback
	g.State = StateMessage
	g.ui.ShowMessageWindow(g)
}