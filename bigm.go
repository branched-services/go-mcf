// SPDX-License-Identifier: BSL-1.0
package mcf

import "math"

// bigM returns the cost assigned to artificial arcs. The (n+1) factor bounds
// potential accumulation along the longest tree path; the /8 factor leaves
// headroom for reduced-cost subtraction. The result strictly dominates any
// real arc's reduced cost unless the problem is infeasible.
//
// Division is ordered as MaxInt64/8/int64(n+1) to avoid overflowing the
// intermediate 8*int64(n+1) product for very large n.
func bigM(n int) int64 {
	return math.MaxInt64 / 8 / int64(n+1)
}

// arcCostWithinBound reports whether cost satisfies the overflow-safety
// precondition: |cost|*(n+1) < MaxInt64/8. This is precisely the condition
// that guarantees reduced-cost arithmetic cannot overflow int64.
//
// The check is written to avoid overflow in the bound computation itself:
// instead of multiplying absCost*int64(n+1) (which may overflow), we
// compare absCost against the precomputed limit MaxInt64/8/int64(n+1).
func arcCostWithinBound(cost int64, n int) bool {
	// math.MinInt64 has no positive counterpart in int64; negation wraps
	// around to itself, so reject it explicitly.
	if cost == math.MinInt64 {
		return false
	}
	absCost := cost
	if absCost < 0 {
		absCost = -absCost
	}
	limit := (math.MaxInt64 / 8) / int64(n+1)
	return absCost <= limit
}
