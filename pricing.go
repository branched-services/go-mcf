// SPDX-License-Identifier: BSL-1.0

package mcf

import "math"

const sentinelPi = math.MaxInt64 / 2

func reducedCost(s *solver, arcIdx int) int64 {
	a := s.arc(arcIdx)
	return a.Cost - s.pi[a.From] + s.pi[a.To]
}

func (s *solver) selectEntering() int {
	totalArcs := len(s.arcs) + len(s.artArcs)
	numBlocks := (totalArcs + s.blockSize - 1) / s.blockSize

	for scanned := 0; scanned < numBlocks; scanned++ {
		b := (s.nextBlock + scanned) % numBlocks
		lo := b * s.blockSize
		hi := lo + s.blockSize
		if hi > totalArcs {
			hi = totalArcs
		}

		bestArc := -1
		bestViolation := int64(0)

		for i := lo; i < hi; i++ {
			st := s.state[i]
			if st == stateTree {
				continue
			}

			a := s.arc(i)
			if s.pi[a.From] >= sentinelPi || s.pi[a.From] <= -sentinelPi ||
				s.pi[a.To] >= sentinelPi || s.pi[a.To] <= -sentinelPi {
				continue
			}

			rc := reducedCost(s, i)

			var violation int64
			switch st {
			case stateLower:
				violation = -rc
			case stateUpper:
				violation = rc
			default:
				continue
			}

			if violation > bestViolation {
				bestViolation = violation
				bestArc = i
			}
		}

		if bestArc >= 0 {
			s.nextBlock = (b + 1) % numBlocks
			return bestArc
		}
	}

	s.nextBlock = 0
	return -1
}
