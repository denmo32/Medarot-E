
package scene

import (
	"errors"
	"image/color"

	"medarot-ebiten/data"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// ================
// Map Scene specific structs
// ================

type MapScene struct {
	resources     *data.SharedResources
	manager       *SceneManager
	entityManager *mapEntityManager
	config        *mapGameConfig
}

type mapGameConfig struct {
	ScreenWidth       int
	ScreenHeight      int
	TileSize          int
	MapWidth          int
	MapHeight         int
	PlayerSpeed       int
	InitialMoveDelay  int
	ContinuousMoveDelay int
}

var mapConfig = mapGameConfig{
	ScreenWidth:        1280,
	ScreenHeight:       720,
	TileSize:          32,
	MapWidth:          40,
	MapHeight:         22,
	PlayerSpeed:       15,
	InitialMoveDelay:  15,
	ContinuousMoveDelay: 15,
}

type mapEntityManager struct {
	entities []mapEntity
}

func (em *mapEntityManager) AddEntity(e mapEntity) {
	em.entities = append(em.entities, e)
}

func (em *mapEntityManager) Update() error {
	for _, e := range em.entities {
		if err := e.Update(); err != nil {
			return err
		}
	}
	return nil
}

func (em *mapEntityManager) Draw(screen *ebiten.Image) {
	for _, e := range em.entities {
		e.Draw(screen)
	}
}

type mapEntity interface {
	Update() error
	Draw(screen *ebiten.Image)
}

type mapInputSystem struct{}

func (is *mapInputSystem) GetDirection() (dx, dy int) {
	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		return 0, -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		return 0, 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		return -1, 0
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		return 1, 0
	}
	return 0, 0
}

type mapLayer struct {
	data [][]int
}

func newMapLayer(width, height int) *mapLayer {
	data := make([][]int, height)
	for y := range data {
		data[y] = make([]int, width)
	}
	return &mapLayer{data: data}
}

func (ml *mapLayer) SetTile(x, y, tileType int) error {
	if x < 0 || x >= len(ml.data[0]) || y < 0 || y >= len(ml.data) {
		return errors.New("tile position out of bounds")
	}
	ml.data[y][x] = tileType
	return nil
}

func (ml *mapLayer) GetTile(x, y int) (int, error) {
	if x < 0 || x >= len(ml.data[0]) || y < 0 || y >= len(ml.data) {
		return -1, errors.New("tile position out of bounds")
	}
	return ml.data[y][x], nil
}

type gameMap struct {
	TerrainLayer *mapLayer
}

func newGameMap() *gameMap {
	terrain := newMapLayer(mapConfig.MapWidth, mapConfig.MapHeight)
	for y := 0; y < mapConfig.MapHeight; y++ {
		for x := 0; x < mapConfig.MapWidth; x++ {
			var tileType int
			switch {
			case x == 0 || x == mapConfig.MapWidth-1 || y == 0 || y == mapConfig.MapHeight-1:
				tileType = 2 // Water
			case x%3 == 0 && y%3 == 0:
				tileType = 1 // Forest
			default:
				tileType = 0 // Grass
			}
			terrain.SetTile(x, y, tileType)
		}
	}
	return &gameMap{TerrainLayer: terrain}
}

func (gm *gameMap) Update() error {
	return nil
}

func (gm *gameMap) IsWalkable(x, y int) bool {
	tile, err := gm.TerrainLayer.GetTile(x, y)
	if err != nil {
		return false
	}
	return tile != 2 // Water is not walkable
}

func (gm *gameMap) Draw(screen *ebiten.Image) {
	for y := 0; y < mapConfig.MapHeight; y++ {
		for x := 0; x < mapConfig.MapWidth; x++ {
			tileType, _ := gm.TerrainLayer.GetTile(x, y)
			var tileColor color.Color
			switch tileType {
			case 0:
				tileColor = color.RGBA{34, 139, 34, 255}
			case 1:
				tileColor = color.RGBA{0, 100, 0, 255}
			case 2:
				tileColor = color.RGBA{0, 0, 255, 255}
			}
			ebitenutil.DrawRect(screen, float64(x*mapConfig.TileSize), float64(y*mapConfig.TileSize), float64(mapConfig.TileSize), float64(mapConfig.TileSize), tileColor)
		}
	}
}

type player struct {
	tileX, tileY int
	moveTimer    int
	isMoving     bool
	moveDirX, moveDirY int
	canMove      bool
	input        *mapInputSystem
	gameMap      *gameMap
	moveDelay    int
	lastDirX, lastDirY int
}

func newPlayer(x, y int, input *mapInputSystem, gameMap *gameMap) *player {
	return &player{
		tileX: x, tileY: y, canMove: true, input: input, gameMap: gameMap,
	}
}

func (p *player) Update() error {
	if p.isMoving {
		p.moveTimer++
		if p.moveTimer >= mapConfig.PlayerSpeed {
			p.isMoving = false
			p.moveTimer = 0
			p.canMove = true
			p.moveDelay = 0
		}
		return nil
	}

	if p.moveDelay > 0 {
		p.moveDelay--
	}

	dx, dy := p.input.GetDirection()
	if dx != 0 || dy != 0 {
		p.lastDirX, p.lastDirY = dx, dy
		if p.canMove && p.moveDelay == 0 {
			p.tryMove(dx, dy)
		}
	} else {
		p.moveDelay = 0
	}
	return nil
}

func (p *player) tryMove(dx, dy int) {
	newX, newY := p.tileX+dx, p.tileY+dy
	if !p.gameMap.IsWalkable(newX, newY) {
		return
	}
	p.tileX, p.tileY = newX, newY
	p.isMoving, p.moveDirX, p.moveDirY = true, dx, dy
	p.moveTimer, p.canMove = 0, false
	if p.moveDelay == 0 {
		p.moveDelay = mapConfig.InitialMoveDelay
	} else {
		p.moveDelay = mapConfig.ContinuousMoveDelay
	}
}

func (p *player) Draw(screen *ebiten.Image) {
	playerX, playerY := float64(p.tileX*mapConfig.TileSize), float64(p.tileY*mapConfig.TileSize)
	if p.isMoving {
		progress := float64(p.moveTimer) / float64(mapConfig.PlayerSpeed)
		playerX = float64(p.tileX-p.moveDirX)*float64(mapConfig.TileSize) + (float64(p.tileX)*float64(mapConfig.TileSize)-float64(p.tileX-p.moveDirX)*float64(mapConfig.TileSize))*progress
		playerY = float64(p.tileY-p.moveDirY)*float64(mapConfig.TileSize) + (float64(p.tileY)*float64(mapConfig.TileSize)-float64(p.tileY-p.moveDirY)*float64(mapConfig.TileSize))*progress
	}
	ebitenutil.DrawCircle(screen, playerX+float64(mapConfig.TileSize)/2, playerY+float64(mapConfig.TileSize)/2, float64(mapConfig.TileSize)/3, color.RGBA{255, 0, 0, 255})
}

// ================
// Scene Implementation
// ================

func NewMapScene(res *data.SharedResources, manager *SceneManager) (Scene, error) {
	ms := &MapScene{
		resources:     res,
		manager:       manager,
		entityManager: &mapEntityManager{},
		config:        &mapConfig,
	}

	inputSystem := &mapInputSystem{}
	gameMap := newGameMap()
	player := newPlayer(mapConfig.MapWidth/2, mapConfig.MapHeight/2, inputSystem, gameMap)

	ms.entityManager.AddEntity(gameMap)
	ms.entityManager.AddEntity(player)

	return ms, nil
}

func (ms *MapScene) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		ms.manager.GoToTitleScene()
		return nil
	}
	return ms.entityManager.Update()
}

func (ms *MapScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	ms.entityManager.Draw(screen)
	ebitenutil.DebugPrint(screen, "Press ESC to return to Title")
}

func (ms *MapScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ms.config.ScreenWidth, ms.config.ScreenHeight
}
