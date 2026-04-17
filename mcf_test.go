// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/holiman/uint256"
)

func u256(v uint64) *uint256.Int {
	return uint256.NewInt(v)
}

func TestSolve_FeasibleDiamond(t *testing.T) {
	// Diamond network: 4 nodes
	//   0 -> 1 (cap 10, cost 1)
	//   0 -> 2 (cap 10, cost 5)
	//   1 -> 3 (cap 10, cost 1)
	//   2 -> 3 (cap 10, cost 1)
	// source=0, sink=3, demand=10
	// Optimal: route all 10 through 0->1->3, total cost = 10*1 + 10*1 = 20
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(10)},
		{From: 0, To: 2, Cost: 5, Capacity: u256(10)},
		{From: 1, To: 3, Cost: 1, Capacity: u256(10)},
		{From: 2, To: 3, Cost: 1, Capacity: u256(10)},
	}

	res, err := Solve(context.Background(), arcs, 4, 0, 3, u256(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.TotalFlow.Eq(u256(10)) {
		t.Errorf("TotalFlow = %s, want 10", res.TotalFlow)
	}
	if res.TotalCost != 20 {
		t.Errorf("TotalCost = %d, want 20", res.TotalCost)
	}

	// Check per-arc flows: 0->1 = 10, 0->2 = 0, 1->3 = 10, 2->3 = 0
	wantFlows := []uint64{10, 0, 10, 0}
	for i, want := range wantFlows {
		if !arcs[i].Flow.Eq(u256(want)) {
			t.Errorf("arcs[%d].Flow = %s, want %d", i, arcs[i].Flow, want)
		}
	}
}

func TestSolve_FeasibleSplit(t *testing.T) {
	// Network where demand must split across two paths.
	//   0 -> 1 (cap 5, cost 1)
	//   0 -> 2 (cap 5, cost 2)
	//   1 -> 3 (cap 5, cost 1)
	//   2 -> 3 (cap 5, cost 1)
	// source=0, sink=3, demand=8
	// Optimal: 5 through 0->1->3 (cost 10), 3 through 0->2->3 (cost 9) = 19
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(5)},
		{From: 0, To: 2, Cost: 2, Capacity: u256(5)},
		{From: 1, To: 3, Cost: 1, Capacity: u256(5)},
		{From: 2, To: 3, Cost: 1, Capacity: u256(5)},
	}

	res, err := Solve(context.Background(), arcs, 4, 0, 3, u256(8))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.TotalFlow.Eq(u256(8)) {
		t.Errorf("TotalFlow = %s, want 8", res.TotalFlow)
	}
	if res.TotalCost != 19 {
		t.Errorf("TotalCost = %d, want 19", res.TotalCost)
	}

	wantFlows := []uint64{5, 3, 5, 3}
	for i, want := range wantFlows {
		if !arcs[i].Flow.Eq(u256(want)) {
			t.Errorf("arcs[%d].Flow = %s, want %d", i, arcs[i].Flow, want)
		}
	}
}

func TestSolve_Infeasible(t *testing.T) {
	// Demand exceeds max-flow capacity.
	//   0 -> 1 (cap 5, cost 1)
	// source=0, sink=1, demand=10
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(5)},
	}

	_, err := Solve(context.Background(), arcs, 2, 0, 1, u256(10))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInfeasible) {
		t.Errorf("error = %v, want ErrInfeasible", err)
	}
}

func TestSolve_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(10)},
		{From: 1, To: 2, Cost: 1, Capacity: u256(10)},
	}

	res, err := Solve(ctx, arcs, 3, 0, 2, u256(5))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	if res.TotalFlow != nil || res.TotalCost != 0 {
		t.Errorf("expected zero Result, got %+v", res)
	}
}

func TestSolve_BestEffortTotalCost_LargeBottleneck(t *testing.T) {
	// Use a capacity so large that bottleneck * cost overflows int64.
	// The solver should still succeed but TotalCost may not include this pivot's contribution.
	huge := new(uint256.Int).SetUint64(math.MaxUint64)
	huge.Lsh(huge, 64)

	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: new(uint256.Int).Set(huge)},
	}

	res, err := Solve(context.Background(), arcs, 2, 0, 1, new(uint256.Int).Set(huge))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.TotalFlow.Eq(huge) {
		t.Errorf("TotalFlow = %s, want %s", res.TotalFlow, huge)
	}
	// TotalCost should be 0 because the bottleneck doesn't fit in uint64
	if res.TotalCost != 0 {
		t.Errorf("TotalCost = %d, want 0 (skipped due to large bottleneck)", res.TotalCost)
	}
}

func TestSolve_ZeroResult(t *testing.T) {
	// Cancelled context returns zero-value Result.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(10)},
	}

	res, err := Solve(ctx, arcs, 2, 0, 1, u256(5))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	var zero Result
	if res != zero {
		t.Errorf("expected zero Result, got %+v", res)
	}
}
