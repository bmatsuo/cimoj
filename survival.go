package main

import "math"

const defaultSurvivalLevelOne = 30
const defaultSurvivalLevelBase = 1.5

// SurvivalDifficulty controls how the game difficulty scales with the player's score.
type SurvivalDifficulty interface {
	// NumBugInit returns the number of bugs to initializer the game with.
	NumBugInit() int

	// BugRateInit returns a constant amount of time between bug spawns during
	// initialization.
	BugRateInit() float64

	// BugDistribution returns the distribution of bugs and colors
	BugDistribution(lvl int) BugDistribution

	// NextLevel returns the total score required to achive the next level.
	NextLevel(lvl int) int64

	// BugRate returns the expected number of seconds between individual bug
	// spawns for the current level.  An normal distribution will determine the
	// actual duration between each spawn.
	BugRate(lvl int) float64

	// ItemRate returns the expected number of seconds between individual item
	// spawns for the current level and the expected number of seconds for a
	// spawned item to despawn.  A normal distribution will determine the
	// actual duration between each spawn.  Another normal distribution
	// determins the duration each spawn exists.  Multiple items may exist at
	// the same time.
	ItemRate(lvl int) (spawn, despawn float64)

	// ItemDistribution returns the distribution of item types seen on the
	// given level.
	ItemDistribution(lvl int) ItemDistribution
}

type simpleSurvivalDifficulty struct{}

func (s *simpleSurvivalDifficulty) NextLevel(lvl int) int64 {
	return int64(float64(defaultSurvivalLevelOne) * math.Pow(defaultSurvivalLevelBase, float64(lvl)))
}

func (s *simpleSurvivalDifficulty) NumBugInit() int {
	return 12
}

func (s *simpleSurvivalDifficulty) BugRateInit() float64 {
	return 0.3
}

func (s *simpleSurvivalDifficulty) BugRate(lvl int) float64 {
	const initialRate = 7 // about every 5 seconds
	const baseReduction = 0.99
	return initialRate * math.Pow(baseReduction, float64(lvl))
}

func (s *simpleSurvivalDifficulty) ItemRate(lvl int) (spawn, despawn float64) {
	const initialSpawnRate = 15   // about every 10 seconds
	const initialDespawnRate = 10 // about 5 seconds
	const baseSpawnReduction = 0.90
	const baseDespawnReduction = 0.96
	spawn = initialSpawnRate * math.Pow(baseSpawnReduction, float64(lvl))
	despawn = initialDespawnRate * math.Pow(baseDespawnReduction, float64(lvl))
	return spawn, despawn
}

func (s *simpleSurvivalDifficulty) ItemDistribution(lvl int) ItemDistribution {
	return itemTypeDistn{
		ItemRowClear: 10,
		ItemPushUp:   10,
		ItemBullet:   10,
		ItemScramble: 10,
		ItemRecolor:  10,
	}
}

func (s *simpleSurvivalDifficulty) BugDistribution(lvl int) BugDistribution {
	if lvl < 3 {
		return &simpleDistribution{
			&bugTypeDistn{
				BugSmall:      500,
				BugLarge:      400,
				BugGnat:       200,
				BugMagic:      0,
				BugBomb:       0,
				BugLightning:  0,
				BugRock:       0,
				BugMultiChain: 0,
			},
			&bugColorCondDistn{
				BugSmall: {ColorBug + 0: 1},
				BugLarge: {ColorBug + 2: 1},
			},
		}
	}
	if lvl < 5 {
		return &simpleDistribution{
			&bugTypeDistn{
				BugSmall:      390,
				BugLarge:      385,
				BugGnat:       195,
				BugMagic:      0,
				BugBomb:       30,
				BugLightning:  0,
				BugRock:       0,
				BugMultiChain: 0,
			},
			&bugColorCondDistn{
				BugSmall: {ColorBug + 0: 1, ColorBug + 1: 1},
				BugLarge: {ColorBug + 2: 1, ColorBug + 3: 0},
			},
		}
	}
	if lvl == 6 {
		return &simpleDistribution{
			&bugTypeDistn{
				BugSmall:      380,
				BugLarge:      375,
				BugGnat:       192,
				BugMagic:      0,
				BugBomb:       15,
				BugLightning:  15,
				BugRock:       10,
				BugMultiChain: 10,
			},
			&bugColorCondDistn{
				BugSmall: {ColorBug + 0: 1, ColorBug + 1: 1},
				BugLarge: {ColorBug + 2: 1, ColorBug + 3: 1},
			},
		}
	}
	if lvl == 7 {
		return &simpleDistribution{
			&bugTypeDistn{
				BugSmall:      373,
				BugLarge:      363,
				BugGnat:       190,
				BugMagic:      10,
				BugBomb:       15,
				BugLightning:  15,
				BugRock:       15,
				BugMultiChain: 10,
			},
			&bugColorCondDistn{
				BugSmall: {ColorBug + 0: 1, ColorBug + 1: 1},
				BugLarge: {ColorBug + 2: 1, ColorBug + 3: 1},
			},
		}
	}
	if lvl == 8 {
		return &simpleDistribution{
			&bugTypeDistn{
				BugSmall:      378,
				BugLarge:      358,
				BugGnat:       180,
				BugMagic:      15,
				BugBomb:       15,
				BugLightning:  15,
				BugRock:       15,
				BugMultiChain: 15,
			},
			&bugColorCondDistn{
				BugSmall: {ColorBug + 0: 1, ColorBug + 1: 1},
				BugLarge: {ColorBug + 2: 1, ColorBug + 3: 1},
			},
		}
	}
	if lvl == 9 {
		return &simpleDistribution{
			&bugTypeDistn{
				BugSmall:      380,
				BugLarge:      350,
				BugGnat:       170,
				BugMagic:      20,
				BugBomb:       20,
				BugLightning:  20,
				BugRock:       20,
				BugMultiChain: 20,
			},
			&bugColorCondDistn{
				BugSmall: {ColorBug + 0: 1, ColorBug + 1: 1},
				BugLarge: {ColorBug + 2: 1, ColorBug + 3: 1},
			},
		}
	}
	if lvl == 10 {
		return &simpleDistribution{
			&bugTypeDistn{
				BugSmall:      383,
				BugLarge:      343,
				BugGnat:       160,
				BugMagic:      25,
				BugBomb:       25,
				BugLightning:  25,
				BugRock:       25,
				BugMultiChain: 25,
			},
			&bugColorCondDistn{
				BugSmall: {ColorBug + 0: 1, ColorBug + 1: 1},
				BugLarge: {ColorBug + 2: 1, ColorBug + 3: 1},
			},
		}
	}
	return &simpleDistribution{
		&bugTypeDistn{
			BugSmall:      390,
			BugLarge:      340,
			BugGnat:       150,
			BugMagic:      30,
			BugBomb:       30,
			BugLightning:  30,
			BugRock:       30,
			BugMultiChain: 30,
		},
		&bugColorCondDistn{
			BugSmall: {ColorBug + 0: 1, ColorBug + 1: 1},
			BugLarge: {ColorBug + 2: 1, ColorBug + 3: 1},
		},
	}
}
