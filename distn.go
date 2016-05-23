package main

import (
	"log"
	"math/rand"
)

// Rand wraps PRNG implementations so that behavior of randomized things can be
// tested more easily.
type Rand interface {
	Intn(n int) int
	Float64() float64
	NormFloat64() float64
}

// BugDistribution destribes how bugs spawn on a level.
type BugDistribution interface {
	// BugTypeProb returns the probability of spawning a t type bug.
	RandBugType(r Rand) BugType

	// RandColor returns the a color c.  This can be thought of as the
	// probability of (T, C) conditioned on T=t.  RandColor will only be called
	// with types BugLarge and BugSmall.
	RandColor(r Rand, t BugType) Color
}

// ItemDistribution returns the relative rates at which different types of
// items are spawned onto bugs.
type ItemDistribution interface {
	RandItemType(r Rand) ItemType
}

type simpleDistribution struct {
	*bugTypeDistn
	*bugColorCondDistn
}

var _ *simpleDistribution = &simpleDistribution{}

func (d *simpleDistribution) RandBugType(r Rand) BugType {
	return d.bugTypeDistn.Rand(r)
}

func (d *simpleDistribution) RandColor(r Rand, t BugType) Color {
	return d.bugColorCondDistn.Rand(r, t)
}

type bugColorCondDistn []bugColorDistn

func (d bugColorCondDistn) Rand(r Rand, t BugType) Color {
	if t < 0 || int(t) >= len(d) {
		log.Panicf("no distribution for bug type: %d", t)
	}
	return d[t].Rand(r)
}

type bugColorDistn intDistn

func (d bugColorDistn) Rand(r Rand) Color {
	return Color((intDistn)(d).Rand(r))
}

type bugTypeDistn intDistn

func (d bugTypeDistn) Rand(r Rand) BugType {
	return BugType((intDistn)(d).Rand(r))
}

type itemTypeDistn intDistn

func (d itemTypeDistn) RandItemType(r Rand) ItemType {
	return ItemType((intDistn)(d).Rand(r))
}

type intDistn []int

func (d intDistn) Rand(r Rand) int {
	var sum int
	for i := range d {
		sum += d[i]
	}
	roll := rand.Intn(sum)
	for i := range d {
		roll -= d[i]
		if roll < 0 {
			return i
		}
	}
	panic("bad computation of distribution")
}
