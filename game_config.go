package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/yohamta/donburi"
)

// PartParameter はパーツのどの数値を参照するかを示す型です
type PartParameter string

const (
	Power      PartParameter = "Power"
	Accuracy   PartParameter = "Accuracy"
	Mobility   PartParameter = "Mobility"
	Propulsion PartParameter = "Propulsion"
	Stability  PartParameter = "Stability"
	Defense    PartParameter = "Defense"
)

// BonusTerm は計算式のボーナス項を定義します
type BonusTerm struct {
	SourceParam PartParameter // どのパラメータを参照するか
	Multiplier  float64       // 乗数
}

// DebuffEffect は発生するデバフ効果を定義します
type DebuffEffect struct {
	Type       DebuffType // デバフの種類
	Multiplier float64    // 効果量（乗数）
}

// ActionFormula はアクションの計算ルール全体を定義します
type ActionFormula struct {
	ID                 string
	SuccessRateBonuses []BonusTerm    // 成功度へのボーナスリスト
	PowerBonuses       []BonusTerm    // 威力度へのボーナスリスト
	CriticalRateBonus  float64        // クリティカル率へのボーナス
	UserDebuffs        []DebuffEffect // チャージ中に自身にかかるデバフのリスト
}

type ActionFormulaConfig struct {
	SuccessRateBonuses []BonusTerm
	PowerBonuses       []BonusTerm
	CriticalRateBonus  float64
	UserDebuffs        []DebuffEffect
}

// BalanceConfig 構造体を新しいルールに合わせて拡張します。
type BalanceConfig struct {
	Time struct {
		PropulsionEffectRate float64
		GameSpeedMultiplier  float64
	}
	HPAnimationSpeed float64 // HPゲージアニメーション速度 (1フレームあたりのHP変化量)
	Factors          struct {
		AccuracyStabilityFactor      float64
		EvasionStabilityFactor       float64
		DefenseStabilityFactor       float64
		PowerStabilityFactor         float64
		MeleeAccuracyMobilityFactor  float64
		BerserkPowerPropulsionFactor float64
	}
	Effects struct {
		Melee struct {
			DefenseRateDebuff float64
			CriticalRateBonus int
		}
		Berserk struct {
			DefenseRateDebuff float64
			EvasionRateDebuff float64
		}
		Shoot struct{}
		Aim   struct {
			EvasionRateDebuff float64
			CriticalRateBonus int
		}
	}
	Damage struct {
		CriticalMultiplier     float64
		MedalSkillFactor       int
		DamageAdjustmentFactor float64
		Critical               struct {
			BaseChance        float64
			SuccessRateFactor float64
			MinChance         float64
			MaxChance         float64
		}
	}
	Hit struct {
		BaseChance float64
		MinChance  float64
		MaxChance  float64
	}
	Defense struct {
		BaseChance float64
		MinChance  float64
		MaxChance  float64
	}
	Formulas map[Trait]ActionFormulaConfig // 新しく追加
}

// AIPartSelectionStrategyFunc はAIパーツ選択戦略の関数シグネチャを定義します。
// 行動するAIエンティティと利用可能なパーツのリストを受け取り、選択されたパーツとそのスロットを返します。
type AIPartSelectionStrategyFunc func(
	actingEntry *donburi.Entry,
	availableParts []AvailablePart,
	world donburi.World,
	battleLogic *BattleLogic,
) (PartSlotKey, *PartDefinition)

// AIPersonality はAIの性格に関連する戦略をカプセル化します。
type AIPersonality struct {
	TargetingStrategy     TargetingStrategy
	PartSelectionStrategy AIPartSelectionStrategyFunc
}

// PersonalityRegistry は、性格名をキーとしてAIPersonalityを保持するグローバルなマップです。
var PersonalityRegistry = map[string]AIPersonality{
	"ハンター": {
		TargetingStrategy:     &HunterStrategy{},
		PartSelectionStrategy: SelectHighestPowerPart,
	},
	"クラッシャー": {
		TargetingStrategy:     &CrusherStrategy{},
		PartSelectionStrategy: SelectHighestPowerPart,
	},
	"ジョーカー": {
		TargetingStrategy:     &JokerStrategy{},
		PartSelectionStrategy: SelectFastestChargePart,
	},
	"リーダー": { // デフォルト/フォールバック用
		TargetingStrategy:     &LeaderStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"アシスト": {
		TargetingStrategy:     &AssistStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"カウンター": {
		TargetingStrategy:     &CounterStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"チェイス": {
		TargetingStrategy:     &ChaseStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"デュエル": {
		TargetingStrategy:     &DuelStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"フォーカス": {
		TargetingStrategy:     &FocusStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"ガード": {
		TargetingStrategy:     &GuardStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
	"インターセプト": {
		TargetingStrategy:     &InterceptStrategy{},
		PartSelectionStrategy: SelectFirstAvailablePart,
	},
}

type Config struct {
	Balance BalanceConfig
	UI      UIConfig
}

type UIConfig struct {
	Screen struct {
		Width  int
		Height int
	}
	Battlefield struct {
		Height                 float32
		Team1HomeX             float32
		Team2HomeX             float32
		Team1ExecutionLineX    float32
		Team2ExecutionLineX    float32
		IconRadius             float32
		HomeMarkerRadius       float32
		LineWidth              float32
		MedarotVerticalSpacing float32
		TargetIndicator        struct {
			Width  float32
			Height float32
		}
	}
	InfoPanel struct {
		Padding           int
		BlockWidth        float32
		BlockHeight       float32
		PartHPGaugeWidth  float32
		PartHPGaugeHeight float32
	}
	ActionModal struct {
		ButtonWidth   float32
		ButtonHeight  float32
		ButtonSpacing int
	}
	Colors struct {
		White      color.Color
		Red        color.Color
		Blue       color.Color
		Yellow     color.Color
		Gray       color.Color
		Team1      color.Color
		Team2      color.Color
		Leader     color.Color
		Broken     color.Color
		HP         color.Color
		HPCritical color.Color
		Background color.Color
	}
}

type SharedResources struct {
	GameData        *GameData
	Config          Config
	Font            text.Face
	GameDataManager *GameDataManager
	ButtonImage     *widget.ButtonImage
}
