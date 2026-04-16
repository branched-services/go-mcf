// SPDX-License-Identifier: BSL-1.0

package mcf

// findJoinNode returns the join node (lowest common ancestor) of u and v in
// the spanning tree. It walks both nodes toward the root using s.parent,
// equalizing depth via s.succNum before the common walk begins.
//
// succNum stores subtree size, which is monotone non-decreasing along the path
// to root, making it a valid depth proxy for LEMON's tree layout.
func (s *solverState) findJoinNode(u, v int) int {
	if u == v {
		return u
	}

	// Advance the deeper node until both are at the same depth level.
	for s.succNum[u] < s.succNum[v] {
		u = s.parent[u]
	}
	for s.succNum[v] < s.succNum[u] {
		v = s.parent[v]
	}

	// Walk both upward in lockstep until they meet.
	for u != v {
		u = s.parent[u]
		v = s.parent[v]
	}

	return u
}
