package battle

import "medarot-ebiten/internal/game"

// FormulaManager はすべてのActionFormulaを管理します。
// ここで各特性（Trait）に対応する計算ルールを定義します。
// これらの値はゲームバランスに直接影響します。
var FormulaManager = make(map[game.Trait]*ActionFormula)

// SetupFormulaManager は Config から FormulaManager を初期化します。
func SetupFormulaManager(cfg *game.Config) {
	FormulaManager = make(map[game.Trait]*ActionFormula)
	for trait, formulaCfg := range cfg.Balance.Formulas {
		FormulaManager[trait] = &ActionFormula{
			ID:                 string(trait),
			SuccessRateBonuses: formulaCfg.SuccessRateBonuses,
			PowerBonuses:       formulaCfg.PowerBonuses,
			CriticalRateBonus:  formulaCfg.CriticalRateBonus,
			UserDebuffs:        formulaCfg.UserDebuffs,
		}
	}
}
