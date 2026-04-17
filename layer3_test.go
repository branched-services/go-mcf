// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/holiman/uint256"
)

// (a) Negative-cost cycle entirely within the input graph.
// Graph: 0->1 cost 2 cap 20, 1->2 cost -3 cap 20, 2->1 cost 1 cap 20, 1->3 cost 1 cap 20.
// The cycle 1->2->1 has cost -3+1 = -2 (negative).
// Source=0, sink=3, demand=10.
// Base path 0->1->3 costs (2+1)*10 = 30, but the solver also circulates flow
// around the negative cycle 1->2->1 (cost -2 per unit) up to 20 units,
// saving 40. Optimal total cost = 30 - 40 = -10.
// Network Simplex handles this natively without special casing.
func TestLayer3_NegativeCostCycle(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 2, Capacity: u256(20)},
		{From: 1, To: 2, Cost: -3, Capacity: u256(20)},
		{From: 2, To: 1, Cost: 1, Capacity: u256(20)},
		{From: 1, To: 3, Cost: 1, Capacity: u256(20)},
	}
	demand := u256(10)

	res, err := solveAndCheck(t, arcs, 4, 0, 3, demand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalCost != -10 {
		t.Errorf("TotalCost = %d, want -10", res.TotalCost)
	}
}

// (b) Parallel arcs with strictly increasing costs.
// 0->1 cost 1 cap 5, 0->1 cost 3 cap 5, 0->1 cost 7 cap 5.
// Source=0, sink=1, demand=12.
// Optimal: saturate cost-1 (5), then cost-3 (5), then cost-7 (2).
// Total = 5*1 + 5*3 + 2*7 = 5 + 15 + 14 = 34.
func TestLayer3_ParallelArcsIncreasingCost(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(5)},
		{From: 0, To: 1, Cost: 3, Capacity: u256(5)},
		{From: 0, To: 1, Cost: 7, Capacity: u256(5)},
	}
	demand := u256(12)

	res, err := solveAndCheck(t, arcs, 2, 0, 1, demand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalCost != 34 {
		t.Errorf("TotalCost = %d, want 34", res.TotalCost)
	}

	if !arcs[0].Flow.Eq(u256(5)) {
		t.Errorf("arc[0].Flow = %s, want 5 (cheapest should saturate first)", arcs[0].Flow)
	}
	if !arcs[1].Flow.Eq(u256(5)) {
		t.Errorf("arc[1].Flow = %s, want 5 (second cheapest saturates next)", arcs[1].Flow)
	}
	if !arcs[2].Flow.Eq(u256(2)) {
		t.Errorf("arc[2].Flow = %s, want 2 (most expensive takes remainder)", arcs[2].Flow)
	}
}

// (c) Sparse multi-hop graph with many intermediate nodes and few direct arcs.
// Chain: 0->1->2->3->4->5, each cap 10, cost 1.
// Source=0, sink=5, demand=7.
// Only path is the chain; total cost = 7*5 = 35.
func TestLayer3_SparseMultiHop(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(10)},
		{From: 1, To: 2, Cost: 1, Capacity: u256(10)},
		{From: 2, To: 3, Cost: 1, Capacity: u256(10)},
		{From: 3, To: 4, Cost: 1, Capacity: u256(10)},
		{From: 4, To: 5, Cost: 1, Capacity: u256(10)},
	}
	demand := u256(7)

	res, err := solveAndCheck(t, arcs, 6, 0, 5, demand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalCost != 35 {
		t.Errorf("TotalCost = %d, want 35", res.TotalCost)
	}
	for i, a := range arcs {
		if !a.Flow.Eq(u256(7)) {
			t.Errorf("arc[%d].Flow = %s, want 7", i, a.Flow)
		}
	}
}

// (d) Single-unit demand routes along the minimum-cost path.
// Diamond: 0->1 cost 1, 1->2 cost 1, 0->2 cost 10, all cap 10.
// Source=0, sink=2, demand=1.
// Min-cost path: 0->1->2, cost=2.
func TestLayer3_SingleUnitDemand(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(10)},
		{From: 1, To: 2, Cost: 1, Capacity: u256(10)},
		{From: 0, To: 2, Cost: 10, Capacity: u256(10)},
	}
	demand := u256(1)

	res, err := solveAndCheck(t, arcs, 3, 0, 2, demand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalCost != 2 {
		t.Errorf("TotalCost = %d, want 2", res.TotalCost)
	}
	if !arcs[0].Flow.Eq(u256(1)) {
		t.Errorf("arc[0].Flow = %s, want 1", arcs[0].Flow)
	}
	if !arcs[1].Flow.Eq(u256(1)) {
		t.Errorf("arc[1].Flow = %s, want 1", arcs[1].Flow)
	}
	if !arcs[2].Flow.IsZero() {
		t.Errorf("arc[2].Flow = %s, want 0", arcs[2].Flow)
	}
}

// (e) Demand equals graph max-flow capacity -- saturates every path.
// Two paths: 0->1->2 cap 5, 0->2 cap 3. Max-flow = 8.
// Source=0, sink=2, demand=8.
func TestLayer3_DemandEqualsMaxFlow(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(5)},
		{From: 1, To: 2, Cost: 1, Capacity: u256(5)},
		{From: 0, To: 2, Cost: 4, Capacity: u256(3)},
	}
	demand := u256(8)

	res, err := solveAndCheck(t, arcs, 3, 0, 2, demand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.TotalFlow.Eq(demand) {
		t.Errorf("TotalFlow = %s, want %s", res.TotalFlow, demand)
	}
	if !arcs[0].Flow.Eq(u256(5)) {
		t.Errorf("arc[0].Flow = %s, want 5", arcs[0].Flow)
	}
	if !arcs[1].Flow.Eq(u256(5)) {
		t.Errorf("arc[1].Flow = %s, want 5", arcs[1].Flow)
	}
	if !arcs[2].Flow.Eq(u256(3)) {
		t.Errorf("arc[2].Flow = %s, want 3", arcs[2].Flow)
	}
}

// (f) Demand = max-flow + 1 returns ErrInfeasible.
func TestLayer3_DemandExceedsMaxFlow(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(5)},
		{From: 1, To: 2, Cost: 1, Capacity: u256(5)},
		{From: 0, To: 2, Cost: 4, Capacity: u256(3)},
	}
	demand := u256(9)

	_, err := Solve(context.Background(), arcs, 3, 0, 2, demand)
	if err == nil {
		t.Fatal("expected ErrInfeasible, got nil")
	}
	if !errors.Is(err, ErrInfeasible) {
		t.Errorf("error = %v, want ErrInfeasible", err)
	}
}

// (g) Large uint256 capacity exceeding MaxUint64.
// Two arcs: 0->1 cost 1 cap huge, 0->1 cost 2 cap huge.
// demand = huge (exceeds uint64). The solver must handle the large bottleneck
// and TotalCost accumulation is skipped per FR-10 because the flow is not
// representable as uint64.
func TestLayer3_LargeUint256Capacity(t *testing.T) {
	huge := new(uint256.Int).Lsh(uint256.NewInt(1), 128)

	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: new(uint256.Int).Set(huge)},
		{From: 0, To: 1, Cost: 2, Capacity: new(uint256.Int).Set(huge)},
	}
	demand := new(uint256.Int).Set(huge)

	res, err := solveAndCheck(t, arcs, 2, 0, 1, demand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.TotalFlow.Eq(demand) {
		t.Errorf("TotalFlow = %s, want %s", res.TotalFlow, demand)
	}

	if res.TotalCost != 0 {
		t.Errorf("TotalCost = %d, want 0 (skipped due to large flow exceeding uint64)", res.TotalCost)
	}
}

func TestSPDXHeaders(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	const expected = "// SPDX-License-Identifier: BSL-1.0"
	var violations []string

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == ".claude" || base == ".workflow" || base == "logs" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		if !scanner.Scan() {
			violations = append(violations, path+": empty file")
			return nil
		}
		line := scanner.Text()
		if line != expected {
			rel, _ := filepath.Rel(root, path)
			violations = append(violations, rel+": got "+line)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	for _, v := range violations {
		t.Errorf("SPDX header violation: %s", v)
	}
}
