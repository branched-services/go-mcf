// SPDX-License-Identifier: BSL-1.0

package mcf

import "testing"

func TestReducedCostSignConvention(t *testing.T) {
	// 3 nodes (0, 1, 2), 2 real arcs.
	s := newSolverState(3, 2)

	// Arc 0: 0 -> 1, cost 10.
	s.source[0] = 0
	s.target[0] = 1
	s.cost[0] = 10

	// Arc 1: 1 -> 2, cost 20.
	s.source[1] = 1
	s.target[1] = 2
	s.cost[1] = 20

	// Node potentials: pi[0]=3, pi[1]=7, pi[2]=1.
	s.pi[0] = 3
	s.pi[1] = 7
	s.pi[2] = 1

	// rc(0) = cost[0] + pi[source[0]] - pi[target[0]] = 10 + 3 - 7 = 6
	if rc := s.reducedCost(0); rc != 6 {
		t.Fatalf("reducedCost(0) = %d, want 6", rc)
	}

	// rc(1) = cost[1] + pi[source[1]] - pi[target[1]] = 20 + 7 - 1 = 26
	if rc := s.reducedCost(1); rc != 26 {
		t.Fatalf("reducedCost(1) = %d, want 26", rc)
	}
}

func TestViolationStateSwitch(t *testing.T) {
	// 3 nodes, 3 real arcs. We only care about state and reduced cost.
	s := newSolverState(3, 3)

	// Set up arcs so that reducedCost returns known values.
	// We want rc = -5, +5, -5 for arcs 0, 1, 2 respectively.
	// Use cost directly with zero potentials for simplicity.
	s.cost[0] = -5 // rc = -5
	s.cost[1] = 5  // rc = +5
	s.cost[2] = -5 // rc = -5

	s.state[0] = stateLower // violation = -(-5) = 5  (eligible)
	s.state[1] = stateLower // violation = -(+5) = -5 (not eligible)
	s.state[2] = stateUpper // violation = +(-5) = -5 (not eligible)

	cases := []struct {
		arc  int
		want int64
	}{
		{0, 5},
		{1, -5},
		{2, -5},
	}

	for _, tc := range cases {
		if got := s.violation(tc.arc); got != tc.want {
			t.Errorf("violation(%d) = %d, want %d", tc.arc, got, tc.want)
		}
	}

	// Only arc 0 is pricing-eligible (violation > 0).
	if v := s.violation(0); v <= 0 {
		t.Errorf("arc 0 should be pricing-eligible, violation = %d", v)
	}
	for _, arc := range []int{1, 2} {
		if v := s.violation(arc); v > 0 {
			t.Errorf("arc %d should not be pricing-eligible, violation = %d", arc, v)
		}
	}
}

func TestViolationTreeArcIsZero(t *testing.T) {
	s := newSolverState(2, 1)

	// Give the arc a non-zero reduced cost.
	s.cost[0] = 42
	s.pi[0] = 10
	s.pi[1] = 5
	// rc = 42 + 10 - 5 = 47, definitely non-zero.

	s.state[0] = stateTree

	if v := s.violation(0); v != 0 {
		t.Fatalf("violation of stateTree arc = %d, want 0", v)
	}
}
