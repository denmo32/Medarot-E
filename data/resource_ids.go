package data

import (
	resource "github.com/quasilyte/ebitengine-resource"
)

// Resource IDs
const (
	_ resource.FontID = iota
	FontMPLUS1pRegular
	FontModalButton
	FontMessageWindow
)

const (
	_ resource.ImageID = iota
	ImageBattleBackground
)

const (
	_ resource.RawID = iota
	RawMedalsCSV
	RawPartsCSV
	RawMedarotsCSV
	RawFormulasJSON
	RawMessagesJSON
)
