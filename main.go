package main

import (
	"image"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/JoelOtter/termloop"
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
	logf, err := os.Create("cimoj-log.txt")
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(logf)

	config := &CrunchConfig{
		NumCol:           6,
		ColSpace:         2,
		ColDepth:         10,
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
		column.Fill(&termloop.Cell{Fg: termloop.ColorGreen, Ch: '|'})
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
	ColorExploded
	ColorPlayer
	ColorBug
)

// ColorMap maps game colors to their actual representation in a terminal.
type ColorMap interface {
	Color(Color) termloop.Attr
}

// SetCellColor sets the foreground of c according to a color map
func SetCellColor(c *termloop.Cell, m ColorMap, fg Color) {
	c.Bg = termloop.ColorBlack
	c.Fg = m.Color(fg)
}

// SetCellColorBg sets the foreground and background of c according to a color
// map
func SetCellColorBg(c *termloop.Cell, m ColorMap, fg, bg Color) {
	c.Fg = m.Color(fg)
	c.Bg = m.Color(bg)
}

// BugType enumerates the types of possible bugs
type BugType uint8

// BugType values that are acceptable
const (
	BugSmall BugType = iota
	BugLarge
	BugGnat
	BugMagic
	BugBomb
)

// Bug is a bug that crawls down the vines.  Bugs have distinct color.  Large
// bugs can only eat smaller bugs of the same color.
type Bug struct {
	Type     BugType
	Color    Color
	RColor   Color
	EColor   Color
	Exploded bool
	Eaten    int8
	Rune     rune
	entity   *termloop.Entity
}

// ColorEffective returns the currently drawn color for the bug.
func (b *Bug) ColorEffective() Color {
	if b.Color != ColorMulti {
		return b.Color
	}
	if b.EColor != ColorNone && b.EColor != ColorMulti {
		return b.EColor
	}
	return b.RColor
}

// CrunchGame contains a player, critters, a score, and other game state.
type CrunchGame struct {
	config        *CrunchConfig
	playerPos     int
	player        *Player
	vines         [][]*Bug
	pendingExplos []image.Point
	pendingChains []image.Point
	pendingMagics []image.Point
	rand          Rand
	spawnTime     time.Time
	multis        map[*Bug]struct{}
	multisTime    time.Time
	level         *termloop.BaseLevel
}

// NewCrunchGame initializes a new CrunchGame.
func NewCrunchGame(config *CrunchConfig, level *termloop.BaseLevel) *CrunchGame {
	g := &CrunchGame{
		config: config,
		rand:   defaultRand(),
		multis: make(map[*Bug]struct{}),
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

func (g *CrunchGame) gameOver() bool {
	for i := range g.vines {
		if len(g.vines[i]) > g.config.ColDepth {
			return true
		}
	}
	return false
}

func (g *CrunchGame) randomColorBug(n int) Color {
	return ColorBug + Color(g.rand.Intn(n))
}

func (g *CrunchGame) randomBug() *Bug {
	roll := g.rand.Intn(100) + 1

	roll -= 30
	if roll < 0 {
		return g.createBug(BugSmall, g.randomColorBug(2))
	}

	roll -= 30
	if roll < 0 {
		return g.createBug(BugLarge, g.randomColorBug(2))
	}

	roll -= 30
	if roll < 0 {
		return g.createBug(BugGnat, ColorNone)
	}

	roll -= 8
	if roll < 0 {
		return g.createBug(BugBomb, ColorBomb)
	}

	return g.createBug(BugMagic, ColorMulti)
}

func (g *CrunchGame) createBug(typ BugType, c Color) *Bug {
	b := &Bug{
		Type:  typ,
		Color: c,
	}
	b.Rune = g.assignRune(b)
	return b
}

func (g *CrunchGame) assignRune(bug *Bug) rune {
	switch bug.Type {
	case BugSmall:
		if bug.Eaten > 0 {
			return '⊛'
		}
		return 'o'
	case BugLarge:
		if bug.Eaten > 0 {
			return '@'
		}
		return 'O'
	case BugGnat:
		const gnats = "`'~"
		return rune(gnats[g.rand.Intn(len(gnats))])
	case BugBomb:
		if bug.Eaten > 0 {
			return '&'
		}
		return '8'
	case BugMagic:
		if bug.Eaten > 0 {
			return '*'
		}
		return '+'
	}
	return 'x'
}

func (g *CrunchGame) spawnBugs() {
	// for now we do something simple and spawn bugs in all rows simultaneously
	for i := range g.vines {
		g.vines[i] = g.vines[i][:len(g.vines[i])+1]
		copy(g.vines[i][1:], g.vines[i][0:]) // shift bugs "down"
		g.vines[i][0] = g.randomBug()
		g.vines[i][0].entity = termloop.NewEntity(0, 0, 1, 1)
		if g.vines[i][0].Color == ColorMulti {
			g.multis[g.vines[i][0]] = struct{}{}
			g.vines[i][0].entity.SetCell(0, 0, &termloop.Cell{
				Fg: defaultColorMap.Color(g.randMultiColor()),
				Ch: g.vines[i][0].Rune,
			})
		} else {
			g.vines[i][0].entity.SetCell(0, 0, &termloop.Cell{
				Fg: defaultColorMap.Color(g.vines[i][0].Color),
				Ch: g.vines[i][0].Rune,
			})
		}
		g.level.AddEntity(g.vines[i][0].entity)
		cx := g.colX(i)
		size := g.config.boardSize()
		for j := range g.vines[i] {
			y := size.Y
			if j < g.config.ColDepth {
				y = 1 + j
			}
			g.vines[i][j].entity.SetPosition(cx, y)
		}
		for k := range g.pendingExplos {
			if g.pendingExplos[k].X == i {
				g.pendingExplos[k].Y++
			}
		}
		for k := range g.pendingMagics {
			if g.pendingMagics[k].X == i {
				g.pendingMagics[k].Y++
			}
		}
		for k := range g.pendingChains {
			if g.pendingChains[k].X == i {
				g.pendingChains[k].Y++
			}
		}
	}
}

func (g *CrunchGame) assignMultiColors() {
	for bug := range g.multis {
		color := ColorBomb
		switch g.rand.Intn(3) {
		case 0:
			color = ColorBug + 0
		case 1:
			color = ColorBug + 1
		}
		bug.RColor = color
		cell := &termloop.Cell{
			Fg: defaultColorMap.Color(bug.ColorEffective()),
			Ch: bug.Rune,
		}
		bug.entity.SetCell(0, 0, cell)
	}
}

func (g *CrunchGame) randMultiColor() Color {
	switch g.rand.Intn(3) {
	case 0:
		return ColorBug + 0
	case 1:
		return ColorBug + 1
	}
	return ColorBomb
}

// Draw implements termloop.Drawable
func (g *CrunchGame) Draw(screen *termloop.Screen) {
	g.level.Draw(screen)

	now := time.Now()

	twinkle := true

	if g.gameOver() {
		// TODO: do something here
	} else {
		if now.Sub(g.spawnTime) > 4*time.Second {
			g.spawnTime = now
			g.spawnBugs()
		}
	}
	if twinkle && now.Sub(g.multisTime) > 100*time.Millisecond {
		g.multisTime = now
		g.assignMultiColors()
	}

	if g.clearExploded() {
		log.Printf("triggering explosions")
		g.triggerExplosions()
	}
	g.level.Draw(screen)
}

func (g *CrunchGame) grabBug(i int) bool {
	if i >= g.config.NumCol {
		return false
	}
	var j int
	for j = len(g.vines[i]) - 1; j >= 0; j-- {
		bug := g.vines[i][j]
		if bug.Exploded {
			continue
		}
		if bug.Eaten >= 2 {
			return false
		}
		g.player.contains = bug
		break
	}
	if g.player.contains == nil {
		return false
	}
	log.Printf("pos=[%d, %d] grab", i, j)
	copy(g.vines[i][j:], g.vines[i][j+1:])
	g.vines[i] = g.vines[i][:len(g.vines[i])-1]
	g.level.RemoveEntity(g.player.contains.entity)
	return true
}

func (g *CrunchGame) bugEats(i, j int, other *Bug) bool {
	if i >= g.config.NumCol {
		return false
	}
	if j < 0 {
		if len(g.vines[i]) == 0 {
			return false
		}
		j = len(g.vines[i]) - 1
	}
	bottom := g.vines[i][j]

	// Determine if the bottom bug can eat the incoming bug.  Large bugs eat
	// small bugs.  Small bugs eat gnats.  Magic bugs and bomb bugs eat
	// anything.
	eats := false
	switch bottom.Type {
	case BugLarge:
		if other.Type == BugSmall && other.Color == bottom.Color {
			eats = true
		}
	case BugSmall:
		if other.Type == BugGnat {
			eats = true
		}
	case BugMagic, BugBomb:
		eats = true
	}

	if bottom.Eaten >= 2 || !eats {
		return false
	}

	bottom.Eaten += 1 + other.Eaten
	bottom.Rune = g.assignRune(bottom)
	bottom.entity.SetCell(0, 0, &termloop.Cell{
		Fg: defaultColorMap.Color(bottom.ColorEffective()),
		Ch: bottom.Rune,
	})

	if bottom.Eaten >= 2 {
		if bottom.Type == BugBomb {
			g.pendingExplos = append(g.pendingExplos, image.Pt(i, j))
		} else if bottom.Type == BugMagic {
			bottom.EColor = other.Color
			g.pendingMagics = append(g.pendingMagics, image.Pt(i, j))
		} else {
			g.pendingChains = append(g.pendingChains, image.Pt(i, j))
		}
	}

	return true
}

// BUG: Bombs do not trigger chain reactions with other bombs.
func (g *CrunchGame) triggerExplosions() {
	for _, pt := range g.pendingExplos {
		log.Printf("pos=[%d, %d] bomb exploded ", pt.X, pt.Y)
		for i := pt.X - 1; i <= pt.X+1; i++ {
			if i >= 0 && i < len(g.vines) {
				for j := pt.Y - 1; j <= pt.Y+1; j++ {
					if j >= 0 && j < len(g.vines[i]) {
						log.Printf("pos=[%d, %d] exploaded by bomb at pos=[%d, %d]", i, j, pt.X, pt.Y)
						g.vines[i][j].Exploded = true
						g.vines[i][j].entity.SetCell(0, 0, &termloop.Cell{
							Fg: defaultColorMap.Color(ColorExploded),
							Ch: g.vines[i][j].Rune,
						})
					}
				}
			}
		}
	}

	for _, pt := range g.pendingMagics {
		i, j := pt.X, pt.Y
		if g.vines[i][j].Exploded {
			continue
		}
		g.vines[i][j].Exploded = true
		g.vines[i][j].entity.SetCell(0, 0, &termloop.Cell{
			Fg: defaultColorMap.Color(ColorExploded),
			Ch: g.vines[i][j].Rune,
		})
		log.Printf("pos=[%d, %d] magic exploded", i, j)
		mcolor := g.vines[i][j].EColor

		for i := range g.vines {
			for j := range g.vines[i] {
				if g.vines[i][j].Color == mcolor {
					log.Printf("pos=[%d, %d] exploaded by magic at pos=[%d, %d]", i, j, pt.X, pt.Y)
					g.vines[i][j].Exploded = true
					g.vines[i][j].entity.SetCell(0, 0, &termloop.Cell{
						Fg: defaultColorMap.Color(ColorExploded),
						Ch: g.vines[i][j].Rune,
					})
				}
			}
		}
	}

	for _, pt := range g.pendingChains {
		i, j := pt.X, pt.Y
		g.explosionChain(i, j, g.vines[i][j].Color)
	}

	g.pendingExplos = g.pendingExplos[:0]
	g.pendingMagics = g.pendingMagics[:0]
	g.pendingChains = g.pendingChains[:0]
}

func (g *CrunchGame) clearExploded() bool {
	g.triggerExplosions()
	consumed := false
	newvine := make([]*Bug, 0, cap(g.vines[0]))
	for i := range g.vines {
		compacted := false
		gapstart := -1
		for j := range g.vines[i] {
			if g.vines[i][j].Exploded {
				compacted = true
				if gapstart < 0 {
					gapstart = j
				}
				g.level.RemoveEntity(g.vines[i][j].entity)
			} else if gapstart >= 0 {
				if !g.bugEats(i, gapstart, g.vines[i][j]) {
					newvine = append(newvine, g.vines[i][j])
				} else {
					g.level.RemoveEntity(g.vines[i][j].entity)
					consumed = true
				}
				gapstart = -1
			} else {
				newvine = append(newvine, g.vines[i][j])
			}
		}
		if compacted {
			copy(g.vines[i], newvine)
			g.vines[i] = g.vines[i][:len(newvine)]
			cx := g.colX(i)
			for j := range g.vines[i] {
				g.vines[i][j].entity.SetPosition(cx, j+1)
			}
			log.Printf("col=%d compacted remaining=%d", i, len(g.vines[i]))
		}
		newvine = newvine[:0]
	}

	return consumed
}

func (g *CrunchGame) explosionChain(i, j int, c Color) {
	if i < 0 {
		return
	}
	if i >= len(g.vines) {
		return
	}
	if j < 0 {
		return
	}
	if j >= len(g.vines[i]) {
		return
	}
	if g.vines[i][j].Exploded {
		return
	}
	if g.vines[i][j].Color != c {
		return
	}

	log.Printf("pos=[%d, %d] exploaded in chain color=%v", i, j, c)
	g.vines[i][j].Exploded = true
	g.vines[i][j].entity.SetCell(0, 0, &termloop.Cell{
		Fg: defaultColorMap.Color(ColorExploded),
		Ch: g.vines[i][j].Rune,
	})

	if i > 0 {
		g.explosionChain(i-1, j, c)
	}
	if i < len(g.vines)-1 {
		g.explosionChain(i+1, j, c)
	}

	if j > 0 {
		g.explosionChain(i, j-1, c)
	}
	if j < len(g.vines[i])-1 {
		g.explosionChain(i, j+1, c)
	}
}

func (g *CrunchGame) spitBug(i int) bool {
	if i >= g.config.NumCol {
		return false
	}

	spat := g.player.contains
	g.player.contains = nil

	if g.bugEats(i, -1, spat) {
		return true
	}

	if len(g.vines[i]) >= g.config.ColDepth {
		log.Printf("col=%d cannot spit", i)
		g.player.contains = spat
		return false
	}

	g.vines[i] = g.vines[i][:len(g.vines[i])+1]
	g.vines[i][len(g.vines[i])-1] = spat
	log.Printf("pos=[%d, %d] spit", i, len(g.vines[i])-1)
	spat.entity.SetPosition(g.colX(i), len(g.vines[i]))
	g.level.AddEntity(spat.entity)

	return true
}

// Tick implements termloop.Drawable
func (g *CrunchGame) Tick(event termloop.Event) {
	if g.gameOver() {
		log.Printf("GAME OVER")
		return
	}

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
				if g.spitBug(g.playerPos) {
					g.player.entity.SetCell(0, 0, g.player.cell())
				}
			} else {
				if g.grabBug(g.playerPos) {
					g.player.entity.SetCell(0, 0, g.player.cell())
				}
			}
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
		cell.Ch = '@'
	} else {
		cell.Ch = 'O'
	}
	SetCellColor(cell, defaultColorMap, ColorPlayer)
	return cell
}

// Tick implements termloop.Drawable
func (p *Player) Tick(event termloop.Event) {
	p.entity.Tick(event)
}

var defaultColorMap = simpleColorMap{
	ColorNone:     termloop.ColorWhite,
	ColorMulti:    termloop.ColorWhite, // ColorMulti is not used
	ColorBomb:     termloop.ColorRed,
	ColorExploded: termloop.ColorBlack,
	ColorPlayer:   termloop.ColorMagenta,

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
