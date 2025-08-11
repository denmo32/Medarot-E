package ui

import (
	"image/color"
	"medarot-ebiten/data"
	"github.com/ebitenui/ebitenui/widget"
)

func createRootContainer() *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewStackedLayout()),
	)
}

func createBaseLayoutContainer() *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{true, false}),
			widget.GridLayoutOpts.Spacing(0, 10),
		)),
	)
}

func createCommonBottomPanel(config *data.Config, uiFactory *UIFactory, gameDataManager *data.GameDataManager) *UIPanel {
	return NewPanel(&PanelOptions{
		PanelWidth:      814,
		PanelHeight:     180,
		Padding:         widget.NewInsetsSimple(5),
		Spacing:         5,
		BackgroundColor: color.NRGBA{50, 50, 70, 200},
		BorderColor:     config.UI.Colors.Gray,
		BorderThickness: 5,
		CenterContent:   true,
	}, uiFactory.imageGenerator, gameDataManager.Font)
}
