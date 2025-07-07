package main

import (
	"fmt"
	"image/color"

	// "math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/yohamta/donburi"
)

type BattleScene struct {
	resources                *SharedResources
	world                    donburi.World
	tickCount                int
	debugMode                bool
	state                    GameState
	playerTeam               TeamID
	ui                       UIInterface
	message                  string
	messageQueue             []string // 複数のメッセージを保持するキュー
	currentMessageIndex      int      // 現在表示しているメッセージのインデックス
	postMessageCallback      func()
	winner                   TeamID           // TeamNone で初期化されます
	playerActionPendingQueue []*donburi.Entry // プレイヤーメダロットの行動選択待ちキュー
	attackingEntity          *donburi.Entry
	targetedEntity           *donburi.Entry
	whitePixel               *ebiten.Image

	currentActionAnimation *ActionAnimationData // 現在実行中のアクションアニメーション

	battleLogic    *BattleLogic
	uiEventChannel chan UIEvent
}

// ActionAnimationData はアクションアニメーションのデータを保持します。
type ActionAnimationData struct {
	Result    ActionResult // ActionResult全体を保持
	StartTime int          // アニメーション開始時のtickCount
}

// NewBattleScene は新しい戦闘シーンを初期化します
func NewBattleScene(res *SharedResources) *BattleScene {
	world := donburi.NewWorld()
	whitePixel := ebiten.NewImage(1, 1)
	whitePixel.Fill(color.White)

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
		whitePixel:               whitePixel,
	}

	bs.battleLogic = NewBattleLogic(bs.world, &bs.resources.Config)

	// ActionQueueComponent を持つワールド状態エンティティが存在することを確認します。
	// これにより、エンティティと ActionQueueComponentData が存在しない場合に作成されます。
	EnsureActionQueueEntity(bs.world)

	CreateMedarotEntities(bs.world, res.GameData, bs.playerTeam)
	bs.uiEventChannel = make(chan UIEvent, 10) // バッファ付きチャネル
	bs.ui = NewUI(bs.world, &bs.resources.Config, bs.uiEventChannel)

	return bs
}

// Update は戦闘シーンのロジックを更新します
func (bs *BattleScene) Update() (SceneType, error) {
	bs.tickCount++
	bs.ui.Update()

	// UIイベントチャネルを処理します
	select {
	case event := <-bs.uiEventChannel:
		switch e := event.(type) {
		case PlayerActionSelectedEvent:
			bs.playerActionPendingQueue, bs.state, bs.message, bs.postMessageCallback = ProcessPlayerActionSelected(
				bs.world, &bs.resources.Config, bs.battleLogic, bs.playerActionPendingQueue, bs.ui, e, bs.ui.GetActionTargetMap())
			if bs.message != "" {
				bs.enqueueMessage(bs.message, bs.postMessageCallback)
			}
		case PlayerActionCancelEvent:
			bs.playerActionPendingQueue, bs.state = ProcessPlayerActionCancel(bs.playerActionPendingQueue, bs.ui, e)
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
		if !bs.ui.IsActionModalVisible() && len(bs.playerActionPendingQueue) == 0 {
			UpdateAIInputSystem(bs.world, bs.battleLogic.PartInfoProvider, bs.battleLogic.TargetSelector, &bs.resources.Config)

			playerInputResult := UpdatePlayerInputSystem(bs.world)
			if len(playerInputResult.PlayerMedarotsToAct) > 0 {
				bs.playerActionPendingQueue = playerInputResult.PlayerMedarotsToAct
			}
		}

		if len(bs.playerActionPendingQueue) > 0 {
			bs.state = StatePlayerActionSelect
		}

		actionQueueComp := GetActionQueueComponent(bs.world)
		if !bs.ui.IsActionModalVisible() && len(bs.playerActionPendingQueue) == 0 && len(actionQueueComp.Queue) == 0 {
			UpdateGaugeSystem(bs.world)
		}

		if bs.state == StatePlaying {
			actionResults, err := UpdateActionQueueSystem(
				bs.world,
				bs.battleLogic,
				&bs.resources.Config,
			)
			if err != nil {
				fmt.Println("アクションキューシステムの処理中にエラーが発生しました:", err)
			}

			for _, result := range actionResults {
				if result.ActingEntry != nil && result.ActingEntry.Valid() {
					// アニメーションデータを設定
					bs.currentActionAnimation = &ActionAnimationData{
						Result:    result,
						StartTime: bs.tickCount,
					}

					// ダメージ適用
					targetParts := PartsComponent.Get(result.TargetEntry)
					if targetParts != nil {
						intendedTargetPartInstance := targetParts.Map[result.TargetPartSlot]
						bs.battleLogic.DamageCalculator.ApplyDamage(result.TargetEntry, intendedTargetPartInstance, result.OriginalDamage)
					}

					// メッセージ表示とクールダウンはアニメーション終了後に行う
					bs.state = StateAnimatingAction // アニメーション状態に遷移
					break                           // 1フレームに1つのアクションのみ処理
				}
			}
		}

		gameEndResult := CheckGameEndSystem(bs.world)
		if gameEndResult.IsGameOver {
			bs.winner = gameEndResult.Winner
			bs.state = StateGameOver
			bs.enqueueMessage(gameEndResult.Message, nil)
		}

		bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)
		bfVM := BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
		bs.ui.SetBattlefieldViewModel(bfVM)
		ProcessStateChangeSystem(bs.world)

	case StateAnimatingAction:
		// UIパネルを更新してHPバーアニメーションを実行
		bs.ui.UpdateInfoPanels(bs.world, &bs.resources.Config)

		// アニメーションのロジック
		anim := bs.currentActionAnimation
		if anim == nil {
			bs.state = StatePlaying // 安全のため
			return SceneTypeBattle, nil
		}

		// アニメーションの進行度を計算
		progress := float64(bs.tickCount - anim.StartTime)

		// アニメーション終了判定
		const travelDuration = 30 // 三角印が移動するフレーム数
		const popupDelay = 30     // 三角印がターゲットに到達してからポップアップ開始までの遅延
		const popupDuration = 60  // ポップアップ表示時間
		const totalAnimationDuration = travelDuration + popupDelay + popupDuration

		if progress >= totalAnimationDuration {
			// アニメーション終了後の処理
			bs.currentActionAnimation = nil // アニメーションデータをクリア

			// アニメーション終了後の処理
			bs.currentActionAnimation = nil // アニメーションデータをクリア

			// 新しいメッセージ形式でメッセージキューを作成
			messages := []string{}
			result := anim.Result
			if result.ActionDidHit {
				// メッセージID: action_initiate
				initiateParams := map[string]interface{}{
					"attacker_name": result.AttackerName,
					"action_name":   result.ActionName,
					"weapon_type":   result.WeaponType,
				}
				messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_initiate", initiateParams))

				// メッセージID: action_defend
				if result.ActionIsDefended {
					defendParams := map[string]interface{}{
						"defender_name":       result.DefenderName,
						"defending_part_type": result.DefendingPartType,
					}
					messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_defend", defendParams))
				}

				// メッセージID: action_damage
				damageParams := map[string]interface{}{
					"defender_name":    result.DefenderName,
					"target_part_type": result.TargetPartType,
					"damage":           result.DamageDealt,
				}
				messages = append(messages, GlobalGameDataManager.Messages.FormatMessage("action_damage", damageParams))

			} else {
				messages = append(messages, result.LogMessage) // 回避などのメッセージ
			}

			bs.enqueueMessageQueue(messages, func() {
				actingEntry := anim.Result.ActingEntry
				if actingEntry.Valid() && StateComponent.Get(actingEntry).Current != StateTypeBroken {
					StartCooldownSystem(actingEntry, bs.world, &bs.resources.Config, bs.battleLogic.PartInfoProvider)
				}
				bs.attackingEntity = nil
				bs.targetedEntity = nil
				bs.ui.ClearCurrentTarget()
			})
			// enqueueMessageがStateMessageに遷移させるため、ここでは何もしない
		}

		// StateAnimatingAction中は、ゲームロジックの進行は停止

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
				// UIに渡す前に、利用可能なパーツとターゲットを計算
				actionTargetMap := make(map[PartSlotKey]ActionTarget)
				availableParts := bs.battleLogic.PartInfoProvider.GetAvailableAttackParts(actingEntry)

				for _, available := range availableParts {
					partDef := available.PartDef
					slotKey := available.Slot

					var targetEntity *donburi.Entry
					var targetPartSlot PartSlotKey

					if partDef.Category == CategoryShoot || partDef.Category == CategoryMelee {
						medal := MedalComponent.Get(actingEntry)
						var strategy TargetingStrategy
						switch medal.Personality {
						case "アシスト":
							strategy = &AssistStrategy{}
						case "クラッシャー":
							strategy = &CrusherStrategy{}
						case "カウンター":
							strategy = &CounterStrategy{}
						case "チェイス":
							strategy = &ChaseStrategy{}
						case "デュエル":
							strategy = &DuelStrategy{}
						case "フォーカス":
							strategy = &FocusStrategy{}
						case "ガード":
							strategy = &GuardStrategy{}
						case "ハンター":
							strategy = &HunterStrategy{}
						case "インターセプト":
							strategy = &InterceptStrategy{}
						case "ジョーカー":
							strategy = &JokerStrategy{}
						default:
							strategy = &LeaderStrategy{}
						}
						targetEntity, targetPartSlot = strategy.SelectTarget(bs.world, actingEntry, bs.battleLogic.TargetSelector, bs.battleLogic.PartInfoProvider)
					}
					actionTargetMap[slotKey] = ActionTarget{Target: targetEntity, Slot: targetPartSlot}
				}
				bs.ui.ShowActionModal(actingEntry, actionTargetMap)
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
			bs.currentMessageIndex++
			if bs.currentMessageIndex < len(bs.messageQueue) {
				// 次のメッセージを表示
				bs.ui.ShowMessageWindow(bs.messageQueue[bs.currentMessageIndex])
			} else {
				// すべて表示完了
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
				}
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

	// アクションアニメーションの描画
	if bs.currentActionAnimation != nil {
		anim := bs.currentActionAnimation
		// アニメーションの進行度を計算
		progress := float64(bs.tickCount - anim.StartTime)

		// 攻撃者とターゲットのアイコン位置を取得
		bfVM := BuildBattlefieldViewModel(bs.world, bs.battleLogic.PartInfoProvider, &bs.resources.Config, bs.debugMode, bs.ui.GetBattlefieldWidgetRect())
		var attackerVM, targetVM *IconViewModel
		for _, icon := range bfVM.Icons {
			if icon.EntryID == uint32(anim.Result.ActingEntry.Id()) {
				attackerVM = icon
			}
			if icon.EntryID == uint32(anim.Result.TargetEntry.Id()) {
				targetVM = icon
			}
		}

		if attackerVM != nil && targetVM != nil {
			// 三角印のアニメーション
			const travelDuration = 30 // 30フレームで移動
			if progress <= travelDuration {
				// 移動中の三角印を描画
				lerpFactor := progress / travelDuration
				currentX := attackerVM.X + (targetVM.X-attackerVM.X)*float32(lerpFactor)
				currentY := attackerVM.Y + (targetVM.Y-attackerVM.Y)*float32(lerpFactor)

				// 三角形の描画 (DrawTargetIndicatorを参考)
				indicatorColor := bs.resources.Config.UI.Colors.Yellow
				iconRadius := bs.resources.Config.UI.Battlefield.IconRadius
				indicatorHeight := bs.resources.Config.UI.Battlefield.TargetIndicator.Height
				indicatorWidth := bs.resources.Config.UI.Battlefield.TargetIndicator.Width
				margin := float32(5)

				p1x := currentX - indicatorWidth/2
				p1y := currentY - iconRadius - margin - indicatorHeight
				p2x := currentX + indicatorWidth/2
				p2y := p1y
				p3x := currentX
				p3y := currentY - iconRadius - margin

				vertices := []ebiten.Vertex{
					{DstX: p1x, DstY: p1y},
					{DstX: p2x, DstY: p2y},
					{DstX: p3x, DstY: p3y},
				}
				r, g, b, a := indicatorColor.RGBA()
				cr := float32(r) / 65535
				cg := float32(g) / 65535
				cb := float32(b) / 65535
				ca := float32(a) / 65535
				for i := range vertices {
					vertices[i].ColorR = cr
					vertices[i].ColorG = cg
					vertices[i].ColorB = cb
					vertices[i].ColorA = ca
				}
				indices := []uint16{0, 1, 2}
				screen.DrawTriangles(vertices, indices, bs.whitePixel, &ebiten.DrawTrianglesOptions{})
			}

			// ダメージ数字のポップアップ
			const popupDelay = 30    // 三角印がターゲットに到達してからポップアップ開始までの遅延
			const popupDuration = 60 // ポップアップ表示時間
			if progress >= travelDuration+popupDelay && progress < travelDuration+popupDelay+popupDuration {
				popupProgress := (progress - (travelDuration + popupDelay)) / popupDuration

				// ポップアップの位置を計算（少し上に移動する）
				x := targetVM.X
				y := targetVM.Y - 20 - (20 * float32(popupProgress)) // 20ピクセル上に移動

				// フェードアウト効果
				alpha := 1.0
				if popupProgress > 0.7 {
					alpha = (1.0 - popupProgress) / 0.3
				}

				// ダメージテキストを描画
				geoM := ebiten.GeoM{}
				geoM.Translate(float64(x), float64(y))
				colorM := ebiten.ColorM{}
				colorM.Scale(1, 1, 1, alpha)

				text.Draw(screen, fmt.Sprintf("%d", anim.Result.OriginalDamage),
					bs.resources.Font,
					&text.DrawOptions{
						DrawImageOptions: ebiten.DrawImageOptions{
							GeoM:   geoM,
							ColorM: colorM,
						},
						LayoutOptions: text.LayoutOptions{
							PrimaryAlign:   text.AlignCenter,
							SecondaryAlign: text.AlignCenter,
						},
					})
			}

		}
	}

	if bs.debugMode {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\nState: %s",
			ebiten.ActualTPS(), ebiten.ActualFPS(), bs.state))
	}
}

// enqueueMessage は単一のメッセージをキューに追加します
func (bs *BattleScene) enqueueMessage(msg string, callback func()) {
	bs.enqueueMessageQueue([]string{msg}, callback)
}

// enqueueMessageQueue は複数のメッセージをキューに追加し、表示を開始します
func (bs *BattleScene) enqueueMessageQueue(messages []string, callback func()) {
	bs.messageQueue = messages
	bs.currentMessageIndex = 0
	bs.postMessageCallback = callback
	bs.state = StateMessage
	if len(bs.messageQueue) > 0 {
		bs.ui.ShowMessageWindow(bs.messageQueue[0])
	}
}
