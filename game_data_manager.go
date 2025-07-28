package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	resource "github.com/quasilyte/ebitengine-resource" // resource パッケージを追加
)

// GameDataManager はパーツやメダルなどのすべての静적ゲームデータ定義とメッセージを保持します。
type GameDataManager struct {
	partDefinitions  map[string]*PartDefinition
	medalDefinitions map[string]*Medal       // Medal構造体は今のところ主に定義情報と仮定
	Messages         *MessageManager         // メッセージマネージャー
	Font             text.Face               // UIで使用するフォント
	Formulas         map[Trait]ActionFormula // 追加: アクション計算式
	// 他のゲームデータ定義もここに追加できます
}

// NewGameDataManager はGameDataManagerの新しいインスタンスを作成し、初期化します。
func NewGameDataManager(font text.Face, assetPaths *AssetPaths, r *resource.Loader) (*GameDataManager, error) {
	gdm := &GameDataManager{
		partDefinitions:  make(map[string]*PartDefinition),
		medalDefinitions: make(map[string]*Medal),
		Font:             font,
		Formulas:         make(map[Trait]ActionFormula), // 初期化
	}

	// メッセージマネージャーの初期化
	messageManager, err := NewMessageManager(assetPaths.Messages, r) // r を渡す
	if err != nil {
		return nil, fmt.Errorf("メッセージマネージャーの初期化に失敗しました: %w", err)
	}
	gdm.Messages = messageManager

	return gdm, nil
}

// AddPartDefinition はパーツ定義をマネージャーに追加します。
func (gdm *GameDataManager) AddPartDefinition(pd *PartDefinition) error {
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
func (gdm *GameDataManager) GetPartDefinition(id string) (*PartDefinition, bool) {
	pd, found := gdm.partDefinitions[id]
	return pd, found
}

// AddMedalDefinition はメダル定義をマネージャーに追加します。
func (gdm *GameDataManager) AddMedalDefinition(md *Medal) error {
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
func (gdm *GameDataManager) GetMedalDefinition(id string) (*Medal, bool) {
	md, found := gdm.medalDefinitions[id]
	return md, found
}

// GetAllPartDefinitions はすべてのパーツ定義のスライスを返します。
// 注意: マップの反復処理は順序を保証しません。順序が必要な場合は、スライスとして格納するかソートしてください。
func (gdm *GameDataManager) GetAllPartDefinitions() []*PartDefinition {
	defs := make([]*PartDefinition, 0, len(gdm.partDefinitions))
	for _, pd := range gdm.partDefinitions {
		defs = append(defs, pd)
	}
	// UIで一貫した順序が必要な場合は、ここでソートを追加
	return defs
}

// GetAllMedalDefinitions はすべてのメダル定義のスライスを返します。
func (gdm *GameDataManager) GetAllMedalDefinitions() []*Medal {
	defs := make([]*Medal, 0, len(gdm.medalDefinitions))
	for _, md := range gdm.medalDefinitions {
		defs = append(defs, md)
	}
	// UIで一貫した順序が必要な場合は、ここでソートを追加
	return defs
}
