// SPDX-License-Identifier: BSL-1.0

package mcf

import "testing"

func TestUpdatePotentialsSubtree(t *testing.T) {
	// Build a solverState by hand: n=5, root=5.
	// Tree structure (rooted at artificial root 5):
	//       5
	//      / \
	//     1   0
	//    / \
	//   2   3
	// Node 4 hangs off root as well.
	//
	// Post-order thread: 2 -> 3 -> 1 -> 4 -> 0 -> 5 -> 2 (cyclic)
	// Subtree of node 1 = {1, 2, 3}; lastSucc[1] = 3 (last in thread
	// before leaving the subtree).

	s := &solverState{}
	s.n = 5
	s.root = 5

	// 6 entries: nodes 0..5
	s.thread = make([]int, 6)
	s.lastSucc = make([]int, 6)
	s.pi = make([]int64, 6)

	// Thread order: 2 -> 3 -> 1 -> 4 -> 0 -> 5 -> back to 2
	s.thread[2] = 3
	s.thread[3] = 1
	s.thread[1] = 4
	s.thread[4] = 0
	s.thread[0] = 5
	s.thread[5] = 2

	// Subtree of node 1 contains {1, 2, 3}.
	// In the thread starting at 1 we visit: 1 -> 4 (stop).
	// But we need thread starting at subtreeRoot=1 to cover 2 and 3 first.
	// Re-examine: the traversal is node := subtreeRoot; stop := thread[lastSucc[subtreeRoot]].
	// If subtreeRoot=1 and lastSucc[1]=3, stop = thread[3] = 1.
	// Walk: node=1, 1!=1? No -- that visits nothing.
	//
	// We need the thread to start at subtreeRoot and walk through the
	// subtree nodes. Let's re-order the thread so that starting from 1
	// we reach 2 and 3 before exiting.
	//
	// Correct thread for subtree {1,2,3} rooted at 1:
	// 1 -> 2 -> 3 -> (exit subtree) -> ...
	// lastSucc[1] = 3, so stop = thread[3].
	//
	// Full thread: 1 -> 2 -> 3 -> 4 -> 0 -> 5 -> 1 (cyclic)
	s.thread[1] = 2
	s.thread[2] = 3
	s.thread[3] = 4
	s.thread[4] = 0
	s.thread[0] = 5
	s.thread[5] = 1

	s.lastSucc[1] = 3 // subtree of 1 = {1, 2, 3}, last in thread = 3

	s.pi = []int64{10, 20, 30, 40, 50, 0}

	s.updatePotentials(1, 7)

	// Nodes 1, 2, 3 should each increase by 7.
	wantPi := []int64{10, 27, 37, 47, 50, 0}
	for i, want := range wantPi {
		if s.pi[i] != want {
			t.Errorf("pi[%d] = %d, want %d", i, s.pi[i], want)
		}
	}
}

func TestSubtreeUpdateOnSingletonSubtree(t *testing.T) {
	// Single-node subtree: subtreeRoot == lastSucc[subtreeRoot].
	s := &solverState{}
	s.n = 3
	s.root = 3

	s.thread = make([]int, 4)
	s.lastSucc = make([]int, 4)
	s.pi = make([]int64, 4)

	// Thread: 0 -> 1 -> 2 -> 3 -> 0 (cyclic)
	s.thread[0] = 1
	s.thread[1] = 2
	s.thread[2] = 3
	s.thread[3] = 0

	// Node 2 is a singleton subtree.
	s.lastSucc[2] = 2 // subtreeRoot == lastSucc

	s.pi = []int64{10, 20, 30, 0}

	s.updatePotentials(2, -5)

	wantPi := []int64{10, 20, 25, 0}
	for i, want := range wantPi {
		if s.pi[i] != want {
			t.Errorf("pi[%d] = %d, want %d", i, s.pi[i], want)
		}
	}
}
