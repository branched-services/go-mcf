// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"math"
	"testing"

	"github.com/holiman/uint256"
)

func makePricingSolver(arcs []Arc, n, source, sink int, pi []int64, states []int) *solver {
	s := newSolver(arcs, n, source, sink, uint256.NewInt(100))
	s.initializeTree()
	copy(s.pi, pi)
	copy(s.state, states)
	return s
}

func TestReducedCost(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100)},
		{From: 1, To: 2, Cost: 20, Capacity: uint256.NewInt(100)},
	}
	pi := make([]int64, 4) // n=3, root=3
	pi[0] = 5
	pi[1] = 3
	pi[2] = 7
	pi[3] = 0

	states := make([]int, len(arcs)+3)

	s := makePricingSolver(arcs, 3, 0, 2, pi, states)

	// rc = cost - pi[from] + pi[to]
	// arc 0: 10 - 5 + 3 = 8
	if got := reducedCost(s, 0); got != 8 {
		t.Errorf("reducedCost(0) = %d, want 8", got)
	}
	// arc 1: 20 - 3 + 7 = 24
	if got := reducedCost(s, 1); got != 24 {
		t.Errorf("reducedCost(1) = %d, want 24", got)
	}
}

func TestSelectEnteringLargestViolation(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100)},
		{From: 0, To: 1, Cost: 5, Capacity: uint256.NewInt(100)},
		{From: 1, To: 0, Cost: 3, Capacity: uint256.NewInt(100)},
	}
	n := 2
	pi := make([]int64, n+1)
	pi[0] = 0
	pi[1] = 20
	pi[2] = 0

	// rc(0) = 10 - 0 + 20 = 30 -> stateLower violation = -30 (not eligible)
	// rc(1) = 5 - 0 + 20 = 25  -> stateLower violation = -25 (not eligible)
	// rc(2) = 3 - 20 + 0 = -17 -> stateLower violation = 17 (eligible!)
	states := make([]int, len(arcs)+n)
	states[0] = stateLower
	states[1] = stateLower
	states[2] = stateLower

	s := makePricingSolver(arcs, n, 0, 1, pi, states)
	s.blockSize = len(arcs) + len(s.artArcs) // one big block
	s.nextBlock = 0

	got := s.selectEntering()
	if got != 2 {
		t.Errorf("selectEntering() = %d, want 2", got)
	}
}

func TestSelectEnteringUpperState(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100)},
		{From: 1, To: 0, Cost: 5, Capacity: uint256.NewInt(100)},
	}
	n := 2
	pi := make([]int64, n+1)
	pi[0] = 0
	pi[1] = 3
	pi[2] = 0

	// rc(0) = 10 - 0 + 3 = 13 -> stateUpper violation = +13 (eligible)
	// rc(1) = 5 - 3 + 0 = 2   -> stateUpper violation = +2 (eligible but smaller)
	states := make([]int, len(arcs)+n)
	states[0] = stateUpper
	states[1] = stateUpper

	s := makePricingSolver(arcs, n, 0, 1, pi, states)
	s.blockSize = len(arcs) + len(s.artArcs)
	s.nextBlock = 0

	got := s.selectEntering()
	if got != 0 {
		t.Errorf("selectEntering() = %d, want 0", got)
	}
}

func TestSelectEnteringOptimality(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100)},
	}
	n := 2
	pi := make([]int64, n+1)
	pi[0] = 0
	pi[1] = 20
	pi[2] = 0

	// rc = 10 - 0 + 20 = 30 -> stateLower violation = -30 (not eligible)
	states := make([]int, len(arcs)+n)
	states[0] = stateLower

	s := makePricingSolver(arcs, n, 0, 1, pi, states)
	s.blockSize = len(arcs) + len(s.artArcs)
	s.nextBlock = 0

	got := s.selectEntering()
	if got != -1 {
		t.Errorf("selectEntering() = %d, want -1", got)
	}
}

func TestSelectEnteringRotation(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 5, Capacity: uint256.NewInt(100)},
		{From: 0, To: 1, Cost: 3, Capacity: uint256.NewInt(100)},
		{From: 1, To: 0, Cost: 2, Capacity: uint256.NewInt(100)},
		{From: 1, To: 0, Cost: 4, Capacity: uint256.NewInt(100)},
	}
	n := 2
	pi := make([]int64, n+1)
	pi[0] = 10
	pi[1] = 0
	pi[2] = 0

	// All stateLower:
	// rc(0) = 5 - 10 + 0 = -5  -> violation = 5 (eligible)
	// rc(1) = 3 - 10 + 0 = -7  -> violation = 7 (eligible)
	// rc(2) = 2 - 0 + 10 = 12  -> violation = -12 (not eligible)
	// rc(3) = 4 - 0 + 10 = 14  -> violation = -14 (not eligible)
	states := make([]int, len(arcs)+n)
	for i := range arcs {
		states[i] = stateLower
	}

	s := makePricingSolver(arcs, n, 0, 1, pi, states)
	s.blockSize = 2 // 2 blocks of real arcs, plus block(s) for art arcs
	s.nextBlock = 0

	got := s.selectEntering()
	// Block 0: arcs 0,1 -> arc 1 has violation 7 (largest)
	if got != 1 {
		t.Errorf("first selectEntering() = %d, want 1", got)
	}

	savedNext := s.nextBlock
	if savedNext != 1 {
		t.Errorf("nextBlock after first call = %d, want 1", savedNext)
	}

	// Call again: starts at block 1 (arcs 2,3 - no eligible), wraps to block 0
	got2 := s.selectEntering()
	if got2 != 1 {
		t.Errorf("second selectEntering() = %d, want 1", got2)
	}
}

func TestSelectEnteringSentinelSkip(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100)},
	}
	n := 2
	pi := make([]int64, n+1)
	pi[0] = sentinelPi
	pi[1] = 0
	pi[2] = 0

	states := make([]int, len(arcs)+n)
	states[0] = stateLower

	s := makePricingSolver(arcs, n, 0, 1, pi, states)
	s.blockSize = len(arcs) + len(s.artArcs)
	s.nextBlock = 0

	got := s.selectEntering()
	if got != -1 {
		t.Errorf("selectEntering() = %d, want -1 (sentinel should skip)", got)
	}
}

func TestSelectEnteringSentinelNegative(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100)},
	}
	n := 2
	pi := make([]int64, n+1)
	pi[0] = 0
	pi[1] = -sentinelPi
	pi[2] = 0

	states := make([]int, len(arcs)+n)
	states[0] = stateLower

	s := makePricingSolver(arcs, n, 0, 1, pi, states)
	s.blockSize = len(arcs) + len(s.artArcs)
	s.nextBlock = 0

	got := s.selectEntering()
	if got != -1 {
		t.Errorf("selectEntering() = %d, want -1 (negative sentinel should skip)", got)
	}
}

func TestSelectEnteringTreeStateIneligible(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100)},
	}
	n := 2
	pi := make([]int64, n+1)
	pi[0] = 20
	pi[1] = 0
	pi[2] = 0

	// rc = 10 - 20 + 0 = -10, would have violation 10 if stateLower
	states := make([]int, len(arcs)+n)
	states[0] = stateTree

	s := makePricingSolver(arcs, n, 0, 1, pi, states)
	s.blockSize = len(arcs) + len(s.artArcs)
	s.nextBlock = 0

	got := s.selectEntering()
	if got != -1 {
		t.Errorf("selectEntering() = %d, want -1 (tree arcs ineligible)", got)
	}
}

func TestBlockSizeCalculation(t *testing.T) {
	arcs := make([]Arc, 100)
	for i := range arcs {
		arcs[i] = Arc{From: 0, To: 1, Cost: 1, Capacity: uint256.NewInt(1)}
	}
	n := 2
	s := newSolver(arcs, n, 0, 1, uint256.NewInt(1))
	totalArcs := len(arcs) + n
	want := int(math.Ceil(math.Sqrt(float64(totalArcs))))
	if s.blockSize != want {
		t.Errorf("blockSize = %d, want %d (ceil(sqrt(%d)))", s.blockSize, want, totalArcs)
	}
	if s.nextBlock != 0 {
		t.Errorf("nextBlock = %d, want 0", s.nextBlock)
	}
}
