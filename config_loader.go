package main

import (
	"encoding/json"
	"fmt" // fmtパッケージを追加
	"image/color"
	"io/ioutil"
	"log"
)

// GameSettings は game_settings.json の構造を定義します。
type GameSettings struct {
	Time struct {
		PropulsionEffectRate float64
		GameSpeedMultiplier  float64
	}
	HPAnimationSpeed float64
	Factors          struct {
		AccuracyStabilityFactor      float64
		EvasionStabilityFactor       float64
		DefenseStabilityFactor       float64
		PowerStabilityFactor         float64
		MeleeAccuracyMobilityFactor  float64
		BerserkPowerPropulsionFactor float64
	}
	Effects struct {
		Melee struct {
			DefenseRateDebuff float64
			CriticalRateBonus int
		}
		Berserk struct {
			DefenseRateDebuff float64
			EvasionRateDebuff float64
		}
		Shoot struct{}
		Aim   struct {
			EvasionRateDebuff float64
			CriticalRateBonus int
		}
	}
	Damage struct {
		CriticalMultiplier     float64
		MedalSkillFactor       int
		DamageAdjustmentFactor float64
		Critical               struct {
			BaseChance        float64
			SuccessRateFactor float64
			MinChance         float64
			MaxChance         float64
		}
	}
	Hit struct {
		BaseChance float64
		MinChance  float64
		MaxChance  float64
	}
	Defense struct {
		BaseChance float64
		MinChance  float64
		MaxChance  float64
	}
	Formulas map[Trait]ActionFormulaConfig
	UI       struct {
		Screen struct {
			Width  int
			Height int
		}
		Battlefield struct {
			Height                       float32
			Team1HomeX                   float32
			Team2HomeX                   float32
			Team1ExecutionLineX          float32
			Team2ExecutionLineX          float32
			IconRadius                   float32
			HomeMarkerRadius             float32
			LineWidth                    float32
			MedarotVerticalSpacingFactor float32
			TargetIndicator              struct {
				Width  float32
				Height float32
			}
		}
		InfoPanel struct {
			Padding           int
			BlockWidth        float32
			BlockHeight       float32
			PartHPGaugeWidth  float32
			PartHPGaugeHeight float32
		}
		ActionModal struct {
			ButtonWidth         float32
			ButtonHeight        float32
			ButtonSpacing       int
			ModalButtonFontSize float64 // 追加
		}
		MessageWindow struct { // 追加
			MessageWindowFontSize float64 // 追加
		}
		Colors struct {
			White      string
			Red        string
			Blue       string
			Yellow     string
			Gray       string
			Team1      string
			Team2      string
			Leader     string
			Broken     string
			HP         string
			HPCritical string
			Background string
		}
	}
}

func LoadConfig() Config {
	// game_settings.json から設定をロード
	var gameSettings GameSettings
	assetPaths := AssetPaths{
		GameSettings: "assets/configs/game_settings.json",
		Messages:     "assets/texts/messages.json",
		MedalsCSV:    "assets/databases/medals.csv",
		PartsCSV:     "assets/databases/parts.csv",
		MedarotsCSV:  "assets/databases/medarots.csv",
		FormulasJSON: "assets/configs/formulas.json",
		Font:         "assets/fonts/MPLUS1p-Regular.ttf",
		Image:        "assets/images/Gemini_Generated_Image_hojkprhojkprhojk.png",
	}

	jsonFile, err := ioutil.ReadFile(assetPaths.GameSettings)
	if err != nil {
		log.Fatalf("Error reading game_settings.json: %v", err)
	}
	err = json.Unmarshal(jsonFile, &gameSettings)
	if err != nil {
		log.Fatalf("Error unmarshalling game_settings.json: %v", err)
	}

	cfg := Config{
		Balance: BalanceConfig{
			Time:             gameSettings.Time,
			HPAnimationSpeed: gameSettings.HPAnimationSpeed,
			Factors:          gameSettings.Factors,
			Effects:          gameSettings.Effects,
			Damage:           gameSettings.Damage,
			Hit:              gameSettings.Hit,
			Defense:          gameSettings.Defense,
			Formulas:         gameSettings.Formulas,
		},
		AssetPaths: assetPaths,
		UI: UIConfig{
			Screen: struct {
				Width  int
				Height int
			}{
				Width:  gameSettings.UI.Screen.Width,
				Height: gameSettings.UI.Screen.Height,
			},
			Battlefield: struct {
				Height                       float32
				Team1HomeX                   float32
				Team2HomeX                   float32
				Team1ExecutionLineX          float32
				Team2ExecutionLineX          float32
				IconRadius                   float32
				HomeMarkerRadius             float32
				LineWidth                    float32
				MedarotVerticalSpacingFactor float32
				TargetIndicator              struct {
					Width  float32
					Height float32
				}
			}{
				Height:                       gameSettings.UI.Battlefield.Height,
				Team1HomeX:                   gameSettings.UI.Battlefield.Team1HomeX,
				Team2HomeX:                   gameSettings.UI.Battlefield.Team2HomeX,
				Team1ExecutionLineX:          gameSettings.UI.Battlefield.Team1ExecutionLineX,
				Team2ExecutionLineX:          gameSettings.UI.Battlefield.Team2ExecutionLineX,
				IconRadius:                   gameSettings.UI.Battlefield.IconRadius,
				HomeMarkerRadius:             gameSettings.UI.Battlefield.HomeMarkerRadius,
				LineWidth:                    gameSettings.UI.Battlefield.LineWidth,
				MedarotVerticalSpacingFactor: gameSettings.UI.Battlefield.MedarotVerticalSpacingFactor,
				TargetIndicator: struct {
					Width  float32
					Height float32
				}{
					Width:  gameSettings.UI.Battlefield.TargetIndicator.Width,
					Height: gameSettings.UI.Battlefield.TargetIndicator.Height,
				},
			},
			InfoPanel: struct {
				Padding           int
				BlockWidth        float32
				BlockHeight       float32
				PartHPGaugeWidth  float32
				PartHPGaugeHeight float32
			}{
				Padding:           gameSettings.UI.InfoPanel.Padding,
				BlockWidth:        gameSettings.UI.InfoPanel.BlockWidth,
				BlockHeight:       gameSettings.UI.InfoPanel.BlockHeight,
				PartHPGaugeWidth:  gameSettings.UI.InfoPanel.PartHPGaugeWidth,
				PartHPGaugeHeight: gameSettings.UI.InfoPanel.PartHPGaugeHeight,
			},
			ActionModal: struct {
				ButtonWidth         float32
				ButtonHeight        float32
				ButtonSpacing       int
				ModalButtonFontSize float64
			}{
				ButtonWidth:         gameSettings.UI.ActionModal.ButtonWidth,
				ButtonHeight:        gameSettings.UI.ActionModal.ButtonHeight,
				ButtonSpacing:       gameSettings.UI.ActionModal.ButtonSpacing,
				ModalButtonFontSize: gameSettings.UI.ActionModal.ModalButtonFontSize,
			},
			MessageWindow: struct {
				MessageWindowFontSize float64
			}{
				MessageWindowFontSize: gameSettings.UI.MessageWindow.MessageWindowFontSize,
			},
			Colors: struct {
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
			}{
				White:      parseHexColor(gameSettings.UI.Colors.White),
				Red:        parseHexColor(gameSettings.UI.Colors.Red),
				Blue:       parseHexColor(gameSettings.UI.Colors.Blue),
				Yellow:     parseHexColor(gameSettings.UI.Colors.Yellow),
				Gray:       parseHexColor(gameSettings.UI.Colors.Gray),
				Team1:      parseHexColor(gameSettings.UI.Colors.Team1),
				Team2:      parseHexColor(gameSettings.UI.Colors.Team2),
				Leader:     parseHexColor(gameSettings.UI.Colors.Leader),
				Broken:     parseHexColor(gameSettings.UI.Colors.Broken),
				HP:         parseHexColor(gameSettings.UI.Colors.HP),
				HPCritical: parseHexColor(gameSettings.UI.Colors.HPCritical),
				Background: parseHexColor(gameSettings.UI.Colors.Background),
			},
		},
	}

	return cfg
}

// parseHexColor は16進数文字列からcolor.Colorをパースします。
func parseHexColor(s string) color.Color {
	var r, g, b uint8
	if len(s) == 6 {
		_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
		if err != nil {
			log.Printf("Failed to parse hex color %s: %v", s, err)
			return color.White // エラー時はデフォルト色
		}
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
