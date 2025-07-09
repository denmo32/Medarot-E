package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ResetAllEffects は全ての効果をリセットします。
func ResetAllEffects(world donburi.World) {
	query.NewQuery(filter.Contains(DefenseDebuffComponent)).Each(world, func(e *donburi.Entry) {
		e.RemoveComponent(DefenseDebuffComponent)
	})
	query.NewQuery(filter.Contains(EvasionDebuffComponent)).Each(world, func(e *donburi.Entry) {
		e.RemoveComponent(EvasionDebuffComponent)
	})
	log.Println("すべての一時的な効果がリセットされました。")
}
