// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"math"

	"github.com/holiman/uint256"
)

// buildInitialState constructs the initial Big-M artificial-root spanning tree
// used by the Network Simplex algorithm. All real arcs start at their lower
// bound (flow 0); each real node gets one artificial arc connecting it to the
// root with cost bigM(n). The initial basis, potentials, and thread linkage
// mirror LEMON 1.3.1's NetworkSimplex::init() exactly.
//
// Inputs are assumed to be already validated by validateSolveInputs.
func buildInitialState(arcs []Arc, n, source, sink int, demand *uint256.Int) *solverState {
	mReal := len(arcs)
	s := newSolverState(n, mReal)

	// ── Real arcs [0, mReal) ─────────────────────────────────────────────
	for i, a := range arcs {
		s.source[i] = a.From
		s.target[i] = a.To
		s.cost[i] = a.Cost
		s.capacity[i] = a.Capacity // alias, not copy
		s.state[i] = stateLower    // all real arcs start at lower bound (flow 0)
	}

	// ── Supply vector ────────────────────────────────────────────────────
	// supply drives artificial-arc orientation: +1 at source, -1 at sink.
	// The magnitude of flow on artificial arcs uses demand (uint256).
	supply := make([]int64, n+1)
	supply[source] = +1
	supply[sink] = -1

	// ── Artificial arcs [mReal, m) ───────────────────────────────────────
	M := bigM(n)
	for v := range n {
		artIdx := mReal + v
		if supply[v] >= 0 {
			s.source[artIdx] = v
			s.target[artIdx] = s.root
			s.dirNode[v] = dirUp
		} else {
			s.source[artIdx] = s.root
			s.target[artIdx] = v
			s.dirNode[v] = dirDown
		}
		s.cost[artIdx] = M
		s.state[artIdx] = stateTree
		s.capacity[artIdx] = new(uint256.Int).Set(demand)

		// Flow on artificial arcs: demand for source/sink nodes (|supply|==1),
		// zero for the rest (already zero from newSolverState).
		if supply[v] != 0 {
			s.flow[artIdx].Set(demand)
		}
	}

	// ── Initial basis linkage (mirrors LEMON's init() exactly) ───────────
	// Every real node is a direct child of the artificial root.
	for v := range n {
		s.parent[v] = s.root
		s.predArc[v] = mReal + v
		s.succNum[v] = 1
		s.lastSucc[v] = v
	}
	s.parent[s.root] = -1
	s.predArc[s.root] = -1
	s.succNum[s.root] = n + 1
	s.lastSucc[s.root] = n - 1

	// Thread: linear traversal root -> 0 -> 1 -> ... -> n-1 -> root.
	s.thread[s.root] = 0
	for i := 0; i < n-1; i++ {
		s.thread[i] = i + 1
	}
	if n > 0 {
		s.thread[n-1] = s.root
	}
	// revThread: inverse of thread.
	for i := 0; i <= n; i++ {
		s.revThread[s.thread[i]] = i
	}

	// ── Initial potentials ───────────────────────────────────────────────
	// pi[root] = 0; for each real node v, set pi so that every artificial
	// arc's reduced cost is zero (they are in the basis).
	s.pi[s.root] = 0
	for v := range n {
		if s.dirNode[v] == dirUp {
			s.pi[v] = -M
		} else {
			s.pi[v] = M
		}
	}

	// ── Block-search pricing state ───────────────────────────────────────
	s.nextArc = 0
	if mReal == 0 {
		s.blockSize = 1
	} else {
		s.blockSize = max(1, int(math.Ceil(math.Sqrt(float64(mReal)))))
	}

	return s
}
