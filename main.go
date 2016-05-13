package main

import (
	"log"
	"os"

	"github.com/JoelOtter/termloop"
)

// GameVersion is used for bookkeeping pursposes
var GameVersion = "v0.0.1"

func main() {
	scorefile, err := NewHighScoreFile("tmp/highscores.json")
	if err != nil {
		log.Fatal(err)
	}

	logf, err := os.Create("tmp/cimoj-log.txt")
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(logf)

	config := &CrunchConfig{
		Difficulty:       &simpleDifficulty{},
		NumCol:           6,
		ColSpace:         2,
		ColDepth:         10,
		CritterSizeSmall: 1,
		CritterSizeLarge: 1,
	}

	size := config.boardSize()
	log.Printf("size: %v", size)

	game := termloop.NewGame()
	app := NewCrunchApp(game, config, scorefile)
	app.Start()
}
