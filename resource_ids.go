package main

import (
	resource "github.com/quasilyte/ebitengine-resource"
)

// Resource IDs
const (
	_ resource.FontID = iota
	FontMPLUS1pRegular
	FontModalButton   // 追加
	FontMessageWindow // 追加
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
	RawMessagesJSON // 追加
)
