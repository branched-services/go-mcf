// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"testing"

	"github.com/holiman/uint256"
)

// buildTestSolver creates a solver with a manually configured tree for
// testing findLeaving. The caller must set up arcs, artArcs, parent,
// predArc, direction, succNum, and state as needed.
func buildTestSolver(n int, realArcs []Arc) *solver {
	demand := uint256.NewInt(100)
	s := newSolver(realArcs, n, 0, n-1, demand)
	s.initializeTree()
	return s
}

func TestFindLeavingBasicBottleneck(t *testing.T) {
	// Network: 0 → 1 → 2, root=3
	// Tree arcs (artificial): 0→3 (flow=10,cap=10), 1→3 (flow=0,cap=10), 3→2 (flow=10,cap=10)
	// Entering arc: 0→1 (real arc index 0), stateLower
	// join = 3 (root)
	//
	// First side (from node 0 to join=3):
	//   predArc[0] = art[0], direction=up, residual = cap-flow = 10-10 = 0
	// Second side (from node 1 to join=3):
	//   predArc[1] = art[1], direction=up, residual = flow = 0
	//
	// Both sides have residual 0. SFT tie-break: second side wins.
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: uint256.NewInt(100)},
	}
	s := buildTestSolver(3, arcs)
	artBase := len(s.arcs)

	// entering arc = 0 (real arc 0→1), stateLower
	s.state[0] = stateLower
	join := s.n // root

	leaving, df, ds := s.findLeaving(0, join)
	_ = df
	_ = ds

	// The artificial arcs at source(0) and node 1 both have
	// residual computed per the tree walk. The leaving arc should
	// be from the second side (SFT), which is art[1].
	if leaving != artBase+1 {
		t.Errorf("leaving = %d, want %d (art[1])", leaving, artBase+1)
	}
}

func TestFindLeavingSelectsMinResidual(t *testing.T) {
	// 4 nodes: 0,1,2,3, root=4
	// Build tree: root→0→1, root→2→3
	// Real arc entering: 1→3 (index 0), stateLower
	// join = root
	//
	// First side: 1→0→root
	//   predArc[1]: cap=20, flow=5, dir=up → residual = cap-flow = 15
	//   predArc[0]: cap=20, flow=18, dir=up → residual = cap-flow = 2
	// Second side: 3→2→root
	//   predArc[3]: cap=20, flow=7, dir=up → residual = flow = 7
	//   predArc[2]: cap=20, flow=12, dir=up → residual = flow = 12
	//
	// First side min = 2 (arc at node 0)
	// Second side min = 7 (arc at node 3)
	// Overall min = 2 → leaving arc is predArc[0]
	arcs := []Arc{
		{From: 1, To: 3, Cost: 1, Capacity: uint256.NewInt(100)},
	}
	n := 4
	s := buildTestSolver(n, arcs)
	root := s.n
	artBase := len(s.arcs)

	// Reparent: 1 under 0, 3 under 2
	s.parent[0] = root
	s.parent[1] = 0
	s.parent[2] = root
	s.parent[3] = 2
	s.succNum[root] = 5
	s.succNum[0] = 2
	s.succNum[1] = 1
	s.succNum[2] = 2
	s.succNum[3] = 1

	// Set up artificial arcs with specific flows/caps
	s.artArcs[0].Capacity = uint256.NewInt(20)
	s.artArcs[0].Flow = uint256.NewInt(18)
	s.artArcs[0].From = 0
	s.artArcs[0].To = root

	s.artArcs[1].Capacity = uint256.NewInt(20)
	s.artArcs[1].Flow = uint256.NewInt(5)
	s.artArcs[1].From = 1
	s.artArcs[1].To = 0

	s.artArcs[2].Capacity = uint256.NewInt(20)
	s.artArcs[2].Flow = uint256.NewInt(12)
	s.artArcs[2].From = 2
	s.artArcs[2].To = root

	s.artArcs[3].Capacity = uint256.NewInt(20)
	s.artArcs[3].Flow = uint256.NewInt(7)
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

	join := root
	leaving, df, ds := s.findLeaving(0, join)

	if !df.Eq(uint256.NewInt(2)) {
		t.Errorf("deltaFirst = %s, want 2", df)
	}
	if !ds.Eq(uint256.NewInt(7)) {
		t.Errorf("deltaSecond = %s, want 7", ds)
	}
	if leaving != artBase+0 {
		t.Errorf("leaving = %d, want %d (predArc[0])", leaving, artBase+0)
	}
}

func TestFindLeavingSFTTieBreak(t *testing.T) {
	// This test constructs a case where the SFT tie-break picks a different
	// arc than a naive first-seen approach.
	//
	// 4 nodes: 0,1,2,3, root=4
	// Tree: root→0→1 (first side), root→2→3 (second side)
	// Entering arc: 1→3, stateLower
	// join = root
	//
	// First side (1→0→root):
	//   predArc[1]: cap=10, flow=5, dir=up → residual = cap-flow = 5
	//   predArc[0]: cap=10, flow=5, dir=up → residual = cap-flow = 5
	//   First-side min = 5
	//
	// Second side (3→2→root):
	//   predArc[3]: cap=10, flow=5, dir=up → residual = flow = 5
	//   predArc[2]: cap=10, flow=5, dir=up → residual = flow = 5
	//   Second-side min = 5
	//
	// All arcs have residual = 5. With SFT tie-break (second side wins,
	// closest to join), leaving should be predArc[2] (closest to join on
	// second side).
	//
	// A naive first-seen approach would pick the first-side arc at node 1
	// (farthest from join), which is WRONG for SFT.
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

	s.artArcs[0].Capacity = uint256.NewInt(10)
	s.artArcs[0].Flow = uint256.NewInt(5)
	s.artArcs[0].From = 0
	s.artArcs[0].To = root

	s.artArcs[1].Capacity = uint256.NewInt(10)
	s.artArcs[1].Flow = uint256.NewInt(5)
	s.artArcs[1].From = 1
	s.artArcs[1].To = 0

	s.artArcs[2].Capacity = uint256.NewInt(10)
	s.artArcs[2].Flow = uint256.NewInt(5)
	s.artArcs[2].From = 2
	s.artArcs[2].To = root

	s.artArcs[3].Capacity = uint256.NewInt(10)
	s.artArcs[3].Flow = uint256.NewInt(5)
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

	// Entering arc at stateLower with large capacity (not the bottleneck)
	s.state[0] = stateLower

	join := root
	leaving, df, ds := s.findLeaving(0, join)

	if !df.Eq(uint256.NewInt(5)) {
		t.Errorf("deltaFirst = %s, want 5", df)
	}
	if !ds.Eq(uint256.NewInt(5)) {
		t.Errorf("deltaSecond = %s, want 5", ds)
	}
	// SFT: second side, closest to join = predArc[2]
	if leaving != artBase+2 {
		t.Errorf("leaving = %d, want %d (predArc[2] = SFT pick)", leaving, artBase+2)
	}
	// Verify this differs from naive first-seen (which would be predArc[1])
	if leaving == artBase+1 {
		t.Error("leaving arc matches naive first-seen; SFT tie-break not applied")
	}
}

func TestFindLeavingDirectionDown(t *testing.T) {
	// Test arcs with directionDown to verify correct residual computation.
	// 2 nodes: 0 (source), 1 (sink), root=2
	// Tree: root→0 (dirDown), 1→root (dirUp)
	// Entering arc: 0→1 (index 0), stateLower
	// join = root
	//
	// First side (from 0 to root):
	//   predArc[0] dirDown: residual = flow = 8
	// Second side (from 1 to root):
	//   predArc[1] dirUp: residual = flow = 3
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: uint256.NewInt(50)},
	}
	n := 2
	s := buildTestSolver(n, arcs)
	root := s.n
	artBase := len(s.arcs)

	s.artArcs[0].Capacity = uint256.NewInt(20)
	s.artArcs[0].Flow = uint256.NewInt(8)
	s.artArcs[0].From = root
	s.artArcs[0].To = 0
	s.direction[0] = directionDown
	s.predArc[0] = artBase + 0

	s.artArcs[1].Capacity = uint256.NewInt(20)
	s.artArcs[1].Flow = uint256.NewInt(3)
	s.artArcs[1].From = 1
	s.artArcs[1].To = root
	s.direction[1] = directionUp
	s.predArc[1] = artBase + 1

	s.state[0] = stateLower

	leaving, df, ds := s.findLeaving(0, root)

	if !df.Eq(uint256.NewInt(8)) {
		t.Errorf("deltaFirst = %s, want 8", df)
	}
	if !ds.Eq(uint256.NewInt(3)) {
		t.Errorf("deltaSecond = %s, want 3", ds)
	}
	if leaving != artBase+1 {
		t.Errorf("leaving = %d, want %d", leaving, artBase+1)
	}
}

func TestFindLeavingZeroAllocs(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: uint256.NewInt(100)},
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
	// When entering arc is stateUpper, first and second are swapped.
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: uint256.NewInt(50)},
	}
	n := 2
	s := buildTestSolver(n, arcs)
	s.arcs[0].Flow.SetUint64(50)
	root := s.n
	artBase := len(s.arcs)

	s.artArcs[0].Capacity = uint256.NewInt(20)
	s.artArcs[0].Flow = uint256.NewInt(3)
	s.artArcs[0].From = 0
	s.artArcs[0].To = root
	s.direction[0] = directionUp
	s.predArc[0] = artBase + 0

	s.artArcs[1].Capacity = uint256.NewInt(20)
	s.artArcs[1].Flow = uint256.NewInt(8)
	s.artArcs[1].From = 1
	s.artArcs[1].To = root
	s.direction[1] = directionUp
	s.predArc[1] = artBase + 1

	s.state[0] = stateUpper

	// stateUpper: first = To = 1, second = From = 0
	// First side (1 to root): predArc[1] dirUp → cap-flow = 20-8 = 12
	// Second side (0 to root): predArc[0] dirUp → flow = 3
	// Entering arc residual: flow = 50
	leaving, df, ds := s.findLeaving(0, root)

	if !df.Eq(uint256.NewInt(12)) {
		t.Errorf("deltaFirst = %s, want 12", df)
	}
	if !ds.Eq(uint256.NewInt(3)) {
		t.Errorf("deltaSecond = %s, want 3", ds)
	}
	if leaving != artBase+0 {
		t.Errorf("leaving = %d, want %d", leaving, artBase+0)
	}
}
