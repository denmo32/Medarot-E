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
    *   内容: ウィンドウの初期化、フォントや設定ファイルの読み込み、ゲーム全体のメインループを開始します。
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
*   `scene_battle.go`: 戦闘シーンの統括。戦闘用のWorld（ECS）、UI、ゲーム状態（StatePlaying, StateGameOverなど）を管理し、各戦闘システムを適切なタイミングで呼び出します。戦闘の進行ロジックの多くは、各種戦闘システムファイルに移譲されています。
*   `scene_customize.go`: メダロットのカスタマイズ画面の実装。
*   `scene_placeholder.go`: 未実装画面などのための、汎用的なプレースホルダー画面。

ECS (エンティティ・コンポーネント・システム)
---------------------------------------

ECSアーキテクチャの主要な要素です。

*   `components.go`
    *   役割: **[データ]** の定義。ECSの「C（コンポーネント）」。
    *   内容: エンティティを構成する部品（`Settings`, `PartsComponentData`, `Gauge`など）や、状態を示すタグ（`IdleStateComponent`, `DefenseDebuffComponent`, `ActingWithBerserkTraitTagComponent`, `JustBecameIdleTagComponent`など）、AI戦略コンポーネント（`TargetingStrategyComponentData`, `AIPartSelectionStrategyComponentData`）、戦闘計算修飾コンポーネント（`ActionModifierComponentData`）のデータ構造をすべて定義します。新しいデータ、状態、エンティティに持たせる特性を追加したい場合は、まずこのファイルを編集します。
*   `action_queue_component.go`
    *   役割: **[データ]** 戦闘中の行動実行待ちキューを保持するコンポーネントの定義。
    *   内容: `ActionQueueComponentData` 構造体（行動するエンティティのキューを持つ）と、それに関連するComponentType、取得・初期化用関数を定義します。ワールドに一つ存在する専用エンティティがこのコンポーネントを持ちます。行動実行の順序制御に関わるデータを変更する場合に編集します。
*   `ecs_setup.go`
    *   役割: 戦闘開始時のエンティティ生成と初期コンポーネント設定。
    *   内容: CSVから読み込んだデータをもとに、戦闘に参加する全メダロットのエンティティと、その初期状態に必要な各種コンポーネント（`Settings`, `Parts`, `Medal`, AI戦略コンポーネント等）を作成・設定します。新しいコンポーネントをエンティティの初期状態に追加する場合や、初期設定ロジックを変更する場合に編集します。
*   `systems.go`
    *   役割: **[ロジック/振る舞い]** ECSの「S（システム）」のうち、現在は使用されていないファイル。
    *   内容: かつては多くの戦闘ロジックを含んでいましたが、リファクタリングにより主要な戦闘システムはそれぞれ専用のファイルに移管されました。このファイルは現在、具体的な処理を持っていません。
*   `action_modifier_system.go`: **[ロジック/振る舞い]** 行動実行時の戦闘計算修飾システム。`ApplyActionModifiersSystem` と `RemoveActionModifiersSystem` を定義します。
*   `action_queue_system.go`: **[ロジック/振る舞い]** 行動実行キューの処理と、実際のアクション実行ロジック。`UpdateActionQueueSystem`, `executeActionLogic`, `StartCooldownSystem`, `StartCharge` を定義します。
*   `action_handler.go`: **[ロジック/振る舞い]** パーツカテゴリ別（射撃、格闘など）の行動処理戦略。`ActionHandler` インターフェースとその実装を定義します。
*   `game_end_system.go`: **[ロジック/振る舞い]** ゲーム終了条件判定システム。`CheckGameEndSystem` を定義します。
*   `gauge_system.go`: **[ロジック/振る舞い]** チャージゲージおよびクールダウンゲージの進行管理システム。`UpdateGaugeSystem` を定義します。
*   `state_effect_system.go`: **[ロジック/振る舞い]** 状態遷移時の副作用処理システム。`ProcessStateEffectsSystem` を定義します。

Game Logic & AI (戦闘ルールと思考)
---------------------------------

戦闘のコアロジックやAIの思考ルーチンです。

*   `battle_logic.go`
    *   役割: 戦闘中のコアな計算ロジック（Calculator群）。
    *   内容: `DamageCalculator`, `HitCalculator`, `TargetSelector`, `PartInfoProvider` を定義します。これらは、具体的な計算式や選択アルゴリズムを実装し、アクション修飾システムやアクション実行ロジックから利用されます。
*   `ai.go`
    *   役割: 敵（AI）の思考ルーチン。
    *   内容: `aiSelectAction`（AIの行動全体を決定するエントリーポイント）、性格別ターゲット選択戦略関数群、パーツ選択戦略関数群を定義します。
*   `ai_input_system.go`: **[ロジック/振る舞い]** AIの入力処理システム。`UpdateAIInputSystem` を定義します。
*   `player_actions.go`: プレイヤーの行動選択に関連するヘルパー関数。
*   `player_input_system.go`: **[ロジック/振る舞い]** プレイヤーの入力処理システム。`UpdatePlayerInputSystem` を定義します。

UI (ユーザーインターフェース)
-----------------------

ゲームのユーザーインターフェース関連のファイルです。

*   `ui.go`: 戦闘画面UI全体のレイアウトと管理。
*   `battlefield_widget.go`: 中央のバトルフィールド描画。
*   `ui_info_panels.go`: 左右の情報パネル（HPゲージなど）の作成と更新。
*   `ui_action_modal.go`: プレイヤーの行動選択モーダルウィンドウ。
*   `ui_message_window.go`: 画面下のメッセージウィンドウ。

Data & Utilities (データ定義と補助機能)
------------------------------------

プロジェクト全体で使用するデータ構造や補助的な機能です。

*   `types.go`
    *   役割: プロジェクト全体で使われる基本的な型や定数の定義。
    *   内容: `TeamID`, `PartSlotKey`, `StateType` のような型定義や、`const` で定義される定数を集約します。
*   `entity_utils.go`
    *   役割: エンティティやコンポーネント操作に関するユーティリティ関数。
    *   内容: `ChangeState`（状態遷移と一時タグ付与）、`ResetAllEffects` など、複数の場所から利用されるエンティティ操作関連のヘルパー関数をまとめます。
*   `csv_loader.go`
    *   役割: `data` フォルダにあるCSVファイルの読み込み/保存。
    *   内容: パーツ、メダル、メダロット構成のデータをファイルから読み込みます。
*   `game_data_manager.go`
    *   役割: 静的なゲームデータ（パーツ定義、メダル定義など）の管理。
    *   内容: `GlobalGameDataManager` インスタンスを通じて、ロードされた各種定義データへのアクセスを提供します。

**今後の検討事項**

*   `action_handler.go`: `HandlerTargetingResult` 構造体 (旧 `ActionResult`) を、`action_queue_system.go` で定義されている `ActionResult` と共通化することを検討。両者で類似の情報を扱っているため、一元管理することでコードの重複を減らし、整合性を保ちやすくなります。
*   `ai.go`: AIのパーツ選択ロジックについて、現在は基本的な戦略（利用可能な最初のパーツを選択など）が実装されていますが、より高度な判断（例：相手の状況に応じたパーツ選択、メダルの性格に基づいた戦略的なパーツ選択など）を導入するために、Strategyパターンなどを活用した拡張を検討。