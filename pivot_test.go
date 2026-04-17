// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"testing"

	"github.com/holiman/uint256"
)

func setupPivotSolver4() (*solver, int) {
	arcs := []Arc{
		{From: 1, To: 3, Cost: 5, Capacity: uint256.NewInt(100)},
	}
	n := 4
	s := newSolver(arcs, n, 0, 3, uint256.NewInt(100))
	s.initializeTree()
	artBase := len(s.arcs)
	root := s.n

	s.parent[0] = root
	s.parent[1] = 0
	s.parent[2] = root
	s.parent[3] = 2
	s.direction[0] = directionUp
	s.direction[1] = directionUp
	s.direction[2] = directionUp
	s.direction[3] = directionUp

	s.artArcs[0] = Arc{From: 0, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(10)}
	s.artArcs[1] = Arc{From: 1, To: 0, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(5)}
	s.artArcs[2] = Arc{From: 2, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(10)}
	s.artArcs[3] = Arc{From: 3, To: 2, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(7)}

	s.predArc[0] = artBase + 0
	s.predArc[1] = artBase + 1
	s.predArc[2] = artBase + 2
	s.predArc[3] = artBase + 3

	s.state[0] = stateLower
	for i := 0; i < n; i++ {
		s.state[artBase+i] = stateTree
	}

	s.rebuildDFS()

	s.pi[0] = 10
	s.pi[1] = 8
	s.pi[2] = 12
	s.pi[3] = 6
	s.pi[root] = 0

	return s, artBase
}

func TestPivotFlowPush(t *testing.T) {
	s, artBase := setupPivotSolver4()

	// Correct bottleneck from findLeaving:
	// First side (1->0->root): dirUp residuals = flow: 5, 10. min=5.
	// Second side (3->2->root): dirUp residuals = cap-flow: 13, 10. min=10.
	// Bottleneck = 5. Leaving = artBase+1 (node 1, flow reaches 0).
	enterArc := 0
	join := s.n
	leaveArc := artBase + 1
	bottleneck := uint256.NewInt(5)

	s.pivot(enterArc, leaveArc, join, bottleneck)

	wantFlows := map[string]uint64{
		"arc[0]": 5,
		"art[0]": 5,
		"art[1]": 0,
		"art[2]": 15,
		"art[3]": 12,
	}
	gotFlows := map[string]uint64{
		"arc[0]": s.arcs[0].Flow.Uint64(),
		"art[0]": s.artArcs[0].Flow.Uint64(),
		"art[1]": s.artArcs[1].Flow.Uint64(),
		"art[2]": s.artArcs[2].Flow.Uint64(),
		"art[3]": s.artArcs[3].Flow.Uint64(),
	}

	for name, want := range wantFlows {
		if gotFlows[name] != want {
			t.Errorf("%s flow = %d, want %d", name, gotFlows[name], want)
		}
	}
}

func TestPivotNoExtraneousFlowChanges(t *testing.T) {
	arcs := []Arc{
		{From: 1, To: 3, Cost: 5, Capacity: uint256.NewInt(100)},
		{From: 0, To: 2, Cost: 99, Capacity: uint256.NewInt(50), Flow: uint256.NewInt(42)},
	}
	n := 4
	s := newSolver(arcs, n, 0, 3, uint256.NewInt(100))
	s.initializeTree()
	artBase := len(s.arcs)
	root := s.n

	s.parent[0] = root
	s.parent[1] = 0
	s.parent[2] = root
	s.parent[3] = 2
	s.direction[0] = directionUp
	s.direction[1] = directionUp
	s.direction[2] = directionUp
	s.direction[3] = directionUp

	s.artArcs[0] = Arc{From: 0, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(10)}
	s.artArcs[1] = Arc{From: 1, To: 0, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(5)}
	s.artArcs[2] = Arc{From: 2, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(10)}
	s.artArcs[3] = Arc{From: 3, To: 2, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(7)}

	s.predArc[0] = artBase + 0
	s.predArc[1] = artBase + 1
	s.predArc[2] = artBase + 2
	s.predArc[3] = artBase + 3

	s.state[0] = stateLower
	s.state[1] = stateLower
	for i := 0; i < n; i++ {
		s.state[artBase+i] = stateTree
	}

	s.rebuildDFS()

	s.arcs[1].Flow.SetUint64(42)

	enterArc := 0
	join := root
	leaveArc := artBase + 1
	bottleneck := uint256.NewInt(5)

	s.pivot(enterArc, leaveArc, join, bottleneck)

	if got := s.arcs[1].Flow.Uint64(); got != 42 {
		t.Errorf("non-cycle arc flow changed: got %d, want 42", got)
	}
}

func TestPivotTreeArrays(t *testing.T) {
	s, artBase := setupPivotSolver4()
	root := s.n

	enterArc := 0
	join := root
	leaveArc := artBase + 1
	bottleneck := uint256.NewInt(5)

	s.pivot(enterArc, leaveArc, join, bottleneck)

	// After pivot: node 1 reparented to 3 (via entering arc 1->3).
	wantParent := []int{root, 3, root, 2, -1}
	for i, w := range wantParent {
		if s.parent[i] != w {
			t.Errorf("parent[%d] = %d, want %d", i, s.parent[i], w)
		}
	}

	if s.predArc[1] != enterArc {
		t.Errorf("predArc[1] = %d, want %d (entering arc)", s.predArc[1], enterArc)
	}
	if s.direction[1] != directionUp {
		t.Errorf("direction[1] = %d, want directionUp(%d)", s.direction[1], directionUp)
	}

	if s.state[enterArc] != stateTree {
		t.Errorf("state[enterArc] = %d, want stateTree(%d)", s.state[enterArc], stateTree)
	}
	if s.state[leaveArc] != stateLower {
		t.Errorf("state[leaveArc] = %d, want stateLower(%d)", s.state[leaveArc], stateLower)
	}

	visited := make(map[int]bool)
	cur := root
	for range s.n + 1 {
		if visited[cur] {
			t.Fatalf("thread revisits node %d", cur)
		}
		visited[cur] = true
		cur = s.thread[cur]
	}
	if cur != root {
		t.Fatalf("thread does not cycle back to root: ends at %d", cur)
	}
	if len(visited) != s.n+1 {
		t.Fatalf("thread visited %d nodes, want %d", len(visited), s.n+1)
	}

	for i := 0; i <= s.n; i++ {
		next := s.thread[i]
		if s.revThread[next] != i {
			t.Errorf("revThread[thread[%d]] = %d, want %d", i, s.revThread[next], i)
		}
	}

	for i := 0; i < s.n; i++ {
		pathLen := 0
		for u := i; u != root; u = s.parent[u] {
			pathLen++
			if pathLen > s.n+1 {
				t.Fatalf("infinite parent loop from node %d", i)
			}
		}
	}

	wantSuccNum := []int{1, 1, 3, 2, 5}
	for i, w := range wantSuccNum {
		if s.succNum[i] != w {
			t.Errorf("succNum[%d] = %d, want %d", i, s.succNum[i], w)
		}
	}
}

func TestPivotPotentialUpdate(t *testing.T) {
	s, artBase := setupPivotSolver4()
	root := s.n

	enterArc := 0
	join := root
	leaveArc := artBase + 1
	bottleneck := uint256.NewInt(5)

	s.pivot(enterArc, leaveArc, join, bottleneck)

	wantPi := []int64{10, 11, 12, 6, 0}
	for i, w := range wantPi {
		if s.pi[i] != w {
			t.Errorf("pi[%d] = %d, want %d", i, s.pi[i], w)
		}
	}

	rc := reducedCost(s, enterArc)
	if rc != 0 {
		t.Errorf("reducedCost(enterArc) = %d after pivot, want 0", rc)
	}
}

func TestPivotDegenerate(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 5, Capacity: uint256.NewInt(10), Flow: uint256.NewInt(10)},
	}
	n := 2
	s := newSolver(arcs, n, 0, 1, uint256.NewInt(10))
	s.initializeTree()
	s.rebuildDFS()

	s.state[0] = stateLower

	snapParent := make([]int, len(s.parent))
	copy(snapParent, s.parent)
	snapThread := make([]int, len(s.thread))
	copy(snapThread, s.thread)
	snapPi := make([]int64, len(s.pi))
	copy(snapPi, s.pi)

	oldFlow := new(uint256.Int).Set(s.arcs[0].Flow)

	enterArc := 0
	bottleneck := uint256.NewInt(0)
	s.pivot(enterArc, enterArc, s.n, bottleneck)

	if s.state[0] != stateUpper {
		t.Errorf("state[0] = %d after degenerate pivot, want stateUpper(%d)", s.state[0], stateUpper)
	}
	if !s.arcs[0].Flow.Eq(oldFlow) {
		t.Errorf("flow changed during degenerate pivot: got %s, want %s", s.arcs[0].Flow, oldFlow)
	}
	for i := range snapParent {
		if s.parent[i] != snapParent[i] {
			t.Errorf("parent[%d] changed during degenerate pivot: %d -> %d", i, snapParent[i], s.parent[i])
		}
	}
	for i := range snapThread {
		if s.thread[i] != snapThread[i] {
			t.Errorf("thread[%d] changed during degenerate pivot", i)
		}
	}
	for i := range snapPi {
		if s.pi[i] != snapPi[i] {
			t.Errorf("pi[%d] changed during degenerate pivot: %d -> %d", i, snapPi[i], s.pi[i])
		}
	}

	s.pivot(enterArc, enterArc, s.n, bottleneck)
	if s.state[0] != stateLower {
		t.Errorf("state[0] = %d after double flip, want stateLower(%d)", s.state[0], stateLower)
	}
}

func setupPivotSolver5() (*solver, int) {
	arcs := []Arc{
		{From: 2, To: 4, Cost: 3, Capacity: uint256.NewInt(50)},
	}
	n := 5
	s := newSolver(arcs, n, 0, 4, uint256.NewInt(100))
	s.initializeTree()
	artBase := len(s.arcs)
	root := s.n

	s.parent[0] = root
	s.parent[1] = 0
	s.parent[2] = 1
	s.parent[3] = root
	s.parent[4] = 3
	s.direction[0] = directionUp
	s.direction[1] = directionUp
	s.direction[2] = directionUp
	s.direction[3] = directionUp
	s.direction[4] = directionUp

	s.artArcs[0] = Arc{From: 0, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(10)}
	s.artArcs[1] = Arc{From: 1, To: 0, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(18)}
	s.artArcs[2] = Arc{From: 2, To: 1, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(5)}
	s.artArcs[3] = Arc{From: 3, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(12)}
	s.artArcs[4] = Arc{From: 4, To: 3, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(7)}

	s.predArc[0] = artBase + 0
	s.predArc[1] = artBase + 1
	s.predArc[2] = artBase + 2
	s.predArc[3] = artBase + 3
	s.predArc[4] = artBase + 4

	s.state[0] = stateLower
	for i := 0; i < n; i++ {
		s.state[artBase+i] = stateTree
	}

	s.rebuildDFS()

	s.pi[0] = 10
	s.pi[1] = 8
	s.pi[2] = 12
	s.pi[3] = 6
	s.pi[4] = 4
	s.pi[root] = 0

	return s, artBase
}

func TestPivotMultiNodeStem(t *testing.T) {
	s, artBase := setupPivotSolver5()
	root := s.n

	// Correct bottleneck from findLeaving:
	// First side (2->1->0->root): dirUp residuals = flow: 5, 18, 10. min=5.
	// Second side (4->3->root): dirUp residuals = cap-flow: 13, 8. min=8.
	// Bottleneck = 5. Leaving = artBase+2 (node 2, flow reaches 0).
	enterArc := 0
	join := root
	leaveArc := artBase + 2
	bottleneck := uint256.NewInt(5)

	s.pivot(enterArc, leaveArc, join, bottleneck)

	wantFlows := map[string]uint64{
		"arc[0]": 5,
		"art[0]": 5,
		"art[1]": 13,
		"art[2]": 0,
		"art[3]": 17,
		"art[4]": 12,
	}
	gotFlows := map[string]uint64{
		"arc[0]": s.arcs[0].Flow.Uint64(),
		"art[0]": s.artArcs[0].Flow.Uint64(),
		"art[1]": s.artArcs[1].Flow.Uint64(),
		"art[2]": s.artArcs[2].Flow.Uint64(),
		"art[3]": s.artArcs[3].Flow.Uint64(),
		"art[4]": s.artArcs[4].Flow.Uint64(),
	}
	for name, want := range wantFlows {
		if gotFlows[name] != want {
			t.Errorf("%s flow = %d, want %d", name, gotFlows[name], want)
		}
	}

	// After pivot: node 2 reparented to 4 (via entering arc 2->4).
	// Chain reversal only applies to uOut=2 (no loop since prev==uOut immediately).
	wantParent := []int{root, 0, 4, root, 3, -1}
	for i, w := range wantParent {
		if s.parent[i] != w {
			t.Errorf("parent[%d] = %d, want %d", i, s.parent[i], w)
		}
	}

	if s.predArc[2] != enterArc {
		t.Errorf("predArc[2] = %d, want %d", s.predArc[2], enterArc)
	}
	if s.direction[2] != directionUp {
		t.Errorf("direction[2] = %d, want directionUp", s.direction[2])
	}

	if s.state[enterArc] != stateTree {
		t.Errorf("state[enterArc] = %d, want stateTree", s.state[enterArc])
	}
	if s.state[leaveArc] != stateLower {
		t.Errorf("state[leaveArc] = %d, want stateLower", s.state[leaveArc])
	}

	wantSuccNum := []int{2, 1, 1, 3, 2, 6}
	for i, w := range wantSuccNum {
		if s.succNum[i] != w {
			t.Errorf("succNum[%d] = %d, want %d", i, s.succNum[i], w)
		}
	}

	visited := make(map[int]bool)
	cur := root
	for range s.n + 1 {
		if visited[cur] {
			t.Fatalf("thread revisits node %d", cur)
		}
		visited[cur] = true
		cur = s.thread[cur]
	}
	if cur != root {
		t.Fatalf("thread does not cycle back to root")
	}

	for i := 0; i <= s.n; i++ {
		next := s.thread[i]
		if s.revThread[next] != i {
			t.Errorf("revThread[thread[%d]] = %d, want %d", i, s.revThread[next], i)
		}
	}

	wantPi := []int64{10, 8, 7, 6, 4, 0}
	for i, w := range wantPi {
		if s.pi[i] != w {
			t.Errorf("pi[%d] = %d, want %d", i, s.pi[i], w)
		}
	}

	rc := reducedCost(s, enterArc)
	if rc != 0 {
		t.Errorf("reducedCost(enterArc) = %d after pivot, want 0", rc)
	}
}

func TestPivotLeavingAtUpperBound(t *testing.T) {
	// Test that leaving arc at capacity gets stateUpper.
	// Setup: second-side arc with cap-flow = bottleneck, so after push
	// flow reaches capacity.
	arcs := []Arc{
		{From: 1, To: 3, Cost: 5, Capacity: uint256.NewInt(100)},
	}
	n := 4
	s := newSolver(arcs, n, 0, 3, uint256.NewInt(100))
	s.initializeTree()
	artBase := len(s.arcs)
	root := s.n

	s.parent[0] = root
	s.parent[1] = 0
	s.parent[2] = root
	s.parent[3] = 2
	s.direction[0] = directionUp
	s.direction[1] = directionUp
	s.direction[2] = directionUp
	s.direction[3] = directionUp

	// Set flows so second-side arc at node 3 has cap-flow=5 (the bottleneck).
	s.artArcs[0] = Arc{From: 0, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(10)}
	s.artArcs[1] = Arc{From: 1, To: 0, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(5)}
	s.artArcs[2] = Arc{From: 2, To: root, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(10)}
	s.artArcs[3] = Arc{From: 3, To: 2, Cost: s.M, Capacity: uint256.NewInt(20), Flow: uint256.NewInt(15)}

	s.predArc[0] = artBase + 0
	s.predArc[1] = artBase + 1
	s.predArc[2] = artBase + 2
	s.predArc[3] = artBase + 3

	s.state[0] = stateLower
	for i := 0; i < n; i++ {
		s.state[artBase+i] = stateTree
	}
	s.rebuildDFS()

	// First side (1->0->root): dirUp residuals = flow: 5, 10. min=5.
	// Second side (3->2->root): dirUp residuals = cap-flow: 5, 10. min=5.
	// Tie: SFT picks second side, closest to join = node 2 (cap-flow=10).
	// But node 3 has cap-flow=5 <= bottleneck(5), so it's checked first.
	// Result: leaveArc=artBase+3, bottleneck=5.
	enterArc := 0
	join := root
	leaveArc := artBase + 3
	bottleneck := uint256.NewInt(5)

	s.pivot(enterArc, leaveArc, join, bottleneck)

	la := s.arc(leaveArc)
	if !la.Flow.Eq(la.Capacity) {
		t.Errorf("leaving arc flow=%s, cap=%s; expected flow==cap for stateUpper", la.Flow, la.Capacity)
	}
	if s.state[leaveArc] != stateUpper {
		t.Errorf("state[leaveArc] = %d, want stateUpper(%d)", s.state[leaveArc], stateUpper)
	}
}

func TestPivotLeavingAtLowerBound(t *testing.T) {
	s, artBase := setupPivotSolver4()
	root := s.n

	// leaveArc=artBase+1 (first side, node 1, flow=5).
	// With bottleneck=5: flow=5-5=0 → stateLower.
	enterArc := 0
	join := root
	leaveArc := artBase + 1
	bottleneck := uint256.NewInt(5)

	s.pivot(enterArc, leaveArc, join, bottleneck)

	la := s.arc(leaveArc)
	if !la.Flow.IsZero() {
		t.Errorf("leaving arc flow=%s; expected 0 for stateLower", la.Flow)
	}
	if s.state[leaveArc] != stateLower {
		t.Errorf("state[leaveArc] = %d, want stateLower(%d)", s.state[leaveArc], stateLower)
	}
}

func TestPivotZeroAllocs(t *testing.T) {
	s, artBase := setupPivotSolver4()
	root := s.n

	enterArc := 0
	join := root
	leaveArc := artBase + 1
	bottleneck := uint256.NewInt(5)

	allocs := testing.AllocsPerRun(100, func() {
		s.artArcs[1].Flow.SetUint64(5)
		s.arcs[0].Flow.Clear()
		s.artArcs[0].Flow.SetUint64(10)
		s.artArcs[2].Flow.SetUint64(10)
		s.artArcs[3].Flow.SetUint64(7)

		s.parent[0] = root
		s.parent[1] = 0
		s.parent[2] = root
		s.parent[3] = 2
		s.direction[0] = directionUp
		s.direction[1] = directionUp
		s.direction[2] = directionUp
		s.direction[3] = directionUp
		s.predArc[0] = artBase + 0
		s.predArc[1] = artBase + 1
		s.predArc[2] = artBase + 2
		s.predArc[3] = artBase + 3
		s.state[0] = stateLower
		s.state[artBase+1] = stateTree

		s.rebuildDFS()

		s.pi[0] = 10
		s.pi[1] = 8
		s.pi[2] = 12
		s.pi[3] = 6
		s.pi[root] = 0

		s.pivot(enterArc, leaveArc, join, bottleneck)
	})
	if allocs != 0 {
		t.Errorf("pivot allocs = %v, want 0", allocs)
	}
}

func TestPivotDegenerateZeroAllocs(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 5, Capacity: uint256.NewInt(10), Flow: uint256.NewInt(10)},
	}
	s := newSolver(arcs, 2, 0, 1, uint256.NewInt(10))
	s.initializeTree()
	s.rebuildDFS()
	s.state[0] = stateLower

	bottleneck := uint256.NewInt(0)

	allocs := testing.AllocsPerRun(100, func() {
		s.state[0] = stateLower
		s.pivot(0, 0, s.n, bottleneck)
	})
	if allocs != 0 {
		t.Errorf("degenerate pivot allocs = %v, want 0", allocs)
	}
}
