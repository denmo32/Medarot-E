package main

import (
	"github.com/yohamta/donburi"
)

// ActionResult (再掲または共通化が必要か検討)
// action_queue_system.go から移動または共通パッケージにすることも検討。
// ここでは一旦、ActionHandler が返す結果として簡易的に定義。
// 実際には action_queue_system.go の ActionResult を利用/拡張する形が良い。
type HandlerTargetingResult struct {
	TargetEntity   *donburi.Entry
	TargetPartSlot PartSlotKey
	Success        bool
	LogMessage     string
}

// ActionHandler defines the interface for category-specific action processing.
type ActionHandler interface {
	// ResolveTarget determines the target(s) for an action based on the category.
	// actingEntry: The entity performing the action.
	// world: The game world.
	// actionComp: The ActionComponent of the acting entity, which might contain pre-selected target info (e.g., for player-controlled shoot).
	// targetSelector: Helper for finding targets.
	// Returns the chosen target entity and target part slot, or an error/log.
	ResolveTarget(
		actingEntry *donburi.Entry,
		world donburi.World,
		actionComp *Action, // ActionComponent.Get(actingEntry)
		targetSelector *TargetSelector,
		partInfoProvider *PartInfoProvider, // May be needed for some target logic
	) HandlerTargetingResult

	// CanExecute (オプション): 行動実行が可能かどうかの事前チェック
	// (例: 射撃なら弾数があるか、など。今回はスコープ外とする可能性)
	// CanExecute(actingEntry *donburi.Entry, world donburi.World, actingPart *Part) bool

	// Execute (オプション): カテゴリ固有の実行ロジックの主要部分。
	// 多くのロジックは executeActionLogic に集約されているため、
	// このメソッドは限定的な役割になるか、あるいは不要かもしれない。
	// executeActionLogic がこのハンドラから情報を得る形にする。
	// 現状では ResolveTarget に集中する。
}

// --- Concrete Handlers ---

// ShootActionHandler handles actions for SHOOT category parts.
type ShootActionHandler struct{}

func (h *ShootActionHandler) ResolveTarget(
	actingEntry *donburi.Entry,
	world donburi.World,
	actionComp *Action,
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) HandlerTargetingResult {
	settings := SettingsComponent.Get(actingEntry)
	// For SHOOT, a target entity and part slot should typically be pre-selected
	// (either by player or AI's initial targeting strategy).
	if actionComp.TargetEntity == nil || actionComp.TargetPartSlot == "" {
		return HandlerTargetingResult{Success: false, LogMessage: settings.Name + "は射撃ターゲットが未選択です。"}
	}
	targetEntry := actionComp.TargetEntity
	targetPartSlot := actionComp.TargetPartSlot

	if targetEntry.HasComponent(BrokenStateComponent) {
		return HandlerTargetingResult{TargetEntity: targetEntry, TargetPartSlot: targetPartSlot, Success: false, LogMessage: settings.Name + "はターゲット(" + SettingsComponent.Get(targetEntry).Name + ")を狙ったが、既に行動不能だった！"}
	}
	targetParts := PartsComponent.Get(targetEntry)
	if targetParts.Map[targetPartSlot] == nil || targetParts.Map[targetPartSlot].IsBroken {
		return HandlerTargetingResult{TargetEntity: targetEntry, TargetPartSlot: targetPartSlot, Success: false, LogMessage: settings.Name + "は" + SettingsComponent.Get(targetEntry).Name + "の" + string(targetPartSlot) + "を狙ったが、パーツは既に破壊されていた！"}
	}

	return HandlerTargetingResult{
		TargetEntity:   targetEntry,
		TargetPartSlot: targetPartSlot,
		Success:        true,
		LogMessage:     settings.Name + "は" + SettingsComponent.Get(targetEntry).Name + "の" + string(targetPartSlot) + "を狙う！",
	}
}

// MeleeActionHandler handles actions for MELEE (FIGHT) category parts.
type MeleeActionHandler struct{}

func (h *MeleeActionHandler) ResolveTarget(
	actingEntry *donburi.Entry,
	world donburi.World,
	actionComp *Action, // Melee might ignore pre-selected target in ActionComponent
	targetSelector *TargetSelector,
	partInfoProvider *PartInfoProvider,
) HandlerTargetingResult {
	settings := SettingsComponent.Get(actingEntry)
	closestEnemy := targetSelector.FindClosestEnemy(actingEntry)
	if closestEnemy == nil {
		return HandlerTargetingResult{Success: false, LogMessage: settings.Name + "は格闘攻撃しようとしたが、相手がいなかった。"}
	}
	if closestEnemy.HasComponent(BrokenStateComponent) { // Should be filtered by FindClosestEnemy, but double check
		return HandlerTargetingResult{TargetEntity: closestEnemy, Success: false, LogMessage: settings.Name + "はターゲット(" + SettingsComponent.Get(closestEnemy).Name + ")を狙ったが、既に行動不能だった！"}
	}

	targetPart := targetSelector.SelectRandomPartToDamage(closestEnemy)
	if targetPart == nil {
		return HandlerTargetingResult{TargetEntity: closestEnemy, Success: false, LogMessage: settings.Name + "は" + SettingsComponent.Get(closestEnemy).Name + "を狙ったが、攻撃できる部位がなかった！"}
	}
	// FindPartSlot is needed to get the PartSlotKey from the Part struct
	targetPartSlot := partInfoProvider.FindPartSlot(closestEnemy, targetPart)
	if targetPartSlot == "" {
		// This should ideally not happen if SelectRandomPartToDamage returns a valid part from the entity
		return HandlerTargetingResult{TargetEntity: closestEnemy, Success: false, LogMessage: settings.Name + "の" + SettingsComponent.Get(closestEnemy).Name + "への攻撃でパーツスロット特定失敗。"}
	}

	return HandlerTargetingResult{
		TargetEntity:   closestEnemy,
		TargetPartSlot: targetPartSlot,
		Success:        true,
		LogMessage:     settings.Name + "は" + SettingsComponent.Get(closestEnemy).Name + "の" + string(targetPartSlot) + "に格闘攻撃！",
	}
}

// Global instances of handlers (or a factory/registry could be used)
var (
	shootHandler = &ShootActionHandler{}
	meleeHandler = &MeleeActionHandler{}
	// supportHandler = &SupportActionHandler{} // Example for future
	// defenseHandler = &DefenseActionHandler{} // Example for future
)

// GetActionHandlerForCategory returns an appropriate ActionHandler based on the part category.
func GetActionHandlerForCategory(category PartCategory) ActionHandler {
	switch category {
	case CategoryShoot:
		return shootHandler
	case CategoryMelee:
		return meleeHandler
	// case CategorySupport:
	//	return supportHandler
	// case CategoryDefense:
	//	return defenseHandler
	default:
		// Return a default handler or nil if unhandled
		// For now, returning nil means executeActionLogic needs a fallback or error handling
		return nil
	}
}
