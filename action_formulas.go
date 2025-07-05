package main

// FormulaManager はすべてのActionFormulaを管理します。
// ここで各特性（Trait）に対応する計算ルールを定義します。
// これらの値はゲームバランスに直接影響します。
var FormulaManager = make(map[Trait]*ActionFormula)

// init はゲーム起動時に自動的に呼び出され、すべての計算式を初期化します。
// ここで定義された値は、ゲームのバランス調整の主要なポイントとなります。
func init() {
	// 「撃つ」の計算式 (ボーナスなし)
	// この特性を持つパーツの攻撃は、基本の計算式のみを使用します。
	FormulaManager[TraitNormal] = &ActionFormula{
		ID:                 string(TraitNormal),
		SuccessRateBonuses: []BonusTerm{},
		PowerBonuses:       []BonusTerm{},
		CriticalRateBonus:  0.0, // クリティカル率への追加ボーナス（%）
		UserDebuffs:        []DebuffEffect{},
	}

	// 「狙い撃ち」の計算式
	// この特性を持つパーツの攻撃は、成功度とクリティカル率にボーナスを得て、自身にデバフがかかります。
	FormulaManager[TraitAim] = &ActionFormula{
		ID: string(TraitAim),
		SuccessRateBonuses: []BonusTerm{
			{SourceParam: Stability, Multiplier: 1.0}, // 成功度に「安定」をプラスする係数
		},
		PowerBonuses: []BonusTerm{},
		CriticalRateBonus:  50.0, // クリティカル率への追加ボーナス（%）
		UserDebuffs: []DebuffEffect{
			{Type: DebuffTypeEvasion, Multiplier: 0.5}, // チャージ中に自身にかかる回避デバフの乗数
		},
	}

	// 「殴る」の計算式
	// この特性を持つパーツの攻撃は、成功度とクリティカル率にボーナスを得て、自身にデバフがかかります。
	FormulaManager[TraitStrike] = &ActionFormula{
		ID: string(TraitStrike),
		SuccessRateBonuses: []BonusTerm{
			{SourceParam: Mobility, Multiplier: 1.0}, // 成功度に「機動」をプラスする係数
		},
		PowerBonuses: []BonusTerm{},
		CriticalRateBonus:  10.0, // クリティカル率への追加ボーナス（%）
		UserDebuffs: []DebuffEffect{
			{Type: DebuffTypeDefense, Multiplier: 0.5}, // チャージ中に自身にかかる防御デバフの乗数
		},
	}

	// 「我武者羅」の計算式
	// この特性を持つパーツの攻撃は、成功度と威力にボーナスを得て、自身にデバフがかかります。
	FormulaManager[TraitBerserk] = &ActionFormula{
		ID: string(TraitBerserk),
		SuccessRateBonuses: []BonusTerm{
			{SourceParam: Mobility, Multiplier: 1.0}, // 成功度に「機動」をプラスする係数
		},
		PowerBonuses: []BonusTerm{
			{SourceParam: Propulsion, Multiplier: 1.0}, // 威力に「推進」をプラスする係数
		},
		CriticalRateBonus:  0.0, // クリティカル率への追加ボーナス（%）
		UserDebuffs: []DebuffEffect{
			{Type: DebuffTypeEvasion, Multiplier: 0.5}, // チャージ中に自身にかかる回避デバフの乗数
			{Type: DebuffTypeDefense, Multiplier: 0.5}, // チャージ中に自身にかかる防御デバフの乗数
		},
	}

	// 他の特性もここに追加できます。
	// 新しい特性を追加する際は、対応するActionFormulaを定義してください。
}
