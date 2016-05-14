package main

import (
	"flag"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/JoelOtter/termloop"
)

// GameVersion is used for bookkeeping pursposes
var GameVersion = "v0.0.1"

func main() {
	showMenu := flag.Bool("m", false, "Montru la menuon antaŭ komencu")
	dataDir := flag.String("d", "tmp", "Dosierujo de ludo datumoj")
	flag.Parse()

	gameDir := GameDir(*dataDir)

	logf, err := os.Create(gameDir.Path("cimoj-log.txt"))
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(logf)

	scorefile, err := NewHighScoreFile(gameDir.Path("cimoj-highscores.json"))
	if err != nil {
		log.Fatal(err)
	}

	alias := "player"
	usr, err := user.Current()
	if err != nil {
		log.Printf("unable to detect username: %v", err)
	} else if usr.Username == "" {
		log.Printf("detected user has no username")
	} else {
		alias = usr.Username
	}

	config := &CrunchConfig{
		Player:           alias,
		Survival:         &simpleSurvivalDifficulty{},
		NumCol:           8,
		ColSpace:         2,
		ColDepth:         7,
		CritterSizeSmall: 1,
		CritterSizeLarge: 1,
	}

	size := config.boardSize()
	log.Printf("size: %v", size)

	game := termloop.NewGame()
	app := NewCrunchApp(game, config, scorefile, *showMenu)
	app.Start()
}

// GameDir is a simple construct for accessing paths under the game's data
// directory.
type GameDir string

// Path returns a path under the game dir.  No attempt is made to validate
// input, so it is imperitive that Path not be used on unvalidated user input
// strings.
func (d GameDir) Path(p string) string {
	return filepath.Join(string(d), p)
}
