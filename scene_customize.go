package main

import (
	"fmt"
	"image/color"
	"log"
	"sort"
	"strings"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CustomizeScene は機体カスタマイズ画面のシーンです
type CustomizeScene struct {
	resources *SharedResources
	ui        *ebitenui.UI
	nextScene SceneType

	statusText              *widget.Text
	medalNameButton         *widget.Button
	headNameButton          *widget.Button
	rArmNameButton          *widget.Button
	lArmNameButton          *widget.Button
	legsNameButton          *widget.Button
	medarotSelectionButtons []*widget.Button

	playerMedarots            []*MedarotData
	currentTargetMedarotIndex int

	medalList     []Medal
	headPartsList []*Part
	rArmPartsList []*Part
	lArmPartsList []*Part
	legsPartsList []*Part

	currentMedalIndex int
	currentHeadIndex  int
	currentRArmIndex  int
	currentLArmIndex  int
	currentLegsIndex  int
}

func NewCustomizeScene(res *SharedResources) *CustomizeScene {
	cs := &CustomizeScene{
		resources: res,
		nextScene: SceneTypeCustomize,
	}

	for i := range res.GameData.Medarots {
		if res.GameData.Medarots[i].Team == Team1 {
			cs.playerMedarots = append(cs.playerMedarots, &res.GameData.Medarots[i])
		}
	}
	sort.Slice(cs.playerMedarots, func(i, j int) bool {
		return cs.playerMedarots[i].DrawIndex < cs.playerMedarots[j].DrawIndex
	})

	if len(cs.playerMedarots) == 0 {
		rootContainer := widget.NewContainer()
		rootContainer.AddChild(widget.NewText(widget.TextOpts.Text("Player team not found.", res.Font, color.White)))
		cs.ui = &ebitenui.UI{Container: rootContainer}
		return cs
	}

	cs.currentTargetMedarotIndex = 0
	cs.setupPartLists()
	rootContainer := cs.createLayout()
	cs.ui = &ebitenui.UI{Container: rootContainer}
	cs.refreshUIForSelectedMedarot()

	return cs
}

func (cs *CustomizeScene) setupPartLists() {
	cs.medalList = cs.resources.GameData.Medals

	allParts := []*Part{}
	for _, p := range cs.resources.GameData.AllParts {
		allParts = append(allParts, p)
	}
	sort.Slice(allParts, func(i, j int) bool { return allParts[i].ID < allParts[j].ID })

	cs.headPartsList = nil
	cs.rArmPartsList = nil
	cs.lArmPartsList = nil
	cs.legsPartsList = nil
	for _, p := range allParts {
		switch p.Type {
		case PartTypeHead:
			cs.headPartsList = append(cs.headPartsList, p)
		case PartTypeRArm:
			cs.rArmPartsList = append(cs.rArmPartsList, p)
		case PartTypeLArm:
			cs.lArmPartsList = append(cs.lArmPartsList, p)
		case PartTypeLegs:
			cs.legsPartsList = append(cs.legsPartsList, p)
		}
	}
}

func (cs *CustomizeScene) createLayout() *widget.Container {
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{true, true}, []bool{true}),
			widget.GridLayoutOpts.Spacing(20, 0),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(20)),
		)),
	)

	leftPanel := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(15),
		)),
	)
	rootContainer.AddChild(leftPanel)

	rightPanel := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(15)),
		)),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{50, 50, 70, 200})),
	)
	rootContainer.AddChild(rightPanel)

	cs.statusText = widget.NewText(
		widget.TextOpts.Text("", cs.resources.Font, color.White),
	)
	rightPanel.AddChild(cs.statusText)

	medarotSelectionContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Spacing(10),
		)),
	)
	leftPanel.AddChild(medarotSelectionContainer)

	// ★★★ このブロックで Image を設定 ★★★
	buttonImage := &widget.ButtonImage{
		Idle:    image.NewNineSliceColor(cs.resources.Config.UI.Colors.Gray),
		Hover:   image.NewNineSliceColor(color.NRGBA{180, 180, 180, 255}),
		Pressed: image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
	}
	for i := 0; i < len(cs.playerMedarots); i++ {
		idx := i
		button := widget.NewButton(
			widget.ButtonOpts.Image(buttonImage), // ★★★ Image設定を追加 ★★★
			widget.ButtonOpts.Text(fmt.Sprintf("機体%d", idx+1), cs.resources.Font, &widget.ButtonTextColor{Idle: color.White}),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				cs.selectMedarot(idx)
			}),
		)
		cs.medarotSelectionButtons = append(cs.medarotSelectionButtons, button)
		medarotSelectionContainer.AddChild(button)
	}

	cs.medalNameButton = cs.createPartSelectionRow(leftPanel, "Medal")
	cs.headNameButton = cs.createPartSelectionRow(leftPanel, "Head")
	cs.rArmNameButton = cs.createPartSelectionRow(leftPanel, "Right Arm")
	cs.lArmNameButton = cs.createPartSelectionRow(leftPanel, "Left Arm")
	cs.legsNameButton = cs.createPartSelectionRow(leftPanel, "Legs")

	saveButton := widget.NewButton(
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle:    image.NewNineSliceColor(cs.resources.Config.UI.Colors.Gray),
			Hover:   image.NewNineSliceColor(color.NRGBA{180, 180, 180, 255}),
			Pressed: image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
		}),
		widget.ButtonOpts.Text("Save & Back to Title", cs.resources.Font, &widget.ButtonTextColor{Idle: color.White}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(10)),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionEnd,
		})),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			err := SaveMedarotLoadouts("data/medarots.csv", cs.resources.GameData.Medarots)
			if err != nil {
				log.Printf("ERROR: Failed to save medarots.csv: %v", err)
			} else {
				log.Println("Successfully saved medarots.csv")
			}
			cs.nextScene = SceneTypeTitle
		}),
	)
	leftPanel.AddChild(saveButton)

	return rootContainer
}

func (cs *CustomizeScene) selectMedarot(index int) {
	if cs.currentTargetMedarotIndex == index {
		return
	}
	cs.currentTargetMedarotIndex = index
	cs.refreshUIForSelectedMedarot()
}

func (cs *CustomizeScene) refreshUIForSelectedMedarot() {
	target := cs.playerMedarots[cs.currentTargetMedarotIndex]

	cs.currentMedalIndex = findIndexByID(cs.medalList, target.MedalID, func(m Medal) string { return m.ID })
	cs.currentHeadIndex = findIndexByID(cs.headPartsList, target.HeadID, func(p *Part) string { return p.ID })
	cs.currentRArmIndex = findIndexByID(cs.rArmPartsList, target.RightArmID, func(p *Part) string { return p.ID })
	cs.currentLArmIndex = findIndexByID(cs.lArmPartsList, target.LeftArmID, func(p *Part) string { return p.ID })
	cs.currentLegsIndex = findIndexByID(cs.legsPartsList, target.LegsID, func(p *Part) string { return p.ID })

	cs.medalNameButton.Text().Label = cs.getCurrentName("Medal")
	cs.headNameButton.Text().Label = cs.getCurrentName("Head")
	cs.rArmNameButton.Text().Label = cs.getCurrentName("Right Arm")
	cs.lArmNameButton.Text().Label = cs.getCurrentName("Left Arm")
	cs.legsNameButton.Text().Label = cs.getCurrentName("Legs")

	cs.updateStatus(target.MedalID)
	cs.updateMedarotSelectionButtons()
}

func (cs *CustomizeScene) updateMedarotSelectionButtons() {
	highlightedImage := &widget.ButtonImage{
		Idle:    image.NewNineSliceColor(color.NRGBA{100, 100, 120, 255}),
		Hover:   image.NewNineSliceColor(color.NRGBA{120, 120, 140, 255}),
		Pressed: image.NewNineSliceColor(color.NRGBA{80, 80, 100, 255}),
	}
	normalImage := &widget.ButtonImage{
		Idle:    image.NewNineSliceColor(cs.resources.Config.UI.Colors.Gray),
		Hover:   image.NewNineSliceColor(color.NRGBA{180, 180, 180, 255}),
		Pressed: image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
	}

	for i, button := range cs.medarotSelectionButtons {
		if i == cs.currentTargetMedarotIndex {
			button.Image = highlightedImage
		} else {
			button.Image = normalImage
		}
	}
}

func (cs *CustomizeScene) createPartSelectionRow(parent *widget.Container, label string) *widget.Button {
	res := cs.resources
	rowContainer := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Stretch: true,
		})),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{false, true, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(10, 0),
		)),
	)
	parent.AddChild(rowContainer)

	buttonImage := &widget.ButtonImage{
		Idle:    image.NewNineSliceColor(res.Config.UI.Colors.Gray),
		Hover:   image.NewNineSliceColor(color.NRGBA{180, 180, 180, 255}),
		Pressed: image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
	}
	textColor := &widget.ButtonTextColor{Idle: color.White}

	leftButton := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("◀", res.Font, textColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) { cs.changeSelection(label, -1) }),
	)
	rowContainer.AddChild(leftButton)

	nameButton := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("", res.Font, textColor),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			currentID := cs.getCurrentID(label)
			cs.updateStatus(currentID)
		}),
	)
	rowContainer.AddChild(nameButton)

	rightButton := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("▶", res.Font, textColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) { cs.changeSelection(label, 1) }),
	)
	rowContainer.AddChild(rightButton)

	nameButton.Text().Label = cs.getCurrentName(label)
	return nameButton
}

func (cs *CustomizeScene) changeSelection(label string, direction int) {
	target := cs.playerMedarots[cs.currentTargetMedarotIndex]
	var listSize int
	var currentIndex *int
	var nameButton *widget.Button

	switch label {
	case "Medal":
		listSize, currentIndex, nameButton = len(cs.medalList), &cs.currentMedalIndex, cs.medalNameButton
	case "Head":
		listSize, currentIndex, nameButton = len(cs.headPartsList), &cs.currentHeadIndex, cs.headNameButton
	case "Right Arm":
		listSize, currentIndex, nameButton = len(cs.rArmPartsList), &cs.currentRArmIndex, cs.rArmNameButton
	case "Left Arm":
		listSize, currentIndex, nameButton = len(cs.lArmPartsList), &cs.currentLArmIndex, cs.lArmNameButton
	case "Legs":
		listSize, currentIndex, nameButton = len(cs.legsPartsList), &cs.currentLegsIndex, cs.legsNameButton
	}

	if listSize == 0 {
		return
	}
	*currentIndex = (*currentIndex + direction + listSize) % listSize

	newID := cs.getCurrentID(label)
	switch label {
	case "Medal":
		target.MedalID = newID
	case "Head":
		target.HeadID = newID
	case "Right Arm":
		target.RightArmID = newID
	case "Left Arm":
		target.LeftArmID = newID
	case "Legs":
		target.LegsID = newID
	}

	nameButton.Text().Label = cs.getCurrentName(label)
	cs.updateStatus(newID)
}

func (cs *CustomizeScene) updateStatus(id string) {
	var sb strings.Builder
	if part, ok := cs.resources.GameData.AllParts[id]; ok {
		sb.WriteString(fmt.Sprintf("Name: %s\n", part.PartName))
		sb.WriteString(fmt.Sprintf("Type: %s\n", part.Type))
		sb.WriteString(fmt.Sprintf("Category: %s\n", part.Category))
		sb.WriteString(fmt.Sprintf("Trait: %s\n\n", part.Trait))
		sb.WriteString(fmt.Sprintf("Armor: %d\n", part.Armor))
		sb.WriteString(fmt.Sprintf("Power: %d\n", part.Power))
		sb.WriteString(fmt.Sprintf("Accuracy: %d\n", part.Accuracy))
		sb.WriteString(fmt.Sprintf("Charge: %d\n", part.Charge))
		sb.WriteString(fmt.Sprintf("Cooldown: %d\n", part.Cooldown))
		if part.Type == PartTypeLegs {
			sb.WriteString(fmt.Sprintf("\nPropulsion: %d\n", part.Propulsion))
			sb.WriteString(fmt.Sprintf("Mobility: %d\n", part.Mobility))
		}
	} else if medal, ok := findMedalDataByID(cs.resources.GameData.Medals, id); ok {
		sb.WriteString(fmt.Sprintf("Name: %s\n", medal.Name))
		sb.WriteString(fmt.Sprintf("Personality: %s\n\n", medal.Personality))
		sb.WriteString(fmt.Sprintf("Skill Level: %d\n", medal.SkillLevel))
	} else {
		sb.WriteString("No data available.")
	}
	cs.statusText.Label = sb.String()
}

func (cs *CustomizeScene) Update() (SceneType, error) {
	cs.ui.Update()
	return cs.nextScene, nil
}

func (cs *CustomizeScene) Draw(screen *ebiten.Image) {
	screen.Fill(cs.resources.Config.UI.Colors.Background)
	cs.ui.Draw(screen)
}

func findIndexByID[T any](slice []T, id string, getID func(T) string) int {
	for i, v := range slice {
		if getID(v) == id {
			return i
		}
	}
	return 0
}

func findMedalDataByID(medals []Medal, id string) (*Medal, bool) {
	for i := range medals {
		if medals[i].ID == id {
			return &medals[i], true
		}
	}
	return nil, false
}

func (cs *CustomizeScene) getCurrentID(label string) string {
	switch label {
	case "Medal":
		if len(cs.medalList) > 0 {
			return cs.medalList[cs.currentMedalIndex].ID
		}
	case "Head":
		if len(cs.headPartsList) > 0 {
			return cs.headPartsList[cs.currentHeadIndex].ID
		}
	case "Right Arm":
		if len(cs.rArmPartsList) > 0 {
			return cs.rArmPartsList[cs.currentRArmIndex].ID
		}
	case "Left Arm":
		if len(cs.lArmPartsList) > 0 {
			return cs.lArmPartsList[cs.currentLArmIndex].ID
		}
	case "Legs":
		if len(cs.legsPartsList) > 0 {
			return cs.legsPartsList[cs.currentLegsIndex].ID
		}
	}
	return ""
}

func (cs *CustomizeScene) getCurrentName(label string) string {
	switch label {
	case "Medal":
		if len(cs.medalList) > 0 {
			return cs.medalList[cs.currentMedalIndex].Name
		}
	case "Head":
		if len(cs.headPartsList) > 0 {
			return cs.headPartsList[cs.currentHeadIndex].PartName
		}
	case "Right Arm":
		if len(cs.rArmPartsList) > 0 {
			return cs.rArmPartsList[cs.currentRArmIndex].PartName
		}
	case "Left Arm":
		if len(cs.lArmPartsList) > 0 {
			return cs.lArmPartsList[cs.currentLArmIndex].PartName
		}
	case "Legs":
		if len(cs.legsPartsList) > 0 {
			return cs.legsPartsList[cs.currentLegsIndex].PartName
		}
	}
	return "N/A"
}
