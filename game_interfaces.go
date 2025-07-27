package main

import (
	"github.com/yohamta/donburi"
)

// TargetingStrategy はAIのターゲット選択アルゴリズムをカプセル化するインターフェースです。
type TargetingStrategy interface {
	SelectTarget(
		world donburi.World,
		actingEntry *donburi.Entry,
		battleLogic *BattleLogic,
	) (*donburi.Entry, PartSlotKey)
}

// TraitActionHandler はカテゴリ固有のアクション処理全体をカプセル化します。
// ActionResultを返し、副作用をなくします。
type TraitActionHandler interface {
	Execute(
		actingEntry *donburi.Entry,
		world donburi.World,
		intent *ActionIntent,
		battleLogic *BattleLogic,
		gameConfig *Config,
		actingPartDef *PartDefinition,
	) ActionResult
}

// WeaponTypeEffectHandler は weapon_type 固有の追加効果を処理します。
// ActionResult を受け取り、デバフ付与などの副作用を適用します。
type WeaponTypeEffectHandler interface {
	ApplyEffect(result *ActionResult, world donburi.World, battleLogic *BattleLogic, actingPartDef *PartDefinition)
}

// PartInfoProviderInterface はパーツの状態や情報を取得・操作するロジックのインターフェースです。
type PartInfoProviderInterface interface {
	// パーツのパラメータ値を取得するメソッド
	GetPartParameterValue(entry *donburi.Entry, partSlot PartSlotKey, param PartParameter) float64

	// パーツスロットを検索するメソッド
	FindPartSlot(entry *donburi.Entry, partToFindInstance *PartInstanceData) PartSlotKey

	// 利用可能な攻撃パーツを取得するメソッド
	GetAvailableAttackParts(entry *donburi.Entry) []AvailablePart

	// 全体的な推進力と機動力を取得するメソッド
	GetOverallPropulsion(entry *donburi.Entry) int
	GetOverallMobility(entry *donburi.Entry) int

	// 脚部パーツの定義を取得するメソッド
	GetLegsPartDefinition(entry *donburi.Entry) (*PartDefinition, bool)

	// 成功度、回避度、防御度を取得するメソッド
	GetSuccessRate(entry *donburi.Entry, actingPartDef *PartDefinition, selectedPartKey PartSlotKey) float64
	GetEvasionRate(entry *donburi.Entry) float64
	GetDefenseRate(entry *donburi.Entry) float64

	// チームの命中率バフ乗数を取得するメソッド
	GetTeamAccuracyBuffMultiplier(entry *donburi.Entry) float64

	// バフを削除するメソッド
	RemoveBuffsFromSource(entry *donburi.Entry, partInst *PartInstanceData) 

	// ゲージの持続時間を計算するメソッド
	CalculateGaugeDuration(baseSeconds float64, entry *donburi.Entry) float64

	// メダロットのX座標を計算するメソッド
	CalculateMedarotXPosition(entry *donburi.Entry, battlefieldWidth float32) float32

	// GameDataManagerへのアクセスを提供するメソッド
	GetGameDataManager() *GameDataManager
}

// StatusEffect は、すべてのステータス効果（バフ・デバフ）が実装すべきインターフェースです。
type StatusEffect interface {
	Apply(world donburi.World, target *donburi.Entry)
	Remove(world donburi.World, target *donburi.Entry)
	Description() string
	Duration() int    // 効果の持続時間（ターン数や秒数など）。0の場合は永続、または即時解除。
	Type() DebuffType // 効果の種類を返すメソッドを追加
}
