package scene

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"sort"

	"medarot-ebiten/core"
	"medarot-ebiten/data"
	"medarot-ebiten/ecs/component"
	"medarot-ebiten/ecs/system"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"medarot-ebiten/donburi"
)

// balanceTestUnit は攻撃側または防御側の単一ユニットの状態を保持します。
type balanceTestUnit struct {
	entry         *donburi.Entry
	medalID       string
	headID        string
	rArmID        string
	lArmID        string
	legsID        string

	// UI Widgets
	medalButton *widget.Button
	headButton  *widget.Button
	rArmButton  *widget.Button
	lArmButton  *widget.Button
	legsButton  *widget.Button
}

// BalanceTestScene はゲームバランス調整用のシーンです
type BalanceTestScene struct {
	resources *data.SharedResources
	manager   *SceneManager
	ui        *ebitenui.UI

	// ECS and Systems
	world            donburi.World
	partInfoProvider system.PartInfoProviderInterface
	damageCalculator *system.DamageCalculator
	hitCalculator    *system.HitCalculator
	targetSelector   *system.TargetSelector
	rand             *rand.Rand

	// Scene State
	attacker *balanceTestUnit
	defender *balanceTestUnit

	// Part Lists
	medalList     []*core.Medal
	headPartsList []*core.PartDefinition
	rArmPartsList []*core.PartDefinition
	lArmPartsList []*core.PartDefinition
	legsPartsList []*core.PartDefinition

	// UI Widgets for results
	expectedDamageText *widget.Text
	hitChanceText      *widget.Text
	defenseChanceText  *widget.Text
	criticalChanceText *widget.Text
	simulationLogText  *widget.Text
}

// mustGetPartDefinition is a helper, panics if part not found.
// This is acceptable for the test scene where we control the data.
func (bs *BalanceTestScene) mustGetPartDefinition(id string) *core.PartDefinition {
	pd, found := bs.resources.GameDataManager.GetPartDefinition(id)
	if !found {
		panic(fmt.Sprintf("Part definition with id %s not found", id))
	}
	return pd
}

// NewBalanceTestScene は新しいバランス調整シーンを作成します
func NewBalanceTestScene(res *data.SharedResources, manager *SceneManager) (*BalanceTestScene, error) {
	bs := &BalanceTestScene{
		resources: res,
		manager:   manager,
		world:     donburi.NewWorld(),
		rand:      rand.New(rand.NewSource(1)), // Use a fixed seed for deterministic tests
	}

	// システムの初期化
	logger := data.NewBattleLogger(bs.resources.GameDataManager)
	bs.partInfoProvider = system.NewPartInfoProvider(bs.world, &bs.resources.Config, bs.resources.GameDataManager)
	bs.damageCalculator = system.NewDamageCalculator(bs.world, &bs.resources.Config, bs.partInfoProvider, bs.resources.GameDataManager, bs.rand, logger)
	bs.hitCalculator = system.NewHitCalculator(bs.world, &bs.resources.Config, bs.partInfoProvider, bs.rand, logger)
	bs.targetSelector = system.NewTargetSelector(bs.world, &bs.resources.Config, bs.partInfoProvider)

	// パーツリストの準備
	bs.setupPartLists()

	// 攻撃側と防御側のユニットを初期化
	bs.attacker = bs.createTestUnit(core.Team1, "Attacker")
	bs.defender = bs.createTestUnit(core.Team2, "Defender")

	// UIの構築
	rootContainer := bs.createLayout()
	bs.ui = &ebitenui.UI{Container: rootContainer}

	// 初期計算の実行
	bs.recalculate()

	return bs, nil
}

func (bs *BalanceTestScene) createLayout() *widget.Container {
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{true, false, true}, []bool{true}),
			widget.GridLayoutOpts.Spacing(20, 0),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(20)),
		)),
	)

	// Left Panel (Attacker)
	attackerPanel := bs.createUnitPanel(bs.attacker, "Attacker")
	rootContainer.AddChild(attackerPanel)

	// Center Panel (Results & Controls)
	centerPanel := bs.createCenterPanel()
	rootContainer.AddChild(centerPanel)

	// Right Panel (Defender)
	defenderPanel := bs.createUnitPanel(bs.defender, "Defender")
	rootContainer.AddChild(defenderPanel)

	return rootContainer
}

func (bs *BalanceTestScene) createUnitPanel(unit *balanceTestUnit, title string) *widget.Container {
	panel := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(15),
		)),
	)

	panel.AddChild(widget.NewText(
		widget.TextOpts.Text(title, bs.resources.Font, color.White),
	))

	unit.medalButton = bs.createPartSelectionRow(panel, unit, "Medal")
	unit.headButton = bs.createPartSelectionRow(panel, unit, "Head")
	unit.rArmButton = bs.createPartSelectionRow(panel, unit, "R-Arm")
	unit.lArmButton = bs.createPartSelectionRow(panel, unit, "L-Arm")
	unit.legsButton = bs.createPartSelectionRow(panel, unit, "Legs")

	return panel
}

func (bs *BalanceTestScene) createCenterPanel() *widget.Container {
	panel := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(15),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(20)),
		)),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{20, 20, 30, 200})),
	)

	panel.AddChild(widget.NewText(widget.TextOpts.Text("Calculation Results", bs.resources.Font, color.White)))

	bs.expectedDamageText = widget.NewText(widget.TextOpts.Text("Expected Damage: ", bs.resources.Font, color.White))
	panel.AddChild(bs.expectedDamageText)

	bs.hitChanceText = widget.NewText(widget.TextOpts.Text("Hit Chance: ", bs.resources.Font, color.White))
	panel.AddChild(bs.hitChanceText)

	bs.defenseChanceText = widget.NewText(widget.TextOpts.Text("Defense Chance: ", bs.resources.Font, color.White))
	panel.AddChild(bs.defenseChanceText)

	bs.criticalChanceText = widget.NewText(widget.TextOpts.Text("Critical Chance: ", bs.resources.Font, color.White))
	panel.AddChild(bs.criticalChanceText)

	runButton := widget.NewButton(
		widget.ButtonOpts.Text("Run Simulation", bs.resources.Font, &widget.ButtonTextColor{Idle: color.White}),
		widget.ButtonOpts.Image(bs.resources.ButtonImage),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			bs.runSimulation()
		}),
	)
	panel.AddChild(runButton)

	bs.simulationLogText = widget.NewText(widget.TextOpts.Text("Log:", bs.resources.Font, color.White))
	panel.AddChild(bs.simulationLogText)

	return panel
}

func (bs *BalanceTestScene) createPartSelectionRow(parent *widget.Container, unit *balanceTestUnit, partType string) *widget.Button {
	rowContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{false, true, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(10, 0),
		)),
	)
	parent.AddChild(rowContainer)

	leftButton := widget.NewButton(
		widget.ButtonOpts.Text("<", bs.resources.Font, &widget.ButtonTextColor{Idle: color.White}),
		widget.ButtonOpts.Image(bs.resources.ButtonImage),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) { bs.changePart(unit, partType, -1) }),
	)
	rowContainer.AddChild(leftButton)

	nameButton := widget.NewButton(
		widget.ButtonOpts.Text(partType, bs.resources.Font, &widget.ButtonTextColor{Idle: color.White}),
		widget.ButtonOpts.Image(bs.resources.ButtonImage),
	)
	rowContainer.AddChild(nameButton)

	rightButton := widget.NewButton(
		widget.ButtonOpts.Text(">", bs.resources.Font, &widget.ButtonTextColor{Idle: color.White}),
		widget.ButtonOpts.Image(bs.resources.ButtonImage),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) { bs.changePart(unit, partType, 1) }),
	)
	rowContainer.AddChild(rightButton)

	return nameButton
}

func (bs *BalanceTestScene) Update() error {
	bs.ui.Update()
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		bs.manager.GoToTitleScene()
	}
	return nil
}

func (bs *BalanceTestScene) Draw(screen *ebiten.Image) {
	screen.Fill(bs.resources.Config.UI.Colors.Background)
	bs.ui.Draw(screen)
}

func (bs *BalanceTestScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return bs.resources.Config.UI.Screen.Width, bs.resources.Config.UI.Screen.Height
}

// --- Logic for the scene ---

func (bs *BalanceTestScene) setupPartLists() {
	bs.medalList = bs.resources.GameDataManager.GetAllMedalDefinitions()
	sort.Slice(bs.medalList, func(i, j int) bool { return bs.medalList[i].ID < bs.medalList[j].ID })

	allParts := bs.resources.GameDataManager.GetAllPartDefinitions()
	sort.Slice(allParts, func(i, j int) bool { return allParts[i].ID < allParts[j].ID })

	for _, p := range allParts {
		switch p.Type {
		case core.PartTypeHead: 
			bs.headPartsList = append(bs.headPartsList, p)
		case core.PartTypeRArm:
			bs.rArmPartsList = append(bs.rArmPartsList, p)
		case core.PartTypeLArm:
			bs.lArmPartsList = append(bs.lArmPartsList, p)
		case core.PartTypeLegs:
			bs.legsPartsList = append(bs.legsPartsList, p)
		}
	}
}

func (bs *BalanceTestScene) createTestUnit(team core.TeamID, name string) *balanceTestUnit {
	entry := bs.world.Entry(bs.world.Create(
		component.SettingsComponent,
		component.PartsComponent,
		component.MedalComponent,
		component.StateComponent,
	))

	unit := &balanceTestUnit{
		entry:   entry,
		medalID: bs.medalList[0].ID,
		headID:  bs.headPartsList[0].ID,
		rArmID:  bs.rArmPartsList[0].ID,
		lArmID:  bs.lArmPartsList[0].ID,
		legsID:  bs.legsPartsList[0].ID,
	}

	component.SettingsComponent.SetValue(entry, core.Settings{Name: name, Team: team})
	component.StateComponent.SetValue(entry, core.State{CurrentState: core.StateIdle})

	bs.updateUnitEntity(unit)
	return unit
}

func (bs *BalanceTestScene) updateUnitEntity(unit *balanceTestUnit) {
	partsMap := make(map[core.PartSlotKey]*core.PartInstanceData)
	partIDMap := map[core.PartSlotKey]string{
		core.PartSlotHead:     unit.headID,
		core.PartSlotRightArm: unit.rArmID,
		core.PartSlotLeftArm:  unit.lArmID,
		core.PartSlotLegs:     unit.legsID,
	}

	for slot, partID := range partIDMap {
		partDef, ok := bs.resources.GameDataManager.GetPartDefinition(partID)
		if !ok {
			log.Printf("Error: Part definition not found for ID %s", partID)
			continue
		}
		partsMap[slot] = &core.PartInstanceData{
			DefinitionID: partDef.ID,
			CurrentArmor: partDef.MaxArmor,
			IsBroken:     false,
		}
	}
	component.PartsComponent.SetValue(unit.entry, core.PartsComponentData{Map: partsMap})

	medalDef, ok := bs.resources.GameDataManager.GetMedalDefinition(unit.medalID)
	if ok {
		component.MedalComponent.SetValue(unit.entry, *medalDef)
	} else {
		log.Printf("Error: Medal definition not found for ID %s", unit.medalID)
	}

	// Update button labels
	if unit.medalButton != nil {
		unit.medalButton.Text().Label = medalDef.Name
	}
	if unit.headButton != nil {
		unit.headButton.Text().Label = bs.mustGetPartDefinition(unit.headID).PartName
	}
	if unit.rArmButton != nil {
		unit.rArmButton.Text().Label = bs.mustGetPartDefinition(unit.rArmID).PartName
	}
	if unit.lArmButton != nil {
		unit.lArmButton.Text().Label = bs.mustGetPartDefinition(unit.lArmID).PartName
	}
	if unit.legsButton != nil {
		unit.legsButton.Text().Label = bs.mustGetPartDefinition(unit.legsID).PartName
	}
}

func (bs *BalanceTestScene) changePart(unit *balanceTestUnit, partType string, direction int) {
	findPartIndex := func(parts []*core.PartDefinition, id string) int {
		for i, p := range parts { if p.ID == id { return i } }
		return -1
	}
	findMedalIndex := func(medals []*core.Medal, id string) int {
		for i, m := range medals { if m.ID == id { return i } }
		return -1
	}

	switch partType {
	case "Medal":
		idx := findMedalIndex(bs.medalList, unit.medalID)
		idx = (idx + direction + len(bs.medalList)) % len(bs.medalList)
		unit.medalID = bs.medalList[idx].ID
	case "Head":
		idx := findPartIndex(bs.headPartsList, unit.headID)
		idx = (idx + direction + len(bs.headPartsList)) % len(bs.headPartsList)
		unit.headID = bs.headPartsList[idx].ID
	case "R-Arm":
		idx := findPartIndex(bs.rArmPartsList, unit.rArmID)
		idx = (idx + direction + len(bs.rArmPartsList)) % len(bs.rArmPartsList)
		unit.rArmID = bs.rArmPartsList[idx].ID
	case "L-Arm":
		idx := findPartIndex(bs.lArmPartsList, unit.lArmID)
		idx = (idx + direction + len(bs.lArmPartsList)) % len(bs.lArmPartsList)
		unit.lArmID = bs.lArmPartsList[idx].ID
	case "Legs":
		idx := findPartIndex(bs.legsPartsList, unit.legsID)
		idx = (idx + direction + len(bs.legsPartsList)) % len(bs.legsPartsList)
		unit.legsID = bs.legsPartsList[idx].ID
	}

	bs.updateUnitEntity(unit)
	bs.recalculate()
}

func (bs *BalanceTestScene) recalculate() {
	// Attacker's right arm is used for calculation by default
	actingPartDef, ok := bs.resources.GameDataManager.GetPartDefinition(bs.attacker.rArmID)
	if !ok { return }

	// --- Hit Chance ---
	successRate := bs.partInfoProvider.GetSuccessRate(bs.attacker.entry, actingPartDef, core.PartSlotRightArm)
	evasion := bs.partInfoProvider.GetEvasionRate(bs.defender.entry)
	hitChance := bs.resources.Config.Hit.BaseChance + (successRate - evasion)
	bs.hitChanceText.Label = fmt.Sprintf("Hit Chance: %.1f%% (S:%.1f vs E:%.1f)", hitChance, successRate, evasion)

	// --- Defense Chance ---
	defenseRate := bs.partInfoProvider.GetDefenseRate(bs.defender.entry)
	defenseChance := bs.resources.Config.Defense.BaseChance + (defenseRate - successRate)
	bs.defenseChanceText.Label = fmt.Sprintf("Defense Chance: %.1f%% (D:%.1f vs S:%.1f)", defenseChance, defenseRate, successRate)

	// --- Damage & Critical ---
	damage, _ := bs.damageCalculator.CalculateDamage(bs.attacker.entry, bs.defender.entry, actingPartDef, core.PartSlotRightArm, false)
	bs.expectedDamageText.Label = fmt.Sprintf("Expected Damage (No Def): %d", damage)

	formula, _ := bs.resources.GameDataManager.Formulas[actingPartDef.Trait]
	criticalChance := bs.resources.Config.Damage.Critical.BaseChance + (successRate * bs.resources.Config.Damage.Critical.SuccessRateFactor) + formula.CriticalRateBonus
	bs.criticalChanceText.Label = fmt.Sprintf("Critical Chance: %.1f%%", criticalChance)
}

func (bs *BalanceTestScene) runSimulation() {
    // For now, we only simulate the attacker's right arm attacking the defender's head
    actingPartDef, ok := bs.resources.GameDataManager.GetPartDefinition(bs.attacker.rArmID)
    if !ok {
        bs.simulationLogText.Label = "Log: Attacker part not found!"
        return
    }

    // 1. Hit Check
    didHit := bs.hitCalculator.CalculateHit(bs.attacker.entry, bs.defender.entry, actingPartDef, core.PartSlotRightArm)
    if !didHit {
        bs.simulationLogText.Label = "Log: Attack Missed!"
        return
    }

    // 2. Defense Check
    defendingPartInst := bs.targetSelector.SelectDefensePart(bs.defender.entry)
    var isDefended bool
	var defendingPartDef *core.PartDefinition
    if defendingPartInst != nil {
        defendingPartDef, _ = bs.partInfoProvider.GetGameDataManager().GetPartDefinition(defendingPartInst.DefinitionID)
        isDefended = bs.hitCalculator.CalculateDefense(bs.attacker.entry, bs.defender.entry, actingPartDef, core.PartSlotRightArm, defendingPartDef)
    } else {
        isDefended = false
    }

    // 3. Damage Calculation
    damage, isCritical := bs.damageCalculator.CalculateDamage(bs.attacker.entry, bs.defender.entry, actingPartDef, core.PartSlotRightArm, isDefended)

    // 4. Format Log Message
    logMsg := "Log: Hit!"
    if isDefended {
        logMsg = fmt.Sprintf("Log: Defended! Dealt %d damage to %s.", damage, defendingPartDef.PartName)
    } else {
		targetPartDef, _ := bs.partInfoProvider.GetGameDataManager().GetPartDefinition(bs.defender.headID) // Default to head
        logMsg = fmt.Sprintf("Log: Hit! Dealt %d damage to %s.", damage, targetPartDef.PartName)
    }

    if isCritical {
        logMsg += " (CRITICAL!)"
    }

    bs.simulationLogText.Label = logMsg
}
