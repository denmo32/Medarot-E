package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// GetPlayerActionQueueComponent はワールド状態エンティティから PlayerActionQueueComponentData を取得します。
// PlayerActionQueueComponentType を持つエンティティが1つだけ存在することを期待します。
func GetPlayerActionQueueComponent(world donburi.World) *PlayerActionQueueComponentData {
	entry, ok := query.NewQuery(filter.Contains(PlayerActionQueueComponent)).First(world)
	if !ok {
		log.Panicln("PlayerActionQueueComponent がワールドに見つかりません。ワールド状態エンティティで初期化する必要があります。")
		return nil
	}
	return PlayerActionQueueComponent.Get(entry)
}

// EnsurePlayerActionQueueEntity は PlayerActionQueueComponentType と worldStateTag を持つエンティティが存在することを保証します。
// 存在しない場合は作成します。これは通常、セットアップ時に一度だけ呼び出されます。
func EnsurePlayerActionQueueEntity(world donburi.World) *donburi.Entry {
	entry, ok := query.NewQuery(filter.And(filter.Contains(PlayerActionQueueComponent), filter.Contains(worldStateTag))).First(world)
	if ok {
		return entry
	}

	log.Println("PlayerActionQueueComponent と worldStateTag を持つ PlayerActionQueueEntity を作成します。")
	newEntry := world.Entry(world.Create(PlayerActionQueueComponent, worldStateTag))
	PlayerActionQueueComponent.SetValue(newEntry, PlayerActionQueueComponentData{
		Queue: make([]*donburi.Entry, 0),
	})
	return newEntry
}
