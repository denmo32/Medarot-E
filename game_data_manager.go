package main

import (
	"fmt"
	"medarot-ebiten/domain"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	resource "github.com/quasilyte/ebitengine-resource" // resource パッケージを追加
)

// GameDataManager はパーツやメダルなどのすべての静적ゲームデータ定義とメッセージを保持します。
type GameDataManager struct {
	partDefinitions  map[string]*domain.PartDefinition
	medalDefinitions map[string]*domain.Medal              // Medal構造体は今のところ主に定義情報と仮定
	Messages         *MessageManager                       // メッセージマネージャー
	Font             text.Face                             // UIで使用するフォント
	Formulas         map[domain.Trait]domain.ActionFormula // 追加: アクション計算式
	// 他のゲームデータ定義もここに追加できます
}

// NewGameDataManager はGameDataManagerの新しいインスタンスを作成し、初期化します。
func NewGameDataManager(font text.Face, assetPaths *AssetPaths, r *resource.Loader) (*GameDataManager, error) {
	messageManager, err := NewMessageManager(assetPaths.Messages, r) // r を渡す
	if err != nil {
		return nil, fmt.Errorf("メッセージマネージャーの初期化に失敗しました: %w", err)
	}

	gdm := &GameDataManager{
		partDefinitions:  make(map[string]*domain.PartDefinition),
		medalDefinitions: make(map[string]*domain.Medal),
		Messages:         messageManager,                              // メッセージマネージャー
		Font:             font,                                        // UIで使用するフォント
		Formulas:         make(map[domain.Trait]domain.ActionFormula), // 初期化
	}
	return gdm, nil
}

// AddPartDefinition はパーツ定義をマネージャーに追加します。
func (gdm *GameDataManager) AddPartDefinition(pd *domain.PartDefinition) error {
	if pd == nil {
		return fmt.Errorf("nilのPartDefinitionを追加できません")
	}
	if _, exists := gdm.partDefinitions[pd.ID]; exists {
		return fmt.Errorf("ID %s のPartDefinitionは既に存在します", pd.ID)
	}
	gdm.partDefinitions[pd.ID] = pd
	return nil
}

// GetPartDefinition はIDによってパーツ定義を取得します。
func (gdm *GameDataManager) GetPartDefinition(id string) (*domain.PartDefinition, bool) {
	pd, found := gdm.partDefinitions[id]
	return pd, found
}

// AddMedalDefinition はメダル定義をマネージャーに追加します。
func (gdm *GameDataManager) AddMedalDefinition(md *domain.Medal) error {
	if md == nil {
		return fmt.Errorf("nilのMedalDefinitionを追加できません")
	}
	if _, exists := gdm.medalDefinitions[md.ID]; exists {
		return fmt.Errorf("ID %s のMedalDefinitionは既に存在します", md.ID)
	}
	gdm.medalDefinitions[md.ID] = md
	return nil
}

// GetMedalDefinition はIDによってメダル定義を取得します。
func (gdm *GameDataManager) GetMedalDefinition(id string) (*domain.Medal, bool) {
	md, found := gdm.medalDefinitions[id]
	return md, found
}

// GetAllPartDefinitions はすべてのパーツ定義のスライスを返します。
// 注意: マップの反復処理は順序を保証しません。順序が必要な場合は、スライスとして格納するかソートしてください。
func (gdm *GameDataManager) GetAllPartDefinitions() []*domain.PartDefinition {
	defs := make([]*domain.PartDefinition, 0, len(gdm.partDefinitions))
	for _, pd := range gdm.partDefinitions {
		defs = append(defs, pd)
	}
	// UIで一貫した順序が必要な場合は、ここでソートを追加
	return defs
}

// GetAllMedalDefinitions はすべてのメダル定義のスライスを返します。
func (gdm *GameDataManager) GetAllMedalDefinitions() []*domain.Medal {
	defs := make([]*domain.Medal, 0, len(gdm.medalDefinitions))
	for _, md := range gdm.medalDefinitions {
		defs = append(defs, md)
	}
	// UIで一貫した順序が必要な場合は、ここでソートを追加
	return defs
}
