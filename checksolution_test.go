// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"context"
	"math"
	"testing"

	"github.com/holiman/uint256"
)

type testReporter interface {
	Helper()
	Errorf(format string, args ...interface{})
}

func checkSolution(t testReporter, arcs []Arc, n, source, sink int, demand *uint256.Int, result Result, snap solverSnapshot) {
	t.Helper()

	// Invariant 1: flow conservation at every node other than source and sink.
	inFlow := make([]*uint256.Int, n)
	outFlow := make([]*uint256.Int, n)
	for i := 0; i < n; i++ {
		inFlow[i] = new(uint256.Int)
		outFlow[i] = new(uint256.Int)
	}
	for i, a := range arcs {
		if a.Flow == nil {
			continue
		}
		if a.From < 0 || a.From >= n || a.To < 0 || a.To >= n {
			t.Errorf("checkSolution: arc %d has out-of-range endpoints: from=%d, to=%d, n=%d", i, a.From, a.To, n)
			continue
		}
		outFlow[a.From] = new(uint256.Int).Add(outFlow[a.From], a.Flow)
		inFlow[a.To] = new(uint256.Int).Add(inFlow[a.To], a.Flow)
	}
	for node := 0; node < n; node++ {
		if node == source || node == sink {
			continue
		}
		if !inFlow[node].Eq(outFlow[node]) {
			t.Errorf("checkSolution: flow conservation violated at node %d: in=%s, out=%s", node, inFlow[node], outFlow[node])
		}
	}

	// Invariant 2: capacity feasibility -- 0 <= Flow <= Capacity per arc.
	for i, a := range arcs {
		if a.Flow == nil {
			continue
		}
		if a.Flow.Sign() < 0 {
			t.Errorf("checkSolution: capacity bound violated at arc %d: flow=%s, capacity=%s (negative flow)", i, a.Flow, a.Capacity)
		}
		if a.Capacity != nil && a.Flow.Gt(a.Capacity) {
			t.Errorf("checkSolution: capacity bound violated at arc %d: flow=%s, capacity=%s", i, a.Flow, a.Capacity)
		}
	}

	// Invariant 3: optimality certificate -- non-tree arcs at stateLower have
	// reduced cost >= 0, at stateUpper have reduced cost <= 0.
	for i, a := range arcs {
		st := snap.State[i]
		if st == stateTree {
			continue
		}
		rc := a.Cost - snap.Pi[a.From] + snap.Pi[a.To]
		if st == stateLower && rc < 0 {
			t.Errorf("checkSolution: optimality certificate violated at arc %d: state=lower, reduced_cost=%d", i, rc)
		}
		if st == stateUpper && rc > 0 {
			t.Errorf("checkSolution: optimality certificate violated at arc %d: state=upper, reduced_cost=%d", i, rc)
		}
	}

	// Invariant 4: demand satisfied -- TotalFlow == demand.
	if result.TotalFlow == nil || !result.TotalFlow.Eq(demand) {
		tf := "nil"
		if result.TotalFlow != nil {
			tf = result.TotalFlow.String()
		}
		t.Errorf("checkSolution: TotalFlow != demand: TotalFlow=%s, demand=%s", tf, demand)
	}

	// Invariant 5: cost consistency -- sum(arc.Cost * arc.Flow) == TotalCost
	// when all flows fit IsUint64; otherwise only assert TotalCost is not
	// obviously wrong (non-negative when all costs are non-negative).
	allFit := true
	var expectedCost int64
	overflow := false
	for _, a := range arcs {
		if a.Flow == nil || a.Flow.IsZero() {
			continue
		}
		if !a.Flow.IsUint64() {
			allFit = false
			break
		}
		f64 := int64(a.Flow.Uint64())
		if f64 < 0 {
			allFit = false
			break
		}
		cost := a.Cost
		if cost == math.MinInt64 {
			allFit = false
			break
		}
		absCost := cost
		if absCost < 0 {
			absCost = -absCost
		}
		if absCost != 0 && f64 > math.MaxInt64/absCost {
			overflow = true
			allFit = false
			break
		}
		product := f64 * cost
		if (cost >= 0 && expectedCost > math.MaxInt64-product) ||
			(cost < 0 && expectedCost < math.MinInt64-product) {
			overflow = true
			allFit = false
			break
		}
		expectedCost += product
	}

	if allFit && !overflow {
		if result.TotalCost != expectedCost {
			t.Errorf("checkSolution: TotalCost inconsistent: expected=%d, actual=%d", expectedCost, result.TotalCost)
		}
	} else {
		allNonNeg := true
		for _, a := range arcs {
			if a.Flow != nil && !a.Flow.IsZero() && a.Cost < 0 {
				allNonNeg = false
				break
			}
		}
		if allNonNeg && result.TotalCost < 0 {
			t.Errorf("checkSolution: TotalCost inconsistent: expected non-negative cost (all arc costs >= 0), actual=%d", result.TotalCost)
		}
	}
}

func solveAndCheck(t *testing.T, arcs []Arc, n, source, sink int, demand *uint256.Int) (Result, error) {
	t.Helper()
	res, s, err := solve(context.Background(), arcs, n, source, sink, demand)
	if err != nil {
		return res, err
	}
	checkSolution(t, arcs, n, source, sink, demand, res, s.snapshot())
	return res, nil
}

func buildSnapshot(n int, pi []int64, state []int) solverSnapshot {
	return solverSnapshot{
		N:     n,
		Pi:    pi,
		State: state,
	}
}
