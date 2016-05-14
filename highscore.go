package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// HighScore is a play record for personal records.  The record contains
// key-value Qual that can contain any qualifying data which can be filtered on
// later.
type HighScore struct {
	GameType string
	Player   string
	Score    int64
	Level    int
	Start    time.Time
	End      time.Time
	Qual     map[string]string
}

// ScoreDB stores high scores, possibly for several different players and
// allows the top-N to be retrieved using the qualifying index.
type ScoreDB interface {
	// WriteHighScore persists a high-score record and returns an error if the
	// record could not be persisted.
	WriteHighScore(*HighScore) error

	// TopHighScores returns the n highest scores that have been persisted.  If
	// player is an empty string then scores for all players are returned.  If
	// an even number of qualpairs are given then the returned scores should
	// all contain the specified Qual data.  Implementations may panic if given
	// an odd number of qual pairs.
	//
	// If an implemention cannot find n HighScore records then the located
	// records are returned along with any error that prevented more from being
	// located.
	TopHighScores(n int, gametype, player string, qualpairs ...string) ([]*HighScore, error)
}

// HighScoreFile is a ScoreDB that maintains a high-score database in a flat
// file of line-delimited json.
type HighScoreFile struct {
	path string
}

// NewHighScoreFile returns a new HighScoreFile that stores HighScore records
// in path.
func NewHighScoreFile(path string) (*HighScoreFile, error) {
	_, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("the directory containing the high score file does not exist: %v", err)
	}
	db := &HighScoreFile{
		path: path,
	}
	return db, nil
}

// WriteHighScore implements HighScoreDB
func (db *HighScoreFile) WriteHighScore(score *HighScore) error {
	// Give the file group write permissions so that it can work executed
	// merely under the 'games' group on linux.
	f, err := os.OpenFile(db.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0664)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(score)
}

// TopHighScores implements HighScoreDB
func (db *HighScoreFile) TopHighScores(n int, gametype, player string, qualpairs ...string) ([]*HighScore, error) {
	if len(qualpairs)%2 != 0 {
		panic("odd length qualifier pairs list")
	}

	f, err := os.Open(db.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var scores []*HighScore

	dec := json.NewDecoder(f)
decloop:
	for {
		var score *HighScore
		err := dec.Decode(&score)
		if err == io.EOF {
			break
		}
		if err != nil {
			return scores, err
		}
		if player != "" && score.Player != player {
			continue
		}
		for i := 0; i < len(qualpairs); i += 2 {
			if score.Qual[qualpairs[i]] != qualpairs[i+1] {
				continue decloop
			}
		}
		scores = append(scores, score)
	}

	return scores, nil
}
