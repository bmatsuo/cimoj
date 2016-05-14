package main

import (
	"image"
	"log"

	"github.com/JoelOtter/termloop"
)

// CrunchConfig defines the crunching board.
type CrunchConfig struct {
	Player           string
	Difficulty       Difficulty
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

// CrunchApp represents the top-level application, a session which may involve
// multiple games.
type CrunchApp struct {
	game    *termloop.Game
	screen  *termloop.Screen
	config  *CrunchConfig
	menu    *CrunchMenu
	current *CrunchGame
	scoreDB ScoreDB
}

// NewCrunchApp creates a new CrunchApp using a static config that can be
// repeatedly played.
func NewCrunchApp(game *termloop.Game, config *CrunchConfig, scores ScoreDB) *CrunchApp {
	app := &CrunchApp{
		game:    game,
		config:  config,
		scoreDB: scores,
	}
	app.menu = NewCrunchMenu(config)

	game.Screen().AddEntity(app)

	return app
}

// Start starts the application/game.
func (app *CrunchApp) Start() {
	app.game.Start()
}

// Draw implements termloop.Drawable
func (app *CrunchApp) Draw(screen *termloop.Screen) {
	if app.current != nil {
		app.current.Draw(screen)
		return
	}
	app.menu.Draw(screen)
}

// Tick implements termloop.Drawable
func (app *CrunchApp) Tick(event termloop.Event) {
	if app.current != nil && !app.current.Finished() {
		app.current.Tick(event)
		return
	}

	if event.Type == termloop.EventKey { // Is it a keyboard event?
		switch event.Key {
		case termloop.KeyEnter:
			if app.current != nil {
				// Just let the old game get garbage collected, it will stop
				// recieved events and draw calls, so the only real worry is lag in
				// the subsequent game.
				app.current = app.createNewGame()
				return
			}
			menuItem, _ := app.menu.GetSelection()
			if menuItem == 0 {
				app.current = app.createNewGame()
			}
			return
		}
	}

	// Pass the keypress onto the menu when the app has not intercepted it by
	// this point.
	app.menu.Tick(event)
}

func (app *CrunchApp) createNewGame() *CrunchGame {
	size := app.config.boardSize()
	log.Printf("size=[%d, %d] new game", size.X, size.Y)

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

	for i := 0; i < app.config.NumCol; i++ {
		posX := 1 + app.config.ColSpace + app.config.CritterSizeLarge/2 + i*(app.config.ColSpace+app.config.CritterSizeLarge)
		column := termloop.NewEntity(posX, 1, 1, app.config.colLength())
		column.Fill(&termloop.Cell{Fg: termloop.ColorGreen, Ch: '|'})
		board.AddEntity(column)
	}

	crunch := NewCrunchGame(app.config, app.scoreDB, board)
	level.AddEntity(crunch)

	return crunch
}
