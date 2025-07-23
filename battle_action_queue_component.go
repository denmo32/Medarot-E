package main

import (
	"log"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"github.com/yohamta/donburi/query"
)

// ActionQueueComponentData は行動準備完了エンティティのキューを格納します。
type ActionQueueComponentData struct {
	Queue []*donburi.Entry
}

// ActionQueueComponentType は ActionQueueComponentData のコンポーネントタイプです。
var ActionQueueComponentType = donburi.NewComponentType[ActionQueueComponentData]()

// GetActionQueueComponent はワールド状態エンティティから ActionQueueComponentData を取得します。
// ActionQueueComponentType を持つエンティティが1つだけ存在することを期待します。
// 注意: worldStateTag のチェックは EnsureActionQueueEntity で行われます。
func GetActionQueueComponent(world donburi.World) *ActionQueueComponentData {
	// ActionQueueComponentType を持つエンティティをクエリします。
	// EnsureActionQueueEntity が既にそのようなエンティティを作成済みであると想定されます。
	// (おそらく worldStateTag も付与されているでしょう)。Get の簡潔さのため、ここではコンポーネントのみを探します。
	entry, ok := query.NewQuery(filter.Contains(ActionQueueComponentType)).First(world)
	if !ok {
		// これは NewBattleScene で正しく初期化されていれば起こりません。
		log.Panicln("ActionQueueComponent がワールドに見つかりません。ワールド状態エンティティで初期化する必要があります。")
		return nil // panic により到達不能のはずです
	}
	return ActionQueueComponentType.Get(entry)
}

// EnsureActionQueueEntity は ActionQueueComponentType と worldStateTag を持つエンティティが存在することを保証します。
// 存在しない場合は作成します。これは通常、セットアップ時に一度だけ呼び出されます。
func EnsureActionQueueEntity(world donburi.World) *donburi.Entry {
	// ActionQueueComponentType と worldStateTag の両方を持つエンティティをクエリします。
	entry, ok := query.NewQuery(filter.And(filter.Contains(ActionQueueComponentType), filter.Contains(worldStateTag))).First(world)
	if ok {
		return entry
	}

	log.Println("ActionQueueComponent と worldStateTag を持つ ActionQueueEntity を作成します。")
	newEntry := world.Entry(world.Create(ActionQueueComponentType, worldStateTag))
	ActionQueueComponentType.SetValue(newEntry, ActionQueueComponentData{
		Queue: make([]*donburi.Entry, 0),
	})
	return newEntry
}
