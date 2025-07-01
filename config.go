package main

import (
	"github.com/ebitenui/ebitenui/widget"
	"image/color"
)

func LoadConfig() Config {
	screenWidth := 1280
	screenHeight := 720

	return Config{
		Balance: BalanceConfig{
			Time: struct {
				PropulsionEffectRate float64
				GameSpeedMultiplier  float64
			}{
				PropulsionEffectRate: 0.01,
				GameSpeedMultiplier:  50,
			},
			// ★★★ この部分を正しく設定 ★★★
			Factors: struct {
				AccuracyStabilityFactor      float64
				EvasionStabilityFactor       float64
				DefenseStabilityFactor       float64
				PowerStabilityFactor         float64
				MeleeAccuracyMobilityFactor  float64
				BerserkPowerPropulsionFactor float64
			}{
				AccuracyStabilityFactor:      0.5,
				EvasionStabilityFactor:       0.5,
				DefenseStabilityFactor:       0.5,
				PowerStabilityFactor:         0.2,
				MeleeAccuracyMobilityFactor:  1.0,
				BerserkPowerPropulsionFactor: 1.0,
			},
			Effects: struct {
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
			}{
				Melee: struct {
					DefenseRateDebuff float64
					CriticalRateBonus int
				}{
					DefenseRateDebuff: 0.5,
					CriticalRateBonus: 10,
				},
				Berserk: struct {
					DefenseRateDebuff float64
					EvasionRateDebuff float64
				}{
					DefenseRateDebuff: 0.5,
					EvasionRateDebuff: 0.5,
				},
				Shoot: struct{}{},
				Aim: struct {
					EvasionRateDebuff float64
					CriticalRateBonus int
				}{
					EvasionRateDebuff: 0.5,
					CriticalRateBonus: 15,
				},
			},
			Damage: struct {
				CriticalMultiplier float64
				MedalSkillFactor   int
			}{
				CriticalMultiplier: 1.5,
				MedalSkillFactor:   2,
			},
		},
		UI: UIConfig{
			Screen: struct {
				Width  int
				Height int
			}{
				Width:  screenWidth,
				Height: screenHeight,
			},
			Battlefield: struct {
				Rect                   *widget.Container
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
			}{
				Height:                 float32(screenHeight) * 0.5,
				Team1HomeX:             float32(screenWidth) * 0.1,
				Team2HomeX:             float32(screenWidth) * 0.9,
				Team1ExecutionLineX:    float32(screenWidth) * 0.4,
				Team2ExecutionLineX:    float32(screenWidth) * 0.6,
				IconRadius:             12,
				HomeMarkerRadius:       15,
				LineWidth:              2,
				MedarotVerticalSpacing: float32(screenHeight) * 0.5 / float32(PlayersPerTeam+1),
				TargetIndicator: struct {
					Width  float32
					Height float32
				}{
					Width:  15,
					Height: 12,
				},
			},
			InfoPanel: struct {
				Padding           int
				BlockWidth        float32
				BlockHeight       float32
				PartHPGaugeWidth  float32
				PartHPGaugeHeight float32
			}{
				Padding:           10,
				BlockWidth:        200,
				BlockHeight:       200,
				PartHPGaugeWidth:  120,
				PartHPGaugeHeight: 10,
			},
			ActionModal: struct {
				ButtonWidth   float32
				ButtonHeight  float32
				ButtonSpacing int
			}{
				ButtonWidth:   250,
				ButtonHeight:  40,
				ButtonSpacing: 10,
			},
			Colors: struct {
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
			}{
				White:      color.White,
				Red:        color.RGBA{R: 255, G: 100, B: 100, A: 255},
				Blue:       color.RGBA{R: 100, G: 100, B: 255, A: 255},
				Yellow:     color.RGBA{R: 255, G: 255, B: 100, A: 255},
				Gray:       color.RGBA{R: 150, G: 150, B: 150, A: 255},
				Team1:      color.RGBA{R: 50, G: 150, B: 255, A: 255},
				Team2:      color.RGBA{R: 255, G: 50, B: 50, A: 255},
				Leader:     color.RGBA{R: 255, G: 215, B: 0, A: 255},
				Broken:     color.RGBA{R: 80, G: 80, B: 80, A: 255},
				HP:         color.RGBA{R: 0, G: 200, B: 100, A: 255},
				HPCritical: color.RGBA{R: 255, G: 100, B: 0, A: 255},
				Background: color.RGBA{R: 30, G: 30, B: 40, A: 255},
			},
		},
	}
}