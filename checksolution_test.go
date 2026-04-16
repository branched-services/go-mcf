// SPDX-License-Identifier: BSL-1.0

// checkSolution is a shared test helper (package-private, _test.go file) that
// verifies flow conservation, capacity feasibility, demand satisfaction, and
// cost consistency on a Solve result. Callers include lemon_test.go,
// structural_test.go, and dimacs_bench_test.go.
//
// Optimality certificate: the public Result does not expose dual potentials
// (pi) or arc state, so checkSolution cannot verify complementary slackness
// directly. Optimality is instead exercised by per-instance LEMON-ported
// expected-cost assertions in the ported test suites, which compare against
// externally-verified optima.

package mcf

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
)

// arcFlowBig returns a *big.Int with the value of a.Flow.
func arcFlowBig(a Arc) *big.Int {
	return a.Flow.ToBig()
}

// checkSolution runs every assertion against the given solution; it does not
// return early after the first failure so that every violated invariant is
// reported.
func checkSolution(t testing.TB, arcs []Arc, n, source, sink int, demand *uint256.Int, res Result) {
	t.Helper()

	// --- (1) Capacity feasibility ---
	for i, a := range arcs {
		if a.Flow == nil {
			t.Errorf("arc %d (%d->%d): Flow is nil", i, a.From, a.To)
			continue
		}
		if a.Flow.Cmp(a.Capacity) > 0 {
			t.Errorf("arc %d (%d->%d): Flow %s exceeds Capacity %s",
				i, a.From, a.To, a.Flow, a.Capacity)
		}
	}

	// --- (2) Flow conservation ---
	// Compute inflow and outflow for every node using big.Int to avoid overflow.
	inflow := make([]*big.Int, n)
	outflow := make([]*big.Int, n)
	for i := range n {
		inflow[i] = new(big.Int)
		outflow[i] = new(big.Int)
	}

	for _, a := range arcs {
		if a.Flow == nil {
			continue
		}
		fb := arcFlowBig(a)
		inflow[a.To].Add(inflow[a.To], fb)
		outflow[a.From].Add(outflow[a.From], fb)
	}

	demandBig := demand.ToBig()

	for v := range n {
		diff := new(big.Int)
		switch v {
		case source:
			// outflow - inflow == demand
			diff.Sub(outflow[v], inflow[v])
			if diff.Cmp(demandBig) != 0 {
				t.Errorf("flow conservation at source %d: outflow-inflow = %s, want demand %s",
					v, diff, demandBig)
			}
		case sink:
			// inflow - outflow == demand
			diff.Sub(inflow[v], outflow[v])
			if diff.Cmp(demandBig) != 0 {
				t.Errorf("flow conservation at sink %d: inflow-outflow = %s, want demand %s",
					v, diff, demandBig)
			}
		default:
			if inflow[v].Cmp(outflow[v]) != 0 {
				t.Errorf("flow conservation at node %d: inflow %s != outflow %s",
					v, inflow[v], outflow[v])
			}
		}
	}

	// --- (3) Demand satisfied ---
	if res.TotalFlow == nil {
		t.Errorf("Result.TotalFlow is nil")
	} else if !res.TotalFlow.Eq(demand) {
		t.Errorf("Result.TotalFlow = %s, want demand %s", res.TotalFlow, demand)
	}

	// --- (4) Cost consistency ---
	// Accumulate arc.Cost * arc.Flow using big.Int. If the sum fits in int64,
	// assert it matches res.TotalCost; otherwise skip (best-effort per spec).
	expectedCost := new(big.Int)
	for _, a := range arcs {
		if a.Flow == nil {
			continue
		}
		costBig := big.NewInt(a.Cost)
		term := new(big.Int).Mul(costBig, arcFlowBig(a))
		expectedCost.Add(expectedCost, term)
	}

	if expectedCost.IsInt64() {
		if res.TotalCost != expectedCost.Int64() {
			t.Errorf("Result.TotalCost = %d, want %d (computed from arc costs*flows)",
				res.TotalCost, expectedCost.Int64())
		}
	}
}
