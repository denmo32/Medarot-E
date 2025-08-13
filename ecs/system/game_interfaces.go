package system

import (
	"image"
	"math/rand"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/event"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

// UIUpdater はUIの更新とイベント収集のインターフェースです。
type UIUpdater interface {
	SetViewModels(infoPanelVMs []core.InfoPanelViewModel, battlefieldVM core.BattlefieldViewModel)
	Update(tickCount int, world donburi.World) []event.GameEvent
	ProcessEvents(world donburi.World, events []event.GameEvent)
	EnqueueMessageQueue(messages []string, callback func())
	IsMessageFinished() bool
	SetCurrentTarget(entityID donburi.Entity)
	ClearCurrentTarget()
	SetAnimation(anim *component.ActionAnimationData)
	ClearAnimation()
	GetBattlefieldWidgetRect() image.Rectangle
	Draw(screen *ebiten.Image, tickCount int, gameDataManager *data.GameDataManager)
}

// ViewModelBuilder はViewModelを構築するインターフェースです。
type ViewModelBuilder interface {
	BuildInfoPanelViewModel(entry *donburi.Entry) (core.InfoPanelViewModel, error)
	BuildBattlefieldViewModel(world donburi.World, battlefieldRect image.Rectangle, config *data.Config) (core.BattlefieldViewModel, error)
	BuildActionModalViewModel(actingEntry *donburi.Entry, actionTargetMap map[core.PartSlotKey]core.ActionTarget) (core.ActionModalViewModel, error)
	GetAvailableAttackParts(entry *donburi.Entry) []core.AvailablePart
}

// TargetingStrategy はAIのターゲット選択アルゴリズムをカプセル化するインターフェースです。
type TargetingStrategy interface {
	SelectTarget(
		world donburi.World,
		actingEntry *donburi.Entry,
		battleLogic *BattleLogic,
	) (*donburi.Entry, core.PartSlotKey)
}

type BattleLogger interface {
	LogHitCheck(attackerName, targetName string, chance, successRate, evasion float64, roll int)
	LogDefenseCheck(targetName string, defenseRate, successRate, chance float64, roll int)
	LogCriticalHit(attackerName string, chance float64)
	LogPartBroken(medarotName, partName, partID string)
	LogActionInitiated(attackerName string, actionTrait core.Trait, weaponType core.WeaponType, category core.PartCategory)
	LogAttackMiss(attackerName, skillName, targetName string)
	LogDamageDealt(defenderName, targetPartType string, damage int)
	LogDefenseSuccess(targetName, defensePartName string, originalDamage, actualDamage int, isCritical bool)
}

// TraitActionHandler はカテゴリ固有のアクション処理全体をカプセル化します。
// ActionResultを返し、副作用をなくします。
type TraitActionHandler interface {
	Execute(
		actingEntry *donburi.Entry,
		world donburi.World,
		intent *core.ActionIntent,
		damageCalculator *DamageCalculator,
		hitCalculator *HitCalculator,
		targetSelector *TargetSelector,
		partInfoProvider PartInfoProviderInterface,
		actingPartDef *core.PartDefinition,
		rand *rand.Rand,
	) component.ActionResult
}

// WeaponTypeEffectHandler は weapon_type 固有の追加効果を処理します。
// ActionResult を受け取り、デバフ付与などの副作用を適用します。
type WeaponTypeEffectHandler interface {
	ApplyEffect(result *component.ActionResult, world donburi.World, damageCalculator *DamageCalculator, hitCalculator *HitCalculator, targetSelector *TargetSelector, partInfoProvider PartInfoProviderInterface, actingPartDef *core.PartDefinition, rand *rand.Rand)
}

// PartInfoProviderInterface はパーツの状態や情報を取得・操作するロジックのインターフェースです。
type PartInfoProviderInterface interface {
	// パーツのパラメータ値を取得するメソッド
	GetPartParameterValue(entry *donburi.Entry, partSlot core.PartSlotKey, param core.PartParameter) float64

	// パーツスロットを検索するメソッド
	FindPartSlot(entry *donburi.Entry, partToFindInstance *core.PartInstanceData) core.PartSlotKey

	// 利用可能な攻撃パーツを取得するメソッド
	GetAvailableAttackParts(entry *donburi.Entry) []core.AvailablePart

	// 全体的な推進力と機動力を取得するメソッド
	GetOverallPropulsion(entry *donburi.Entry) int
	GetOverallMobility(entry *donburi.Entry) int

	// 脚部パーツの定義を取得するメソッド
	GetLegsPartDefinition(entry *donburi.Entry) (*core.PartDefinition, bool)

	// 成功度、回避度、防御度を取得するメソッド
	GetSuccessRate(entry *donburi.Entry, actingPartDef *core.PartDefinition, selectedPartKey core.PartSlotKey) float64
	GetEvasionRate(entry *donburi.Entry) float64
	GetDefenseRate(entry *donburi.Entry) float64

	// チームの命中率バフ乗数を取得するメソッド
	GetTeamAccuracyBuffMultiplier(entry *donburi.Entry) float64

	// バフを削除するメソッド
	RemoveBuffsFromSource(entry *donburi.Entry, partInst *core.PartInstanceData)

	// ゲージの持続時間を計算するメソッド
	CalculateGaugeDuration(baseSeconds float64, entry *donburi.Entry) float64

	// メダロットの正規化された行動進行度を取得するメソッド
	GetNormalizedActionProgress(entry *donburi.Entry) float32

	// GameDataManagerへのアクセスを提供するメソッド
	GetGameDataManager() *data.GameDataManager
}
