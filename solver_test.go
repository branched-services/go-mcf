// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"math"
	"testing"

	"github.com/holiman/uint256"
)

func TestBigM(t *testing.T) {
	n := 4
	want := math.MaxInt64 / (8 * int64(n+1))
	s := newSolver([]Arc{}, n, 0, 1, uint256.NewInt(100))
	if s.M != want {
		t.Errorf("M = %d, want %d", s.M, want)
	}
}

func TestScratchBuffersNonNil(t *testing.T) {
	s := newSolver([]Arc{}, 2, 0, 1, uint256.NewInt(1))
	if s.bottleneck == nil {
		t.Fatal("bottleneck is nil")
	}
	if s.tmpFlow == nil {
		t.Fatal("tmpFlow is nil")
	}
	ptr1 := s.bottleneck
	ptr2 := s.tmpFlow
	if ptr1 == ptr2 {
		t.Fatal("bottleneck and tmpFlow share the same pointer")
	}
}

func TestInitializeTreeSpanningTree(t *testing.T) {
	demand := uint256.NewInt(50)
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100)},
		{From: 1, To: 2, Cost: 20, Capacity: uint256.NewInt(100)},
		{From: 2, To: 3, Cost: 30, Capacity: uint256.NewInt(100)},
	}
	n := 4
	source, sink := 0, 3
	s := newSolver(arcs, n, source, sink, demand)
	s.initializeTree()
	snap := s.snapshot()
	root := n

	for i := 0; i < n; i++ {
		if snap.Parent[i] != root {
			t.Errorf("parent[%d] = %d, want %d", i, snap.Parent[i], root)
		}
	}
	if snap.Parent[root] != -1 {
		t.Errorf("parent[root] = %d, want -1", snap.Parent[root])
	}

	visited := make(map[int]bool)
	cur := root
	for range n + 1 {
		if visited[cur] {
			t.Fatalf("thread revisits node %d", cur)
		}
		visited[cur] = true
		cur = snap.Thread[cur]
	}
	if cur != root {
		t.Fatalf("thread does not cycle back to root: ends at %d", cur)
	}
	if len(visited) != n+1 {
		t.Fatalf("thread visited %d nodes, want %d", len(visited), n+1)
	}

	for i := 0; i <= n; i++ {
		next := snap.Thread[i]
		if snap.RevThread[next] != i {
			t.Errorf("revThread[thread[%d]] = %d, want %d", i, snap.RevThread[next], i)
		}
	}
}

func TestInitializeTreeArtificialArcFlows(t *testing.T) {
	demand := uint256.NewInt(50)
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100)},
		{From: 1, To: 2, Cost: 20, Capacity: uint256.NewInt(100)},
		{From: 2, To: 3, Cost: 30, Capacity: uint256.NewInt(100)},
	}
	n := 4
	source, sink := 0, 3
	s := newSolver(arcs, n, source, sink, demand)
	s.initializeTree()
	snap := s.snapshot()
	root := n

	srcArt := &snap.ArtArcs[source]
	if srcArt.From != source || srcArt.To != root {
		t.Errorf("source art arc: from=%d to=%d, want from=%d to=%d", srcArt.From, srcArt.To, source, root)
	}
	if !srcArt.Flow.Eq(demand) {
		t.Errorf("source art arc flow = %s, want %s", srcArt.Flow, demand)
	}

	sinkArt := &snap.ArtArcs[sink]
	if sinkArt.From != root || sinkArt.To != sink {
		t.Errorf("sink art arc: from=%d to=%d, want from=%d to=%d", sinkArt.From, sinkArt.To, root, sink)
	}
	if !sinkArt.Flow.Eq(demand) {
		t.Errorf("sink art arc flow = %s, want %s", sinkArt.Flow, demand)
	}

	for i := 0; i < n; i++ {
		if i == source || i == sink {
			continue
		}
		art := &snap.ArtArcs[i]
		if !art.Flow.IsZero() {
			t.Errorf("art arc[%d] flow = %s, want 0", i, art.Flow)
		}
	}
}

func TestInitializeTreeRealArcState(t *testing.T) {
	demand := uint256.NewInt(50)
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100)},
		{From: 1, To: 2, Cost: 20, Capacity: uint256.NewInt(100)},
		{From: 2, To: 3, Cost: 30, Capacity: uint256.NewInt(100)},
	}
	s := newSolver(arcs, 4, 0, 3, demand)
	s.initializeTree()
	snap := s.snapshot()

	for i := range snap.Arcs {
		if snap.State[i] != stateLower {
			t.Errorf("arcs[%d] state = %d, want %d", i, snap.State[i], stateLower)
		}
		if !snap.Arcs[i].Flow.IsZero() {
			t.Errorf("arcs[%d] flow = %s, want 0", i, snap.Arcs[i].Flow)
		}
	}
}

func TestInitializeTreeReducedCosts(t *testing.T) {
	demand := uint256.NewInt(50)
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100)},
		{From: 1, To: 2, Cost: 20, Capacity: uint256.NewInt(100)},
		{From: 2, To: 3, Cost: 30, Capacity: uint256.NewInt(100)},
	}
	n := 4
	s := newSolver(arcs, n, 0, 3, demand)
	s.initializeTree()
	snap := s.snapshot()

	for i := 0; i < n; i++ {
		art := &snap.ArtArcs[i]
		rc := art.Cost - snap.Pi[art.From] + snap.Pi[art.To]
		if rc != 0 {
			t.Errorf("art arc[%d] reduced cost = %d, want 0 (cost=%d, pi[%d]=%d, pi[%d]=%d)",
				i, rc, art.Cost, art.From, snap.Pi[art.From], art.To, snap.Pi[art.To])
		}
	}
}

func TestInitializeTreeSuccNumLastSucc(t *testing.T) {
	demand := uint256.NewInt(50)
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100)},
	}
	n := 3
	s := newSolver(arcs, n, 0, 2, demand)
	s.initializeTree()
	snap := s.snapshot()
	root := n

	for i := 0; i < n; i++ {
		if snap.SuccNum[i] != 1 {
			t.Errorf("succNum[%d] = %d, want 1", i, snap.SuccNum[i])
		}
		if snap.LastSucc[i] != i {
			t.Errorf("lastSucc[%d] = %d, want %d", i, snap.LastSucc[i], i)
		}
	}
	if snap.SuccNum[root] != n+1 {
		t.Errorf("succNum[root] = %d, want %d", snap.SuccNum[root], n+1)
	}
}

func TestInitializeTreeArtArcStates(t *testing.T) {
	demand := uint256.NewInt(10)
	arcs := []Arc{
		{From: 0, To: 1, Cost: 5, Capacity: uint256.NewInt(20)},
	}
	n := 2
	s := newSolver(arcs, n, 0, 1, demand)
	s.initializeTree()
	snap := s.snapshot()
	artBase := len(snap.Arcs)

	for i := 0; i < n; i++ {
		if snap.State[artBase+i] != stateTree {
			t.Errorf("state[artArc %d] = %d, want %d", i, snap.State[artBase+i], stateTree)
		}
	}
}

func TestInitializeTreeNilFlowAllocated(t *testing.T) {
	demand := uint256.NewInt(10)
	arcs := []Arc{
		{From: 0, To: 1, Cost: 5, Capacity: uint256.NewInt(20), Flow: nil},
	}
	s := newSolver(arcs, 2, 0, 1, demand)
	s.initializeTree()

	if s.arcs[0].Flow == nil {
		t.Fatal("Flow was not allocated for arc with nil Flow")
	}
	if !s.arcs[0].Flow.IsZero() {
		t.Errorf("Flow = %s, want 0", s.arcs[0].Flow)
	}
}
