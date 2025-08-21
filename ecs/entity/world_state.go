package entity

import (
	"log"

	"medarot-ebiten/donburi"
	"medarot-ebiten/ecs/component"
)

// GetActionQueueComponent はワールド状態エンティティから ActionQueueComponentData を取得します。
// ActionQueueComponentType を持つエンティティが1つだけ存在することを期待します。
// 注意: worldStateTag のチェックは EnsureActionQueueEntity で行われます。
func GetActionQueueComponent(world donburi.World) *component.ActionQueueComponentData {
	// ActionQueueComponentType を持つエンティティをクエリします。
	// EnsureActionQueueEntity が既にそのようなエンティティを作成済みであると想定されます。
	// (おそらく worldStateTag も付与されているでしょう)。Get の簡潔さのため、ここではコンポーネントのみを探します。
	entry, ok := donburi.NewQuery(donburi.Contains(component.ActionQueueComponentType)).First(world)
	if !ok {
		// これは NewBattleScene で正しく初期化されていれば起こりません。
		log.Panicln("ActionQueueComponent がワールドに見つかりません。ワールド状態エンティティで初期化する必要があります。")
		return nil // panic により到達不能のはずです
	}
	return component.ActionQueueComponentType.Get(entry)
}

// EnsureActionQueueEntity は ActionQueueComponentType と worldStateTag を持つエンティティが存在することを保証します。
// 存在しない場合は作成します。これは通常、セットアップ時に一度だけ呼び出されます。
func EnsureActionQueueEntity(world donburi.World) *donburi.Entry {
	// ActionQueueComponentType と worldStateTag の両方を持つエンティティをクエリします。
	entry, ok := donburi.NewQuery(donburi.And(donburi.Contains(component.ActionQueueComponentType), donburi.Contains(component.WorldStateTag))).First(world)
	if ok {
		return entry
	}

	log.Println("ActionQueueComponent と worldStateTag を持つ ActionQueueEntity を作成します。")
	newEntry := world.Entry(world.Create(component.ActionQueueComponentType, component.WorldStateTag))
	component.ActionQueueComponentType.SetValue(newEntry, component.ActionQueueComponentData{
		Queue: make([]*donburi.Entry, 0),
	})
	return newEntry
}

// GetPlayerActionQueueComponent はワールド状態エンティティから PlayerActionQueueComponentData を取得します。
// PlayerActionQueueComponentType を持つエンティティが1つだけ存在することを期待します。
func GetPlayerActionQueueComponent(world donburi.World) *component.PlayerActionQueueComponentData {
	entry, ok := donburi.NewQuery(donburi.Contains(component.PlayerActionQueueComponent)).First(world)
	if !ok {
		log.Panicln("PlayerActionQueueComponent がワールドに見つかりません。ワールド状態エンティティで初期化する必要があります。")
		return nil
	}
	return component.PlayerActionQueueComponent.Get(entry)
}

// EnsurePlayerActionQueueEntity は PlayerActionQueueComponentType と worldStateTag を持つエンティティが存在することを保証します。
// 存在しない場合は作成します。これは通常、セットアップ時に一度だけ呼び出されます。
func EnsurePlayerActionQueueEntity(world donburi.World) *donburi.Entry {
	entry, ok := donburi.NewQuery(donburi.And(donburi.Contains(component.PlayerActionQueueComponent), donburi.Contains(component.WorldStateTag))).First(world)
	if ok {
		return entry
	}

	log.Println("PlayerActionQueueComponent と worldStateTag を持つ PlayerActionQueueEntity を作成します。")
	newEntry := world.Entry(world.Create(component.PlayerActionQueueComponent, component.WorldStateTag))
	component.PlayerActionQueueComponent.SetValue(newEntry, component.PlayerActionQueueComponentData{
		Queue: make([]*donburi.Entry, 0),
	})
	return newEntry
}
