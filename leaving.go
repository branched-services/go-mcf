// SPDX-License-Identifier: BSL-1.0

package mcf

import "github.com/holiman/uint256"

// findLeavingArc walks the cycle induced by the entering arc to find the
// bottleneck (minimum-residual) arc that must leave the basis. It returns:
//   - leaving:       index of the arc that leaves the basis
//   - bottleneck:    pointer to s.bottleneck (caller treats as read-only)
//   - deltaDirFirst: true if the bottleneck was found on the source-side walk
//
// The algorithm follows LEMON 1.3.1's findLeavingArc with the strongly-feasible
// tree tie-breaking rule.
func (s *solverState) findLeavingArc(entering int) (leaving int, bottleneck *uint256.Int, deltaDirFirst bool) {
	u := s.source[entering]
	v := s.target[entering]
	j := s.findJoinNode(u, v)

	// scratch is a stack-local uint256 used to compute residuals without
	// allocating. We reuse it across both walks.
	var scratch uint256.Int

	// ── Initialize with the entering arc's residual ─────────────────────
	if s.state[entering] == stateLower {
		// Pushing forward (u->v): residual = capacity - flow.
		s.bottleneck.Sub(s.capacity[entering], s.flow[entering])
	} else {
		// stateUpper, pushing backward (v->u): residual = flow.
		s.bottleneck.Set(s.flow[entering])
	}
	leaving = entering
	deltaDirFirst = true

	// ── First walk: source (u) up to join node (j) ──────────────────────
	// Residual in the direction of cycle flow:
	//   dirUp  (arc aligned with cycle): capacity - flow
	//   dirDown (arc opposes cycle):     flow
	for cur := u; cur != j; cur = s.parent[cur] {
		a := s.predArc[cur]
		if s.dirNode[cur] == dirUp {
			scratch.Sub(s.capacity[a], s.flow[a])
		} else {
			scratch.Set(s.flow[a])
		}
		if scratch.Lt(s.bottleneck) {
			s.bottleneck.Set(&scratch)
			leaving = a
			deltaDirFirst = true
		}
	}

	// ── Second walk: target (v) up to join node (j) ─────────────────────
	// Sign convention flips on this walk:
	//   dirUp:   flow
	//   dirDown: capacity - flow
	//
	// ┌─────────────────────────────────────────────────────────────────┐
	// │ STRONGLY-FEASIBLE TIE-BREAK — DO NOT REMOVE                   │
	// │                                                                │
	// │ When the second-walk residual EQUALS the current bottleneck,   │
	// │ we REPLACE the running choice with the second-walk arc. This   │
	// │ is the strongly-feasible-tree tie-breaking rule from LEMON.    │
	// │                                                                │
	// │ This tie-break is NOT optional. It is the termination proof    │
	// │ for degenerate pivots in the Network Simplex algorithm.        │
	// │ Bland's rule alone is insufficient to guarantee finite         │
	// │ termination under degeneracy; the strongly-feasible spanning   │
	// │ tree property is required. Future maintainers must not delete  │
	// │ this as 'redundant' — removing it risks infinite cycling on    │
	// │ degenerate instances.                                          │
	// └─────────────────────────────────────────────────────────────────┘
	for cur := v; cur != j; cur = s.parent[cur] {
		a := s.predArc[cur]
		if s.dirNode[cur] == dirUp {
			scratch.Set(s.flow[a])
		} else {
			scratch.Sub(s.capacity[a], s.flow[a])
		}
		// <= : ties go to the second walk (strongly-feasible tie-break).
		if scratch.Lt(s.bottleneck) || scratch.Eq(s.bottleneck) {
			s.bottleneck.Set(&scratch)
			leaving = a
			deltaDirFirst = false
		}
	}

	return leaving, s.bottleneck, deltaDirFirst
}
