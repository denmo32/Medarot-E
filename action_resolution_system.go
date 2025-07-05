package main

import (
	"fmt"
	"log"

	"github.com/yohamta/donburi"
)

// ResolveActionSystem はアクションのターゲットを解決し、ActionContextを準備します。
// 成功した場合は true を、解決に失敗した場合は false を返します。
func ResolveActionSystem(
	actingEntry *donburi.Entry,
	world donburi.World,
	actionResult *ActionResult,
	partInfoProvider *PartInfoProvider,
	gameConfig *Config,
	targetSelector *TargetSelector,
) (*PartDefinition, *PartInstanceData, *PartInstanceData, *PartDefinition, bool) {
	intent := ActionIntentComponent.Get(actingEntry)
	settings := SettingsComponent.Get(actingEntry)
	partsComp := PartsComponent.Get(actingEntry)
	actingPartInstance := partsComp.Map[intent.SelectedPartKey]

	if actingPartInstance == nil {
		log.Printf("エラー: ResolveActionSystem - %s の行動パーツインスタンスがnilです。パーツキー: %s", settings.Name, intent.SelectedPartKey)
		actionResult.LogMessage = fmt.Sprintf("%sは行動パーツの取得に失敗しました。", settings.Name)
		return nil, nil, nil, nil, false
	}

	actingPartDef, defFound := GlobalGameDataManager.GetPartDefinition(actingPartInstance.DefinitionID)
	if !defFound {
		log.Printf("エラー: ResolveActionSystem - ID %s (エンティティ: %s) のPartDefinitionが見つかりません。", actingPartInstance.DefinitionID, settings.Name)
		actionResult.LogMessage = fmt.Sprintf("%sはパーツ定義(%s)の取得に失敗しました。", settings.Name, actingPartInstance.DefinitionID)
				return nil, nil, nil, nil, false
	}

	// ApplyActionModifiersSystem(world, actingEntry, gameConfig, partInfoProvider)

	
	handler := GetActionHandlerForCategory(actingPartDef.Category)
	if handler == nil {
		actionResult.LogMessage = fmt.Sprintf("%sのパーツカテゴリ '%s' に対応する行動ハンドラがありません。", settings.Name, actingPartDef.Category)
				return nil, nil, nil, nil, false
	}

	targetEntry, targetPartSlot, success := handler.ResolveTarget(actingEntry, world, intent, targetSelector, partInfoProvider, actionResult)
	if !success {
		// 失敗した場合、LogMessageはハンドラによってresultに設定されています
		return nil, nil, nil, nil, false
	}

	// ターゲット情報をTargetComponentに保存
	target := TargetComponent.Get(actingEntry)
	target.TargetEntity = targetEntry
	target.TargetPartSlot = targetPartSlot

	actionResult.TargetEntry = targetEntry
	actionResult.TargetPartSlot = targetPartSlot

	var intendedTargetPartInstance *PartInstanceData
	var intendedTargetPartDef *PartDefinition

	// ターゲット情報の検証
	if actionResult.TargetEntry != nil && actionResult.TargetPartSlot != "" {
		targetPartsComp := PartsComponent.Get(actionResult.TargetEntry)
		if targetPartsComp == nil {
			actionResult.LogMessage = fmt.Sprintf("%sは%sを狙ったが、ターゲットにパーツコンポーネントがありません。", settings.Name, SettingsComponent.Get(actionResult.TargetEntry).Name)
			return nil, nil, nil, nil, false
		}

		intendedTargetPartInstance = targetPartsComp.Map[actionResult.TargetPartSlot]
		if intendedTargetPartInstance == nil {
			actionResult.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、ターゲットパーツインスタンスが見つかりませんでした。", settings.Name, SettingsComponent.Get(actionResult.TargetEntry).Name, actionResult.TargetPartSlot)
			return nil, nil, nil, nil, false
		}

		var tDefFound bool
		intendedTargetPartDef, tDefFound = GlobalGameDataManager.GetPartDefinition(intendedTargetPartInstance.DefinitionID)
		if !tDefFound {
			actionResult.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、ターゲットパーツ定義(%s)が見つかりませんでした。", settings.Name, SettingsComponent.Get(actionResult.TargetEntry).Name, actionResult.TargetPartSlot, intendedTargetPartInstance.DefinitionID)
			return nil, nil, nil, nil, false
		}

		if intendedTargetPartInstance.IsBroken {
			actionResult.LogMessage = fmt.Sprintf("%sは%sの%sを狙ったが、パーツは既に破壊されていました。", settings.Name, SettingsComponent.Get(actionResult.TargetEntry).Name, actionResult.TargetPartSlot)
			return nil, nil, nil, nil, false
		}
	}

	// 攻撃アクションで有効なターゲット部位がない場合のエラー
	if (actingPartDef.Category == CategoryShoot || actingPartDef.Category == CategoryMelee) && actionResult.TargetEntry != nil && intendedTargetPartInstance == nil {
		actionResult.LogMessage = fmt.Sprintf("%s は %s を攻撃しようとしましたが、有効な対象部位がありませんでした。", settings.Name, SettingsComponent.Get(actionResult.TargetEntry).Name)
		return nil, nil, nil, nil, false
	}

	return actingPartDef, actingPartInstance, intendedTargetPartInstance, intendedTargetPartDef, true
}
