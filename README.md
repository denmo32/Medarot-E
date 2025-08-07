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
*   `scene_manager.go`
    *   役割: シーンの切り替えと管理を行います。
    *   内容: `bamenn` ライブラリを使用して、ゲーム内の異なるシーン（タイトル、バトル、カスタマイズなど）間の遷移を制御します。

Domain (ゲームのコア)
-------------------

ゲームのルール、データ構造、インターフェースなど、プロジェクトの核心となるドメイン知識を定義します。
このパッケージは、特定の技術（UIライブラリ、ECSフレームワークなど）から独立していることを目指します。

*   `domain/types.go`: **[データ]** ゲーム全体で使われる、`donburi`に依存しない基本的な型、定数、データ構造を定義します。
*   `ecs/components.go`: **[データ]** `donburi.Entity`など、ECSの概念に依存する型定義を定義します。
*   `ecs/events.go`: **[定義]** ゲーム内で発生するイベントの定義。
*   `ecs/ui_viewmodels.go`
    *   役割: UI表示に必要な整形済みデータ（ViewModel）の定義。
    *   内容: `InfoPanelViewModel`, `ActionModalViewModel`, `BattlefieldViewModel`など、UIがECSの内部構造に直接依存しないためのデータ構造を定義します。
*   `game_interfaces.go`: **[定義]** ゲーム全体で利用される主要なインターフェースを定義します。 `TargetingStrategy` や `TraitActionHandler` など、特定の振る舞いを抽象化するためのインターフェースが含まれます。

ECS (エンティティ・コンポーネント・システム)
---------------------------------------

ECSアーキテクチャの構成要素を定義します。

*   `ecs_components.go`: **[データ]** ECSの「C（コンポーネント）」を`donburi`に登録します。各コンポーネントが保持するデータ構造自体は`domain`パッケージで定義されます。
*   `ecs_setup_logic.go`: 戦闘開始時のエンティティ生成と初期コンポーネント設定。
*   `player_action_queue_component.go`: **[データ]** プレイヤーの行動実行待ちキューを保持するコンポーネントの定義。
*   `battle_action_queue_component.go`: **[データ]** 戦闘中の行動実行待ちキューを保持するコンポーネントの定義。

Scene (各画面の実装)
-------------------

ゲーム内の各画面（シーン）の実装です。

*   `scene.go`
    *   役割: すべてのシーンが満たすべき共通のルール（インターフェース）を定義します。
    *   内容: `Update`, `Draw` メソッドの型定義や、シーン間で共有するリソース（`SharedResources`）を定義します。
*   `scene_title.go`: タイトル画面の実装。
*   `scene_battle.go`: 戦闘シーンの統括。戦闘用のWorld（ECS）、UI、および戦闘全体の進行を管理するステートマシン（`GameState`）を保持します。`BattleScene`は、現在の状態（`GameState`）に応じて適切な処理（ゲージ進行、行動選択、アニメーションなど）を実行し、状態間の遷移を制御します。戦闘のコアな計算ロジックは`BattleLogic`構造体に集約されています。
*   `scene_customize.go`: メダロットのカスタマイズ画面の実装。
*   `scene_placeholder.go`: 未実装画面などのための、汎用的なプレースホルダー画面。

Battle Action (メダロットの行動)
---------------------------------------

戦闘中の各メダロットが実行するアクション定義や処理です。


*   `ai_action_selection.go`: **[ロジック/振る舞い]** AI制御のメダロットの行動選択ロジックを定義します。
*   `ai_personalities.go`: **[データ]** AIの性格定義と、それに対応する行動戦略をマッピングします。
*   `ai_target_strategies.go`: **[ロジック/振る舞い]** AIのターゲット選択戦略の具体的な実装を定義します。
*   `battle_action_queue_system.go`: **[ロジック/振る舞い]** 行動実行キューを処理し、適切な `ActionExecutor` を呼び出して行動を実行します。（チャージ開始ロジックは `charge_initiation_system.go` に移動しました）
*   `battle_action_executor.go`: **[ロジック/振る舞い]** アクションの実行に関する主要なロジックをカプセル化します。特性や武器タイプごとの具体的な処理は、`battle_trait_handlers.go` および `battle_weapon_effect_handlers.go` に委譲されます。
*   `battle_trait_handlers.go`: **[ロジック/振る舞い]** 各特性（Trait）に応じたアクションの実行ロジックを定義します。`BaseAttackHandler`、`SupportTraitExecutor`、`ObstructTraitExecutor` などが含まれます。
*   `battle_weapon_effect_handlers.go`: **[ロジック/振る舞い]** 各武器タイプ（WeaponType）に応じた追加効果の適用ロジックを定義します。`ThunderEffectHandler`、`MeltEffectHandler`、`VirusEffectHandler` などが含まれます。
*   `charge_initiation_system.go`: **[ロジック/振る舞い]** メダロットが行動を開始する際のチャージ状態の開始ロジックを管理します。`StartCharge` メソッドを提供します。
*   `post_action_effect_system.go`: **[ロジック/振る舞い]** アクション実行後のステータス効果の適用やパーツ破壊による状態遷移などを処理します。


Battle Logic & AI (戦闘ルールと思考)
---------------------------------

戦闘のコアロジックやAIの思考ルーチンです。

*   `battle_logger.go`: **[ロジック/振る舞い]** 戦闘中のログメッセージの生成と管理を行います。
*   `data_game_states.go`: **[ロジック/振る舞い]** 戦闘全体の進行を制御する各`GameState`（`GaugeProgressState`, `PlayerActionSelectState`, `ActionExecutionState`など）の具体的なロジックを実装します。各状態は、戦闘フローの特定のフェーズ（ゲージ進行、行動選択、アニメーションなど）を担当します。
*   `battle_logic.go`
    *   役割: 戦闘中のコアな計算ロジック（Calculator群）。
    *   内容: 戦闘に関連するヘルパー群（`DamageCalculator`, `HitCalculator`, `TargetSelector`, `PartInfoProvider`, `ChargeInitiationSystem`）を内包する`BattleLogic`構造体を定義します。これにより、`BattleScene`からの依存関係が単純化され、戦闘ロジックが一元管理されます。具体的な計算式や選択アルゴリズムは各ヘルパー内に実装されています。
*   `battle_damage_calculator.go`: **[ロジック/振る舞い]** ダメージ計算に関するロジックを扱います。
*   `battle_hit_calculator.go`: **[ロジック/振る舞い]** 命中・回避・防御判定に関するロジックを扱います。
*   `battle_part_info_provider.go`: **[ロジック/振る舞い]** パーツの状態や情報を取得・操作するロジックを扱います。
*   `battle_target_selector.go`: **[ロジック/振る舞い]** ターゲット選択やパーツ選択に関するロジックを扱います。
*   `battle_end_system.go`: **[ロジック/振る舞い]** ゲーム終了条件判定システム。`CheckGameEndSystem` を定義します。
*   `battle_gauge_system.go`: **[ロジック/振る舞い]** チャージゲージおよびクールダウンゲージの進行管理システム。`UpdateGaugeSystem` を定義します。
*   `battle_intention_system.go`: **[ロジック/振る舞い]** プレイヤーとAIの入力を処理し、行動の「意図（Intention）」を生成するシステムです。
*   `status_effect_system.go`: **[ロジック/振る舞い]** ステータス効果の適用、更新、解除を管理するシステム。
*   `battle_history_system.go`: **[ロジック/振る舞い]** アクションの結果に基づいてAIの行動履歴を更新するシステム。
*   `status_effect_implementations.go`: **[ロジック/振る舞い]** 各種ステータス効果の具体的な適用・解除ロジックを定義します。
*   `utils.go`: 文字列のパースや状態表示名取得などの汎用ユーティリティ関数。

UI (ユーザーインターフェース)
-----------------------

ゲームのユーザーインターフェース関連のファイルです。
UIはECSアーキテクチャの原則に基づき、ゲームロジックから明確に分離することを目標にしています。

*   `ui.go`
    *   役割: UI全体のレイアウトと管理、およびUIの状態管理（モーダルの表示状態など）。
    *   内容: EbitenUIのルートコンテナを構築し、各UI要素を配置します。UIイベントのハブとしても機能し、`BattleScene`に抽象化されたUIイベントを通知します。アニメーション描画の責務は`ui_animation_drawer.go`に委譲されています。
*   `ui_interfaces.go`
    *   役割: UIコンポーネントが満たすべきインターフェースを定義します。
    *   内容: `UIInterface`など、UIの描画とイベント処理に必要なメソッドを定義します。
*   `ui_factory.go`
    *   役割: UIコンポーネントの生成とスタイリングを一元的に管理するファクトリ。
    *   内容: `NewCyberpunkButton`など、共通のスタイルを持つUI要素を生成するメソッドを提供します。
*   `ui_image_generator.go`
    *   役割: UIコンポーネントの画像生成ロジックをカプセル化します。
    *   内容: サイバーパンク風のボタンやパネルの背景画像を生成するメソッドを提供します。
*   `ui_panel.go`
    *   役割: 汎用的なUIパネル（枠付きウィンドウ）の作成と管理。
    *   内容: 他のUI要素（情報パネル、アクションモーダル、メッセージウィンドウなど）の基盤となる、再利用可能なパネルコンポーネントを提供します。
*   `ui_view_model_factory.go`: **[ロジック/振る舞い]** ECSのデータからUI表示用のViewModelを構築するファクトリ。`InfoPanelViewModel`や`BattlefieldViewModel`など、UIが必要とする整形されたデータを生成します。これにより、UIはECSの内部構造に直接依存しません。
*   `ui_battlefield_widget.go`: 中央のバトルフィールド描画。ViewModelを受け取って描画します。
*   `ui_info_panels.go`: 左右の情報パネル（HPゲージなど）の作成と更新。ViewModelを受け取って描画します。
*   `ui_action_modal.go`: プレイヤーの行動選択モーダルウィンドウ。下部の共通パネル上に、背景を透過させてパーツ選択ボタンを表示します。UIイベントを発行し、ViewModelを使用して表示します。
*   `ui_action_modal_manager.go`: **[ロジック/振る舞い]** アクション選択モーダルの表示と状態を管理します。
*   `ui_message_window.go`: 画面下のメッセージウィンドウ。下部の共通パネル上に、背景を透過させてメッセージを表示します。
*   `ui_message_display_manager.go`: **[ロジック/振る舞い]** ゲーム内のメッセージ表示（キュー管理、ウィンドウ表示/非表示、進行）を管理します。
*   `ui_animation_drawer.go`: **[ロジック/振る舞い]** 戦闘中のアクションアニメーションの具体的な描画処理。
*   `ui_event_processor_system.go`: **[ロジック/振る舞い]** UIから発行されたイベント（プレイヤーの行動選択など）を処理し、対応するゲームイベントを発行します。ゲームロジックとは直接やり取りせず、イベントを介して間接的に連携します。
*   `ui_system.go`: **[ロジック/振る舞い]** UI関連のECSシステムを定義します。主に`UpdateInfoPanelViewModelSystem`が含まれ、情報パネルのViewModelを更新します。
*   `ui_target_indicator_manager.go`: **[ロジック/振る舞い]** ターゲットインジケーターの表示と状態を管理します。
*   `ui_events.go`: **[データ]** UIからゲームロジックへ通知されるイベントの定義。

Configuration & Resources (設定とリソース)
------------------------------------

ゲームの設定値や外部リソースの読み込み・管理に関するファイル群です。

*   `config_loader.go`: ゲームの固定設定値（画面サイズ、色など）をロードします。
*   `game_config.go`: ゲームバランスに関する設定値やUIの固定値など、アプリケーション全体の設定（`Config`構造体）を定義します。ドメインルールに関する定義は`domain/types.go`に分離されました。
*   `resource_ids.go`: `ebitengine-resource` ライブラリで使用するリソースIDを定義します。
*   `resource_loader.go`: `ebitengine-resource` を使用したゲームリソース（CSVデータ、フォントなど）の読み込みと管理。
*   `game_data_manager.go`: 静的なゲームデータ（パーツ定義、メダル定義など）の管理とアクセスを提供します。
*   `message_manager.go`: ゲーム内のメッセージテンプレートの読み込みとフォーマットを管理します。
*   `csv_saver.go`: メダロット構成のデータをCSVファイルに保存します。
