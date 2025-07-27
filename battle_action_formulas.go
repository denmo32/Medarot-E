package main

// SetupFormulaManager は Config から FormulaManager を初期化します。
func SetupFormulaManager(cfg *Config) {
	FormulaManager = make(map[Trait]ActionFormula)
	for trait, formulaCfg := range cfg.Balance.Formulas {
		FormulaManager[trait] = ActionFormula{
			ID:                 string(trait),
			SuccessRateBonuses: formulaCfg.SuccessRateBonuses,
			PowerBonuses:       formulaCfg.PowerBonuses,
			CriticalRateBonus:  formulaCfg.CriticalRateBonus,
			UserDebuffs:        formulaCfg.UserDebuffs,
		}
	}
}
