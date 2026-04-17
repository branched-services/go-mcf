// SPDX-License-Identifier: BSL-1.0

package mcf

import "github.com/holiman/uint256"

func (s *solver) findLeaving(enterArc, join int) (int, *uint256.Int, *uint256.Int) {
	a := s.arc(enterArc)
	var first, second int
	if s.state[enterArc] == stateLower {
		first, second = a.From, a.To
	} else {
		first, second = a.To, a.From
	}

	s.deltaFirst.SetAllOne()
	s.deltaSecond.SetAllOne()

	if s.state[enterArc] == stateLower {
		s.bottleneck.Sub(a.Capacity, a.Flow)
	} else {
		s.bottleneck.Set(a.Flow)
	}
	leavingArc := enterArc

	for u := first; u != join; u = s.parent[u] {
		e := s.predArc[u]
		ea := s.arc(e)
		if s.direction[u] == directionUp {
			s.scratch.Sub(ea.Capacity, ea.Flow)
		} else {
			s.scratch.Set(ea.Flow)
		}
		if s.scratch.Cmp(s.deltaFirst) <= 0 {
			s.deltaFirst.Set(s.scratch)
		}
		if s.scratch.Lt(s.bottleneck) {
			s.bottleneck.Set(s.scratch)
			leavingArc = e
		}
	}

	// This tie-break maintains the strongly-feasible-tree invariant.
	// Do not remove: Bland's rule alone cannot prevent cycling in
	// degenerate pivots. The <= comparison ensures that ties are broken
	// in favor of the arc closest to the join node on this (second) side,
	// which is the side opposite the entering arc's flow orientation.
	// This is the real anti-cycling mechanism for the Network Simplex
	// algorithm under degeneracy.
	for u := second; u != join; u = s.parent[u] {
		e := s.predArc[u]
		ea := s.arc(e)
		if s.direction[u] == directionUp {
			s.scratch.Set(ea.Flow)
		} else {
			s.scratch.Sub(ea.Capacity, ea.Flow)
		}
		if s.scratch.Cmp(s.deltaSecond) <= 0 {
			s.deltaSecond.Set(s.scratch)
		}
		if s.scratch.Cmp(s.bottleneck) <= 0 {
			s.bottleneck.Set(s.scratch)
			leavingArc = e
		}
	}

	return leavingArc, s.deltaFirst, s.deltaSecond
}
