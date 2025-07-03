package main

import (
	"fmt"
	"log"
)

// ResolveActionSystem はアクションのターゲットを解決し、ActionContextを準備します。
// 成功した場合は true を、解決に失敗した場合は false を返します。
func ResolveActionSystem(ctx *ActionContext) bool {
	intent := ActionIntentComponent.Get(ctx.ActingEntry)
	settings := SettingsComponent.Get(ctx.ActingEntry)
	partsComp := PartsComponent.Get(ctx.ActingEntry)
	actingPartInstance := partsComp.Map[intent.SelectedPartKey]

	if actingPartInstance == nil {
		log.Printf("エラー: ResolveActionSystem - %s の行動パーツインスタンスがnilです。パーツキー: %s", settings.Name, intent.SelectedPartKey)
		ctx.ActionResult.LogMessage = fmt.Sprintf("%sは行動パーツの取得に失敗しました。", settings.Name)
		return false
	}
	ctx.ActingPartInstance = actingPartInstance

	actingPartDef, defFound := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID)
	if !defFound {
		log.Printf("エラー: ResolveActionSystem - ID %s (エンティティ: %s) のPartDefinitionが見つかりません。", actingPartInstance.DefinitionID, settings.Name)
		ctx.ActionResult.LogMessage = fmt.Sprintf("%sはパーツ定義(%s)の取得に失敗しました。", settings.Name, actingPartInstance.DefinitionID)
		return false
	}
	ctx.ActingPartDef = actingPartDef

	ApplyActionModifiersSystem(ctx.World, ctx.ActingEntry, ctx.GameConfig, ctx.PartInfoProvider)

	handler := GetActionHandlerForCategory(actingPartDef.Category)
	if handler == nil {
		ctx.ActionResult.LogMessage = fmt.Sprintf("%sのパーツカテゴリ '%s' に対応する行動ハンドラがありません。", settings.Name, actingPartDef.Category)
		return false
	}
	ctx.ActionHandler = handler

	targetEntry, targetPartSlot, success := handler.ResolveTarget(ctx.ActingEntry, ctx.World, intent, ctx.TargetSelector, ctx.PartInfoProvider, ctx.ActionResult)
	if !success {
		// 失敗した場合、LogMessageはハンドラによってresultに設定されています
		return false
	}

	// ターゲット情報をTargetComponentに保存
	target := TargetComponent.Get(ctx.ActingEntry)
	target.TargetEntity = targetEntry
	target.TargetPartSlot = targetPartSlot

	ctx.TargetEntry = targetEntry
	ctx.TargetPartSlot = targetPartSlot

	// ターゲット情報の検証
	if ctx.TargetEntry != nil && ctx.TargetPartSlot != "" {
		targetPartsComp := PartsComponent.Get(ctx.TargetEntry)
		if targetPartsComp == nil {
			ctx.ActionResult.LogMessage = fmt.Sprintf("%sは%sを狙ったが、ターゲットにパーツコンポーネントがありません。", settings.Name, SettingsComponent.Get(ctx.TargetEntry).Name)
			return false
		}

		intendedTargetPartInstance := targetPartsComp.Map[ctx.TargetPartSlot]
		if intendedTargetPartInstance == nil {
			ctx.ActionResult.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、ターゲットパーツインスタンスが見つかりませんでした。", settings.Name, SettingsComponent.Get(ctx.TargetEntry).Name, ctx.TargetPartSlot)
			return false
		}
		ctx.IntendedTargetPartInstance = intendedTargetPartInstance

		intendedTargetPartDef, tDefFound := GlobalGameDataManager.GetPartDefinition(intendedTargetPartInstance.DefinitionID)
		if !tDefFound {
			ctx.ActionResult.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、ターゲットパーツ定義(%s)が見つかりませんでした。", settings.Name, SettingsComponent.Get(ctx.TargetEntry).Name, ctx.TargetPartSlot, intendedTargetPartInstance.DefinitionID)
			return false
		}
		ctx.IntendedTargetPartDef = intendedTargetPartDef

		if intendedTargetPartInstance.IsBroken {
			ctx.ActionResult.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、パーツは既に破壊されていました。", settings.Name, SettingsComponent.Get(ctx.TargetEntry).Name, ctx.TargetPartSlot)
			return false
		}
	}

	// 攻撃アクションで有効なターゲット部位がない場合のエラー
	if (actingPartDef.Category == CategoryShoot || actingPartDef.Category == CategoryMelee) && ctx.TargetEntry != nil && ctx.IntendedTargetPartInstance == nil {
		ctx.ActionResult.LogMessage = fmt.Sprintf("%s は %s を攻撃しようとしましたが、有効な対象部位がありませんでした。", settings.Name, SettingsComponent.Get(ctx.TargetEntry).Name)
		return false
	}

	return true
}
