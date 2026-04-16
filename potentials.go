// SPDX-License-Identifier: BSL-1.0

package mcf

// updatePotentials adds delta to the node potential of every node in the
// subtree rooted at subtreeRoot. The contiguous thread-range
// [subtreeRoot .. lastSucc[subtreeRoot]] is exactly the subtree in
// post-order, so a simple walk along s.thread covers every subtree node
// without allocations.
func (s *solverState) updatePotentials(subtreeRoot int, delta int64) {
	stop := s.thread[s.lastSucc[subtreeRoot]]
	for node := subtreeRoot; node != stop; node = s.thread[node] {
		s.pi[node] += delta
	}
}
