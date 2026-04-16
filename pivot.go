// SPDX-License-Identifier: BSL-1.0

package mcf

import "github.com/holiman/uint256"

// pushFlow pushes bottleneck units of flow around the cycle induced by the
// entering arc. The cycle is defined by walking from source[entering] and
// target[entering] up to joinNode via the spanning-tree parent pointers.
//
// Flow direction along the entering arc is determined by its state:
//   - stateLower: forward (source -> target), flow increases
//   - stateUpper: backward (target -> source), flow decreases
//
// If bottleneck is zero (degenerate pivot), the walks still execute to
// maintain structural correctness, but no Add/Sub calls are issued. The
// state flip for the entering/leaving arcs is handled by the tree-update
// step, not here.
//
// All mutations are in-place on pre-allocated *uint256.Int entries in
// s.flow — no heap allocations occur.
//
// Invariant (assert, not runtime-checked): after the push, every arc on
// the cycle satisfies 0 <= flow[a] <= capacity[a].
func (s *solverState) pushFlow(entering, joinNode int, bottleneck *uint256.Int) {
	zero := bottleneck.IsZero()

	// ── First walk: source side (u) up to joinNode ──────────────────
	// Sign convention mirrors findLeavingArc's first walk:
	//   dirUp  → cycle flow aligns with arc direction → Add
	//   dirDown → cycle flow opposes arc direction   → Sub
	for cur := s.source[entering]; cur != joinNode; cur = s.parent[cur] {
		a := s.predArc[cur]
		if !zero {
			if s.dirNode[cur] == dirUp {
				s.flow[a].Add(s.flow[a], bottleneck)
			} else {
				s.flow[a].Sub(s.flow[a], bottleneck)
			}
		}
	}

	// ── Second walk: target side (v) up to joinNode ─────────────────
	// Sign convention flips relative to the first walk:
	//   dirUp  → cycle flow opposes arc direction → Sub
	//   dirDown → cycle flow aligns with arc direction → Add
	for cur := s.target[entering]; cur != joinNode; cur = s.parent[cur] {
		a := s.predArc[cur]
		if !zero {
			if s.dirNode[cur] == dirUp {
				s.flow[a].Sub(s.flow[a], bottleneck)
			} else {
				s.flow[a].Add(s.flow[a], bottleneck)
			}
		}
	}

	// ── Entering arc ────────────────────────────────────────────────
	if !zero {
		if s.state[entering] == stateLower {
			s.flow[entering].Add(s.flow[entering], bottleneck)
		} else {
			s.flow[entering].Sub(s.flow[entering], bottleneck)
		}
	}
}
