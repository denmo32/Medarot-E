package main

import (
	"log"
)

// =============================================================================
// 初期化 (Initializer)
// =============================================================================

// InitializeAllMedarots は全てのメダロットデータをロードし、インスタンスを生成する
func InitializeAllMedarots(gameData *GameData) []*Medarot {
	var allMedarots []*Medarot

	for _, loadout := range gameData.Medarots {
		medal := findMedalByID(gameData.Medals, loadout.MedalID)
		if medal == nil {
			log.Printf("警告: メダルID '%s' が見つかりません。'%s'にはデフォルトメダルを使用します。", loadout.MedalID, loadout.Name)
			medal = &Medal{ID: "fallback", Name: "フォールバック", SkillLevel: 1}
		}

		medarot := NewMedarot(
			loadout.ID,
			loadout.Name,
			loadout.Team,
			medal,
			loadout.IsLeader,
			loadout.DrawIndex,
		)

		partIDMap := map[PartSlotKey]string{
			PartSlotHead:     loadout.HeadID,
			PartSlotRightArm: loadout.RightArmID,
			PartSlotLeftArm:  loadout.LeftArmID,
			PartSlotLegs:     loadout.LegsID,
		}

		for slot, partID := range partIDMap {
			if p, exists := gameData.AllParts[partID]; exists {
				// パーツの状態をリセットするために、ここでコピーを行うのが最も安全
				newPart := *p
				newPart.IsBroken = false
				medarot.Parts[slot] = &newPart
			} else {
				log.Printf("警告: パーツID '%s' が見つかりません。'%s'の%sスロットは空になります。", partID, medarot.Name, slot)
				placeholderPart := &Part{ID: "placeholder", PartName: "なし", IsBroken: true}
				medarot.Parts[slot] = placeholderPart
			}
		}
		allMedarots = append(allMedarots, medarot)
	}

	log.Printf("%d体のメダロットを初期化しました。", len(allMedarots))
	return allMedarots
}

// findMedalByID はIDでメダルを探す（コピーを返す）
func findMedalByID(allMedals []Medal, id string) *Medal {
	for _, medal := range allMedals {
		if medal.ID == id {
			newMedal := medal
			return &newMedal
		}
	}
	return nil
}

// =============================================================================
// メダロットのメソッド (Methods)
// =============================================================================

// NewMedarot は新しいメダロットのインスタンスを生成する（コンストラクタ）
func NewMedarot(id, name string, team TeamID, medal *Medal, isLeader bool, drawIndex int) *Medarot {
	return &Medarot{
		ID:        id,
		Name:      name,
		Team:      team,
		Medal:     medal,
		Parts:     make(map[PartSlotKey]*Part),
		IsLeader:  isLeader,
		DrawIndex: drawIndex,
		State:     StateIdle,
		Gauge:     0.0,
	}
}

// ChangeState はメダロットの状態を変更し、関連するカウンターをリセットする
// (これは状態変更に伴うリセット処理が密結合しているため、ヘルパーとして残す)
func (m *Medarot) ChangeState(newState MedarotState) {
	if m.State == newState {
		return
	}
	log.Printf("%s のステートが %s から %s に変更されました。", m.Name, m.State, newState)
	m.State = newState

	// 状態変更時にリセットするべき値をここで管理
	switch newState {
	case StateIdle:
		m.Gauge = 0
		m.ProgressCounter = 0
		m.TotalDuration = 0
		m.SelectedPartKey = ""
		m.TargetedMedarot = nil
	case StateReady:
		m.Gauge = 100
	case StateBroken:
		m.Gauge = 0
	}
}


// =============================================================================
// ヘルパー/ゲッターメソッド (Helpers / Getters)
// (これらは外部からデータを取得するための純粋な関数なので、メソッドとして残して問題ない)
// =============================================================================

// GetPart は指定されたスロットのパーツを取得する
func (m *Medarot) GetPart(slotKey PartSlotKey) *Part {
	part, exists := m.Parts[slotKey]
	if !exists || part == nil {
		return nil
	}
	return part
}

// GetAvailableAttackParts は攻撃に使用可能なパーツのリストを取得する
func (m *Medarot) GetAvailableAttackParts() []*Part {
	var availableParts []*Part
	slotsToConsider := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm}
	for _, slot := range slotsToConsider {
		part := m.GetPart(slot)
		if part != nil && !part.IsBroken && part.Category != CategoryNone {
			availableParts = append(availableParts, part)
		}
	}
	return availableParts
}

// GetOverallPropulsion は脚部の推進力を取得する
func (m *Medarot) GetOverallPropulsion() int {
	legs := m.GetPart(PartSlotLegs)
	if legs == nil || legs.IsBroken {
		return 1
	}
	return legs.Propulsion
}

// GetOverallMobility は脚部の機動力を取得する
func (m *Medarot) GetOverallMobility() int {
	legs := m.GetPart(PartSlotLegs)
	if legs == nil || legs.IsBroken {
		return 1
	}
	return legs.Mobility
}