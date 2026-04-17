// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"context"
	"errors"
	"testing"
)

// Layer 1 tests ported from LEMON 1.3.1 test/min_cost_flow_test.cc.
// Ground-truth optimal costs taken directly from the LEMON test file.
// Graph inputs translated from LEMON's multi-supply model to single
// source/sink demand; see per-case comments for the correspondence.

// TestLayer1_TrivialSingleArc ports the simplest feasible case from
// LEMON test_neg2_lgf: a single arc graph with demand well within capacity.
// LEMON source: test/min_cost_flow_test.cc, test_neg2_lgf graph (nodes n1,n2).
func TestLayer1_TrivialSingleArc(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 5, Capacity: u256(100)},
	}
	demand := u256(10)

	res, s, err := solve(context.Background(), arcs, 2, 0, 1, demand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	snap := s.snapshot()
	checkSolution(t, arcs, 2, 0, 1, demand, res, snap)

	if res.TotalCost != 50 {
		t.Errorf("TotalCost = %d, want 50", res.TotalCost)
	}
	if !arcs[0].Flow.Eq(u256(10)) {
		t.Errorf("arc[0].Flow = %s, want 10", arcs[0].Flow)
	}
}

// TestLayer1_ParallelArcs verifies that parallel arcs with different costs
// are correctly handled: cheaper arc saturates before the more expensive one.
// Inspired by LEMON test_lgf pattern of multiple arcs sharing endpoints.
// LEMON source: test/min_cost_flow_test.cc, parallel arc semantics.
func TestLayer1_ParallelArcs(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(5)},
		{From: 0, To: 1, Cost: 3, Capacity: u256(5)},
	}
	demand := u256(8)

	res, s, err := solve(context.Background(), arcs, 2, 0, 1, demand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	snap := s.snapshot()
	checkSolution(t, arcs, 2, 0, 1, demand, res, snap)

	// Optimal: 5 units on cost-1 arc + 3 units on cost-3 arc = 5 + 9 = 14.
	if res.TotalCost != 14 {
		t.Errorf("TotalCost = %d, want 14", res.TotalCost)
	}
	if !arcs[0].Flow.Eq(u256(5)) {
		t.Errorf("arc[0].Flow = %s, want 5", arcs[0].Flow)
	}
	if !arcs[1].Flow.Eq(u256(3)) {
		t.Errorf("arc[1].Flow = %s, want 3", arcs[1].Flow)
	}
}

// TestLayer1_NegativeCostArc ports LEMON test_neg2_lgf: a single negative-cost
// arc. The solver must route flow along it and report a negative TotalCost.
// LEMON source: test/min_cost_flow_test.cc, test_neg2_lgf (n1->n2, cost=-1,
// cap=1000, supply n1=100, n2=-100 under EQ type). Expected cost = -100.
func TestLayer1_NegativeCostArc(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: -1, Capacity: u256(1000)},
	}
	demand := u256(100)

	res, s, err := solve(context.Background(), arcs, 2, 0, 1, demand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	snap := s.snapshot()
	checkSolution(t, arcs, 2, 0, 1, demand, res, snap)

	if res.TotalCost != -100 {
		t.Errorf("TotalCost = %d, want -100", res.TotalCost)
	}
	if !arcs[0].Flow.Eq(u256(100)) {
		t.Errorf("arc[0].Flow = %s, want 100", arcs[0].Flow)
	}
}

// TestLayer1_Infeasible verifies that demand exceeding the network's max-flow
// capacity returns ErrInfeasible. Ported from the LEMON pattern of testing
// infeasible supply configurations (sup2 with insufficient capacity).
// LEMON source: test/min_cost_flow_test.cc, infeasibility checks on
// NetworkSimplex with INFEASIBLE expected result.
func TestLayer1_Infeasible(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(5)},
		{From: 1, To: 2, Cost: 1, Capacity: u256(3)},
	}
	demand := u256(5)

	_, err := Solve(context.Background(), arcs, 3, 0, 2, demand)
	if err == nil {
		t.Fatal("expected ErrInfeasible, got nil")
	}
	if !errors.Is(err, ErrInfeasible) {
		t.Errorf("error = %v, want ErrInfeasible", err)
	}
}

// TestLayer1_ZeroCapacityArc verifies that a zero-capacity arc is inert: flow
// bypasses it even if its cost is lower. Ported from the LEMON pattern of
// arcs with lower==upper==0 (effectively zero capacity).
// LEMON source: test/min_cost_flow_test.cc, zero lower/upper bound semantics.
func TestLayer1_ZeroCapacityArc(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 5, Capacity: u256(10)},
		{From: 0, To: 1, Cost: 1, Capacity: u256(0)},
	}
	demand := u256(10)

	res, s, err := solve(context.Background(), arcs, 2, 0, 1, demand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	snap := s.snapshot()
	checkSolution(t, arcs, 2, 0, 1, demand, res, snap)

	// All flow must go through the cost-5 arc; cost-1 arc has zero capacity.
	if res.TotalCost != 50 {
		t.Errorf("TotalCost = %d, want 50", res.TotalCost)
	}
	if !arcs[0].Flow.Eq(u256(10)) {
		t.Errorf("arc[0].Flow = %s, want 10", arcs[0].Flow)
	}
	if !arcs[1].Flow.IsZero() {
		t.Errorf("arc[1].Flow = %s, want 0 (zero-capacity arc should be inert)", arcs[1].Flow)
	}
}

// TestLayer1_LEMONMainGraph ports LEMON test_lgf with supply config sup2:
// single source node 1 (supply=27), single sink node 12 (demand=-27).
// This maps directly to source=0, sink=11, demand=27 (0-indexed).
// All lower bounds are 0 (low1) so no transformation is needed.
// LEMON source: test/min_cost_flow_test.cc, test_lgf graph, sup2 map.
// Expected optimal cost: 7620 (from LEMON test expectations).
func TestLayer1_LEMONMainGraph(t *testing.T) {
	// 12 nodes (0-11), 21 arcs. Node indices shifted from LEMON's 1-based to 0-based.
	arcs := []Arc{
		{From: 0, To: 1, Cost: 70, Capacity: u256(11)},   // LEMON: 1->2
		{From: 0, To: 2, Cost: 150, Capacity: u256(3)},   // LEMON: 1->3
		{From: 0, To: 3, Cost: 80, Capacity: u256(15)},   // LEMON: 1->4
		{From: 1, To: 7, Cost: 80, Capacity: u256(12)},   // LEMON: 2->8
		{From: 2, To: 4, Cost: 140, Capacity: u256(5)},   // LEMON: 3->5
		{From: 3, To: 5, Cost: 60, Capacity: u256(10)},   // LEMON: 4->6
		{From: 3, To: 6, Cost: 80, Capacity: u256(2)},    // LEMON: 4->7
		{From: 3, To: 7, Cost: 110, Capacity: u256(3)},   // LEMON: 4->8
		{From: 4, To: 6, Cost: 60, Capacity: u256(14)},   // LEMON: 5->7
		{From: 4, To: 10, Cost: 120, Capacity: u256(12)}, // LEMON: 5->11
		{From: 5, To: 2, Cost: 0, Capacity: u256(3)},     // LEMON: 6->3
		{From: 5, To: 8, Cost: 140, Capacity: u256(4)},   // LEMON: 6->9
		{From: 5, To: 9, Cost: 90, Capacity: u256(8)},    // LEMON: 6->10
		{From: 6, To: 0, Cost: 30, Capacity: u256(5)},    // LEMON: 7->1
		{From: 7, To: 11, Cost: 60, Capacity: u256(16)},  // LEMON: 8->12
		{From: 8, To: 11, Cost: 50, Capacity: u256(6)},   // LEMON: 9->12
		{From: 9, To: 11, Cost: 70, Capacity: u256(13)},  // LEMON: 10->12
		{From: 9, To: 1, Cost: 100, Capacity: u256(7)},   // LEMON: 10->2
		{From: 9, To: 6, Cost: 60, Capacity: u256(10)},   // LEMON: 10->7
		{From: 10, To: 9, Cost: 20, Capacity: u256(14)},  // LEMON: 11->10
		{From: 11, To: 10, Cost: 30, Capacity: u256(10)}, // LEMON: 12->11
	}
	demand := u256(27)

	res, s, err := solve(context.Background(), arcs, 12, 0, 11, demand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	snap := s.snapshot()
	checkSolution(t, arcs, 12, 0, 11, demand, res, snap)

	if res.TotalCost != 7620 {
		t.Errorf("TotalCost = %d, want 7620", res.TotalCost)
	}
	if !res.TotalFlow.Eq(demand) {
		t.Errorf("TotalFlow = %s, want 27", res.TotalFlow)
	}
}

// TestLayer1_NegativeCostPath verifies that the solver correctly routes flow
// through a negative-cost arc when it lies on the optimal path.
// Inspired by LEMON test_neg1_lgf negative-cost cycle structure, adapted to
// single source/sink: the negative-cost arc is on the cheapest s-t path.
// LEMON source: test/min_cost_flow_test.cc, test_neg1_lgf negative cost patterns.
func TestLayer1_NegativeCostPath(t *testing.T) {
	// Graph: 0->1 cost 10, 1->2 cost -5, 0->2 cost 8
	// Source=0, sink=2, demand=10
	// Path 0->1->2: cost 10+(-5)=5 per unit
	// Path 0->2:     cost 8 per unit
	// Optimal: all 10 units through 0->1->2, total cost = 50
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: u256(20)},
		{From: 1, To: 2, Cost: -5, Capacity: u256(20)},
		{From: 0, To: 2, Cost: 8, Capacity: u256(20)},
	}
	demand := u256(10)

	res, s, err := solve(context.Background(), arcs, 3, 0, 2, demand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	snap := s.snapshot()
	checkSolution(t, arcs, 3, 0, 2, demand, res, snap)

	if res.TotalCost != 50 {
		t.Errorf("TotalCost = %d, want 50", res.TotalCost)
	}
	if !arcs[0].Flow.Eq(u256(10)) {
		t.Errorf("arc[0].Flow = %s, want 10 (should use negative-cost path)", arcs[0].Flow)
	}
	if !arcs[1].Flow.Eq(u256(10)) {
		t.Errorf("arc[1].Flow = %s, want 10", arcs[1].Flow)
	}
	if !arcs[2].Flow.IsZero() {
		t.Errorf("arc[2].Flow = %s, want 0 (direct path is more expensive)", arcs[2].Flow)
	}
}
