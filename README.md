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
    *   内容: ウィンドウの初期化、フォントや設定ファイルの読み込み（`ui.go`の`loadFont`経由）、ゲーム全体のメインループを開始します。
*   `game.go`
    *   役割: ゲーム全体のシーン（画面）を管理するシーンマネージャー。
    *   内容: 現在のシーンを保持し、シーン間の遷移（例：タイトル→バトル）を制御します。`Update` と `Draw` の司令塔です。
*   `config.go`
    *   役割: ゲームの固定設定値を管理します。
    *   内容: ゲームバランス（攻撃力係数など）、UIのサイズや色といった、マジックナンバーになりがちな値を一元管理します。

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

ECS (エンティティ・コンポーネント・システム)
---------------------------------------

ECSアーキテクチャの主要な要素です。データ（Component）、振る舞い（System）、そしてそれらの入れ物（Entity）を分離して管理します。

*   `components.go`
    *   役割: **[データ]** の定義。ECSの「C（コンポーネント）」。
    *   内容: エンティティを構成する部品（`Settings`, `PartsComponentData`, `Gauge`など）、状態を示すタグ（`DefenseDebuffComponent`, `ActingWithBerserkTraitTagComponent`など）、行動の意図（`ActionIntent`）、ターゲット情報（`Target`）、AI関連データ（`AI`）、戦闘計算修飾コンポーネント（`ActionModifierComponentData`）のデータ構造をすべて定義します。新しいデータ、状態、エンティティに持たせる特性を追加したい場合は、まずこのファイルを編集します。
*   `action_queue_component.go`
    *   役割: **[データ]** 戦闘中の行動実行待ちキューを保持するコンポーネントの定義。
    *   内容: `ActionQueueComponentData` 構造体（行動するエンティティのキューを持つ）と、それに関連するComponentType、取得・初期化用関数を定義します。ワールドに一つ存在する専用エンティティがこのコンポーネントを持ちます。行動実行の順序制御に関わるデータを変更する場合に編集します。
*   `ecs_setup.go`
    *   役割: 戦闘開始時のエンティティ生成と初期コンポーネント設定。
    *   内容: CSVから読み込んだデータをもとに、戦闘に参加する全メダロットのエンティティと、その初期状態に必要な各種コンポーネント（`Settings`, `Parts`, `Medal`, `AIComponent`、`ActionIntentComponent`、`TargetComponent`等）を作成・設定します。新しいコンポーネントをエンティティの初期状態に追加する場合や、初期設定ロジックを変更する場合に編集します。
*   `systems.go`
    *   役割: **[ロジック/振る舞い]** ECSの「S（システム）」のうち、現在は使用されていないファイル。
    *   内容: かつては多くの戦闘ロジックを含んでいましたが、リファクタリングにより主要な戦闘システムはそれぞれ専用のファイルに移管されました。現在は、将来的な共通システムの実装のために残されています。
*   `action_modifier_system.go`: **[ロジック/振る舞い]** 行動実行時の戦闘計算修飾システム。`ApplyActionModifiersSystem` と `RemoveActionModifiersSystem` を定義します。
*   `action_queue_system.go`: **[ロジック/振る舞い]** 行動実行キューの処理と、実際のアクション実行ロジック。`UpdateActionQueueSystem`, `executeActionLogic`, `StartCooldownSystem`, `StartCharge` を定義します。
*   `action_handler.go`: **[ロジック/振る舞い]** パーツカテゴリ別（射撃、格闘など）の行動処理戦略。`ActionHandler` インターフェースとその実装を定義します。
*   `game_end_system.go`: **[ロジック/振る舞い]** ゲーム終了条件判定システム。`CheckGameEndSystem` を定義します。
*   `gauge_system.go`: **[ロジック/振る舞い]** チャージゲージおよびクールダウンゲージの進行管理システム。`UpdateGaugeSystem` を定義します。
*   `state_change_systems.go`: **[ロジック/振る舞い]** 状態遷移時の副作用処理システム。`ProcessStateChangeSystem` を定義します。

Game Logic & AI (戦闘ルールと思考)
---------------------------------

戦闘のコアロジックやAIの思考ルーチンです。

*   `battle_logic.go`
    *   役割: 戦闘中のコアな計算ロジック（Calculator群）。
    *   内容: 戦闘に関連するヘルパー群（`DamageCalculator`, `HitCalculator`, `TargetSelector`, `PartInfoProvider`）を内包する`BattleLogic`構造体を定義します。これにより、`BattleScene`からの依存関係が単純化され、戦闘ロジックが一元管理されます。具体的な計算式や選択アルゴリズムは各ヘルパー内に実装されています。
*   `ai.go`
    *   役割: 敵（AI）の思考ルーチン。
    *   内容: `aiSelectAction`（AIの行動全体を決定するエントリーポイント）を定義します。性格別のターゲット選択戦略やパーツ選択戦略は`AIComponent`内にカプセル化され、`ecs_setup.go`でAIエンティティに動的に割り当てられます。
*   `ai_input_system.go`: **[ロジック/振る舞い]** AIの入力処理システム。`UpdateAIInputSystem` を定義します。
*   `player_actions.go`: プレイヤーの行動選択に関連するヘルパー関数（例：ランダムターゲット選択）。
*   `player_input_system.go`: **[ロジック/振る舞い]** プレイヤーの入力処理システム。`UpdatePlayerInputSystem` を定義します。

UI (ユーザーインターフェース)
-----------------------

ゲームのユーザーインターフェース関連のファイルです。
UIはECSアーキテクチャの原則に基づき、ゲームロジックから明確に分離されるようにリファクタリングされました。

*   `ui.go`
    *   役割: UI全体のレイアウトと管理、およびUIの状態管理（モーダルの表示状態など）。フォントの読み込みと管理も行います。
    *   内容: EbitenUIのルートコンテナを構築し、各UI要素を配置します。UIイベントのハブとしても機能し、`BattleScene`に抽象化されたUIイベントを通知します。
*   `ui_events.go`
    *   役割: (削除済み) UIからゲームロジックへ通知されるイベントの定義は`types.go`に移動しました。
*   `ui_event_system.go`
    *   役割: UIイベントを処理し、対応するゲームロジックをトリガーするシステム。
    *   内容: `BattleScene`から受け取ったUIイベントを解釈し、`StartCharge`などのゲームシステムを呼び出します。
*   `view_model_builder.go`
    *   役割: ECSのデータからUI表示用のViewModelを構築するヘルパー。
    *   内容: `InfoPanelViewModel`や`BattlefieldViewModel`など、UIが必要とする整形されたデータを生成します。これにより、UIはECSの内部構造に直接依存しません。
*   `battlefield_widget.go`: 中央のバトルフィールド描画。ViewModelを受け取って描画します。
*   `ui_info_panels.go`: 左右の情報パネル（HPゲージなど）の作成と更新。ViewModelを受け取って描画します。
*   `ui_action_modal.go`: プレイヤーの行動選択モーダルウィンドウ。UIイベントを発行し、ViewModelを使用して表示します。
*   `ui_message_window.go`: 画面下のメッセージウィンドウ。

Data & Utilities (データ定義と補助機能)
------------------------------------

プロジェクト全体で使用するデータ構造や補助的な機能です。

*   `types.go`
    *   役割: プロジェクト全体で使われる基本的な型や定数、UI用のViewModel、およびUIイベントの定義。
    *   内容: `TeamID`, `PartSlotKey`, `StateType` のような型定義や、`const` で定義される定数を集約します。UIイベントの型定義も含まれます。
*   `entity_utils.go`
    *   役割: エンティティやコンポーネント操作に関するユーティリティ関数。
    *   内容: `ChangeState`（状態遷移と一時タグ付与）、`ResetAllEffects` など、複数の場所から利用されるエンティティ操作関連のヘルパー関数をまとめます。
*   `csv_loader.go`
    *   役割: `data` フォルダにあるCSVファイルの読み込み/保存。
    *   内容: パーツ、メダル、メダロット構成のデータをファイルから読み込みます。
*   `game_data_manager.go`
    *   役割: 静的なゲームデータ（パーツ定義、メダル定義、フォントなど）の管理。
    *   内容: `GlobalGameDataManager` インスタンスを通じて、ロードされた各種定義データへのアクセスを提供します。

**今後の検討事項**

*   `ai.go`: AIの行動決定ロジックは、現在性格に基づいた戦略の組み合わせで動作していますが、今後はより複雑な状況判断（例：相手のパーツの特性や残弾数を考慮する、複数の敵の状況を比較して最適なターゲットを選ぶなど）を組み込むことで、さらに高度化する可能性があります。
*   **UIのさらなる抽象化:** `BattleScene`が`UIInterface`のみを介してUIとやり取りするようにリファクタリングが完了しました。これにより、UIの実装詳細からの疎結合が実現され、テスト容易性が向上しました。射撃アクション選択モーダルでのターゲットインジケーター表示不具合も修正済みです。
*   **テストカバレッジの向上:** リファクタリングされたUIロジックとViewModelのテストを追加することで、コードの堅牢性を高めます。
