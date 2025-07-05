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
	ui                       UIInterface
	message                  string
	postMessageCallback      func()
	winner                   TeamID // TeamNone で初期化されます
	playerActionPendingQueue []*donburi.Entry // プレイヤーメダロットの行動選択待ちキュー
	attackingEntity          *donburi.Entry
	targetedEntity           *donburi.Entry

	battleLogic *BattleLogic
	uiEventChannel chan UIEvent
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
		playerActionPendingQueue: make([]*donburi.Entry, 0), // 新しいキューを初期化
		attackingEntity:          nil,
		targetedEntity:           nil,
		winner:                   TeamNone,
	}

	bs.battleLogic = NewBattleLogic(bs.world, &bs.resources.Config)

	// ActionQueueComponent を持つワールド状態エンティティが存在することを確認します。
	// これにより、エンティティと ActionQueueComponentData が存在しない場合に作成されます。
	EnsureActionQueueEntity(bs.world)

	CreateMedarotEntities(bs.world, bs.resources.GameData, bs.playerTeam)
	bs.uiEventChannel = make(chan UIEvent, 10) // バッファ付きチャネル
	bs.ui = NewUI(bs.world, &bs.resources.Config, bs.uiEventChannel, bs.battleLogic.PartInfoProvider, bs.battleLogic.TargetSelector)

	return bs
}

// Update は戦闘シーンのロジックを更新します
func (bs *BattleScene) Update() (SceneType, error) {
	bs.ui.Update()

	// UIイベントチャネルを処理します
	select {
	case event := <-bs.uiEventChannel:
		switch e := event.(type) {
		case PlayerActionSelectedEvent:
			bs.playerActionPendingQueue, bs.state, bs.targetedEntity, bs.attackingEntity, bs.message, bs.postMessageCallback = processPlayerActionSelected(
				bs.world, &bs.resources.Config, bs.battleLogic, bs.playerActionPendingQueue, bs.ui, e)
			if bs.message != "" {
				bs.enqueueMessage(bs.message, bs.postMessageCallback)
			}
		case PlayerActionCancelEvent:
			bs.playerActionPendingQueue, bs.state = processPlayerActionCancel(bs.playerActionPendingQueue, bs.ui, e)
		case SetCurrentTargetEvent:
			bs.ui.SetCurrentTarget(e.Target)
		case ClearCurrentTargetEvent:
			bs.ui.ClearCurrentTarget()
		}
	default:
		// イベントがない場合は何もしない
	}

	switch bs.state {
	case StatePlaying:
		bs.tickCount++

		// 現在プレイヤーがアクションを選択しておらず、かつ保留キューが空の場合に限り入力を処理します。
		// これは、複数ユニットのプレイヤーアクションシーケンスの途中ではないことを意味します。
		if !bs.ui.IsActionModalVisible() && len(bs.playerActionPendingQueue) == 0 {
			UpdateAIInputSystem(bs.world, bs.battleLogic.PartInfoProvider, bs.battleLogic.TargetSelector, &bs.resources.Config)

			playerInputResult := UpdatePlayerInputSystem(bs.world)
			if len(playerInputResult.PlayerMedarotsToAct) > 0 {
				bs.playerActionPendingQueue = playerInputResult.PlayerMedarotsToAct
				// ここではまだ playerMedarotToAct を設定したり状態を変更したりせず、
				// 以下のロジックでキューから選択するようにします。
			}
		}

		// 保留キューにプレイヤーがいる場合、状態をアクション選択に移行します。
		if len(bs.playerActionPendingQueue) > 0 {
			bs.state = StatePlayerActionSelect
		}

		// プレイヤーのアクションが保留されておらず（現在選択中でもキュー内でもない）、
		// かつメインのアクション実行キューも空の場合にのみゲージを更新します。
		actionQueueComp := GetActionQueueComponent(bs.world) // アクションキューコンポーネントを取得
		if !bs.ui.IsActionModalVisible() && len(bs.playerActionPendingQueue) == 0 && len(actionQueueComp.Queue) == 0 {
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
					bs.ui.SetCurrentTarget(result.TargetEntry)

					// メッセージをキューに入れ、クールダウンのコールバックを設定
					bs.enqueueMessage(result.LogMessage, func() {
						// クールダウンを開始する前に、有効性と状態を再度確認
						if result.ActingEntry.Valid() && StateComponent.Get(result.ActingEntry).Current != StateTypeBroken {
							StartCooldownSystem(result.ActingEntry, bs.world, &bs.resources.Config, bs.battleLogic.PartInfoProvider)
						}
						bs.attackingEntity = nil // アクション処理後にクリア
						bs.targetedEntity = nil  // アクション処理後にクリア
						bs.ui.ClearCurrentTarget()
					})
					// メッセージがキューに入れられた場合、状態が StateMessage に変わるため、
					// このフレームでの StatePlaying でのさらなる処理を避けるために、早期に break/return することがあります。
					if bs.state == StateMessage {
						break // 状態が変更されたため、actionResults ループを終了
					}
				}
			}
		}

		// ゲーム終了判定
		gameEndResult := CheckGameEndSystem(bs.world)
		if gameEndResult.IsGameOver {
			bs.winner = gameEndResult.Winner
			bs.state = StateGameOver
			bs.enqueueMessage(gameEndResult.Message, nil)
		}

		// UIの更新
		bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
		bfVM := BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
		bs.ui.SetBattlefieldViewModel(bfVM)
		ProcessStateChangeSystem(bs.world)

	case StatePlayerActionSelect:
		// プレイヤーがアクションを選択している間、UIのみを更新し、ゲームロジックの進行は停止
		bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
		bfVM := BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
		bs.ui.SetBattlefieldViewModel(bfVM)

		// アクションモーダルがまだ表示されておらず、プレイヤーユニットが行動する必要がある場合は表示します。
		if !bs.ui.IsActionModalVisible() && len(bs.playerActionPendingQueue) > 0 {
			// 選択されたメダロットがまだ有効でアイドル状態であることを確認します
			actingEntry := bs.playerActionPendingQueue[0]
			if actingEntry.Valid() && StateComponent.Get(actingEntry).Current == StateTypeIdle {
				bs.ui.ShowActionModal(actingEntry)
			} else {
				// メダロットが有効でなくなったか、アイドル状態でないためキューから削除します。
				bs.playerActionPendingQueue = bs.playerActionPendingQueue[1:]
				// キューが空になったらプレイング状態に戻ります
				if len(bs.playerActionPendingQueue) == 0 {
					bs.state = StatePlaying
				}
			}
		}
	case StateMessage:
		// メッセージ表示中はUIのみを更新し、ゲームロジックの進行は停止
		bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
		bfVM := BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
		bs.ui.SetBattlefieldViewModel(bfVM)

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
	bs.ui.Draw(screen)

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
	bs.ui.ShowMessageWindow(bs.message)
}