package main

import (
	"log"
	"os"
	"os/user"

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

	alias := "player"
	usr, err := user.Current()
	if err != nil {
		log.Printf("unable to detect username: %v", err)
	} else if usr.Username != "" {
		log.Printf("detected user has no username")
	} else {
		alias = usr.Username
	}

	config := &CrunchConfig{
		Player:           alias,
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
