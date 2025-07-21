package battle

import (
	"medarot-ebiten/internal/game"
)

func LoadConfig() game.Config {
	return game.Config{
		Balance: game.BalanceConfig{
			// 時間に関する設定
			Time: struct {
				PropulsionEffectRate float64 // 脚部パーツの推進力がチャージ・クールダウン時間に与える影響度
				GameSpeedMultiplier  float64 // ゲーム全体の時間進行速度（大きいほど速い）
			}{
				PropulsionEffectRate: 0.1,
				GameSpeedMultiplier:  10,
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
	}
}
