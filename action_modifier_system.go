package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// ApplyActionModifiersSystem は、行動エンティティの特性やメダルなどに基づいて、
// 一時的なアクション修飾子を計算して適用します。
// このシステムは、命中/ダメージ計算の前に呼び出す必要があります。
func ApplyActionModifiersSystem(
	world donburi.World, // ワールドは、エフェクトがグローバルな状態や他のエンティティに依存する場合に必要になることがあります
	actingEntry *donburi.Entry,
	gameConfig *Config, // バランス数値へのアクセス用
	partInfoProvider *PartInfoProvider, // バーサークの推進力などのため
) {
	if actingEntry == nil || !actingEntry.Valid() {
		return
	}

	modifiers := ActionModifierComponentData{
		// デフォルト値または中立値で初期化
		CriticalRateBonus:     0,
		CriticalMultiplier:    0, // 0 は gameConfig.Balance.Damage.CriticalMultiplier を使用することを意味します
		PowerAdditiveBonus:    0,
		PowerMultiplierBonus:  1.0, // 乗数は1.0から開始（変更なし）
		DamageAdditiveBonus:   0,
		DamageMultiplierBonus: 1.0,
		AccuracyAdditiveBonus: 0,
	}

	settings := SettingsComponent.Get(actingEntry) // ログ用

	// 特性からの修飾子を適用
	if actingEntry.HasComponent(ActingWithAimTraitTagComponent) {
		// AIM特性は射撃カテゴリパーツに特有ですが、パーツにAIMがあればタグが追加されます。
		// AIMのクリティカルボーナスに対する射撃カテゴリのチェックは、通常DamageCalculatorで行われます。
		// ここでは、AIMタグが存在すればボーナスを適用するだけです。
		// 設定ファイルが実際のボーナス値を保持します。
		modifiers.CriticalRateBonus += gameConfig.Balance.Effects.Aim.CriticalRateBonus
		log.Printf("%s: AIM特性によりクリティカル率ボーナス+%d適用", settings.Name, gameConfig.Balance.Effects.Aim.CriticalRateBonus)
	}

	if actingEntry.HasComponent(ActingWithBerserkTraitTagComponent) {
		// BERSERK特性: 推進力を威力に加算します。
		// これは加算的な威力ボーナスです。
		if partInfoProvider != nil {
			propulsion := partInfoProvider.GetOverallPropulsion(actingEntry)
			powerBonusFromPropulsion := float64(propulsion) * gameConfig.Balance.Factors.BerserkPowerPropulsionFactor
			modifiers.PowerAdditiveBonus += int(powerBonusFromPropulsion) // BerserkPowerPropulsionFactorが整数スケールのボーナスになると仮定
			log.Printf("%s: BERSERK特性により推進力(%d)から威力ボーナス+%d適用", settings.Name, propulsion, int(powerBonusFromPropulsion))
		}
	}

	// MedalComponentからの修飾子を適用（例：スキルレベルが威力やクリティカルに影響）
	if medalComp := MedalComponent.Get(actingEntry); medalComp != nil {
		// 例：メダルスキルレベルが威力に加算される（既にDamageCalculatorにあるが、
		// すべての修飾子を一元化したい場合はここに移動可能）
		// 現状では、メダルスキル係数は後でDamageCalculatorで適用されるか、
		// ここで加算ボーナスとして扱われると仮定します：
		// modifiers.PowerAdditiveBonus += medalComp.SkillLevel * gameConfig.Balance.Damage.MedalSkillFactor
		// log.Printf("%s: メダルスキルにより威力ボーナス+%d適用", settings.Name, medalComp.SkillLevel*gameConfig.Balance.Damage.MedalSkillFactor)

		// 例：メダルスキルレベルがクリティカル率に加算される（既にDamageCalculatorにある）
		// modifiers.CriticalRateBonus += medalComp.SkillLevel * 2 // 例の係数
	}

	// エンティティのActionModifierComponentを追加または更新
	if actingEntry.HasComponent(ActionModifierComponent) {
		ActionModifierComponent.SetValue(actingEntry, modifiers)
	} else {
		donburi.Add(actingEntry, ActionModifierComponent, &modifiers)
	}
	log.Printf("%s: ActionModifierComponent更新完了: %+v", settings.Name, modifiers)
}

// RemoveActionModifiersSystem はエンティティから一時的なActionModifierComponentを削除します。
// これは、命中/ダメージ計算が完了した後（例：StartCooldownSystemまたはexecuteActionLogicの最後）に呼び出す必要があります。
func RemoveActionModifiersSystem(actingEntry *donburi.Entry) {
	if actingEntry == nil || !actingEntry.Valid() {
		return
	}
	if actingEntry.HasComponent(ActionModifierComponent) {
		actingEntry.RemoveComponent(ActionModifierComponent)
		// ログ用にSettingsComponentを正しく取得
		if actingEntry.HasComponent(SettingsComponent) {
			settingsComp := SettingsComponent.Get(actingEntry)
			log.Printf("%s: ActionModifierComponent解除", settingsComp.Name)
		} else {
			// SettingsComponentが何らかの理由で存在しない場合のフォールバックログ（メダロットでは起こらないはず）
			log.Println("ActionModifierComponent解除 (対象エンティティ名不明)")
		}
	}
}
