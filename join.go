// SPDX-License-Identifier: BSL-1.0

package mcf

func (s *solver) findJoin(u, v int) int {
	for u != v {
		if s.succNum[u] < s.succNum[v] {
			u = s.parent[u]
		} else {
			v = s.parent[v]
		}
	}
	return u
}
