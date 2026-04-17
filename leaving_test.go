// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"testing"

	"github.com/holiman/uint256"
)

func buildTestSolver(n int, realArcs []Arc) *solver {
	demand := uint256.NewInt(100)
	s := newSolver(realArcs, n, 0, n-1, demand)
	s.initializeTree()
	return s
}

func TestFindLeavingBasicBottleneck(t *testing.T) {
	// Network: 0 -> 1 -> 2, root=3
	// Entering arc: 0->1 (real arc 0), stateLower. first=0, second=1.
	// join = root(3)
	//
	// First side (0 to root):
	//   predArc[0]=art[0] (0->root), dirUp. residual = flow = 100
	// Second side (1 to root):
	//   predArc[1]=art[1] (1->root), dirUp. residual = cap-flow = 100-0 = 100
	//
	// Entering arc: cap=100, flow=0. bottleneck = 100.
	// First side: 100 < 100? No (strict). No update.
	// Second side: 100 <= 100? Yes. SFT picks art[1].
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: uint256.NewInt(100)},
	}
	s := buildTestSolver(3, arcs)
	artBase := len(s.arcs)

	s.state[0] = stateLower
	join := s.n

	leaving, _, _ := s.findLeaving(0, join)

	if leaving != artBase+1 {
		t.Errorf("leaving = %d, want %d (art[1], SFT pick)", leaving, artBase+1)
	}
}

func TestFindLeavingSelectsMinResidual(t *testing.T) {
	// 4 nodes: 0,1,2,3, root=4
	// Tree: root->0->1, root->2->3
	// Entering arc: 1->3 (index 0), stateLower. first=1, second=3.
	// join = root
	//
	// First side (1->0->root), all dirUp:
	//   node 1: residual = flow = 5
	//   node 0: residual = flow = 18
	// deltaFirst = 5
	//
	// Second side (3->2->root), all dirUp:
	//   node 3: residual = cap-flow = 20-7 = 13
	//   node 2: residual = cap-flow = 20-12 = 8
	// deltaSecond = 8
	//
	// Bottleneck = min(100, 5) = 5 -> leaving = artBase+1
	arcs := []Arc{
		{From: 1, To: 3, Cost: 1, Capacity: uint256.NewInt(100)},
	}
	n := 4
	s := buildTestSolver(n, arcs)
	root := s.n
	artBase := len(s.arcs)

	s.parent[0] = root
	s.parent[1] = 0
	s.parent[2] = root
	s.parent[3] = 2
	s.succNum[root] = 5
	s.succNum[0] = 2
	s.succNum[1] = 1
	s.succNum[2] = 2
	s.succNum[3] = 1

	s.artArcs[0] = Arc{From: 0, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(18)}
	s.artArcs[1] = Arc{From: 1, To: 0, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(5)}
	s.artArcs[2] = Arc{From: 2, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(12)}
	s.artArcs[3] = Arc{From: 3, To: 2, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(7)}

	s.predArc[0] = artBase + 0
	s.predArc[1] = artBase + 1
	s.predArc[2] = artBase + 2
	s.predArc[3] = artBase + 3

	s.direction[0] = directionUp
	s.direction[1] = directionUp
	s.direction[2] = directionUp
	s.direction[3] = directionUp

	s.state[0] = stateLower

	join := root
	leaving, df, ds := s.findLeaving(0, join)

	if !df.Eq(uint256.NewInt(5)) {
		t.Errorf("deltaFirst = %s, want 5", df)
	}
	if !ds.Eq(uint256.NewInt(8)) {
		t.Errorf("deltaSecond = %s, want 8", ds)
	}
	if leaving != artBase+1 {
		t.Errorf("leaving = %d, want %d (predArc[1])", leaving, artBase+1)
	}
}

func TestFindLeavingSFTTieBreak(t *testing.T) {
	// 4 nodes: 0,1,2,3, root=4
	// Tree: root->0->1 (first side), root->2->3 (second side)
	// Entering arc: 1->3, stateLower. first=1, second=3.
	// join = root
	//
	// First side (1->0->root), all dirUp:
	//   node 1: residual = flow = 5
	//   node 0: residual = flow = 5
	// deltaFirst = 5
	//
	// Second side (3->2->root), all dirUp:
	//   node 3: residual = cap-flow = 10-5 = 5
	//   node 2: residual = cap-flow = 10-5 = 5
	// deltaSecond = 5
	//
	// All residuals = 5. SFT tie-break: second side wins (<=).
	// On second side, closest to join = node 2. leaving = predArc[2].
	arcs := []Arc{
		{From: 1, To: 3, Cost: 1, Capacity: uint256.NewInt(100)},
	}
	n := 4
	s := buildTestSolver(n, arcs)
	root := s.n
	artBase := len(s.arcs)

	s.parent[0] = root
	s.parent[1] = 0
	s.parent[2] = root
	s.parent[3] = 2
	s.succNum[root] = 5
	s.succNum[0] = 2
	s.succNum[1] = 1
	s.succNum[2] = 2
	s.succNum[3] = 1

	s.artArcs[0] = Arc{From: 0, To: root, Cost: s.M, Capacity: uint256.NewInt(10), Flow: uint256.NewInt(5)}
	s.artArcs[1] = Arc{From: 1, To: 0, Cost: s.M, Capacity: uint256.NewInt(10), Flow: uint256.NewInt(5)}
	s.artArcs[2] = Arc{From: 2, To: root, Cost: s.M, Capacity: uint256.NewInt(10), Flow: uint256.NewInt(5)}
	s.artArcs[3] = Arc{From: 3, To: 2, Cost: s.M, Capacity: uint256.NewInt(10), Flow: uint256.NewInt(5)}

	s.predArc[0] = artBase + 0
	s.predArc[1] = artBase + 1
	s.predArc[2] = artBase + 2
	s.predArc[3] = artBase + 3

	s.direction[0] = directionUp
	s.direction[1] = directionUp
	s.direction[2] = directionUp
	s.direction[3] = directionUp

	s.state[0] = stateLower

	join := root
	leaving, df, ds := s.findLeaving(0, join)

	if !df.Eq(uint256.NewInt(5)) {
		t.Errorf("deltaFirst = %s, want 5", df)
	}
	if !ds.Eq(uint256.NewInt(5)) {
		t.Errorf("deltaSecond = %s, want 5", ds)
	}
	if leaving != artBase+2 {
		t.Errorf("leaving = %d, want %d (predArc[2] = SFT pick)", leaving, artBase+2)
	}
	if leaving == artBase+1 {
		t.Error("leaving arc matches naive first-seen; SFT tie-break not applied")
	}
}

func TestFindLeavingDirectionDown(t *testing.T) {
	// 2 nodes: 0 (source), 1 (sink), root=2
	// Entering arc: 0->1 (index 0), stateLower. first=0, second=1.
	// join = root
	//
	// First side (0 to root):
	//   predArc[0]: root->0, dirDown. residual = cap-flow = 20-8 = 12
	// Second side (1 to root):
	//   predArc[1]: 1->root, dirUp. residual = cap-flow = 20-3 = 17
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: uint256.NewInt(50)},
	}
	n := 2
	s := buildTestSolver(n, arcs)
	root := s.n
	artBase := len(s.arcs)

	s.artArcs[0] = Arc{From: root, To: 0, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(8)}
	s.direction[0] = directionDown
	s.predArc[0] = artBase + 0

	s.artArcs[1] = Arc{From: 1, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(3)}
	s.direction[1] = directionUp
	s.predArc[1] = artBase + 1

	s.state[0] = stateLower

	leaving, df, ds := s.findLeaving(0, root)

	if !df.Eq(uint256.NewInt(12)) {
		t.Errorf("deltaFirst = %s, want 12", df)
	}
	if !ds.Eq(uint256.NewInt(17)) {
		t.Errorf("deltaSecond = %s, want 17", ds)
	}
	if leaving != artBase+0 {
		t.Errorf("leaving = %d, want %d", leaving, artBase+0)
	}
}

func TestFindLeavingZeroAllocs(t *testing.T) {
	arcs := []Arc{
		{From: 1, To: 3, Cost: 1, Capacity: uint256.NewInt(100)},
	}
	n := 4
	s := buildTestSolver(n, arcs)
	root := s.n
	artBase := len(s.arcs)

	s.parent[0] = root
	s.parent[1] = 0
	s.parent[2] = root
	s.parent[3] = 2
	s.succNum[root] = 5
	s.succNum[0] = 2
	s.succNum[1] = 1
	s.succNum[2] = 2
	s.succNum[3] = 1

	for i := 0; i < n; i++ {
		s.artArcs[i].Capacity = uint256.NewInt(20)
		s.artArcs[i].Flow = uint256.NewInt(5)
	}
	s.artArcs[0].From = 0
	s.artArcs[0].To = root
	s.artArcs[1].From = 1
	s.artArcs[1].To = 0
	s.artArcs[2].From = 2
	s.artArcs[2].To = root
	s.artArcs[3].From = 3
	s.artArcs[3].To = 2

	s.predArc[0] = artBase + 0
	s.predArc[1] = artBase + 1
	s.predArc[2] = artBase + 2
	s.predArc[3] = artBase + 3

	s.direction[0] = directionUp
	s.direction[1] = directionUp
	s.direction[2] = directionUp
	s.direction[3] = directionUp

	s.state[0] = stateLower

	allocs := testing.AllocsPerRun(1000, func() {
		s.findLeaving(0, root)
	})
	if allocs != 0 {
		t.Errorf("findLeaving allocs = %v, want 0", allocs)
	}
}

func TestFindJoinZeroAllocs(t *testing.T) {
	s := setupTreeSolver(5, 0, 4, nil)
	root := s.n

	s.parent[0] = root
	s.parent[1] = 0
	s.parent[2] = root
	s.parent[3] = 2
	s.parent[4] = 3
	s.succNum[root] = 6
	s.succNum[0] = 2
	s.succNum[1] = 1
	s.succNum[2] = 3
	s.succNum[3] = 2
	s.succNum[4] = 1

	allocs := testing.AllocsPerRun(1000, func() {
		s.findJoin(1, 4)
	})
	if allocs != 0 {
		t.Errorf("findJoin allocs = %v, want 0", allocs)
	}
}

func TestFindLeavingStateUpper(t *testing.T) {
	// Entering arc 0->1, stateUpper. first=To=1, second=From=0.
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: uint256.NewInt(50)},
	}
	n := 2
	s := buildTestSolver(n, arcs)
	s.arcs[0].Flow.SetUint64(50)
	root := s.n
	artBase := len(s.arcs)

	s.artArcs[0] = Arc{From: 0, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(3)}
	s.direction[0] = directionUp
	s.predArc[0] = artBase + 0

	s.artArcs[1] = Arc{From: 1, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(8)}
	s.direction[1] = directionUp
	s.predArc[1] = artBase + 1

	s.state[0] = stateUpper

	// stateUpper: first = To = 1, second = From = 0
	// First side (1 to root): dirUp, residual = flow = 8
	// Second side (0 to root): dirUp, residual = cap-flow = 20-3 = 17
	// Entering arc residual: flow = 50
	// Bottleneck = 8 (first side), leaving = artBase+1
	leaving, df, ds := s.findLeaving(0, root)

	if !df.Eq(uint256.NewInt(8)) {
		t.Errorf("deltaFirst = %s, want 8", df)
	}
	if !ds.Eq(uint256.NewInt(17)) {
		t.Errorf("deltaSecond = %s, want 17", ds)
	}
	if leaving != artBase+1 {
		t.Errorf("leaving = %d, want %d", leaving, artBase+1)
	}
}
