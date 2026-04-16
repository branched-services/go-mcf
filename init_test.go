// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"math"
	"testing"

	"github.com/holiman/uint256"
)

func TestBuildInitialStateSmallGraph(t *testing.T) {
	// 3-node graph: 0 -> 1 -> 2, source=0, sink=2, demand=5.
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: uint256.NewInt(10)},
		{From: 1, To: 2, Cost: 1, Capacity: uint256.NewInt(10)},
	}
	demand := uint256.NewInt(5)
	s := buildInitialState(arcs, 3, 0, 2, demand)

	// Dimensions.
	if s.mReal != 2 {
		t.Fatalf("mReal = %d, want 2", s.mReal)
	}
	if s.m != 5 {
		t.Fatalf("m = %d, want 5", s.m)
	}
	if s.root != 3 {
		t.Fatalf("root = %d, want 3", s.root)
	}

	// Real arcs: state == stateLower, flow == 0.
	for i := 0; i < s.mReal; i++ {
		if s.state[i] != stateLower {
			t.Errorf("real arc %d: state = %d, want %d", i, s.state[i], stateLower)
		}
		if !s.flow[i].IsZero() {
			t.Errorf("real arc %d: flow = %s, want 0", i, s.flow[i])
		}
	}

	// Artificial arcs: state == stateTree.
	for i := s.mReal; i < s.m; i++ {
		if s.state[i] != stateTree {
			t.Errorf("artificial arc %d: state = %d, want %d", i, s.state[i], stateTree)
		}
	}

	// Artificial arc for source (node 0, index mReal+0=2): flow == demand.
	artSrc := s.mReal + 0
	if s.flow[artSrc].Cmp(demand) != 0 {
		t.Errorf("artificial arc %d (source): flow = %s, want %s", artSrc, s.flow[artSrc], demand)
	}
	if s.cost[artSrc] != bigM(3) {
		t.Errorf("artificial arc %d (source): cost = %d, want %d", artSrc, s.cost[artSrc], bigM(3))
	}

	// Artificial arc for sink (node 2, index mReal+2=4): flow == demand.
	artSink := s.mReal + 2
	if s.flow[artSink].Cmp(demand) != 0 {
		t.Errorf("artificial arc %d (sink): flow = %s, want %s", artSink, s.flow[artSink], demand)
	}

	// Artificial arc for node 1 (index mReal+1=3): flow == 0.
	artMid := s.mReal + 1
	if !s.flow[artMid].IsZero() {
		t.Errorf("artificial arc %d (node 1): flow = %s, want 0", artMid, s.flow[artMid])
	}

	// Parent linkage.
	for v := 0; v < 3; v++ {
		if s.parent[v] != s.root {
			t.Errorf("parent[%d] = %d, want %d", v, s.parent[v], s.root)
		}
	}

	// Potentials: pi[root] == 0, |pi[v]| == bigM(3).
	M := bigM(3)
	if s.pi[s.root] != 0 {
		t.Errorf("pi[root] = %d, want 0", s.pi[s.root])
	}
	for v := 0; v < 3; v++ {
		absPI := s.pi[v]
		if absPI < 0 {
			absPI = -absPI
		}
		if absPI != M {
			t.Errorf("|pi[%d]| = %d, want %d", v, absPI, M)
		}
	}

	// Invoke reducedCost on every real arc to prove no panic.
	for i := 0; i < s.mReal; i++ {
		_ = s.reducedCost(i)
	}
}

func TestBuildInitialStateBlockSize(t *testing.T) {
	for _, mReal := range []int{1, 4, 7, 16, 100} {
		want := max(1, int(math.Ceil(math.Sqrt(float64(mReal)))))
		arcs := make([]Arc, mReal)
		for i := range arcs {
			arcs[i] = Arc{From: 0, To: 1, Cost: 1, Capacity: uint256.NewInt(1)}
		}
		s := buildInitialState(arcs, 2, 0, 1, uint256.NewInt(1))
		if s.blockSize != want {
			t.Errorf("mReal=%d: blockSize = %d, want %d", mReal, s.blockSize, want)
		}
	}
}
