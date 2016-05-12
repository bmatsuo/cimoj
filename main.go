package main

import (
	"image"
	"log"
	"math/rand"
	"time"

	"github.com/Ariemeth/termloop"
)

// CrunchConfig defines the crunching board.
type CrunchConfig struct {
	NumCol           int
	ColVSpace        int
	ColSpace         int
	ColDepth         int
	CritterSizeSmall int
	CritterSizeLarge int
}

func (conf *CrunchConfig) boardSize() image.Point {
	return image.Point{
		X: conf.NumCol*conf.CritterSizeLarge + (conf.NumCol+1)*conf.ColSpace,
		Y: (conf.ColDepth+1)*(conf.CritterSizeLarge+conf.ColVSpace) + conf.CritterSizeLarge, // extra depth for death indication -- below vine
	}
}

func (conf *CrunchConfig) colLength() int {
	return conf.ColDepth * (conf.CritterSizeLarge + conf.ColVSpace)
}

func main() {
	config := &CrunchConfig{
		NumCol:           6,
		ColSpace:         2,
		ColDepth:         5,
		CritterSizeSmall: 1,
		CritterSizeLarge: 1,
	}

	size := config.boardSize()
	log.Printf("size: %v", size)

	game := termloop.NewGame()

	level := termloop.NewBaseLevel(termloop.Cell{
		Bg: termloop.ColorBlack,
		Fg: termloop.ColorWhite,
	})

	board := termloop.NewBaseLevel(termloop.Cell{})
	board.SetOffset(2, 1)

	border := termloop.NewEntity(0, 0, size.X+2, size.Y+2)
	for i := 0; i < size.X+2; i++ {
		border.SetCell(i, 0, &termloop.Cell{Fg: termloop.ColorGreen, Ch: '~'})
		border.SetCell(i, size.Y+1, &termloop.Cell{Fg: termloop.ColorGreen, Ch: 'v'})
	}
	for j := 1; j < size.Y+1; j++ {
		border.SetCell(0, j, &termloop.Cell{Fg: termloop.ColorGreen, Ch: '|'})
		border.SetCell(size.X+1, j, &termloop.Cell{Fg: termloop.ColorGreen, Ch: '|'})
	}
	board.AddEntity(border)
	//board.AddEntity(termloop.NewRectangle(1, 1, size.X, size.Y, termloop.ColorCyan))

	for i := 0; i < config.NumCol; i++ {
		posX := 1 + config.ColSpace + config.CritterSizeLarge/2 + i*(config.ColSpace+config.CritterSizeLarge)
		column := termloop.NewEntity(posX, 1, 1, config.colLength())
		for j := 0; j < config.colLength(); j++ {
			column.SetCell(0, j, &termloop.Cell{Fg: termloop.ColorGreen, Ch: '|'})
		}
		board.AddEntity(column)
	}

	crunch := NewCrunchGame(config, board)

	level.AddEntity(crunch)

	game.Screen().SetLevel(level)
	game.Start()
}

// Color is the a color in a crunch game.
type Color uint8

// Color constants with special significance.
const (
	ColorNone Color = iota
	ColorMulti
	ColorBomb
	ColorPlayer
	ColorBug
)

// ColorMap maps game colors to their actual representation in a terminal.
type ColorMap interface {
	Color(Color) termloop.Attr
}

// SetCellColor sets the foreground of m according to a color map
func SetCellColor(c *termloop.Cell, m ColorMap, fg Color) {
	c.Fg = m.Color(fg)
}

// BugType enumerates the types of possible bugs
type BugType uint8

// BugType values that are acceptable
const (
	BugSmall BugType = iota
	BugLarge
	BugMagic
	BugBomb
)

// Bug is a bug that crawls down the vines.  Bugs have distinct color.  Large
// bugs can only eat smaller bugs of the same color.
type Bug struct {
	Type   BugType
	Color  Color
	entity *termloop.Entity
}

// CrunchGame contains a player, critters, a score, and other game state.
type CrunchGame struct {
	config    *CrunchConfig
	playerPos int
	player    *Player
	vines     [][]*Bug
	rand      Rand
	level     *termloop.BaseLevel
}

// NewCrunchGame initializes a new CrunchGame.
func NewCrunchGame(config *CrunchConfig, level *termloop.BaseLevel) *CrunchGame {
	g := &CrunchGame{
		config: config,
		rand:   defaultRand(),
		level:  level,
	}
	g.vines = make([][]*Bug, config.NumCol)
	for i := range g.vines {
		g.vines[i] = make([]*Bug, 0, config.ColDepth+1)
	}

	g.playerPos = config.NumCol
	g.player = &Player{
		entity: termloop.NewEntity(g.colX(g.playerPos), g.config.boardSize().Y, 1, 1),
		level:  level,
	}
	g.player.entity.SetCell(0, 0, g.player.cell())
	g.level.AddEntity(g.player.entity)
	return g
}

func (g *CrunchGame) colX(i int) int {
	if i >= g.config.NumCol {
		return g.config.boardSize().X
	}
	return 1 + g.config.ColSpace + i*(g.config.ColSpace+1+g.config.CritterSizeLarge/2)
}

func defaultRand() Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (g *CrunchGame) randomColorBug(n int) Color {
	return ColorBug + Color(g.rand.Intn(n))
}

func (g *CrunchGame) randomBug() *Bug {
	roll := g.rand.Intn(100) + 1

	roll -= 45
	if roll < 0 {
		return &Bug{
			Type:  BugSmall,
			Color: g.randomColorBug(2),
		}
	}

	roll -= 45
	if roll < 0 {
		return &Bug{
			Type:  BugLarge,
			Color: g.randomColorBug(2),
		}
	}

	roll -= 8
	if roll < 0 {
		return &Bug{
			Type:  BugBomb,
			Color: ColorBomb,
		}
	}

	return &Bug{
		Type:  BugMagic,
		Color: ColorMulti,
	}
}

func (g *CrunchGame) spawnBugs() {
	// for now we do something simple and spawn bugs in all rows simultaneously
	for i := range g.vines {
		g.vines[i] = g.vines[i][:len(g.vines[i])+1]
		copy(g.vines[i], g.vines[i][1:])
		g.vines[i][0] = g.randomBug()
	}
}

// Draw implements termloop.Drawable
func (g *CrunchGame) Draw(screen *termloop.Screen) {
	g.level.Draw(screen)
}

// Tick implements termloop.Drawable
func (g *CrunchGame) Tick(event termloop.Event) {
	if event.Type == termloop.EventKey { // Is it a keyboard event?
		switch event.Ch { // If so, switch on the pressed key.
		case 'l':
			if g.playerPos < g.config.NumCol {
				g.playerPos++
				g.player.entity.SetPosition(g.colX(g.playerPos), g.config.boardSize().Y)
			}
		case 'h':
			if g.playerPos > 0 {
				g.playerPos--
				g.player.entity.SetPosition(g.colX(g.playerPos), g.config.boardSize().Y)
			}
		case 'k':
			if g.player.contains != nil {
				g.player.contains = nil
			} else {
				g.player.contains = &Bug{Type: BugSmall, Color: ColorBug}
			}
			g.player.entity.SetCell(0, 0, g.player.cell())
		case 'j':
			// TODO: puke on the side of the screen when your buddy is around
		}
	}
}

// Player is a player in a CrunchGame
type Player struct {
	config   *CrunchConfig
	entity   *termloop.Entity
	contains *Bug // any contained bug will have its entity removed
	prevX    int
	prevY    int
	level    *termloop.BaseLevel
}

// Draw implements termloop.Drawable
func (p *Player) Draw(screen *termloop.Screen) {
	p.entity.Draw(screen)
}

func (p Player) cell() *termloop.Cell {
	cell := &termloop.Cell{}
	if p.contains != nil {
		cell.Ch = '⊙'
		SetCellColor(cell, defaultColorMap, p.contains.Color)
	} else {
		cell.Ch = 'o'
		SetCellColor(cell, defaultColorMap, ColorPlayer)
	}
	return cell
}

// Tick implements termloop.Drawable
func (p *Player) Tick(event termloop.Event) {
	p.entity.Tick(event)
}

var defaultColorMap = simpleColorMap{
	ColorNone:   termloop.ColorBlack,
	ColorMulti:  termloop.ColorBlack, // ColorMulti is not used
	ColorBomb:   termloop.ColorRed,
	ColorPlayer: termloop.ColorMagenta,

	ColorBug + 0: termloop.ColorYellow,
	ColorBug + 1: termloop.ColorBlue,
}

type simpleColorMap []termloop.Attr

func (m simpleColorMap) Color(c Color) termloop.Attr {
	if len(m) == 0 {
		panic("empty color map")
	}
	if int(c) < len(m) {
		return m[c]
	}
	return m[ColorNone]
}

func cell(c rune) *termloop.Cell {
	return &termloop.Cell{Ch: c}
}

// Rand wraps PRNG implementations so that behavior of randomized things can be
// tested more easily.
type Rand interface {
	Intn(n int) int
}
