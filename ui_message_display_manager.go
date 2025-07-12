package main

import (
	"log"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// UIMessageDisplayManager はゲーム内のメッセージ表示を管理します。
type UIMessageDisplayManager struct {
	messageWindow       widget.PreferredSizeLocateableWidget
	messageQueue        []string
	currentMessageIndex int
	postMessageCallback func()
	config              *Config
	font                text.Face
	uiContainer         *widget.Container // メッセージウィンドウを追加するUIのルートコンテナ
	isWaitingForClick   bool
}

// NewUIMessageDisplayManager は新しいUIMessageDisplayManagerのインスタンスを作成します。
func NewUIMessageDisplayManager(config *Config, font text.Face, uiContainer *widget.Container) *UIMessageDisplayManager {
	return &UIMessageDisplayManager{
		messageQueue: make([]string, 0),
		config:      config,
		font:        font,
		uiContainer: uiContainer,
		isWaitingForClick: false,
	}
}

// EnqueueMessage は単一のメッセージをキューに追加します。
func (mm *UIMessageDisplayManager) EnqueueMessage(msg string, callback func()) {
	mm.EnqueueMessageQueue([]string{msg}, callback)
}

// EnqueueMessageQueue は複数のメッセージをキューに追加します。
func (mm *UIMessageDisplayManager) EnqueueMessageQueue(messages []string, callback func()) {
	mm.messageQueue = messages
	mm.currentMessageIndex = 0
	mm.postMessageCallback = callback
	mm.ShowCurrentMessage()
}

// ShowCurrentMessage は現在のメッセージをウィンドウに表示します。
func (mm *UIMessageDisplayManager) ShowCurrentMessage() {
	if len(mm.messageQueue) > 0 {
		mm.ShowMessageWindow(mm.messageQueue[mm.currentMessageIndex])
	}
}

// ShowMessageWindow はメッセージウィンドウを表示します。
func (mm *UIMessageDisplayManager) ShowMessageWindow(message string) {
	if mm.messageWindow != nil {
		mm.HideMessageWindow()
	}
	win := createMessageWindow(message, mm.config, mm.font)
	mm.messageWindow = win
	mm.uiContainer.AddChild(mm.messageWindow)
	log.Println("メッセージウィンドウを表示しました:", message)
}

// HideMessageWindow はメッセージウィンドウを非表示にします。
func (mm *UIMessageDisplayManager) HideMessageWindow() {
	if mm.messageWindow != nil {
		mm.uiContainer.RemoveChild(mm.messageWindow)
		mm.messageWindow = nil
		log.Println("メッセージウィンドウを非表示にしました。")
	}
}

// Update はメッセージマネージャーの状態を更新します。
// StateMessage のロジックをここに移動します。
func (mm *UIMessageDisplayManager) Update(state GameState) (GameState, bool) {
	if state != StateMessage {
		return state, false // メッセージ状態でない場合は何もしない
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !mm.isWaitingForClick {
		mm.isWaitingForClick = true
		mm.currentMessageIndex++
		if mm.currentMessageIndex < len(mm.messageQueue) {
			mm.ShowCurrentMessage()
		} else {
			mm.HideMessageWindow()
			if mm.postMessageCallback != nil {
				mm.postMessageCallback()
				mm.postMessageCallback = nil
			}
			return StatePlaying, true // メッセージ表示完了、StatePlayingに戻る
		}
	} else if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		mm.isWaitingForClick = false
	}
	return StateMessage, false // メッセージ表示中
}
