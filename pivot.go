// SPDX-License-Identifier: BSL-1.0

package mcf

import "github.com/holiman/uint256"

func (s *solver) pushFlow(enterArc, join int, bottleneck *uint256.Int) {
	if bottleneck.IsZero() {
		return
	}

	a := s.arc(enterArc)
	var first, second int
	if s.state[enterArc] == stateLower {
		first, second = a.From, a.To
	} else {
		first, second = a.To, a.From
	}

	for u := first; u != join; u = s.parent[u] {
		ea := s.arc(s.predArc[u])
		if s.direction[u] == directionUp {
			ea.Flow.Add(ea.Flow, bottleneck)
		} else {
			ea.Flow.Sub(ea.Flow, bottleneck)
		}
	}

	for u := second; u != join; u = s.parent[u] {
		ea := s.arc(s.predArc[u])
		if s.direction[u] == directionDown {
			ea.Flow.Add(ea.Flow, bottleneck)
		} else {
			ea.Flow.Sub(ea.Flow, bottleneck)
		}
	}

	if s.state[enterArc] == stateLower {
		a.Flow.Add(a.Flow, bottleneck)
	} else {
		a.Flow.Sub(a.Flow, bottleneck)
	}
}

func (s *solver) updateTree(enterArc, leaveArc, join int) int {
	ea := s.arc(enterArc)
	var first, second int
	if s.state[enterArc] == stateLower {
		first, second = ea.From, ea.To
	} else {
		first, second = ea.To, ea.From
	}

	var uOut int
	onFirstSide := false
	for u := first; u != join; u = s.parent[u] {
		if s.predArc[u] == leaveArc {
			uOut = u
			onFirstSide = true
			break
		}
	}
	if !onFirstSide {
		for u := second; u != join; u = s.parent[u] {
			if s.predArc[u] == leaveArc {
				uOut = u
				break
			}
		}
	}

	var stemNode, otherNode int
	if onFirstSide {
		stemNode, otherNode = first, second
	} else {
		stemNode, otherNode = second, first
	}

	oldPar := s.parent[stemNode]
	oldPred := s.predArc[stemNode]
	oldDir := s.direction[stemNode]

	s.parent[stemNode] = otherNode
	s.predArc[stemNode] = enterArc
	if ea.From == stemNode {
		s.direction[stemNode] = directionUp
	} else {
		s.direction[stemNode] = directionDown
	}

	prev := stemNode
	for prev != uOut {
		cur := oldPar
		nextPar := s.parent[cur]
		nextPred := s.predArc[cur]
		nextDir := s.direction[cur]

		s.parent[cur] = prev
		s.predArc[cur] = oldPred
		if oldDir == directionUp {
			s.direction[cur] = directionDown
		} else {
			s.direction[cur] = directionUp
		}

		prev = cur
		oldPar = nextPar
		oldPred = nextPred
		oldDir = nextDir
	}

	s.state[enterArc] = stateTree

	la := s.arc(leaveArc)
	if la.Flow.IsZero() {
		s.state[leaveArc] = stateLower
	} else {
		s.state[leaveArc] = stateUpper
	}

	s.rebuildDFS()

	return stemNode
}

func (s *solver) rebuildDFS() {
	root := s.n
	total := root + 1

	for i := 0; i < total; i++ {
		s.childHead[i] = -1
	}
	for i := total - 1; i >= 0; i-- {
		p := s.parent[i]
		if p >= 0 {
			s.childNext[i] = s.childHead[p]
			s.childHead[p] = i
		}
	}

	s.succNum[root] = 1
	s.lastSucc[root] = root
	prev := root
	v := s.childHead[root]

	for v >= 0 {
		s.thread[prev] = v
		s.revThread[v] = prev
		s.succNum[v] = 1
		prev = v

		if s.childHead[v] >= 0 {
			v = s.childHead[v]
		} else {
			s.lastSucc[v] = v
			for {
				p := s.parent[v]
				s.succNum[p] += s.succNum[v]
				s.lastSucc[p] = s.lastSucc[v]

				ns := s.childNext[v]
				if ns >= 0 {
					v = ns
					break
				}
				v = p
				if v == root {
					v = -1
					break
				}
			}
		}
	}

	s.thread[prev] = root
	s.revThread[root] = prev
}

func (s *solver) updatePotentials(enterArc, subtreeRoot int) {
	ea := s.arc(enterArc)
	rc := reducedCost(s, enterArc)

	var delta int64
	if subtreeRoot == ea.From {
		delta = rc
	} else {
		delta = -rc
	}

	if delta == 0 {
		return
	}

	last := s.lastSucc[subtreeRoot]
	v := subtreeRoot
	for {
		s.pi[v] += delta
		if v == last {
			break
		}
		v = s.thread[v]
	}
}

func (s *solver) pivot(enterArc, leaveArc, join int, bottleneck *uint256.Int) {
	if bottleneck.IsZero() && leaveArc == enterArc {
		if s.state[enterArc] == stateLower {
			s.state[enterArc] = stateUpper
		} else {
			s.state[enterArc] = stateLower
		}
		return
	}

	s.pushFlow(enterArc, join, bottleneck)
	subtreeRoot := s.updateTree(enterArc, leaveArc, join)
	s.updatePotentials(enterArc, subtreeRoot)
}
