// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"fmt"
	"strings"
	"testing"
)

func TestCheckSolution_ValidSolution(t *testing.T) {
	// Diamond: 0->1 (cap 10, cost 1), 0->2 (cap 10, cost 5),
	//          1->3 (cap 10, cost 1), 2->3 (cap 10, cost 1)
	// Optimal for demand=10: all flow through 0->1->3, cost=20.
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(10), Flow: u256(10)},
		{From: 0, To: 2, Cost: 5, Capacity: u256(10), Flow: u256(0)},
		{From: 1, To: 3, Cost: 1, Capacity: u256(10), Flow: u256(10)},
		{From: 2, To: 3, Cost: 1, Capacity: u256(10), Flow: u256(0)},
	}
	// Potentials consistent with optimality:
	// pi[0]=0, pi[1]=-1, pi[2]=0, pi[3]=-2
	// Arc 0 (0->1): rc = 1 - 0 + (-1) = 0 (tree)
	// Arc 1 (0->2): rc = 5 - 0 + 0 = 5 >= 0 (lower, ok)
	// Arc 2 (1->3): rc = 1 - (-1) + (-2) = 0 (tree)
	// Arc 3 (2->3): rc = 1 - 0 + (-2) = -1 >= 0? No.
	// Adjust: pi[2] = -4 => arc 1: rc = 5 - 0 + (-4) = 1 >= 0 (lower, ok)
	//                        arc 3: rc = 1 - (-4) + (-2) = 3 >= 0 (lower, ok)
	pi := []int64{0, -1, -4, -2}
	state := []int{stateTree, stateLower, stateTree, stateLower}
	snap := buildSnapshot(4, pi, state)
	result := Result{TotalFlow: u256(10), TotalCost: 20}

	checkSolution(t, arcs, 4, 0, 3, u256(10), result, snap)
}

func TestCheckSolution_FlowConservationViolated(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(10), Flow: u256(10)},
		{From: 1, To: 2, Cost: 1, Capacity: u256(10), Flow: u256(5)},
	}
	pi := []int64{0, -1, -2}
	state := []int{stateTree, stateTree}
	snap := buildSnapshot(3, pi, state)
	result := Result{TotalFlow: u256(10), TotalCost: 15}

	ft := &fakeT{}
	checkSolution(ft, arcs, 3, 0, 2, u256(10), result, snap)

	if !ft.hasError("flow conservation violated") {
		t.Errorf("expected flow conservation violation, got errors: %v", ft.errors)
	}
}

func TestCheckSolution_CapacityBoundViolated(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(5), Flow: u256(10)},
	}
	pi := []int64{0, -1}
	state := []int{stateTree}
	snap := buildSnapshot(2, pi, state)
	result := Result{TotalFlow: u256(10), TotalCost: 10}

	ft := &fakeT{}
	checkSolution(ft, arcs, 2, 0, 1, u256(10), result, snap)

	if !ft.hasError("capacity bound violated") {
		t.Errorf("expected capacity bound violation, got errors: %v", ft.errors)
	}
}

func TestCheckSolution_OptimalityCertificateViolated(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(10), Flow: u256(10)},
		{From: 0, To: 1, Cost: 1, Capacity: u256(10), Flow: u256(0)},
	}
	// Arc 1 at stateLower with negative reduced cost violates optimality.
	// pi[0]=0, pi[1]=0 => rc = 1 - 0 + 0 = 1 >= 0, ok.
	// Need negative rc for lower: pi[0]=10, pi[1]=0 => rc = 1 - 10 + 0 = -9 < 0, violation.
	pi := []int64{10, 0}
	state := []int{stateTree, stateLower}
	snap := buildSnapshot(2, pi, state)
	result := Result{TotalFlow: u256(10), TotalCost: 10}

	ft := &fakeT{}
	checkSolution(ft, arcs, 2, 0, 1, u256(10), result, snap)

	if !ft.hasError("optimality certificate violated") {
		t.Errorf("expected optimality certificate violation, got errors: %v", ft.errors)
	}
}

func TestCheckSolution_DemandNotSatisfied(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: u256(10), Flow: u256(5)},
	}
	pi := []int64{0, -1}
	state := []int{stateTree}
	snap := buildSnapshot(2, pi, state)
	result := Result{TotalFlow: u256(5), TotalCost: 5}

	ft := &fakeT{}
	checkSolution(ft, arcs, 2, 0, 1, u256(10), result, snap)

	if !ft.hasError("TotalFlow != demand") {
		t.Errorf("expected demand violation, got errors: %v", ft.errors)
	}
}

func TestCheckSolution_TotalCostInconsistent(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 2, Capacity: u256(10), Flow: u256(10)},
	}
	pi := []int64{0, -2}
	state := []int{stateTree}
	snap := buildSnapshot(2, pi, state)
	// Correct cost would be 20, but we say 99.
	result := Result{TotalFlow: u256(10), TotalCost: 99}

	ft := &fakeT{}
	checkSolution(ft, arcs, 2, 0, 1, u256(10), result, snap)

	if !ft.hasError("TotalCost inconsistent") {
		t.Errorf("expected TotalCost inconsistency, got errors: %v", ft.errors)
	}
}

type fakeT struct {
	errors []string
}

func (f *fakeT) Helper() {}

func (f *fakeT) Errorf(format string, args ...interface{}) {
	f.errors = append(f.errors, fmt.Sprintf(format, args...))
}

func (f *fakeT) hasError(keyword string) bool {
	for _, e := range f.errors {
		if strings.Contains(e, keyword) {
			return true
		}
	}
	return false
}
