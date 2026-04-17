// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"testing"

	"github.com/holiman/uint256"
)

func setupTreeSolver(n, source, sink int, arcs []Arc) *solver {
	demand := uint256.NewInt(10)
	s := newSolver(arcs, n, source, sink, demand)
	s.initializeTree()
	return s
}

func TestFindJoinSameNode(t *testing.T) {
	s := setupTreeSolver(4, 0, 3, nil)
	if got := s.findJoin(2, 2); got != 2 {
		t.Errorf("findJoin(2,2) = %d, want 2", got)
	}
}

func TestFindJoinSiblings(t *testing.T) {
	s := setupTreeSolver(4, 0, 3, nil)
	root := s.n
	got := s.findJoin(0, 1)
	if got != root {
		t.Errorf("findJoin(0,1) = %d, want root=%d", got, root)
	}
}

func TestFindJoinAllPairs(t *testing.T) {
	s := setupTreeSolver(5, 0, 4, nil)
	root := s.n
	for i := 0; i < s.n; i++ {
		for j := 0; j < s.n; j++ {
			got := s.findJoin(i, j)
			if i == j {
				if got != i {
					t.Errorf("findJoin(%d,%d) = %d, want %d", i, j, got, i)
				}
			} else {
				if got != root {
					t.Errorf("findJoin(%d,%d) = %d, want root=%d", i, j, got, root)
				}
			}
		}
	}
}

func TestFindJoinAsymmetricDepth(t *testing.T) {
	s := setupTreeSolver(5, 0, 4, nil)

	// Manually build a deeper tree: 0 → 1 → 2, 3, 4 all under root=5
	// Reparent: 1's parent = 0, 2's parent = 1
	// Nodes 3,4 remain children of root
	root := s.n
	s.parent[0] = root
	s.succNum[0] = 3
	s.lastSucc[0] = 2

	s.parent[1] = 0
	s.succNum[1] = 2
	s.lastSucc[1] = 2

	s.parent[2] = 1
	s.succNum[2] = 1
	s.lastSucc[2] = 2

	s.parent[3] = root
	s.succNum[3] = 1
	s.lastSucc[3] = 3

	s.parent[4] = root
	s.succNum[4] = 1
	s.lastSucc[4] = 4

	s.succNum[root] = 6

	if got := s.findJoin(2, 3); got != root {
		t.Errorf("findJoin(2,3) = %d, want root=%d", got, root)
	}
	if got := s.findJoin(2, 1); got != 1 {
		t.Errorf("findJoin(2,1) = %d, want 1", got)
	}
	if got := s.findJoin(2, 0); got != 0 {
		t.Errorf("findJoin(2,0) = %d, want 0", got)
	}
	if got := s.findJoin(1, 0); got != 0 {
		t.Errorf("findJoin(1,0) = %d, want 0", got)
	}
	if got := s.findJoin(3, 4); got != root {
		t.Errorf("findJoin(3,4) = %d, want root=%d", got, root)
	}
}

func TestFindJoinSymmetricDepth(t *testing.T) {
	s := setupTreeSolver(5, 0, 4, nil)

	// Build symmetric tree: root → 0, root → 3
	// 0 → 1, 0 → 2
	// 3 → 4
	root := s.n
	s.parent[0] = root
	s.succNum[0] = 3
	s.parent[1] = 0
	s.succNum[1] = 1
	s.parent[2] = 0
	s.succNum[2] = 1
	s.parent[3] = root
	s.succNum[3] = 2
	s.parent[4] = 3
	s.succNum[4] = 1
	s.succNum[root] = 6

	if got := s.findJoin(1, 2); got != 0 {
		t.Errorf("findJoin(1,2) = %d, want 0", got)
	}
	if got := s.findJoin(1, 4); got != root {
		t.Errorf("findJoin(1,4) = %d, want root=%d", got, root)
	}
	if got := s.findJoin(4, 2); got != root {
		t.Errorf("findJoin(4,2) = %d, want root=%d", got, root)
	}
}

func TestFindJoinNodeWithRoot(t *testing.T) {
	s := setupTreeSolver(3, 0, 2, nil)
	root := s.n
	for i := 0; i < s.n; i++ {
		got := s.findJoin(i, root)
		if got != root {
			t.Errorf("findJoin(%d, root) = %d, want root=%d", i, got, root)
		}
	}
}
