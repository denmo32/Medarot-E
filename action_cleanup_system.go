package main

import (
	"log"

	"github.com/yohamta/donburi"
)

// CleanupActionSystem はアクション実行後のクリーンアップ処理を行います。
func CleanupActionSystem(actingEntry *donburi.Entry, world donburi.World) {
	settings := SettingsComponent.Get(actingEntry)

	// 行動中に付与された自身へのデバフを解除
	if actingEntry.HasComponent(DefenseDebuffComponent) {
		actingEntry.RemoveComponent(DefenseDebuffComponent)
		log.Printf("%s の防御デバフを解除", settings.Name)
	}
	if actingEntry.HasComponent(EvasionDebuffComponent) {
		actingEntry.RemoveComponent(EvasionDebuffComponent)
		log.Printf("%s の回避デバフを解除", settings.Name)
	}

	// BERSERK特性の効果を処理
	// if actingEntry.HasComponent(ActingWithBerserkTraitTagComponent) { ... } のような古いロジックは削除されました

	// ActingWithBerserkTraitTagComponent はリファクタリングにより不要になりました
	// if actingEntry.HasComponent(ActingWithBerserkTraitTagComponent) {
	// 	actingEntry.RemoveComponent(ActingWithBerserkTraitTagComponent)
	// }
}
