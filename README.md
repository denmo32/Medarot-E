**【メダロットE】解説書**

**プロジェクト概要**

このプロジェクトは、人気ゲーム「メダロット」の戦闘システムをリスペクトした、EbitenとECS（エンティティ・コンポーネント・システム）アーキテクチャによるバトルシミュレーションゲームです。
ECSライブラリには [donburi](https://github.com/yohamta/donburi) を採用しており、データ指向の設計に基づいています。

**主要ファイル構成**

プロジェクトは、以下の主要なカテゴリのファイル群で構成されています。

Core & Entry Point (中核・起動)
-----------------------------

プログラムの起動とゲーム全体の管理を行います。

*   `main.go`
    *   役割: プログラムの起動点（エントリーポイント）。
    *   内容: ウィンドウの初期化、フォントや設定ファイルの読み込み、ゲーム全体のメインループを開始し、`SceneManager` をセットアップします。
*   `scene/scene_manager.go`
    *   役割: シーンの切り替えと管理を行います。
    *   内容: `bamenn` ライブラリを使用して、ゲーム内の異なるシーン（タイトル、バトル、カスタマイズなど）間の遷移を制御します。

Domain (ゲームのコア)
-------------------

ゲームのルール、データ構造、インターフェースなど、プロジェクトの核心となるドメイン知識を定義します。
このパッケージは、特定の技術（UIライブラリ、ECSフレームワークなど）から独立していることを目指します。

*   `core/types.go`: **[データ]** ゲーム全体で使われる、`donburi`に依存しない基本的な型、定数、データ構造を定義します。また、UI表示に必要な整形済みデータ（ViewModel）の定義もここに含まれます。
*   `ecs/component/component_data.go`: **[データ]** `donburi`フレームワークに依存するコンポーネントのデータ構造（`donburi.Entry`を含む構造体など）を定義します。
*   `event/events.go`: **[定義]** ゲーム内で発生するイベントの定義。

ECS (エンティティ・コンポーネント・システム)
---------------------------------------

ECSアーキテクチャの構成要素を定義します。

*   `ecs/component/component_types.go`: **[データ]** ECSの「C（コンポーネント）」を`donburi`に登録します。各コンポーネントが保持するデータ構造自体は`ecs/component/component_data.go`で定義されます。
*   `ecs/entity/ecs_setup_logic.go`: 戦闘開始時のエンティティ生成と初期コンポーネント設定を行います。
*   `ecs/entity/world_state.go`: **[ロジック/ヘルパー]** `PlayerActionQueueComponent`や`ActionQueueComponent`など、ワールド全体の状態を管理するシングルトンエンティティへのアクセスと操作を提供します。

Scene (各画面の実装)
-------------------

ゲーム内の各画面（シーン）の実装です。

*   `scene/scene.go`
    *   役割: すべてのシーンが満たすべき共通のルール（インターフェース）を定義します。
    *   内容: `Update`, `Draw` メソッドの型定義や、シーン間で共有するリソース（`SharedResources`）を定義します。
*   `scene/scene_title.go`: タイトル画面の実装。
*   `scene/scene_battle.go`: 戦闘シーンの統括。戦闘用のWorld（ECS）と、戦闘全体の進行を管理するステートマシン（`GameState`）を保持します。UI関連のロジックは`ui.BattleUIManager`に委譲され、`BattleScene`は現在の状態（`GameState`）に応じて適切な処理（ゲージ進行、行動選択、アニメーションなど）を実行し、状態間の遷移を制御します。
*   `scene/scene_customize.go`: メダロットのカスタマイズ画面の実装。
*   `scene/scene_placeholder.go`: 未実装画面などのための、汎用的なプレースホルダー画面。

Battle Action (メダロットの行動)
---------------------------------------

戦闘中の各メダロットが実行するアクション定義や処理です。

*   `ecs/system/ai_action_selection.go`: **[ロジック/振る舞い]** AI制御のメダロットの行動選択ロジックを定義します。
*   `ecs/system/ai_personalities.go`: **[データ]** AIの性格定義と、それに対応する行動戦略をマッピングします。
*   `ecs/system/ai_target_strategies.go`: **[ロジック/振る舞い]** AIのターゲット選択戦略の具体的な実装を定義します。
*   `ecs/system/battle_action_queue_system.go`: **[ロジック/振る舞い]** 行動実行キューを処理し、適切な `ActionExecutor` を呼び出して行動を実行します。
*   `ecs/system/battle_action_executor.go`: **[ロジック/振る舞い]** アクションの実行に関する主要なロジックをカプセル化します。特性や武器タイプごとの具体的な処理は、`battle_trait_handlers.go` および `battle_weapon_effect_handlers.go` に委譲されます。
*   `ecs/system/battle_trait_handlers.go`: **[ロジック/振る舞い]** 各特性（Trait）に応じたアクションの実行ロジックを定義します。`BaseAttackHandler`、`SupportTraitExecutor`、`ObstructTraitExecutor` などが含まれます。共通の攻撃ロジックヘルパー関数は `ecs/system/battle_logic_helpers.go` に移動されました。
*   `ecs/system/battle_weapon_effect_handlers.go`: **[ロジック/振る舞い]** 各武器タイプ（WeaponType）に応じた追加効果の適用ロジックを定義します。`ThunderEffectHandler`、`MeltEffectHandler`、`VirusEffectHandler` などが含まれます。
*   `ecs/system/charge_initiation_system.go`: **[ロジック/振る舞い]** メダロットが行動を開始する際のチャージ状態の開始ロジックを管理します。`StartCharge` メソッドを提供します。
*   `ecs/system/post_action_effect_system.go`: **[ロジック/振る舞い]** アクション実行後のステータス効果の適用やパーツ破壊による状態遷移などを処理します。

Battle Logic & AI (戦闘ルールと思考)
---------------------------------

戦闘のコアロジックやAIの思考ルーチンです。

*   `ecs/system/battle_logic_helpers.go`: **[ロジック/ヘルパー]** 戦闘ロジック内で共通して利用されるヘルパー関数群（命中判定、ダメージ適用、ターゲット解決など）を定義します。

*   `data/battle_logger.go`: **[ロジック/振る舞い]** 戦闘中のログメッセージの生成と管理を行います。
*   `ecs/system/game_states.go`: **[ロジック/振る舞い]** 戦闘全体の進行を制御する各`GameState`（`GaugeProgressState`, `PlayerActionSelectState`, `ActionExecutionState`など）の具体的なロジックを実装します。各状態は、戦闘フローの特定のフェーズ（ゲージ進行、行動選択、アニメーションなど）を担当します。
*   `ecs/system/battle_logic.go`
    *   役割: 戦闘中のコアな計算ロジック（Calculator群）をカプセル化します。
    *   内容: 戦闘に関連するヘルパー群（`DamageCalculator`, `HitCalculator`, `TargetSelector`, `PartInfoProvider`, `ChargeInitiationSystem`）を内包する`BattleLogic`構造体を定義します。これにより、`BattleScene`からの依存関係が単純化され、戦闘ロジックが一元管理されます。具体的な計算式や選択アルゴリズムは各ヘルパー内に実装されています。
*   `ecs/system/battle_damage_calculator.go`: **[ロジック/振る舞い]** ダメージ計算に関するロジックを扱います。
*   `ecs/system/battle_hit_calculator.go`: **[ロジック/振る舞い]** 命中・回避・防御判定に関するロジックを扱います。
*   `ecs/system/battle_part_info_provider.go`: **[ロジック/振る舞い]** パーツの状態や情報を取得・操作するロジックを扱います。
*   `ecs/system/battle_target_selector.go`: **[ロジック/振る舞い]** ターゲット選択やパーツ選択に関するロジックを扱います。
*   `ecs/system/battle_end_system.go`: **[ロジック/振る舞い]** ゲーム終了条件判定システム。`CheckGameEndSystem` を定義します。
*   `ecs/system/battle_gauge_system.go`: **[ロジック/振る舞い]** チャージゲージおよびクールダウンゲージの進行管理システム。`UpdateGaugeSystem` を定義します。
*   `ecs/system/battle_intention_system.go`: **[ロジック/振る舞い]** プレイヤーとAIの入力を処理し、行動の「意図（Intention）」を生成するシステムです。
*   `ecs/system/status_effect_system.go`: **[ロジック/振る舞い]** ステータス効果の適用、更新、解除を管理するシステム。
*   `ecs/system/battle_history_system.go`: **[ロジック/振る舞い]** アクションの結果に基づいてAIの行動履歴を更新するシステム。
*   `ecs/system/status_effect_implementations.go`: **[ロジック/振る舞い]** 各種ステータス効果の具体的な適用・解除ロジックを定義します。
*   `ecs/system/game_interfaces.go`: **[定義]** ゲーム全体で利用される主要なインターフェースを定義します。 `TargetingStrategy` や `TraitActionHandler` など、特定の振る舞いを抽象化するためのインターフェースが含まれます。

UI (ユーザーインターフェース)
-----------------------

ゲームのユーザーインターフェース関連のファイルです。
UIはECSアーキテクチャの原則に基づき、ゲームロジックから明確に分離することを目標にしています。

*   `ui/battle_ui_manager.go`: **[ロジック/振る舞い]** バトルUI全体の司令塔。UIの初期化、更新、描画を統括します。UIウィジェットからの入力を受け取り、対応する**ゲームイベント**を`BattleScene`に通知する役割も担います。各種サブマネージャ（`InfoPanelManager`, `ActionModalManager`など）を内包し、UIのライフサイクル全体を管理します。
*   `ui/ui_layout.go`: **[ヘルパー]** `ebitenui`のコンテナなど、UIの基本的なレイアウト構造を生成するためのヘルパー関数を定義します。
*   `ui/state.go`: **[データ]** UIの状態を保持するシングルトンコンポーネント`BattleUIState`を定義します。戦闘UIの表示/非表示、選択中のパーツ、ターゲットなどの状態を管理します。
*   `ui/ui_interfaces.go`
    *   役割: UIコンポーネントが満たすべきインターフェースを定義します。
    *   内容: `UIInterface`など、UIの描画とイベント処理に必要なメソッドを定義します。`DrawBackground`メソッドは削除されました。かつて`UIMediator`インターフェースが定義されていましたが、`ecs/system/game_interfaces.go`に移動されました。
*   `ui/ui_factory.go`
    *   役割: UIコンポーネントの生成とスタイリングを一元的に管理するファクトリ。
    *   内容: `NewCyberpunkButton`など、共通のスタイルを持つUI要素を生成するメソッドを提供します。
*   `ui/ui_image_generator.go`
    *   役割: UIコンポーネントの画像生成ロジックをカプセル化します。
    *   内容: サイバーパンク風のボタンやパネルの背景画像を生成するメソッドを提供します。
*   `ui/ui_panel.go`
    *   役割: 汎用的なUIパネル（枠付きウィンドウ）の作成と管理。
    *   内容: 他のUI要素（情報パネル、アクションモーダル、メッセージウィンドウなど）の基盤となる、再利用可能なパネルコンポーネントを提供します。
*   `ui/ui_view_model_factory.go`: **[ロジック/振る舞い]** ECSのデータからUI表示用のViewModelを構築するファクトリ。`InfoPanelViewModel`や`BattlefieldViewModel`など、UIが必要とする整形されたデータを生成します。これにより、UIはECSの内部構造に直接依存しません。
*   `ui/ui_battlefield_widget.go`: 中央のバトルフィールド描画。ViewModelを受け取って描画します。メダロットのモデル、HPバー、状態アイコンなどを表示します。
*   `ui/ui_info_panels.go`: 左右の情報パネル（HPゲージなど）の作成と更新。ViewModelを受け取って描画します。メダロットのHP、チャージゲージ、クールダウンゲージ、ステータス効果などを表示します。
*   `ui/ui_action_modal.go`: プレイヤーの行動選択モーダルウィンドウ。下部の共通パネル上に、背景を透過させてパーツ選択ボタンを表示します。UIイベントを発行し、ViewModelを使用して表示します。
*   `ui/ui_action_modal_manager.go`: **[ロジック/振る舞い]** アクション選択モーダルの表示と状態を管理します。プレイヤーの入力に応じてモーダルを開閉し、選択されたアクションを処理します。
*   `ui/ui_message_window.go`: 画面下のメッセージウィンドウ。下部の共通パネル上に、背景を透過させてメッセージを表示します。戦闘中のイベントやシステムメッセージを表示します。
*   `ui/ui_message_display_manager.go`: **[ロジック/振る舞い]** ゲーム内のメッセージ表示（キュー管理、ウィンドウ表示/非表示、進行）を管理します。
*   `ui/ui_animation_drawer.go`: **[ロジック/振る舞い]** 戦闘中のアクションアニメーションの具体的な描画処理。攻撃エフェクト、ダメージ表示、ステータス効果アニメーションなどを担当します。
*   `ui/ui_system.go`: **[ロジック/振る舞い]** UI関連のECSシステムを定義します。主に`UpdateInfoPanelViewModelSystem`が含まれ、情報パネルのViewModelを更新します。
*   `ui/ui_target_indicator_manager.go`: **[ロジック/振る舞い]** ターゲットインジケーターの表示と状態を管理します。プレイヤーがターゲットを選択する際に、対象のメダロットやパーツを視覚的に示します。

Configuration & Resources (設定とリソース)
------------------------------------

ゲームの設定値や外部リソースの読み込み・管理に関するファイル群です。

*   `assets/`: 音声、設定ファイル、データベース、フォント、画像、テキストメッセージなど、ゲームで使用される各種リソースを格納します。
*   `data/config.go`: ゲームバランスに関する設定値やUIの固定値など、アプリケーション全体の設定（`Config`構造体）を定義します。
*   `data/config_loader.go`: ゲームの固定設定値（画面サイズ、色など）をロードします。
*   `data/resource_ids.go`: `ebitengine-resource` ライブラリで使用するリソースIDを定義します。
*   `data/resource_loader.go`: `ebitengine-resource` を使用したゲームリソース（CSVデータ、フォントなど）の読み込みと管理。
*   `data/game_data_manager.go`: 静的なゲームデータ（パーツ定義、メダル定義など）の管理とアクセスを提供します。
*   `data/message_manager.go`: ゲーム内のメッセージテンプレートの読み込みとフォーマットを管理します。
*   `data/csv_saver.go`: メダロット構成のデータをCSVファイルに保存します。
*   `data/shared.go`: シーン間で共有されるリソースを定義します。
*   `data/utils.go`: 文字列のパースなどの汎用ユーティリティ関数。
