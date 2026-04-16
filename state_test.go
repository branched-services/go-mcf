// SPDX-License-Identifier: BSL-1.0

package mcf

import "testing"

func TestNewSolverStateAllocations(t *testing.T) {
	const n, mReal = 5, 7
	s := newSolverState(n, mReal)

	if s.m != 12 {
		t.Fatalf("m = %d, want 12", s.m)
	}
	if s.root != 5 {
		t.Fatalf("root = %d, want 5", s.root)
	}

	// Arc-indexed slices must have length m.
	arcSlices := map[string]int{
		"source":   len(s.source),
		"target":   len(s.target),
		"cost":     len(s.cost),
		"capacity": len(s.capacity),
		"flow":     len(s.flow),
		"state":    len(s.state),
	}
	for name, got := range arcSlices {
		if got != s.m {
			t.Errorf("len(%s) = %d, want %d", name, got, s.m)
		}
	}

	// Node-indexed slices must have length n+1.
	nodes := n + 1
	nodeSlices := map[string]int{
		"supply":    len(s.supply),
		"pi":        len(s.pi),
		"parent":    len(s.parent),
		"predArc":   len(s.predArc),
		"thread":    len(s.thread),
		"revThread": len(s.revThread),
		"succNum":   len(s.succNum),
		"lastSucc":  len(s.lastSucc),
		"dirNode":   len(s.dirNode),
	}
	for name, got := range nodeSlices {
		if got != nodes {
			t.Errorf("len(%s) = %d, want %d", name, got, nodes)
		}
	}

	// Every flow entry must be non-nil and zero.
	for i := 0; i < s.m; i++ {
		if s.flow[i] == nil {
			t.Fatalf("flow[%d] is nil", i)
		}
		if !s.flow[i].IsZero() {
			t.Errorf("flow[%d] = %s, want 0", i, s.flow[i])
		}
	}

	// Scratch buffers must be non-nil.
	if s.bottleneck == nil {
		t.Fatal("bottleneck is nil")
	}
	if s.delta == nil {
		t.Fatal("delta is nil")
	}
}
