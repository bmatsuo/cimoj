// BUG: Spawn times don't update immediately when you level up but after the
// next spawn.  This doesn't really seem right.

package main

import (
	"fmt"
	"image"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/JoelOtter/termloop"
)

// CrunchGame contains a player, critters, a score, and other game state.
type CrunchGame struct {
	config            *CrunchConfig
	tutStep           int
	score             int64
	scoreThreshold    int64
	skillLevel        uint32
	playerPos         int
	player            *Player
	vines             [][]*Bug
	pendingExplos     []image.Point
	pendingChains     []image.Point
	pendingMagics     []image.Point
	rand              Rand
	bugSpawnInit      bool
	bugSpawnInitRem   int
	bugSpawnInitDelay time.Duration
	bugRate           float64
	bugSpawnTime      time.Time
	itemSpawnRate     float64
	itemDespawnRate   float64
	itemSpawnTime     time.Time
	goTime            time.Time
	multis            map[*Bug]struct{}
	multisTime        time.Time
	showingGameOver   bool
	dying             bool
	textScore         *termloop.Text
	textLevel         *termloop.Text
	textHintID        string
	textHint          [4]*termloop.Text
	textGameOver      [2]*termloop.Text
	level             *termloop.BaseLevel
	startTime         time.Time
	endTime           time.Time
	scoreDB           ScoreDB
	scoreWriteStarted bool
	scoreWriteResult  chan error
	finishTime        time.Time
	finishTimeout     time.Time
	finished          bool
}

// NewCrunchGame initializes a new CrunchGame.
func NewCrunchGame(config *CrunchConfig, scores ScoreDB, level *termloop.BaseLevel) *CrunchGame {
	now := time.Now()
	g := &CrunchGame{
		config:        config,
		rand:          defaultRand(),
		multis:        make(map[*Bug]struct{}),
		level:         level,
		bugSpawnTime:  now,
		itemSpawnTime: now,
		scoreDB:       scores,
		startTime:     now,
	}
	g.vines = make([][]*Bug, config.NumCol)
	for i := range g.vines {
		g.vines[i] = make([]*Bug, 0, config.ColDepth+1)
	}

	size := config.boardSize()
	/*
		textWidth := 72 - size - 8
		if textWidth < 0 {
			textWidth = 20
		}
	*/
	textLevel := termloop.NewBaseLevel(termloop.Cell{})
	textLevel.SetOffset(size.X+8, 2)

	const textValuePad = 12

	g.textGameOver[0] = termloop.NewText(3+size.X/2-10, size.Y/2-1, "     La Ludo    ", termloop.ColorMagenta, 0)
	g.textGameOver[1] = termloop.NewText(3+size.X/2-10, size.Y/2+1, "     Finiĝis    ", termloop.ColorMagenta, 0)

	textTitle := termloop.NewText(0, 0, "Cimoj", termloop.ColorGreen, 0)
	textLevel.AddEntity(textTitle)

	textLevelLabel := termloop.NewText(0, 2, "Etaĝo No.:", termloop.ColorGreen, 0)
	textLevel.AddEntity(textLevelLabel)
	g.textLevel = termloop.NewText(textValuePad, 2, "0", termloop.ColorWhite, 0)
	textLevel.AddEntity(g.textLevel)

	textScoreLabel := termloop.NewText(0, 4, "Punktoj:", termloop.ColorGreen, 0)
	textLevel.AddEntity(textScoreLabel)
	g.textScore = termloop.NewText(textValuePad, 4, "0", termloop.ColorWhite, 0)
	textLevel.AddEntity(g.textScore)

	g.initHint(textLevel, 0, 6)
	level.AddEntity(textLevel)

	// Set the initial hint for the game.  The basic controls hint helps new
	// players and only shows during the beginning of the game.
	g.setHint("controls")

	g.playerPos = config.NumCol
	g.player = &Player{
		entity: termloop.NewEntity(g.colX(g.playerPos), g.config.boardSize().Y, 1, 1),
		level:  level,
	}
	g.player.entity.SetCell(0, 0, g.player.cell())
	g.level.AddEntity(g.player.entity)

	g.updateDifficulty()
	g.calcBugSpawnTime()
	g.calcItemSpawnTime()

	return g
}

func (g *CrunchGame) initHint(level termloop.Level, x, y int) {
	for i := range g.textHint {
		g.textHint[i] = termloop.NewText(x, y+i, "", termloop.ColorCyan, 0)
	}
	for i := range g.textHint {
		level.AddEntity(g.textHint[i])
	}
}

func (g *CrunchGame) clearHint(id string) {
	if g.textHintID != id {
		return
	}
	for i := range g.textHint {
		g.textHint[i].SetText("")
	}
}

func (g *CrunchGame) setHint(id string) {
	g.textHintID = id

	hint, ok := hints[id]
	if !ok {
		hint[0] = "Unknown hint id:"
		hint[1] = "    " + id
	}
	for i := range g.textHint {
		g.textHint[i].SetText(hint[i])
	}
}

func (g *CrunchGame) calcHighScore() *HighScore {
	return &HighScore{
		GameType: "survival",
		Player:   g.config.Player,
		Score:    g.score,
		Level:    int(g.skillLevel),
		Start:    g.startTime,
		End:      g.endTime,
		Qual: map[string]string{
			"GameVersion": GameVersion,
		},
	}
}

func (g *CrunchGame) calcItemSpawnTime() {
	g.itemSpawnTime = g.itemSpawnTime.Add(time.Duration(float64(time.Second) * g.rand.ExpFloat64() * g.itemSpawnRate))
}

func (g *CrunchGame) calcBugSpawnTime() {
	// board initialization has completed -- enter the normal code path.
	if g.bugSpawnInitRem == 0 {
		g.bugSpawnTime = g.bugSpawnTime.Add(time.Duration(float64(time.Second) * g.rand.ExpFloat64() * g.bugRate))
		return
	}

	// unknown number of bugs to spawn initially
	if g.bugSpawnInitDelay == 0 {
		g.bugSpawnTime = g.bugSpawnTime.Add(time.Second)
		return
	}

	g.bugSpawnTime = g.bugSpawnTime.Add(g.bugSpawnInitDelay)
}

func (g *CrunchGame) updateDifficulty() bool {
	levelup := false
	for g.scoreThreshold >= 0 && g.score >= g.scoreThreshold {
		levelup = true
		g.skillLevel++
		g.scoreThreshold = g.config.Difficulty.NextLevel(int(g.skillLevel))
	}
	if levelup {
		log.Printf("level=%d", g.skillLevel)
		diff := g.config.Difficulty
		g.textLevel.SetText(fmt.Sprint(g.skillLevel))
		g.bugRate = diff.BugRate(int(g.skillLevel))
		spawn, despawn := diff.ItemRate(int(g.skillLevel))
		g.itemSpawnRate = spawn
		g.itemDespawnRate = despawn
		if !g.bugSpawnInit {
			g.bugSpawnInit = true
			g.bugSpawnInitRem = diff.NumBugInit()
			if g.bugSpawnInitRem == 0 {
				g.bugSpawnInitRem = 3
			}
			g.bugSpawnInitDelay = time.Duration(float64(time.Second) * diff.BugRateInit())
		}
	}
	return levelup
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

// Finished will return true when the game screen can be cleared and a new game
// can start.
func (g *CrunchGame) Finished() bool {
	return g.finished
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

	roll -= 31
	if roll <= 0 {
		return g.createBug(BugSmall, g.randomColorBug(2))
	}

	roll -= 31
	if roll <= 0 {
		return g.createBug(BugLarge, 2+g.randomColorBug(2))
	}

	roll -= 31
	if roll <= 0 {
		return g.createBug(BugGnat, ColorNone)
	}

	roll -= 2
	if roll <= 0 {
		return g.createBug(BugBomb, ColorBomb)
	}

	roll -= 2
	if roll <= 0 {
		return g.createBug(BugLightning, ColorBomb)
	}

	roll -= 1
	if roll <= 0 {
		return g.createBug(BugMultiChain, ColorMulti)
	}

	roll -= 1
	if roll <= 0 {
		return g.createBug(BugRock, ColorNone)
	}

	roll -= 1
	if roll <= 0 {
		return g.createBug(BugMagic, ColorMulti)
	}

	panic("bug distribution is bad")
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
	case BugLightning:
		if bug.Eaten > 0 {
			return 'X'
		}
		return 'x'
	case BugRock:
		return '▀'
	case BugMultiChain:
		return '*'
	case BugMagic:
		return '%'
	}
	return '?'
}

func (g *CrunchGame) spawnBugs() {
	// the board state is initialized by rapidly spawning single bugs before
	// bugs start coming in more predictable waves.
	if g.bugSpawnInitRem > 0 {
		log.Printf("INIT SPAWN")
		g.bugSpawnInitRem--
		g.spawnBugOnVine(g.rand.Intn(len(g.vines)))
		return
	}

	log.Printf("ROW SPAWN")
	// for now we do something simple and spawn bugs in all rows simultaneously
	for i := range g.vines {
		g.spawnBugOnVine(i)
	}
}

func (g *CrunchGame) spawnBugOnVine(i int) {
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
	now := time.Now()

	twinkle := true

	if g.tutStep == 0 && g.score > 5 {
		g.tutStep++
		g.setHint("scoring")
	}
	if g.gameOver() {
		if g.endTime.IsZero() {
			g.setHint("continuing")
			g.endTime = now
		}
		if !g.scoreWriteStarted {
			g.finishTime = now.Add(500 * time.Millisecond)
			g.finishTimeout = now.Add(20 * time.Second)
			record := g.calcHighScore()

			g.scoreWriteStarted = true
			g.scoreWriteResult = make(chan error, 1)
			go func() {
				if g.scoreDB != nil {
					g.scoreWriteResult <- g.scoreDB.WriteHighScore(record)
				} else {
					g.scoreWriteResult <- nil
				}
			}()
		} else if now.After(g.finishTime) {
			select {
			case err := <-g.scoreWriteResult:
				g.scoreWriteResult = nil
				g.finishTimeout = time.Time{}

				if err != nil {
					log.Printf("unable to write high score: %v", err)
				}

				// It is OK to exit the game.
				g.finished = true
			default:
				if !g.finishTimeout.IsZero() && now.After(g.finishTimeout) {
					panic("hanging while writing the high score record")
				}
			}
		}

		if now.Sub(g.goTime) > time.Second {
			g.goTime = now
			if g.showingGameOver {
				g.level.RemoveEntity(g.textGameOver[0])
				g.level.RemoveEntity(g.textGameOver[1])
			} else {
				g.level.AddEntity(g.textGameOver[0])
				g.level.AddEntity(g.textGameOver[1])
			}
			g.showingGameOver = !g.showingGameOver
		}
	} else {
		if now.Sub(g.bugSpawnTime) >= 0 {
			g.bugSpawnTime = now
			g.spawnBugs()
			g.calcBugSpawnTime()
			for i := range g.vines {
				if len(g.vines[i]) == g.config.ColDepth {
					g.dying = true
					g.setHint("dying")
				}
			}
		}
		g.textLevel.SetText(fmt.Sprint(g.skillLevel))
		g.textScore.SetText(fmt.Sprint(g.score))
	}
	if twinkle && now.Sub(g.multisTime) > 100*time.Millisecond {
		g.multisTime = now
		g.assignMultiColors()
	}

	if g.clearExploded() {
		log.Printf("triggering explosions")
		g.triggerExplosions()
	}

	if g.dying {
		remedied := true
		for i := range g.vines {
			if len(g.vines[i]) == g.config.ColDepth {
				remedied = false
				break
			}
		}
		if remedied {
			g.dying = false
			g.clearHint("dying")
		}
	}

	g.updateDifficulty()

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
	// small bugs.  Small bugs eat gnats.  Lightning bugs and bomb bugs eat
	// anything.
	eats := false
	switch bottom.Type {
	case BugLarge:
		if other.Type == BugSmall {
			eats = true
		}
	case BugSmall:
		if other.Type == BugGnat {
			eats = true
		}
	case BugLightning, BugBomb:
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
		if bottom.Type == BugBomb || bottom.Type == BugLightning {
			g.pendingExplos = append(g.pendingExplos, image.Pt(i, j))
		} else {
			g.pendingChains = append(g.pendingChains, image.Pt(i, j))
		}
	}

	return true
}

// BUG: Bombs do not trigger chain reactions with other bombs.
func (g *CrunchGame) triggerExplosions() {
	for _, pt := range g.pendingExplos {
		g.bombChain(pt.X, pt.Y)
	}
	g.pendingExplos = g.pendingExplos[:0]

domagics:
	for _, pt := range g.pendingMagics {
		i, j := pt.X, pt.Y
		if g.vines[i][j].Exploded {
			continue
		}
		g.vines[i][j].Exploded = true
		log.Printf("pos=[%d, %d] magic exploded color=%v", i, j, g.vines[i][j].EColor)
		mcolor := g.vines[i][j].EColor
		g.score++

		for i := range g.vines {
			for j := range g.vines[i] {
				if g.vines[i][j].Color == mcolor {
					log.Printf("pos=[%d, %d] exploaded by magic at pos=[%d, %d]", i, j, pt.X, pt.Y)
					g.vines[i][j].Exploded = true
					g.vines[i][j].entity.SetCell(0, 0, &termloop.Cell{
						Fg: defaultColorMap.Color(ColorExploded),
						Ch: g.vines[i][j].Rune,
					})
					g.score++
				}
			}
		}
	}
	g.pendingMagics = g.pendingMagics[:0]

	for _, pt := range g.pendingChains {
		i, j := pt.X, pt.Y
		g.explosionChain(i, j, g.vines[i][j].Color)
	}
	g.pendingChains = g.pendingChains[:0]

	if len(g.pendingMagics) > 0 {
		goto domagics
	}
}

func decreasePtY(pts *[]image.Point, min int) {
	next := 0
	for k := range *pts {
		if (*pts)[k].Y < min {
			(*pts)[next] = (*pts)[k]
			next++
			continue
		}
		if (*pts)[k].Y == min {
			continue
		}
		(*pts)[next] = (*pts)[k]
		(*pts)[next].Y--
		next++
	}
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
				decreasePtY(&g.pendingExplos, j)
				decreasePtY(&g.pendingMagics, j)
				decreasePtY(&g.pendingChains, j)
				g.level.RemoveEntity(g.vines[i][j].entity)
			} else if gapstart >= 0 {
				if j == len(g.vines[i])-1 && !bugClimbs(g.vines[i][j].Type) {
					log.Printf("pos=[%d, %d] dropped from the vines", i, j)
					// BUG: Bombs should explode on the ground and kill the
					// player when they drop in this way.
					g.level.RemoveEntity(g.vines[i][j].entity)
					consumed = true
				} else if gapstart >= 0 {
					if g.bugEats(i, gapstart-1, g.vines[i][j]) {
						g.level.RemoveEntity(g.vines[i][j].entity)
						consumed = true
					} else {
						newvine = append(newvine, g.vines[i][j])
					}
				} else {
					newvine = append(newvine, g.vines[i][j])
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

func (g *CrunchGame) bombChain(i, j int) {
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

	g.vines[i][j].Exploded = true
	g.vines[i][j].entity.SetCell(0, 0, &termloop.Cell{
		Fg: defaultColorMap.Color(ColorExploded),
		Ch: g.vines[i][j].Rune,
	})
	g.score++

	log.Printf("pos=[%d, %d] exploded by bomb", i, j)
	if g.vines[i][j].Type == BugBomb {
		log.Printf("pos=[%d, %d] bomb exploded", i, j)
		// Explode nearby bugs; out of bounds accesses are handled in the call.
		// The following nested loop will call g.explosionChain(i, j) again but
		// we should have already exploded index (i,j) and no infinite
		// recursion will occur.
		for ik := i - 1; ik <= i+1; ik++ {
			for jk := j - 1; jk <= j+1; jk++ {
				g.bombChain(ik, jk)
				g.bombChain(ik, jk)
			}
		}
	} else if g.vines[i][j].Type == BugLightning {
		g.bombChain(i+1, j+1)
		g.bombChain(i+1, j-1)
		g.bombChain(i-1, j+1)
		g.bombChain(i-1, j-1)
		g.bombChain(i+2, j+2)
		g.bombChain(i+2, j-2)
		g.bombChain(i-2, j+2)
		g.bombChain(i-2, j-2)
	}
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
	if g.vines[i][j].Type != BugSmall && g.vines[i][j].Type != BugLarge && g.vines[i][j].Type != BugMultiChain {
		if g.vines[i][j].Type == BugMagic && g.vines[i][j].EColor == ColorNone && c != ColorMulti {
			log.Printf("pos=[%d, %d] magic triggered", i, j)
			g.vines[i][j].EColor = c
			g.pendingMagics = append(g.pendingMagics, image.Pt(i, j))
		}
		return
	}

	// Check the input color and adjust the color for recursive calls if
	// necessary.
	if g.vines[i][j].Color == ColorMulti {
		c = ColorMulti
	} else if c == ColorMulti {
		c = g.vines[i][j].Color
	} else if g.vines[i][j].Color != c {
		return
	}

	log.Printf("pos=[%d, %d] exploaded in chain color=%v", i, j, c)
	g.vines[i][j].Exploded = true
	g.vines[i][j].entity.SetCell(0, 0, &termloop.Cell{
		Fg: defaultColorMap.Color(ColorExploded),
		Ch: g.vines[i][j].Rune,
	})
	g.score++

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
					g.score++
					g.player.entity.SetCell(0, 0, g.player.cell())
				}
			} else {
				if g.grabBug(g.playerPos) {
					g.score++
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
	ColorPlayer:   termloop.ColorWhite,

	ColorBug + 0: termloop.ColorYellow,
	ColorBug + 1: termloop.ColorBlue,
	ColorBug + 2: termloop.ColorMagenta,
	ColorBug + 3: termloop.ColorCyan,
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
	ExpFloat64() float64
}

type simpleDifficulty struct{}

func (s *simpleDifficulty) NextLevel(lvl int) int64 {
	if lvl <= 0 {
		return 0
	}
	if lvl < 63 {
		return 1 << uint(lvl)
	}
	return -1
}

func (s *simpleDifficulty) NumBugInit() int {
	return 12
}

func (s *simpleDifficulty) BugRateInit() float64 {
	return 0.3
}

func (s *simpleDifficulty) BugRate(lvl int) float64 {
	const initialRate = 5 // about every 5 seconds
	const baseReduction = 0.95
	return initialRate * math.Pow(baseReduction, float64(lvl))
}

func (s *simpleDifficulty) ItemRate(lvl int) (spawn, despawn float64) {
	const initialSpawnRate = 10  // about every 10 seconds
	const initialDespawnRate = 5 // about 5 seconds
	const baseSpawnReduction = 0.90
	const baseDespawnReduction = 0.96
	spawn = initialSpawnRate * math.Pow(baseSpawnReduction, float64(lvl))
	despawn = initialDespawnRate * math.Pow(baseDespawnReduction, float64(lvl))
	return spawn, despawn
}

// Difficulty controls how the game difficulty scales with the player's score.
type Difficulty interface {
	// NumBugInit returns the number of bugs to initializer the game with.
	NumBugInit() int

	// BugRateInit returns a constant amount of time between bug spawns during
	// initialization.
	BugRateInit() float64

	// NextLevel returns the total score required to achive the next level.
	NextLevel(lvl int) int64

	// BugRate returns the expected number of seconds between individual bug
	// spawns for the current level.  An exponential distribution will
	// determine the actual duration between each spawn.
	BugRate(lvl int) float64

	// ItemRate returns the expected number of seconds between individual item
	// spawns for the current level and the expected number of seconds for a
	// spawned item to despawn.  An exponential distribution will determine the
	// actual duration between each spawn.  Another exponential distribution
	// determins the duration each spawn exists.  Multiple items may exist at
	// the same time.
	ItemRate(lvl int) (spawn, despawn float64)
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
	BugLightning
	BugRock
	BugMultiChain
	bugMax = iota - 1
)

var _bugFalls = []bool{
	BugBomb:      true,
	BugRock:      true,
	BugLightning: true,
	bugMax:       false,
}

func bugClimbs(b BugType) bool {
	return !_bugFalls[b]
}

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
