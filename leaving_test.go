// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"testing"

	"github.com/holiman/uint256"
)

// buildTriangleSolver constructs a minimal solverState for a 3-node triangle
// (nodes 0, 1, 2) with root=3 and a specific spanning tree, used by leaving-arc
// tests.
//
// Tree structure:
//
//	    2 (join)
//	   / \
//	  0   1
//
// Tree arcs (in basis):
//
//	arc 0: 0->2  (predArc of node 0, dirUp)
//	arc 1: 1->2  (predArc of node 1, dirUp)
//
// Entering arc:
//
//	arc 2: 0->1  (non-tree arc, state configurable)
//
// The cycle induced by entering arc 2 (0->1) is: 0->1 (entering), 1->2 (tree),
// 2->0 (tree reversed). Join node = 2.
//
// First walk: from u=0 up to j=2, traverses arc 0 (node 0, dirUp).
// Second walk: from v=1 up to j=2, traverses arc 1 (node 1, dirUp).
func buildTriangleSolver(capArc0, flowArc0, capArc1, flowArc1, capEntering, flowEntering uint64, enteringState int8) *solverState {
	s := &solverState{
		n:     3,
		mReal: 3,
		m:     3,
		root:  3,

		source:  []int{0, 1, 0},
		target:  []int{2, 2, 1},
		capacity: []*uint256.Int{
			uint256.NewInt(capArc0),
			uint256.NewInt(capArc1),
			uint256.NewInt(capEntering),
		},
		flow: []*uint256.Int{
			uint256.NewInt(flowArc0),
			uint256.NewInt(flowArc1),
			uint256.NewInt(flowEntering),
		},
		state: []int8{stateTree, stateTree, enteringState},

		parent:   []int{2, 2, 3, -1},
		predArc:  []int{0, 1, -1, -1},
		succNum:  []int{1, 1, 3, 4},
		lastSucc: []int{0, 1, 1, 2},
		dirNode:  []int8{dirUp, dirUp, dirUp, 0},

		bottleneck: new(uint256.Int),
		delta:      new(uint256.Int),
	}
	return s
}

// TestLeavingArcPicksMinResidual verifies that findLeavingArc selects the arc
// with the smallest residual capacity in the cycle direction.
//
// Setup: entering arc 2 (0->1, stateLower) with residual 10 (cap=10, flow=0).
// First-walk arc 0 (node 0, dirUp): residual = cap-flow = 10-6 = 4.
// Second-walk arc 1 (node 1, dirUp): residual = flow = 6.
// Minimum is 4 on arc 0 (first walk).
func TestLeavingArcPicksMinResidual(t *testing.T) {
	s := buildTriangleSolver(
		10, 6, // arc 0: cap=10, flow=6 -> first-walk residual (dirUp) = cap-flow = 4
		10, 6, // arc 1: cap=10, flow=6 -> second-walk residual (dirUp) = flow = 6
		10, 0, // entering: cap=10, flow=0 -> residual = cap-flow = 10
		stateLower,
	)

	leaving, bn, deltaDirFirst := s.findLeavingArc(2)

	if leaving != 0 {
		t.Fatalf("leaving = %d, want 0 (arc with residual 4)", leaving)
	}
	if bn.Uint64() != 4 {
		t.Fatalf("bottleneck = %d, want 4", bn.Uint64())
	}
	if !deltaDirFirst {
		t.Fatalf("deltaDirFirst = false, want true (first walk)")
	}
}

// TestLeavingArcTieBreakPicksSecondWalk verifies the strongly-feasible-tree
// tie-breaking: when all residuals are equal, the second-walk arc wins.
//
// Setup: all residuals equal 5.
// Entering arc 2 (stateLower): cap=5, flow=0 -> residual=5.
// First-walk arc 0 (dirUp): cap=10, flow=5 -> residual = cap-flow = 5.
// Second-walk arc 1 (dirUp): cap=10, flow=5 -> residual = flow = 5.
// Tie-break: second walk wins.
func TestLeavingArcTieBreakPicksSecondWalk(t *testing.T) {
	s := buildTriangleSolver(
		10, 5, // arc 0: cap=10, flow=5 -> first-walk residual (dirUp) = cap-flow = 5
		10, 5, // arc 1: cap=10, flow=5 -> second-walk residual (dirUp) = flow = 5
		5, 0, // entering: cap=5, flow=0 -> residual = 5
		stateLower,
	)

	leaving, bn, deltaDirFirst := s.findLeavingArc(2)

	if leaving != 1 {
		t.Fatalf("leaving = %d, want 1 (second-walk arc wins tie)", leaving)
	}
	if bn.Uint64() != 5 {
		t.Fatalf("bottleneck = %d, want 5", bn.Uint64())
	}
	if deltaDirFirst {
		t.Fatalf("deltaDirFirst = true, want false (second walk wins tie)")
	}
}

// TestLeavingArcEnteringItselfWinsWhenMin verifies that when the entering arc
// has the strictly smallest residual, it is selected as the leaving arc.
//
// Setup: entering arc 2 (stateLower): cap=3, flow=0 -> residual=3.
// First-walk arc 0 (dirUp): cap=20, flow=10 -> residual = cap-flow = 10.
// Second-walk arc 1 (dirUp): cap=20, flow=10 -> residual = flow = 10.
// Entering arc wins with residual 3.
func TestLeavingArcEnteringItselfWinsWhenMin(t *testing.T) {
	s := buildTriangleSolver(
		20, 10, // arc 0: residual = cap-flow = 10
		20, 10, // arc 1: residual = flow = 10
		3, 0,   // entering: cap=3, flow=0 -> residual = 3
		stateLower,
	)

	leaving, bn, deltaDirFirst := s.findLeavingArc(2)

	if leaving != 2 {
		t.Fatalf("leaving = %d, want 2 (entering arc)", leaving)
	}
	if bn.Uint64() != 3 {
		t.Fatalf("bottleneck = %d, want 3", bn.Uint64())
	}
	if !deltaDirFirst {
		t.Fatalf("deltaDirFirst = false, want true (entering arc initialized as first)")
	}
}
