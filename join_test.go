// SPDX-License-Identifier: BSL-1.0

package mcf

import "testing"

// Tree used by TestFindJoinNodeBasic:
//
//	        4 (root)
//	       / \
//	      2   3
//	     / \
//	    0   1
//
// parent:  [2, 2, 4, 4, 4]  (root is its own parent)
// succNum: [1, 1, 3, 1, 5]  (subtree sizes)
func TestFindJoinNodeBasic(t *testing.T) {
	s := &solverState{
		parent:  []int{2, 2, 4, 4, 4},
		succNum: []int{1, 1, 3, 1, 5},
	}

	tests := []struct {
		u, v, want int
	}{
		{0, 1, 2},
		{0, 3, 4},
		{2, 3, 4},
		{1, 1, 1}, // degenerate: same node
	}

	for _, tt := range tests {
		got := s.findJoinNode(tt.u, tt.v)
		if got != tt.want {
			t.Errorf("findJoinNode(%d, %d) = %d, want %d", tt.u, tt.v, got, tt.want)
		}
	}
}

func TestFindJoinNodeRootItself(t *testing.T) {
	// Single-node tree: root is 0, both u and v are root.
	s := &solverState{
		parent:  []int{0},
		succNum: []int{1},
	}

	got := s.findJoinNode(0, 0)
	if got != 0 {
		t.Errorf("findJoinNode(0, 0) = %d, want 0", got)
	}
}
