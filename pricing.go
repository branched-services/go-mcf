// SPDX-License-Identifier: BSL-1.0

package mcf

// findEnteringArc uses block-search pricing to select the next entering arc.
//
// It scans blocks of s.blockSize non-tree arcs starting at s.nextArc,
// wrapping around [0, s.m). Within each block it tracks the arc with the
// maximum violation; only arcs with violation > 0 are eligible. After at
// most ceil(s.m / s.blockSize) blocks (one full pass) it either returns the
// best arc found or signals optimality with ok == false.
//
// The cursor s.nextArc is always advanced so that successive calls rotate
// through the arc space. No heap allocations.
func (s *solverState) findEnteringArc() (arc int, ok bool) {
	blocks := (s.m + s.blockSize - 1) / s.blockSize // ceil(m / blockSize)

	bestArc := -1

	cursor := s.nextArc
	for range blocks {
		blockBest := -1
		var blockBestViol int64

		for range s.blockSize {
			a := cursor % s.m
			cursor++
			v := s.violation(a)
			if v > blockBestViol {
				blockBestViol = v
				blockBest = a
			}
		}

		// Advance nextArc past the scanned block.
		s.nextArc = cursor % s.m

		if blockBest >= 0 {
			bestArc = blockBest
			break
		}
	}

	if bestArc < 0 {
		return 0, false
	}
	return bestArc, true
}
