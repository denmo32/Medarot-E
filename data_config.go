package main

import (
	"github.com/ebitenui/ebitenui/widget"
	"image/color"
)

func LoadConfig() Config {
	screenWidth := 1280
	screenHeight := 720

	cfg := Config{
		Balance: BalanceConfig{
			// 時間に関する設定
			Time: struct {
				PropulsionEffectRate float64 // 脚部パーツの推進力がチャージ・クールダウン時間に与える影響度
				GameSpeedMultiplier  float64 // ゲーム全体の時間進行速度（大きいほど速い）
			}{
				PropulsionEffectRate: 0.01,
				GameSpeedMultiplier:  50,
			},
			HPAnimationSpeed: 1.0, // 1フレームあたりのHP変化量
			// 各種計算式の調整係数（現在は未使用ですが、将来的な拡張のために残されています）
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

			// ダメージ計算に関する設定
			Damage: struct {
				CriticalMultiplier     float64 // クリティカル時のダメージ倍率（現在は未使用）
				MedalSkillFactor       int     // メダルスキルがダメージに与える影響度（現在は未使用）
				DamageAdjustmentFactor float64 // ダメージ計算式の調整固定値（この値が大きいほどダメージが小さくなる）
				Critical               struct {
					BaseChance        float64 // クリティカル発生の基本確率（攻撃側の成功度0の場合の初期値）
					SuccessRateFactor float64 // 攻撃側の成功度がクリティカル率に与える影響度（成功度1につき何%上昇するか）
					MinChance         float64 // クリティカル発生率の最低保証値
					MaxChance         float64 // クリティカル発生率の最高保証値
				}
			}{
				CriticalMultiplier:     1.5,
				MedalSkillFactor:       2,
				DamageAdjustmentFactor: 10.0,
				Critical: struct {
					BaseChance        float64
					SuccessRateFactor float64
					MinChance         float64
					MaxChance         float64
				}{
					BaseChance:        5.0,
					SuccessRateFactor: 0.5,
					MinChance:         5.0,
					MaxChance:         95.0,
				},
			},
			// 命中判定に関する設定
			Hit: struct {
				BaseChance float64 // 命中判定の基本確率（攻撃側の成功度と防御側の回避度が同じ場合の初期値）
				MinChance  float64 // 命中率の最低保証値
				MaxChance  float64 // 命中率の最高保証値
			}{
				BaseChance: 50.0,
				MinChance:  5.0,
				MaxChance:  95.0,
			},
			// 自動防御に関する設定
			Defense: struct {
				BaseChance float64 // 自動防御成功の基本確率（防御側の防御度と攻撃側の成功度が同じ場合の初期値）
				MinChance  float64 // 自動防御成功率の最低保証値
				MaxChance  float64 // 自動防御成功率の最高保証値
			}{
				BaseChance: 10.0,
				MinChance:  5.0,
				MaxChance:  95.0,
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

	cfg.Balance.Formulas = map[Trait]ActionFormulaConfig{
		TraitNormal: {
			SuccessRateBonuses: []BonusTerm{},
			PowerBonuses:       []BonusTerm{},
			CriticalRateBonus:  0.0,
			UserDebuffs:        []DebuffEffect{},
		},
		TraitAim: {
			SuccessRateBonuses: []BonusTerm{{SourceParam: Stability, Multiplier: 1.0}},
			PowerBonuses:       []BonusTerm{},
			CriticalRateBonus:  50.0,
			UserDebuffs:        []DebuffEffect{{Type: DebuffTypeEvasion, Multiplier: 0.5}},
		},
		TraitStrike: {
			SuccessRateBonuses: []BonusTerm{{SourceParam: Mobility, Multiplier: 1.0}},
			PowerBonuses:       []BonusTerm{},
			CriticalRateBonus:  10.0,
			UserDebuffs:        []DebuffEffect{{Type: DebuffTypeDefense, Multiplier: 0.5}},
		},
		TraitBerserk: {
			SuccessRateBonuses: []BonusTerm{{SourceParam: Mobility, Multiplier: 1.0}},
			PowerBonuses:       []BonusTerm{{SourceParam: Propulsion, Multiplier: 1.0}},
			CriticalRateBonus:  0.0,
			UserDebuffs: []DebuffEffect{
				{Type: DebuffTypeEvasion, Multiplier: 0.5},
				{Type: DebuffTypeDefense, Multiplier: 0.5},
			},
		},
		TraitSupport: {
			SuccessRateBonuses: []BonusTerm{},
			PowerBonuses:       []BonusTerm{},
			CriticalRateBonus:  0.0,
			UserDebuffs:        []DebuffEffect{},
		},
	}

	return cfg
}
