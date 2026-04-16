// SPDX-License-Identifier: BSL-1.0
package mcf

import (
	"math"
	"testing"
)

func TestBigMMonotone(t *testing.T) {
	m2 := bigM(2)
	m1000 := bigM(1000)
	if m2 <= m1000 {
		t.Fatalf("bigM(2)=%d should be > bigM(1000)=%d", m2, m1000)
	}
	if m1000 <= 0 {
		t.Fatalf("bigM(1000)=%d should be > 0", m1000)
	}
}

func TestArcCostBoundAccepts(t *testing.T) {
	if !arcCostWithinBound(1_000_000, 100) {
		t.Fatal("cost 1_000_000 with n=100 should be within bound")
	}
}

func TestArcCostBoundRejects(t *testing.T) {
	if arcCostWithinBound(math.MaxInt64, 100) {
		t.Fatal("cost MaxInt64 should be rejected")
	}
}

func TestArcCostBoundRejectsMinInt64(t *testing.T) {
	if arcCostWithinBound(math.MinInt64, 100) {
		t.Fatal("cost MinInt64 should be rejected")
	}
}

func TestBigMDominatesRealCost(t *testing.T) {
	const n = 1000
	m := bigM(n)
	// For any cost accepted by the bound check, (n+1)*|cost| < bigM(n).
	for _, c := range []int64{0, 1, -1, 1_000_000, -1_000_000} {
		if !arcCostWithinBound(c, n) {
			continue
		}
		absC := c
		if absC < 0 {
			absC = -absC
		}
		product := int64(n+1) * absC
		if product >= m {
			t.Fatalf("(n+1)*|%d| = %d >= bigM(%d) = %d", c, product, n, m)
		}
	}
}
