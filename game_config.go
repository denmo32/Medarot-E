package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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



type AssetPaths struct {
	GameSettings string
	Messages     string
	MedalsCSV    string
	PartsCSV     string
	MedarotsCSV  string
	FormulasJSON string
	Font         string
	Image        string
}

type Config struct {
	Balance    BalanceConfig
	UI         UIConfig
	AssetPaths AssetPaths // 新しく追加
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
