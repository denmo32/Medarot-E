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
	resources                *SharedResources
	world                    donburi.World
	tickCount                int
	debugMode                bool
	state                    GameState
	playerTeam               TeamID
	ui                       *UI
	message                  string
	postMessageCallback      func()
	winner                   TeamID // TeamNone で初期化されます
	playerMedarotToAct       *donburi.Entry
	playerActionPendingQueue []*donburi.Entry // プレイヤーメダロットの行動選択待ちキュー
	currentTarget            *donburi.Entry
	attackingEntity          *donburi.Entry
	targetedEntity           *donburi.Entry

	battleLogic *BattleLogic
}

// NewBattleScene は新しい戦闘シーンを初期化します
func NewBattleScene(res *SharedResources) *BattleScene {
	world := donburi.NewWorld()

	bs := &BattleScene{
		resources:                res,
		world:                    world,
		tickCount:                0,
		debugMode:                true,
		state:                    StatePlaying,
		playerTeam:               Team1,
		playerMedarotToAct:       nil,
		playerActionPendingQueue: make([]*donburi.Entry, 0), // 新しいキューを初期化
		attackingEntity:          nil,
		targetedEntity:           nil,
		currentTarget:            nil,
		winner:                   TeamNone,
	}

	bs.battleLogic = NewBattleLogic(bs.world, &bs.resources.Config)

	// ActionQueueComponent を持つワールド状態エンティティが存在することを確認します。
	// これにより、エンティティと ActionQueueComponentData が存在しない場合に作成されます。
	EnsureActionQueueEntity(bs.world)

	CreateMedarotEntities(bs.world, bs.resources.GameData, bs.playerTeam)
	bs.ui = NewUI(bs) // UIの初期化はヘルパーの後でも問題ありません

	return bs
}

// Update は戦闘シーンのロジックを更新します
func (bs *BattleScene) Update() (SceneType, error) {
	bs.ui.ebitenui.Update()

	switch bs.state {
	case StatePlaying:
		bs.tickCount++

		// 現在プレイヤーがアクションを選択しておらず、かつ保留キューが空の場合に限り入力を処理します。
		// これは、複数ユニットのプレイヤーアクションシーケンスの途中ではないことを意味します。
		if bs.playerMedarotToAct == nil && len(bs.playerActionPendingQueue) == 0 {
			UpdateAIInputSystem(bs.world, bs.battleLogic.PartInfoProvider, bs.battleLogic.TargetSelector, &bs.resources.Config)

			playerInputResult := UpdatePlayerInputSystem(bs.world)
			if len(playerInputResult.PlayerMedarotsToAct) > 0 {
				bs.playerActionPendingQueue = playerInputResult.PlayerMedarotsToAct
				// ここではまだ playerMedarotToAct を設定したり状態を変更したりせず、
				// 以下のロジックでキューから選択するようにします。
			}
		}

		// 保留キューにプレイヤーがいて、現在誰も行動していない場合（現在のユニットのモーダルがまだ表示されていない場合）
		if bs.playerMedarotToAct == nil && len(bs.playerActionPendingQueue) > 0 {
			bs.playerMedarotToAct = bs.playerActionPendingQueue[0]
			bs.state = StatePlayerActionSelect
		}

		// プレイヤーのアクションが保留されておらず（現在選択中でもキュー内でもない）、
		// かつメインのアクション実行キューも空の場合にのみゲージを更新します。
		actionQueueComp := GetActionQueueComponent(bs.world) // アクションキューコンポーネントを取得
		if bs.playerMedarotToAct == nil && len(bs.playerActionPendingQueue) == 0 && len(actionQueueComp.Queue) == 0 {
			UpdateGaugeSystem(bs.world)
		}

		// まだ StatePlaying の場合（例：AIがアクションをキューに入れたか、入力が不要だった、プレイヤーの保留がない）、
		// アクションキューを処理します。これは shouldSkipGaugeUpdateThisFrame に関係なく実行する必要があります。
		// なぜなら、AIのアクションや以前にキューに入れられたプレイヤーのアクションは実行されるべきだからです。
		// ただし、状態が PlayerActionSelect に変更された場合、この部分はスキップされるか、異なる方法で処理される可能性があります。
		if bs.state == StatePlaying { // 状態が PlayerActionSelect に変更された可能性があるため、再確認
			actionResults, err := UpdateActionQueueSystem(
				bs.world,
				bs.battleLogic,
				&bs.resources.Config, // gameConfig を渡す
			)
			if err != nil {
				// エラーを適切に処理
				fmt.Println("アクションキューシステムの処理中にエラーが発生しました:", err)
			}

			for _, result := range actionResults {
				if result.ActingEntry != nil && result.ActingEntry.Valid() {
					logComp := LogComponent.Get(result.ActingEntry)
					if logComp != nil {
						logComp.LastActionLog = result.LogMessage
					}
					bs.attackingEntity = result.ActingEntry // UI表示用
					bs.targetedEntity = result.TargetEntry  // UI表示用

					// メッセージをキューに入れ、クールダウンのコールバックを設定
					bs.enqueueMessage(result.LogMessage, func() {
						// クールダウンを開始する前に、有効性と状態を再度確認
						if result.ActingEntry.Valid() && StateComponent.Get(result.ActingEntry).Current != StateTypeBroken {
							StartCooldownSystem(result.ActingEntry, bs.world, &bs.resources.Config)
						}
						bs.attackingEntity = nil // アクション処理後にクリア
						bs.targetedEntity = nil  // アクション処理後にクリア
					})
					// メッセージがキューに入れられた場合、状態が StateMessage に変わるため、
					// このフレームでの StatePlaying でのさらなる処理を避けるために、早期に break/return することがあります。
					if bs.state == StateMessage {
						break // 状態が変更されたため、actionResults ループを終了
					}
				}
			}
		}

		// まだ StatePlaying で、かつメッセージが表示されていない場合にのみゲーム終了を確認
		// (actionResults からの enqueueMessage が状態を StateMessage に変更する可能性があるため)
		if bs.state == StatePlaying {
			gameEndResult := CheckGameEndSystem(bs.world)
			if gameEndResult.IsGameOver {
				bs.winner = gameEndResult.Winner
				bs.state = StateGameOver
				bs.enqueueMessage(gameEndResult.Message, nil)
			}
		}
		updateAllInfoPanels(bs)
		if bs.ui.battlefieldWidget != nil {
			bs.ui.battlefieldWidget.UpdatePositions()
		}
		ProcessStateChangeSystem(bs.world)
	case StatePlayerActionSelect:
		// プレイヤーがアクションを選択している間、バトルフィールドのアイコンが更新されるようにします
		if bs.ui.battlefieldWidget != nil {
			bs.ui.battlefieldWidget.UpdatePositions()
		}
		// アクションモーダルがまだ表示されておらず、プレイヤーユニットが行動する必要がある場合は表示します。
		if bs.ui.actionModal == nil && bs.playerMedarotToAct != nil {
			// 選択されたメダロットがまだ有効でアイドル状態であることを確認します
			if bs.playerMedarotToAct.Valid() && StateComponent.Get(bs.playerMedarotToAct).Current == StateTypeIdle {
				bs.ui.ShowActionModal(bs, bs.playerMedarotToAct)
			} else {
				// メダロットが有効でなくなったか、アイドル状態でないためリセットします。
				bs.playerMedarotToAct = nil
				bs.state = StatePlaying // 再評価のためにプレイング状態に戻ります
			}
		}
	case StateMessage:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			bs.ui.HideMessageWindow()
			if bs.postMessageCallback != nil {
				bs.postMessageCallback()
				bs.postMessageCallback = nil
			}

			// 勝者が宣言されたかどうかに基づいて次の状態を決定します
			if bs.winner != TeamNone { // 勝者が設定された場合（ゲームオーバー）
				bs.state = StateGameOver
			} else {
				bs.state = StatePlaying
				// bs.playerMedarotToAct が nil の場合、次の StatePlaying イテレーションで入力システム（AI/プレイヤー）が呼び出されます。
			}
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
