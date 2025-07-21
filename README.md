**【メダロットE】解説書**

**プロジェクト概要**

このプロジェクトは、人気ゲーム「メダロット」の戦闘システムをリスペクトした、EbitenとECS（エンティティ・コンポーネント・システム）アーキテクチャによるバトルシミュレーションゲームです。
ECSライブラリには [donburi](https://github.com/yohamta/donburi) を採用しており、データ指向の設計に基づいています。
リソース管理には [ebitengine-resource](https://github.com/quasilyte/ebitengine-resource) ライブラリを導入し、ゲーム内の各種アセット（CSVデータ、フォントなど）の効率的な読み込みと管理を実現しています。

**主要ファイル構成**

プロジェクトは、以下の主要なカテゴリのファイル群で構成されています。

Core & Entry Point (中核・起動)
-----------------------------

プログラムの起動とゲーム全体の管理を行います。

*   `main.go`
    *   役割: プログラムの起動点（エントリーポイント）。
    *   内容: ウィンドウの初期化、フォントや設定ファイルの読み込み、ゲーム全体のメインループを開始し、`SceneManager` をセットアップします。

Scene (各画面の実装)
-------------------

ゲーム内の各画面（シーン）の実装です。

*   `scene.go`
    *   役割: すべてのシーンが満たすべき共通のルール（インターフェース）を定義します。
    *   内容: `Update`, `Draw` メソッドの型定義や、シーン間で共有するリソース（`SharedResources`）を定義します。
*   `scene_title.go`: タイトル画面の実装。
*   `scene_battle.go`: 戦闘シーンの統括。戦闘用のWorld（ECS）、UI、ゲーム状態（StatePlaying, StateGameOverなど）を管理します。戦闘のコアロジックは`BattleLogic`構造体に集約されており、`BattleScene`はそれを介してダメージ計算やターゲット選択を行います。また、各戦闘システム（ゲージ進行、行動キュー処理など）を適切なタイミングで呼び出します。
*   `scene_customize.go`: メダロットのカスタマイズ画面の実装。
*   `scene_placeholder.go`: 未実装画面などのための、汎用的なプレースホルダー画面。

Battle Action (メダロットの行動)
---------------------------------------

戦闘中の各メダロットが実行するアクション定義や処理です。

*   `battle_action_formulas.go`: **[データ]** 戦闘アクション(行動)の計算式定義と管理。
*   `battle_action_queue_component.go`
    *   役割: **[データ]** 戦闘中の行動実行待ちキューを保持するコンポーネントの定義。
    *   内容: `ActionQueueComponentData` 構造体（行動するエンティティのキューを持つ）と、それに関連するComponentType、取得・初期化用関数を定義します。ワールドに一つ存在する専用エンティティがこのコンポーネントを持ちます。行動実行の順序制御に関わるデータを変更する場合に編集します。
*   `battle_action_queue_system.go`: **[ロジック/振る舞い]** 行動実行キューを処理し、適切な `ActionExecutor` を呼び出して行動を実行します。
*   `battle_action_executor.go`: **[ロジック/振る舞い]** パーツカテゴリ別（射撃、格闘など）の具体的な行動実行ロジックをカプセル化します。`ActionExecutor` インターフェースとその実装（`ShootActionExecutor`, `MeleeActionExecutor`など）を定義します。
*   `ui_event_processor_system.go`: **[ロジック/振る舞い]** UIから発行されたイベント（プレイヤーの行動選択など）を処理し、対応するゲームイベントを発行します。ゲームロジックとは直接やり取りせず、イベントを介して間接的に連携します。
*   `battle_animation_manager.go`: **[ロジック/振る舞い]** 戦闘中のアクションアニメーションの再生と管理を行います。

Battle Logic & AI (戦闘ルールと思考)
---------------------------------

戦闘のコアロジックやAIの思考ルーチンです。

*   `battle_logic.go`
    *   役割: 戦闘中のコアな計算ロジック（Calculator群）。
    *   内容: 戦闘に関連するヘルパー群（`DamageCalculator`, `HitCalculator`, `TargetSelector`, `PartInfoProvider`）を内包する`BattleLogic`構造体を定義します。これにより、`BattleScene`からの依存関係が単純化され、戦闘ロジックが一元管理されます。具体的な計算式や選択アルゴリズムは各ヘルパー内に実装されています。
*   `battle_end_system.go`: **[ロジック/振る舞い]** ゲーム終了条件判定システム。`CheckGameEndSystem` を定義します。
*   `battle_gauge_system.go`: **[ロジック/振る舞い]** チャージゲージおよびクールダウンゲージの進行管理システム。`UpdateGaugeSystem` を定義します。
*   `battle_intention_system.go`: **[ロジック/振る舞い]** プレイヤーとAIの入力を処理し、行動の「意図（Intention）」を生成するシステムです。
*   `battle_targeting_algorithm.go`: **[ロジック/振る舞い]** ターゲット選択の具体的なアルゴリズムを定義します。
*   `battle_targeting_strategy.go`: **[ロジック/振る舞い]** AIのターゲット選択戦略を定義します。

UI (ユーザーインターフェース)
-----------------------

ゲームのユーザーインターフェース関連のファイルです。
UIはECSアーキテクチャの原則に基づき、ゲームロジックから明確に分離することを目標にしています。

*   `ui_panel.go`
    *   役割: 汎用的なUIパネル（枠付きウィンドウ）の作成と管理。
    *   内容: 他のUI要素（情報パネル、アクションモーダル、メッセージウィンドウなど）の基盤となる、再利用可能なパネルコンポーネントを提供します。
*   `ui.go`
    *   役割: UI全体のレイアウトと管理、およびUIの状態管理（モーダルの表示状態など）。
    *   内容: EbitenUIのルートコンテナを構築し、各UI要素を配置します。UIイベントのハブとしても機能し、`BattleScene`に抽象化されたUIイベントを通知します。アニメーション描画の責務は`ui_animation_drawer.go`に委譲されています。
*   `ui_animation_drawer.go`
    *   役割: 戦闘中のアクションアニメーションの具体的な描画処理。
    *   内容: `BattleAnimationManager`から受け取ったアニメーションデータに基づき、画面へのエフェクト描画（座標計算、拡大縮小、フェードなど）を行います。
*   `ui_view_model_builder.go`
    *   役割: ECSのデータからUI表示用のViewModelを構築するヘルパー。
    *   内容: `InfoPanelViewModel`や`BattlefieldViewModel`など、UIが必要とする整形されたデータを生成します。これにより、UIはECSの内部構造に直接依存しません。
*   `ui_battlefield_widget.go`: 中央のバトルフィールド描画。ViewModelを受け取って描画します。
*   `ui_info_panels.go`: 左右の情報パネル（HPゲージなど）の作成と更新。ViewModelを受け取って描画します。
*   `ui_action_modal.go`: プレイヤーの行動選択モーダルウィンドウ。UIイベントを発行し、ViewModelを使用して表示します。
*   `ui_message_window.go`: 画面下のメッセージウィンドウ。
*   `ui_message_display_manager.go`: **[ロジック/振る舞い]** ゲーム内のメッセージ表示（キュー管理、ウィンドウ表示/非表示、進行）を管理します。

Data & Utilities (データ定義と補助機能)
------------------------------------

プロジェクト全体で使用するデータ構造や補助的な機能です。

*   `data_types.go`
    *   役割: プロジェクト全体で使われる基本的な型や定数、UI用のViewModel、およびUIイベントの定義。
    *   内容: `TeamID`, `PartSlotKey`, `StateType` のような型定義や、`const` で定義される定数を集約します。UIイベントの型定義も含まれます。
*   `data_config.go`
    *   役割: ゲームの固定設定値を管理します。
    *   内容: ゲームバランス（攻撃力係数など）、UIのサイズや色といった、マジックナンバーになりがちな値を一元管理します。
*   `data_csv_saver.go`
    *   役割: メダロット構成のデータをCSVファイルに保存します。
    *   内容: `scene_customize.go` から呼び出され、現在のメダロットのロードアウトを `data/medarots.csv` に保存します。
*   `data_resource_manager.go`
    *   役割: `ebitengine-resource` を使用したゲームリソース（CSVデータ、フォントなど）の読み込みと管理。
    *   内容: リソースの登録、キャッシュ、および `text.Face` への変換機能を提供します。CSVファイルのロード機能もここに集約されています。
*   `data_game_manager.go`
    *   役割: 静的なゲームデータ（パーツ定義、メダル定義、フォントなど）の管理。
    *   内容: `GlobalGameDataManager` インスタンスを通じて、ロードされた各種定義データへのアクセスを提供します。
*   `data_components.go`
    *   役割: **[データ]** の定義。ECSの「C（コンポーネント）」。
    *   内容: エンティティを構成する部品（`Settings`, `PartsComponentData`, `Gauge`など）、状態を示すタグ（`DefenseDebuffComponent`, `EvasionDebuffComponent`など）、行動の意図（`ActionIntent`）、ターゲット情報（`Target`）、AI関連データ（`AI`）のデータ構造をすべて定義します。新しいデータ、状態、エンティティに持たせる特性を追加したい場合は、まずこのファイルを編集します。
*   `data_setup.go`
    *   役割: 戦闘開始時のエンティティ生成と初期コンポーネント設定。
    *   内容: CSVから読み込んだデータをもとに、戦闘に参加する全メダロットのエンティティと、その初期状態に必要な各種コンポーネント（`Settings`, `Parts`, `Medal`, `AIComponent`、`ActionIntentComponent`、`TargetComponent`等）を作成・設定します。新しいコンポーネントをエンティティの初期状態に追加する場合や、初期設定ロジックを変更する場合に編集します。
*   `data_message_manager.go`: ゲーム内のメッセージを管理します。

	
**今後の検討事項**
*   **テストカバレッジの向上:** リファクタリングされたUIロジックとViewModelのテストを追加することで、コードの堅牢性を高めます。
