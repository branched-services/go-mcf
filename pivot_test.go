// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"testing"

	"github.com/holiman/uint256"
)

// TestPushFlowInPlace verifies that pushFlow correctly distributes the
// bottleneck around a 3-arc cycle using in-place uint256 arithmetic.
//
// Triangle: nodes 0, 1, 2; join = 2.
//
//	arc 0: 0->2 (tree, predArc of node 0, dirUp)
//	arc 1: 1->2 (tree, predArc of node 1, dirUp)
//	arc 2: 0->1 (entering, stateLower)
//
// Cycle when entering arc 2 (0->1) at stateLower:
//   - Entering arc 2: forward (0->1), Add bottleneck  → flow = 0+4 = 4
//   - First walk: node 0, arc 0 (dirUp), Add           → flow = 0+4 = 4
//   - Second walk: node 1, arc 1 (dirUp), Sub           → flow = 10-4 = 6
//
// Second-walk arc 1 starts at flow=10 so Sub yields 6 (not negative).
func TestPushFlowInPlace(t *testing.T) {
	s := buildTriangleSolver(
		20, 0, // arc 0: cap=20, flow=0
		20, 10, // arc 1: cap=20, flow=10
		20, 0, // arc 2 (entering): cap=20, flow=0
		stateLower,
	)

	bn := uint256.NewInt(4)
	s.pushFlow(2, 2, bn)

	if got := s.flow[0].Uint64(); got != 4 {
		t.Fatalf("flow[0] = %d, want 4", got)
	}
	if got := s.flow[1].Uint64(); got != 6 {
		t.Fatalf("flow[1] = %d, want 6", got)
	}
	if got := s.flow[2].Uint64(); got != 4 {
		t.Fatalf("flow[2] = %d, want 4", got)
	}
}

// TestPushFlowZeroBottleneckIsNoop verifies that a degenerate pivot with
// bottleneck=0 leaves all flows unchanged.
func TestPushFlowZeroBottleneckIsNoop(t *testing.T) {
	s := buildTriangleSolver(
		20, 5, // arc 0: cap=20, flow=5
		20, 7, // arc 1: cap=20, flow=7
		20, 3, // arc 2: cap=20, flow=3
		stateLower,
	)

	bn := uint256.NewInt(0)
	s.pushFlow(2, 2, bn)

	if got := s.flow[0].Uint64(); got != 5 {
		t.Fatalf("flow[0] = %d, want 5 (unchanged)", got)
	}
	if got := s.flow[1].Uint64(); got != 7 {
		t.Fatalf("flow[1] = %d, want 7 (unchanged)", got)
	}
	if got := s.flow[2].Uint64(); got != 3 {
		t.Fatalf("flow[2] = %d, want 3 (unchanged)", got)
	}
}

// TestPushFlowLargeUint256 verifies correctness with a bottleneck exceeding
// the uint64 range (2^100).
func TestPushFlowLargeUint256(t *testing.T) {
	// bottleneck = 2^100
	bn := new(uint256.Int).Lsh(uint256.NewInt(1), 100)

	// Starting flows: all zero for first-walk and entering arcs,
	// arc 1 starts at 2^101 so Sub(2^100) yields 2^100.
	startFlow1 := new(uint256.Int).Lsh(uint256.NewInt(1), 101) // 2^101
	bigCap := new(uint256.Int).Lsh(uint256.NewInt(1), 200)     // large capacity

	s := &solverState{
		n:     3,
		mReal: 3,
		m:     3,
		root:  3,

		source:  []int{0, 1, 0},
		target:  []int{2, 2, 1},
		capacity: []*uint256.Int{
			new(uint256.Int).Set(bigCap),
			new(uint256.Int).Set(bigCap),
			new(uint256.Int).Set(bigCap),
		},
		flow: []*uint256.Int{
			uint256.NewInt(0),            // arc 0
			new(uint256.Int).Set(startFlow1), // arc 1: 2^101
			uint256.NewInt(0),            // arc 2
		},
		state: []int8{stateTree, stateTree, stateLower},

		parent:   []int{2, 2, 3, -1},
		predArc:  []int{0, 1, -1, -1},
		succNum:  []int{1, 1, 3, 4},
		lastSucc: []int{0, 1, 1, 2},
		dirNode:  []int8{dirUp, dirUp, dirUp, 0},

		bottleneck: new(uint256.Int),
		delta:      new(uint256.Int),
	}

	s.pushFlow(2, 2, bn)

	// arc 0: 0 + 2^100 = 2^100
	if s.flow[0].Cmp(bn) != 0 {
		t.Fatalf("flow[0] = %s, want 2^100", s.flow[0].ToBig().String())
	}

	// arc 1: 2^101 - 2^100 = 2^100
	if s.flow[1].Cmp(bn) != 0 {
		t.Fatalf("flow[1] = %s, want 2^100", s.flow[1].ToBig().String())
	}

	// arc 2 (entering): 0 + 2^100 = 2^100
	if s.flow[2].Cmp(bn) != 0 {
		t.Fatalf("flow[2] = %s, want 2^100", s.flow[2].ToBig().String())
	}
}
