// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"math"

	"github.com/holiman/uint256"
)

const (
	stateLower = 0
	stateTree  = -1
	stateUpper = 1

	directionDown = -1
	directionUp   = 1
)

type solver struct {
	n      int
	source int
	sink   int
	demand *uint256.Int

	arcs    []Arc
	artArcs []Arc

	pi        []int64
	parent    []int
	predArc   []int
	thread    []int
	revThread []int
	succNum   []int
	lastSucc  []int
	direction []int
	state     []int

	M int64

	bottleneck *uint256.Int
	tmpFlow    *uint256.Int
}

func newSolver(arcs []Arc, n, source, sink int, demand *uint256.Int) *solver {
	total := n + 1
	s := &solver{
		n:          n,
		source:     source,
		sink:       sink,
		demand:     demand,
		arcs:       arcs,
		artArcs:    make([]Arc, 0, n),
		pi:         make([]int64, total),
		parent:     make([]int, total),
		predArc:    make([]int, total),
		thread:     make([]int, total),
		revThread:  make([]int, total),
		succNum:    make([]int, total),
		lastSucc:   make([]int, total),
		direction:  make([]int, total),
		state:      make([]int, len(arcs)+n),
		M:          math.MaxInt64 / (8 * int64(n+1)),
		bottleneck: new(uint256.Int),
		tmpFlow:    new(uint256.Int),
	}

	for i := range s.arcs {
		if s.arcs[i].Flow == nil {
			s.arcs[i].Flow = new(uint256.Int)
		}
		s.arcs[i].Flow.Clear()
	}

	return s
}

func (s *solver) arc(idx int) *Arc {
	if idx < len(s.arcs) {
		return &s.arcs[idx]
	}
	return &s.artArcs[idx-len(s.arcs)]
}

func (s *solver) initializeTree() {
	root := s.n
	artBase := len(s.arcs)

	for i := 0; i < s.n; i++ {
		s.parent[i] = root
		s.succNum[i] = 1
		s.lastSucc[i] = i

		var a Arc
		a.Cost = s.M

		if i == s.source {
			a.From = s.source
			a.To = root
			a.Capacity = new(uint256.Int).Set(s.demand)
			a.Flow = new(uint256.Int).Set(s.demand)
			s.direction[i] = directionUp
			s.pi[i] = s.M
		} else if i == s.sink {
			a.From = root
			a.To = s.sink
			a.Capacity = new(uint256.Int).Set(s.demand)
			a.Flow = new(uint256.Int).Set(s.demand)
			s.direction[i] = directionDown
			s.pi[i] = -s.M
		} else {
			a.From = i
			a.To = root
			a.Capacity = new(uint256.Int)
			a.Flow = new(uint256.Int)
			s.direction[i] = directionUp
			s.pi[i] = s.M
		}

		s.artArcs = append(s.artArcs, a)
		s.predArc[i] = artBase + i
		s.state[artBase+i] = stateTree
	}

	s.parent[root] = -1
	s.predArc[root] = -1
	s.pi[root] = 0
	s.succNum[root] = s.n + 1
	s.lastSucc[root] = s.n - 1

	s.thread[root] = 0
	for i := 0; i < s.n-1; i++ {
		s.thread[i] = i + 1
	}
	s.thread[s.n-1] = root

	s.revThread[0] = root
	for i := 1; i < s.n; i++ {
		s.revThread[i] = i - 1
	}
	s.revThread[root] = s.n - 1
}

type solverSnapshot struct {
	N         int
	Source    int
	Sink      int
	M         int64
	Pi        []int64
	Parent    []int
	PredArc   []int
	Thread    []int
	RevThread []int
	SuccNum   []int
	LastSucc  []int
	Direction []int
	State     []int
	Arcs      []Arc
	ArtArcs   []Arc
}

func (s *solver) snapshot() solverSnapshot {
	return solverSnapshot{
		N:         s.n,
		Source:    s.source,
		Sink:      s.sink,
		M:         s.M,
		Pi:        s.pi,
		Parent:    s.parent,
		PredArc:   s.predArc,
		Thread:    s.thread,
		RevThread: s.revThread,
		SuccNum:   s.succNum,
		LastSucc:  s.lastSucc,
		Direction: s.direction,
		State:     s.state,
		Arcs:      s.arcs,
		ArtArcs:   s.artArcs,
	}
}
