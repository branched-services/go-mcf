// SPDX-License-Identifier: BSL-1.0

package mcf

import "testing"

// newPricingTestState builds a minimal solverState for pricing tests.
// m=8, blockSize=3, all arcs stateLower.
// Reduced costs: [+5, -2, +1, -7, +3, -4, 0, -1].
// Violations (−rc for stateLower): [-5, +2, -1, +7, -3, +4, 0, +1].
func newPricingTestState() *solverState {
	const m = 8
	// With pi=0 and source==target==0, rc == cost.
	costs := []int64{5, -2, 1, -7, 3, -4, 0, -1}
	states := make([]int8, m)
	src := make([]int, m)
	tgt := make([]int, m)
	pi := []int64{0} // single node, index 0
	for i := range states {
		states[i] = stateLower
	}
	return &solverState{
		m:         m,
		blockSize: 3,
		nextArc:   0,
		cost:      costs,
		state:     states,
		source:    src,
		target:    tgt,
		pi:        pi,
	}
}

func TestFindEnteringArcSelectsMaxViolationInBlock(t *testing.T) {
	s := newPricingTestState()
	// Scan arcs 0,1,2. Violations: -5, +2, -1. Best eligible: arc 1 (v=2).
	arc, ok := s.findEnteringArc()
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if arc != 1 {
		t.Fatalf("expected arc 1, got %d", arc)
	}
	if s.nextArc != 3 {
		t.Fatalf("expected nextArc=3, got %d", s.nextArc)
	}
}

func TestFindEnteringArcRotates(t *testing.T) {
	s := newPricingTestState()
	s.nextArc = 3
	// Scan arcs 3,4,5. Violations: +7, -3, +4. Best eligible: arc 3 (v=7).
	arc, ok := s.findEnteringArc()
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if arc != 3 {
		t.Fatalf("expected arc 3, got %d", arc)
	}
	if s.nextArc != 6 {
		t.Fatalf("expected nextArc=6, got %d", s.nextArc)
	}
}

func TestFindEnteringArcReturnsFalseWhenOptimal(t *testing.T) {
	s := newPricingTestState()
	for i := range s.state {
		s.state[i] = stateTree
	}
	arc, ok := s.findEnteringArc()
	if ok {
		t.Fatalf("expected ok=false, got arc=%d", arc)
	}
}
