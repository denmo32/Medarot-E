package main

import (
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
	messageManager      *MessageManager
	config              *Config
	font                text.Face
	uiContainer         *widget.Container
	uiFactory           *UIFactory // 追加
}

// NewUIMessageDisplayManager は新しいUIMessageDisplayManagerのインスタンスを作成します。
func NewUIMessageDisplayManager(messageManager *MessageManager, config *Config, font text.Face, uiContainer *widget.Container, uiFactory *UIFactory) *UIMessageDisplayManager {
	return &UIMessageDisplayManager{
		messageQueue:   make([]string, 0),
		messageManager: messageManager,
		config:         config,
		font:           font,
		uiContainer:    uiContainer,
		uiFactory:      uiFactory, // 追加
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
	win := createMessageWindow(message, mm.uiFactory) // uiFactoryを渡す
	mm.messageWindow = win
	mm.uiContainer.AddChild(mm.messageWindow)
}

// HideMessageWindow はメッセージウィンドウを非表示にします。
func (mm *UIMessageDisplayManager) HideMessageWindow() {
	if mm.messageWindow != nil {
		mm.uiContainer.RemoveChild(mm.messageWindow)
		mm.messageWindow = nil
	}
}

// Update はメッセージマネージャーの状態を更新します。
func (mm *UIMessageDisplayManager) Update() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mm.currentMessageIndex++
		if mm.currentMessageIndex < len(mm.messageQueue) {
			mm.ShowCurrentMessage()
		} else {
			mm.HideMessageWindow()
			if mm.postMessageCallback != nil {
				mm.postMessageCallback()
				mm.postMessageCallback = nil
			}
			mm.messageQueue = make([]string, 0) // メッセージキューをクリア
		}
	}
}

// IsFinished はメッセージキューが空で、かつメッセージウィンドウが表示されていない場合にtrueを返します。
func (mm *UIMessageDisplayManager) IsFinished() bool {
	isFinished := len(mm.messageQueue) == 0 && mm.messageWindow == nil
	return isFinished
}
