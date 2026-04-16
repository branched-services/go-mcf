// SPDX-License-Identifier: BSL-1.0

package mcf

// reducedCost returns the reduced cost of arc in LEMON's sign convention:
//
//	rc = c_ij + pi_i - pi_j
//
// This equals c_ij - (pi_j - pi_i). Future maintainers must not flip the
// signs: the convention is load-bearing for the violation helper and the
// pricing loop. O(1), no allocation.
func (s *solverState) reducedCost(arc int) int64 {
	return s.cost[arc] + s.pi[s.source[arc]] - s.pi[s.target[arc]]
}

// violation returns the pricing-eligibility measure for arc.
//
// An arc is pricing-eligible iff violation > 0:
//   - Lower-bounded arcs (stateLower) enter the basis when rc < 0, because
//     pushing flow along a negative reduced-cost arc reduces total cost.
//     violation = -rc, so violation > 0 exactly when rc < 0.
//   - Upper-bounded arcs (stateUpper) enter the basis when rc > 0, because
//     reducing flow on a positive reduced-cost arc reduces total cost.
//     violation = +rc, so violation > 0 exactly when rc > 0.
//   - Tree arcs (stateTree) are already in the basis; violation = 0.
//
// Removing the sign flip would make saturated arcs permanently stuck.
// O(1), no allocation.
func (s *solverState) violation(arc int) int64 {
	switch s.state[arc] {
	case stateLower:
		return -s.reducedCost(arc)
	case stateUpper:
		return s.reducedCost(arc)
	default: // stateTree
		return 0
	}
}
