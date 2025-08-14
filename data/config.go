package data

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
)

// Configは、ゲーム全体のコンフィグレーションを保持します。
// この構造体は、game_settings.jsonから直接デシリアライズされる部分と、
// コード内で後から設定される部分（AssetPaths, Game）で構成されます。
type Config struct {
	// --- Balance Settings (from game_settings.json) ---
	// BalanceConfig構造体を廃止し、フィールドをConfig直下に配置することで、
	// game_settings.jsonのフラットな構造と直接対応させます。
	Time struct {
		PropulsionEffectRate float64 `json:"PropulsionEffectRate"`
		GameSpeedMultiplier  float64 `json:"GameSpeedMultiplier"`
	} `json:"Time"`
	HPAnimationSpeed float64 `json:"HPAnimationSpeed"`
	Factors          struct {
		AccuracyStabilityFactor      float64 `json:"AccuracyStabilityFactor"`
		EvasionStabilityFactor       float64 `json:"EvasionStabilityFactor"`
		DefenseStabilityFactor       float64 `json:"DefenseStabilityFactor"`
		PowerStabilityFactor         float64 `json:"PowerStabilityFactor"`
		MeleeAccuracyMobilityFactor  float64 `json:"MeleeAccuracyMobilityFactor"`
		BerserkPowerPropulsionFactor float64 `json:"BerserkPowerPropulsionFactor"`
	} `json:"Factors"`
	Effects struct {
		Melee struct {
			DefenseRateDebuff float64 `json:"DefenseRateDebuff"`
			CriticalRateBonus int     `json:"CriticalRateBonus"`
		} `json:"Melee"`
		Berserk struct {
			DefenseRateDebuff float64 `json:"DefenseRateDebuff"`
			EvasionRateDebuff float64 `json:"EvasionRateDebuff"`
		} `json:"Berserk"`
		Shoot struct{} `json:"Shoot"`
		Aim   struct {
			EvasionRateDebuff float64 `json:"EvasionRateDebuff"`
			CriticalRateBonus int     `json:"CriticalRateBonus"`
		} `json:"Aim"`
	} `json:"Effects"`
	Damage struct {
		CriticalMultiplier     float64 `json:"CriticalMultiplier"`
		MedalSkillFactor       int     `json:"MedalSkillFactor"`
		DamageAdjustmentFactor float64 `json:"DamageAdjustmentFactor"`
		Critical               struct {
			BaseChance        float64 `json:"BaseChance"`
			SuccessRateFactor float64 `json:"SuccessRateFactor"`
			MinChance         float64 `json:"MinChance"`
			MaxChance         float64 `json:"MaxChance"`
		} `json:"Critical"`
	} `json:"Damage"`
	Hit struct {
		BaseChance float64 `json:"BaseChance"`
		MinChance  float64 `json:"MinChance"`
		MaxChance  float64 `json:"MaxChance"`
	} `json:"Hit"`
	Defense struct {
		BaseChance float64 `json:"BaseChance"`
		MinChance  float64 `json:"MinChance"`
		MaxChance  float64 `json:"MaxChance"`
	} `json:"Defense"`

	// UI設定はUIConfig構造体にマッピングされます。
	UI UIConfig `json:"UI"`

	// --- Non-JSON fields ---
	// 以下のフィールドはJSONファイルからロードされず、コード内で設定されます。
	AssetPaths AssetPaths
	Game       GameConfig
	// Formulasフィールドを削除: この責務はGameDataManagerが担うため冗長でした。
	// Formulas   map[core.Trait]core.ActionFormulaConfig
}

// AssetPaths は各種アセットへのパスを保持します。
type AssetPaths struct {
	GameSettings string
	Messages     string
	MedalsCSV    string
	PartsCSV     string
	MedarotsCSV  string
	FormulasJSON string
	Font         string
	Image        string
}

// GameConfig はゲームプレイ固有の設定を保持します。
type GameConfig struct {
	RandomSeed int64
}

// UIConfig は game_settings.json の "UI" セクションとマッピングされます。
// 色設定はstringで受け取り、後から color.Color に変換されます。
type UIConfig struct {
	Screen struct {
		Width  int `json:"Width"`
		Height int `json:"Height"`
	} `json:"Screen"`
	Battlefield struct {
		Height                       float32 `json:"Height"`
		Team1HomeX                   float32 `json:"Team1HomeX"`
		Team2HomeX                   float32 `json:"Team2HomeX"`
		Team1ExecutionLineX          float32 `json:"Team1ExecutionLineX"`
		Team2ExecutionLineX          float32 `json:"Team2ExecutionLineX"`
		IconRadius                   float32 `json:"IconRadius"`
		HomeMarkerRadius             float32 `json:"HomeMarkerRadius"`
		LineWidth                    float32 `json:"LineWidth"`
		MedarotVerticalSpacingFactor float32 `json:"MedarotVerticalSpacingFactor"`
		TargetIndicator              struct {
			Width  float32 `json:"Width"`
			Height float32 `json:"Height"`
		} `json:"TargetIndicator"`
	} `json:"Battlefield"`
	InfoPanel struct {
		Padding           int     `json:"Padding"`
		BlockWidth        float32 `json:"BlockWidth"`
		BlockHeight       float32 `json:"BlockHeight"`
		PartHPGaugeWidth  float32 `json:"PartHPGaugeWidth"`
		PartHPGaugeHeight float32 `json:"PartHPGaugeHeight"`
	} `json:"InfoPanel"`
	ActionModal struct {
		ButtonWidth         float32 `json:"ButtonWidth"`
		ButtonHeight        float32 `json:"ButtonHeight"`
		ButtonSpacing       int     `json:"ButtonSpacing"`
		ModalButtonFontSize float64 `json:"ModalButtonFontSize"`
	} `json:"ActionModal"`
	MessageWindow struct {
		MessageWindowFontSize float64 `json:"MessageWindowFontSize"`
	} `json:"MessageWindow"`

	// ColorsフィールドはJSONから直接デシリアライズされる際に、
	// 下記で定義されたカスタムのUnmarshalJSONメソッドによってパースされます。
	Colors ParsedColors `json:"Colors"`
}

// ParsedColors はパース済みの色情報を保持します。
type ParsedColors struct {
	White      color.Color
	Red        color.Color
	Blue       color.Color
	Yellow     color.Color
	Gray       color.Color
	Team1      color.Color
	Team2      color.Color
	Leader     color.Color
	Broken     color.Color
	HP         color.Color
	HPCritical color.Color
	Background color.Color
	Black      color.Color
}

// UnmarshalJSON は ParsedColors 型のカスタムデシリアライザです。
// JSONの "Colors" オブジェクト（キーが色名、値が16進数文字列のマップ）を
// ParsedColors 構造体の各 color.Color フィールドに変換します。
func (p *ParsedColors) UnmarshalJSON(data []byte) error {
	// JSONの "Colors" オブジェクトを、16進数文字列を保持する一時的な構造体にデコードします。
	// この匿名構造体は、このメソッド内でのみ使用される使い捨ての型です。
	var raw struct {
		White      string `json:"White"`
		Red        string `json:"Red"`
		Blue       string `json:"Blue"`
		Yellow     string `json:"Yellow"`
		Gray       string `json:"Gray"`
		Team1      string `json:"Team1"`
		Team2      string `json:"Team2"`
		Leader     string `json:"Leader"`
		Broken     string `json:"Broken"`
		HP         string `json:"HP"`
		HPCritical string `json:"HPCritical"`
		Background string `json:"Background"`
		Black      string `json:"Black"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("色データのJSONアンマーシャルに失敗しました: %w", err)
	}

	// 16進数文字列をパースし、ParsedColors のフィールドに設定します。
	// このアプローチはリフレクションよりも直感的で、パフォーマンスも優れています。
	p.White = parseHexColor(raw.White)
	p.Red = parseHexColor(raw.Red)
	p.Blue = parseHexColor(raw.Blue)
	p.Yellow = parseHexColor(raw.Yellow)
	p.Gray = parseHexColor(raw.Gray)
	p.Team1 = parseHexColor(raw.Team1)
	p.Team2 = parseHexColor(raw.Team2)
	p.Leader = parseHexColor(raw.Leader)
	p.Broken = parseHexColor(raw.Broken)
	p.HP = parseHexColor(raw.HP)
	p.HPCritical = parseHexColor(raw.HPCritical)
	p.Background = parseHexColor(raw.Background)
	p.Black = parseHexColor(raw.Black)

	return nil
}

// parseHexColor は16進数文字列からcolor.Colorをパースします。
// UnmarshalJSONから利用されるため、このファイルに配置します。
func parseHexColor(s string) color.Color {
	var r, g, b uint8
	// 期待する長さ(6)でなければデフォルト色を返す
	if len(s) != 6 {
		log.Printf("無効な16進数カラーコードです: %s。デフォルト色を使用します。", s)
		return color.White
	}
	// 16進数文字列をRGBの各要素にスキャンします
	_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		log.Printf("16進数カラーコード '%s' のパースに失敗しました: %v", s, err)
		return color.White // エラー時はデフォルト色として白を返す
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
