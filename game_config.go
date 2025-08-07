package main

import (
	"image/color"
	"math/rand" // 追加

	"medarot-ebiten/domain"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

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
	Formulas map[domain.Trait]domain.ActionFormulaConfig // 新しく追加
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
	Game       GameConfig // 追加
}

type GameConfig struct {
	RandomSeed int64
}

type UIConfig struct {
	Screen struct {
		Width  int
		Height int
	}
	Battlefield struct {
		Height                       float32
		Team1HomeX                   float32
		Team2HomeX                   float32
		Team1ExecutionLineX          float32
		Team2ExecutionLineX          float32
		IconRadius                   float32
		HomeMarkerRadius             float32
		LineWidth                    float32
		MedarotVerticalSpacingFactor float32
		TargetIndicator              struct {
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
		ButtonWidth         float32
		ButtonHeight        float32
		ButtonSpacing       int
		ModalButtonFontSize float64 // 追加
	}
	MessageWindow struct { // 追加
		MessageWindowFontSize float64 // 追加
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
		Black      color.Color
	}
}

type SharedResources struct {
	GameData          *domain.GameData
	Config            Config
	Font              text.Face
	ModalButtonFont   text.Face // 追加
	MessageWindowFont text.Face // 追加
	GameDataManager   *GameDataManager
	ButtonImage       *widget.ButtonImage
	Rand              *rand.Rand   // 追加
	BattleLogger      BattleLogger // 追加
}